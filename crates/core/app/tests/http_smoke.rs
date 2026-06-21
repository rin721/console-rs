use app::app::App;
use app::config::{NotificationDriver, Settings};
use axum::body::{Body, to_bytes};
use chrono::{Duration, Utc};
use futures_util::StreamExt;
use http::header::{AUTHORIZATION, CONTENT_DISPOSITION, CONTENT_TYPE, COOKIE, SET_COOKIE};
use http::{HeaderMap, Request, StatusCode};
use serde_json::json;
use sqlx::Row;
use std::{
    collections::{BTreeSet, HashSet},
    fs,
    path::PathBuf,
};
use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::net::TcpListener;
use tower::ServiceExt;
use uuid::Uuid;

use app::transport::http::route_registry;

fn test_settings() -> Settings {
    let mut settings = Settings::default();
    settings.database.url = format!("sqlite://target/test-dbs/{}.sqlite", Uuid::new_v4());
    settings.database.max_connections = 1;
    settings.storage.local_dir = test_temp_dir("media").to_string_lossy().into_owned();
    settings.notification.local_dir = test_temp_dir("notifications")
        .to_string_lossy()
        .into_owned();
    settings
}

fn test_settings_with_webui() -> Settings {
    let mut settings = test_settings();
    let dist_dir = webui_fixture_dir();
    let assets_dir = dist_dir.join("assets");
    fs::create_dir_all(&assets_dir).expect("create webui test dir");
    fs::write(
        dist_dir.join("index.html"),
        "<!doctype html><title>Aoi WebUI</title><main id=\"root\">Aoi WebUI Shell</main>",
    )
    .expect("write index");
    fs::write(assets_dir.join("app.js"), "console.log('console-webui');").expect("write asset");
    settings.webui.enabled = true;
    settings.webui.dist_dir = dist_dir.to_string_lossy().into_owned();
    settings
}

fn webui_fixture_dir() -> PathBuf {
    test_temp_dir("webui").join(Uuid::new_v4().to_string())
}

fn test_temp_dir(kind: &str) -> PathBuf {
    let base = std::env::var_os("CARGO_TARGET_TMPDIR")
        .map(PathBuf::from)
        .unwrap_or_else(|| std::env::temp_dir().join("console-tests"));
    base.join(kind).join(Uuid::new_v4().to_string())
}

async fn test_router() -> (axum::Router, String) {
    let (app, db_url, _) = test_app().await;
    (app.router(), db_url)
}

async fn test_app() -> (App, String, PathBuf) {
    let settings = test_settings();
    let db_url = settings.database.url.clone();
    let notification_dir = PathBuf::from(&settings.notification.local_dir);
    let app = App::boot(settings).await.expect("boot app");
    (app, db_url, notification_dir)
}

async fn spawn_probe_http_target(status_line: &'static str) -> String {
    spawn_probe_http_target_sequence(vec![status_line]).await
}

async fn spawn_probe_http_target_sequence(status_lines: Vec<&'static str>) -> String {
    let listener = TcpListener::bind("127.0.0.1:0")
        .await
        .expect("bind probe target");
    let addr = listener.local_addr().expect("probe target addr");
    tokio::spawn(async move {
        for status_line in status_lines {
            let Ok((mut socket, _)) = listener.accept().await else {
                break;
            };
            let mut request = [0_u8; 1024];
            let _ = socket.read(&mut request).await;
            let response =
                format!("HTTP/1.1 {status_line}\r\nContent-Length: 0\r\nConnection: close\r\n\r\n");
            let _ = socket.write_all(response.as_bytes()).await;
        }
    });
    format!("http://{addr}/health")
}

fn set_cookie_headers(headers: &HeaderMap) -> Vec<String> {
    headers
        .get_all(SET_COOKIE)
        .iter()
        .filter_map(|value| value.to_str().ok())
        .map(ToOwned::to_owned)
        .collect()
}

fn cookie_header_from_set_cookie(headers: &HeaderMap) -> String {
    set_cookie_headers(headers)
        .into_iter()
        .filter_map(|cookie| cookie.split(';').next().map(ToOwned::to_owned))
        .collect::<Vec<_>>()
        .join("; ")
}

fn cookie_pair_by_name(headers: &HeaderMap, name: &str) -> String {
    set_cookie_headers(headers)
        .into_iter()
        .filter_map(|cookie| cookie.split(';').next().map(ToOwned::to_owned))
        .find(|pair| pair.starts_with(&format!("{name}=")))
        .unwrap_or_else(|| panic!("{name} Set-Cookie header"))
}

fn cookie_pair_value(pair: &str) -> String {
    pair.split_once('=')
        .map(|(_, value)| value.to_string())
        .expect("cookie pair value")
}

#[tokio::test]
async fn health_ready_and_openapi_are_available() {
    let app = App::boot(test_settings()).await.expect("boot app");
    let router = app.router();

    let health = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/health")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(health.status(), StatusCode::OK);

    let ready = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/ready")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(ready.status(), StatusCode::OK);

    let openapi = router
        .oneshot(
            Request::builder()
                .uri("/openapi.yaml")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(openapi.status(), StatusCode::OK);
    let body = body_text(openapi).await;
    assert!(body.contains("/api/v1/system/apis"));
    assert!(!body.to_ascii_lowercase().contains("plugin"));
}

#[tokio::test]
async fn route_registry_syncs_runtime_api_catalog_and_permissions() {
    let settings = test_settings();
    let db_url = settings.database.url.clone();
    let expected_contracts = route_registry::contracts(&settings);
    let expected_catalog: BTreeSet<_> = expected_contracts
        .iter()
        .filter(|contract| contract.include_catalog)
        .map(|contract| {
            (
                contract.id.clone(),
                contract.method.clone(),
                contract.path.clone(),
                contract.tag.clone(),
                contract.summary.clone(),
                contract.access.clone(),
                contract.permission.clone(),
                contract.scope.clone(),
                contract.product_code.clone(),
            )
        })
        .collect();
    let expected_permissions: BTreeSet<_> = expected_contracts
        .iter()
        .filter_map(|contract| {
            contract.permission.as_ref().map(|permission| {
                (
                    contract.product_code.clone(),
                    contract.scope.clone(),
                    permission.clone(),
                )
            })
        })
        .collect();

    App::boot(settings).await.expect("boot app");
    let pool = sqlx::SqlitePool::connect(&db_url).await.unwrap();

    let catalog_rows = sqlx::query(
        "select id, method, path, tag, summary, access, permission, scope, product_code
         from system_apis",
    )
    .fetch_all(&pool)
    .await
    .unwrap();
    let actual_catalog: BTreeSet<_> = catalog_rows
        .into_iter()
        .map(|row| {
            (
                row.get::<String, _>("id"),
                row.get::<String, _>("method"),
                row.get::<String, _>("path"),
                row.get::<String, _>("tag"),
                row.get::<String, _>("summary"),
                row.get::<String, _>("access"),
                row.get::<Option<String>, _>("permission"),
                row.get::<String, _>("scope"),
                row.get::<String, _>("product_code"),
            )
        })
        .collect();
    assert_eq!(actual_catalog, expected_catalog);

    let permission_rows = sqlx::query("select product_code, scope, code from iam_permissions")
        .fetch_all(&pool)
        .await
        .unwrap();
    let actual_permissions: BTreeSet<_> = permission_rows
        .into_iter()
        .map(|row| {
            (
                row.get::<String, _>("product_code"),
                row.get::<String, _>("scope"),
                row.get::<String, _>("code"),
            )
        })
        .collect();
    assert_eq!(actual_permissions, expected_permissions);
}

#[tokio::test]
async fn webui_static_hosting_does_not_swallow_api_or_probe_routes() {
    let app = App::boot(test_settings_with_webui())
        .await
        .expect("boot app");
    let router = app.router();

    let root = router
        .clone()
        .oneshot(Request::builder().uri("/").body(Body::empty()).unwrap())
        .await
        .unwrap();
    assert_eq!(root.status(), StatusCode::OK);
    assert_eq!(
        root.headers()
            .get(CONTENT_TYPE)
            .and_then(|value| value.to_str().ok()),
        Some("text/html; charset=utf-8")
    );
    assert!(body_text(root).await.contains("Aoi WebUI Shell"));

    let admin = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/admin/iam")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(admin.status(), StatusCode::OK);
    assert!(body_text(admin).await.contains("Aoi WebUI Shell"));

    let asset = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/assets/app.js")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(asset.status(), StatusCode::OK);
    assert_eq!(
        asset
            .headers()
            .get(CONTENT_TYPE)
            .and_then(|value| value.to_str().ok()),
        Some("text/javascript; charset=utf-8")
    );
    assert!(body_text(asset).await.contains("console-webui"));

    let missing_api = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/does-not-exist")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(missing_api.status(), StatusCode::NOT_FOUND);
    assert!(!body_text(missing_api).await.contains("Aoi WebUI Shell"));

    let public_settings = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/public-settings")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(public_settings.status(), StatusCode::OK);
    assert!(body_text(public_settings).await.contains("console"));

    let health = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/health")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(health.status(), StatusCode::OK);

    let openapi = router
        .oneshot(
            Request::builder()
                .uri("/openapi.yaml")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(openapi.status(), StatusCode::OK);
    assert!(!body_text(openapi).await.contains("Aoi WebUI Shell"));
}

#[tokio::test]
async fn setup_config_checks_are_structured_and_do_not_expose_secrets() {
    let app = App::boot(test_settings()).await.expect("boot app");
    let router = app.router();

    let checks = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/setup/config-checks")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(checks.status(), StatusCode::OK);
    let body = body_text(checks).await;
    assert!(body.contains("\"ready\":true"));
    assert!(body.contains("\"key\":\"database\""));
    assert!(body.contains("\"key\":\"secrets\""));
    assert!(body.contains("\"status\":\"warning\""));
    assert!(!body.contains("dev-session-secret-change-me-32-bytes"));
    assert!(!body.contains("dev-mfa-secret-change-me-32-bytes"));
    assert!(!body.contains("dev-notification-secret-change-me-32-bytes"));

    let run = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/setup/runs")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({"reason": "config-check-test"}).to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(run.status(), StatusCode::OK);
    let run_body = body_text(run).await;
    let run_value: serde_json::Value = serde_json::from_str(&run_body).unwrap();
    let run_id = run_value["id"].as_str().expect("run id");
    assert_eq!(run_value["reason"].as_str(), Some("config-check-test"));
    assert!(run_value["updated_at"].as_str().is_some());

    let runs = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/setup/runs")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(runs.status(), StatusCode::OK);
    let runs_body = body_text(runs).await;
    assert!(runs_body.contains(run_id));
    assert!(runs_body.contains("config-check-test"));

    let logs = router
        .clone()
        .oneshot(
            Request::builder()
                .uri(format!("/api/v1/setup/runs/{run_id}/logs"))
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(logs.status(), StatusCode::OK);
    let logs_body = body_text(logs).await;
    assert!(logs_body.contains("\"step_key\":\"config-checks\""));
    assert!(logs_body.contains("配置检测完成"));
}

#[tokio::test]
async fn setup_completion_requires_ready_checks_and_initial_admin() {
    let app = App::boot(test_settings()).await.expect("boot app");
    let router = app.router();

    let initial_status = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/setup/status")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(initial_status.status(), StatusCode::OK);
    let initial_status_value: serde_json::Value =
        serde_json::from_str(&body_text(initial_status).await).unwrap();
    assert_eq!(initial_status_value["completed"].as_bool(), Some(false));
    assert_eq!(
        setup_step_status(&initial_status_value, "database"),
        "ready"
    );
    assert_eq!(
        setup_step_status(&initial_status_value, "config-checks"),
        "ready"
    );
    assert_eq!(
        setup_step_status(&initial_status_value, "complete"),
        "blocked"
    );

    let run = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/setup/runs")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({"reason": "completion-flow-test"}).to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(run.status(), StatusCode::OK);
    let run_value: serde_json::Value = serde_json::from_str(&body_text(run).await).unwrap();
    let run_id = run_value["id"].as_str().expect("setup run id");

    let rejected = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/setup/complete")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({"confirm": true, "run_id": run_id}).to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(rejected.status(), StatusCode::CONFLICT);
    assert!(body_text(rejected).await.contains("必须先创建首个管理员"));

    let _ = create_initial_admin(router.clone()).await;

    let ready_status = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/setup/status")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(ready_status.status(), StatusCode::OK);
    let ready_status_value: serde_json::Value =
        serde_json::from_str(&body_text(ready_status).await).unwrap();
    assert_eq!(
        ready_status_value["has_initial_admin"].as_bool(),
        Some(true)
    );
    assert_eq!(setup_step_status(&ready_status_value, "complete"), "ready");

    let completed = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/setup/complete")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({"confirm": true, "run_id": run_id}).to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(completed.status(), StatusCode::OK);
    assert!(body_text(completed).await.contains("\"completed\":true"));

    let completed_status = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/setup/status")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(completed_status.status(), StatusCode::OK);
    let completed_status_value: serde_json::Value =
        serde_json::from_str(&body_text(completed_status).await).unwrap();
    assert_eq!(completed_status_value["completed"].as_bool(), Some(true));
    assert_eq!(
        setup_step_status(&completed_status_value, "complete"),
        "done"
    );

    let logs = router
        .clone()
        .oneshot(
            Request::builder()
                .uri(format!("/api/v1/setup/runs/{run_id}/logs"))
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(logs.status(), StatusCode::OK);
    let logs_body = body_text(logs).await;
    assert!(logs_body.contains("\"step_key\":\"complete\""));
    assert!(logs_body.contains("初始化完成状态已写入"));

    let runs = router
        .oneshot(
            Request::builder()
                .uri("/api/v1/setup/runs")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(runs.status(), StatusCode::OK);
    let runs_body = body_text(runs).await;
    assert!(runs_body.contains(run_id));
    assert!(runs_body.contains("\"status\":\"completed\""));
}

#[tokio::test]
async fn csrf_cookie_and_header_are_required_when_enabled() {
    let mut settings = test_settings();
    settings.auth.csrf.enabled = true;
    let app = App::boot(settings).await.expect("boot csrf app");
    let router = app.router();

    let payload = json!({
        "email": "owner@example.com",
        "password": "change-me-123",
        "display_name": "平台所有者",
        "organization_code": "main",
        "organization_name": "主组织"
    });

    let rejected = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/setup/initial-admin")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(payload.to_string()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(rejected.status(), StatusCode::FORBIDDEN);

    let public_settings = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/public-settings")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(public_settings.status(), StatusCode::OK);
    let csrf_cookie = cookie_pair_by_name(public_settings.headers(), "console_csrf");
    let csrf_set_cookie = set_cookie_headers(public_settings.headers())
        .into_iter()
        .find(|cookie| cookie.starts_with("console_csrf="))
        .expect("csrf set-cookie");
    assert!(!csrf_set_cookie.contains("HttpOnly"));
    assert!(
        body_text(public_settings)
            .await
            .contains("\"csrf_enabled\":true")
    );

    let accepted = router
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/setup/initial-admin")
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &csrf_cookie)
                .header("X-CSRF-Token", cookie_pair_value(&csrf_cookie))
                .body(Body::from(payload.to_string()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(accepted.status(), StatusCode::OK);
    let set_cookies = set_cookie_headers(accepted.headers());
    assert!(
        set_cookies
            .iter()
            .any(|cookie| cookie.starts_with("console_session="))
    );
}

#[tokio::test]
async fn initial_admin_creates_http_only_session_token_without_body_token_leak() {
    let (router, _) = test_router().await;

    let payload = json!({
        "email": "owner@example.com",
        "password": "change-me-123",
        "display_name": "平台所有者",
        "organization_code": "main",
        "organization_name": "主组织"
    });
    let response = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/setup/initial-admin")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(payload.to_string()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(response.status(), StatusCode::OK);
    let set_cookies = set_cookie_headers(response.headers());
    assert!(
        set_cookies
            .iter()
            .any(|cookie| cookie.starts_with("console_session="))
    );
    assert!(
        set_cookies
            .iter()
            .any(|cookie| cookie.starts_with("console_refresh="))
    );
    assert!(set_cookies.iter().all(|cookie| cookie.contains("HttpOnly")));
    let cookie_pair = cookie_header_from_set_cookie(response.headers());
    let body = body_text(response).await;
    assert!(body.contains("owner@example.com"));
    assert!(!body.contains("session_token_"));
    assert!(!body.contains("refresh_token_"));

    let session = router
        .oneshot(
            Request::builder()
                .uri("/api/v1/me/session")
                .header(COOKIE, cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(session.status(), StatusCode::OK);
    let body = body_text(session).await;
    assert!(body.contains("\"authenticated\":true"));
    assert!(body.contains("api_token:create"));
    assert!(body.contains("config:read"));
    assert!(body.contains("config:write"));
    assert!(body.contains("dictionary:read"));
    assert!(body.contains("dictionary:write"));
    assert!(body.contains("media:read"));
    assert!(body.contains("media:write"));
    assert!(body.contains("menu:read"));
    assert!(body.contains("operation_record:read"));
    assert!(body.contains("parameter:read"));
    assert!(body.contains("parameter:write"));
    assert!(body.contains("permission:read"));
    assert!(body.contains("server:read"));
    assert!(body.contains("traffic_probe:read"));
    assert!(body.contains("traffic_probe:write"));
    assert!(body.contains("version_package:read"));
    assert!(body.contains("version_package:write"));
}

#[tokio::test]
async fn refresh_token_cookie_is_hashed_rotated_and_revoked() {
    let (router, db_url) = test_router().await;
    let (cookie_pair, _) = create_initial_admin(router.clone()).await;
    let session_token_pair = cookie_pair
        .split("; ")
        .find(|pair| pair.starts_with("console_session="))
        .expect("session cookie pair")
        .to_string();
    let refresh_token_pair = cookie_pair
        .split("; ")
        .find(|pair| pair.starts_with("console_refresh="))
        .expect("refresh cookie pair")
        .to_string();
    let raw_session = session_token_pair
        .strip_prefix("console_session=")
        .expect("raw session");
    let raw_refresh = refresh_token_pair
        .strip_prefix("console_refresh=")
        .expect("raw refresh");

    let pool = sqlx::SqlitePool::connect(&db_url).await.unwrap();
    let row = sqlx::query(
        "select session_token_hash, refresh_token_hash, refresh_expires_at from iam_sessions limit 1",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    let session_token_hash: String = row.get("session_token_hash");
    let refresh_token_hash: String = row.get("refresh_token_hash");
    let refresh_token_expires_at: String = row.get("refresh_expires_at");
    assert_ne!(session_token_hash, raw_session);
    assert_ne!(refresh_token_hash, raw_refresh);
    assert!(!refresh_token_expires_at.is_empty());

    let refreshed = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/refresh")
                .header(COOKIE, &refresh_token_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(refreshed.status(), StatusCode::OK);
    let refreshed_cookie = cookie_header_from_set_cookie(refreshed.headers());
    let new_refresh_token_pair = cookie_pair_by_name(refreshed.headers(), "console_refresh");
    assert_ne!(new_refresh_token_pair, refresh_token_pair);
    let refresh_token_body = body_text(refreshed).await;
    assert!(refresh_token_body.contains("\"authenticated\":true"));
    assert!(!refresh_token_body.contains("session_token_"));
    assert!(!refresh_token_body.contains("refresh_token_"));

    let replay = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/refresh")
                .header(COOKIE, &refresh_token_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(replay.status(), StatusCode::UNAUTHORIZED);

    let session = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/me/session")
                .header(COOKIE, &refreshed_cookie)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(session.status(), StatusCode::OK);

    let logout = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/logout")
                .header(COOKIE, &refreshed_cookie)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(logout.status(), StatusCode::OK);
    let cleared = set_cookie_headers(logout.headers());
    assert!(cleared.iter().all(|cookie| cookie.contains("Max-Age=0")));

    let after_logout = router
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/refresh")
                .header(COOKIE, new_refresh_token_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(after_logout.status(), StatusCode::UNAUTHORIZED);

    let leaked_audit_count: i64 =
        sqlx::query_scalar("select count(*) from iam_audit_logs where detail like '%' || ? || '%'")
            .bind(raw_refresh)
            .fetch_one(&pool)
            .await
            .unwrap();
    assert_eq!(leaked_audit_count, 0);
}

#[tokio::test]
async fn api_token_is_returned_once_and_stored_as_hash() {
    let (router, db_url) = test_router().await;
    let (cookie_pair, org_id) = create_initial_admin(router.clone()).await;

    let response = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!("/api/v1/orgs/{org_id}/api-tokens"))
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(json!({"expires_in_days": 7}).to_string()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(response.status(), StatusCode::OK);
    let body = body_text(response).await;
    let value: serde_json::Value = serde_json::from_str(&body).unwrap();
    let token = value["token"].as_str().expect("raw api token");
    let prefix = value["item"]["token_prefix"]
        .as_str()
        .expect("token prefix");
    assert!(token.starts_with("api_token_"));
    assert!(token.starts_with(prefix));

    let pool = sqlx::SqlitePool::connect(&db_url).await.unwrap();
    let row = sqlx::query("select token_hash, token_prefix from iam_api_tokens limit 1")
        .fetch_one(&pool)
        .await
        .unwrap();
    let stored_hash: String = row.get("token_hash");
    let stored_prefix: String = row.get("token_prefix");
    assert_ne!(stored_hash, token);
    assert_eq!(stored_prefix, prefix);

    let list = router
        .oneshot(
            Request::builder()
                .uri(format!("/api/v1/orgs/{org_id}/api-tokens"))
                .header(COOKIE, cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(list.status(), StatusCode::OK);
    let list_body = body_text(list).await;
    assert!(!list_body.contains(token));
    assert!(list_body.contains(prefix));
}

#[tokio::test]
async fn totp_mfa_is_encrypted_verified_and_required_on_login() {
    let (router, db_url) = test_router().await;
    let (cookie_pair, org_id) = create_initial_admin(router.clone()).await;

    let token = create_api_token(router.clone(), &cookie_pair, org_id).await;
    let token_setup = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/mfa/setup")
                .header(AUTHORIZATION, format!("Bearer {token}"))
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(token_setup.status(), StatusCode::FORBIDDEN);

    let token_list = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/auth/mfa/factors")
                .header(AUTHORIZATION, format!("Bearer {token}"))
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(token_list.status(), StatusCode::FORBIDDEN);

    let setup = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/mfa/setup")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(setup.status(), StatusCode::OK);
    let setup_body = body_text(setup).await;
    let setup_value: serde_json::Value = serde_json::from_str(&setup_body).unwrap();
    let factor_id = setup_value["factor"]["id"].as_i64().expect("factor id");
    let secret = setup_value["secret"].as_str().expect("mfa secret");
    assert!(
        setup_value["otpauth_url"]
            .as_str()
            .unwrap()
            .starts_with("otpauth://totp/")
    );
    assert_eq!(setup_value["factor"]["status"], "pending");

    let pending_list = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/auth/mfa/factors")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(pending_list.status(), StatusCode::OK);
    let pending_list_body = body_text(pending_list).await;
    assert!(pending_list_body.contains("\"status\":\"pending\""));
    assert!(pending_list_body.contains(&factor_id.to_string()));
    assert!(!pending_list_body.contains(secret));
    assert!(!pending_list_body.contains("secret_ciphertext"));

    let pool = sqlx::SqlitePool::connect(&db_url).await.unwrap();
    let stored_ciphertext: String =
        sqlx::query_scalar("select secret_ciphertext from iam_mfa_factors where id = ?")
            .bind(factor_id)
            .fetch_one(&pool)
            .await
            .unwrap();
    assert_ne!(stored_ciphertext, secret);
    assert!(!stored_ciphertext.contains(secret));

    let me_before = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/me/session")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(me_before.status(), StatusCode::OK);
    let me_before_value: serde_json::Value =
        serde_json::from_str(&body_text(me_before).await).unwrap();
    assert_eq!(me_before_value["mfa_enabled"], false);

    let wrong_verify = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/mfa/verify")
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(json!({"code": "000000"}).to_string()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(wrong_verify.status(), StatusCode::UNAUTHORIZED);

    let code = crypto::totp_code_for_now(secret).expect("totp code");
    let verify = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/mfa/verify")
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(json!({ "code": code }).to_string()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(verify.status(), StatusCode::OK);
    let verify_body = body_text(verify).await;
    let verify_value: serde_json::Value = serde_json::from_str(&verify_body).unwrap();
    assert_eq!(verify_value["verified"], true);
    let recovery_codes: Vec<String> = verify_value["recovery_codes"]
        .as_array()
        .expect("mfa recovery codes")
        .iter()
        .map(|code| code.as_str().expect("recovery code").to_string())
        .collect();
    assert_eq!(recovery_codes.len(), 8);
    assert_eq!(
        recovery_codes.iter().collect::<HashSet<_>>().len(),
        recovery_codes.len()
    );
    for recovery_code in &recovery_codes {
        assert!(recovery_code.starts_with("mfa_recovery_code_"));
        assert!(!recovery_code.contains(secret));
        let raw_recovery_hits: i64 = sqlx::query_scalar(
            "select count(*) from iam_mfa_recovery_codes
             where code_hash = ? or code_hash like '%' || ? || '%'",
        )
        .bind(recovery_code)
        .bind(recovery_code)
        .fetch_one(&pool)
        .await
        .unwrap();
        assert_eq!(raw_recovery_hits, 0);
    }
    let active_recovery_count: i64 =
        sqlx::query_scalar("select count(*) from iam_mfa_recovery_codes where status = 'active'")
            .fetch_one(&pool)
            .await
            .unwrap();
    assert_eq!(active_recovery_count, 8);

    let active_list = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/auth/mfa/factors")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(active_list.status(), StatusCode::OK);
    let active_list_body = body_text(active_list).await;
    assert!(active_list_body.contains("\"status\":\"active\""));
    assert!(!active_list_body.contains(secret));

    let recovery_list = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/auth/mfa/recovery-codes")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(recovery_list.status(), StatusCode::OK);
    let recovery_list_body = body_text(recovery_list).await;
    let recovery_list_value: serde_json::Value = serde_json::from_str(&recovery_list_body).unwrap();
    assert_eq!(recovery_list_value.as_array().unwrap().len(), 8);
    assert!(recovery_list_body.contains("\"status\":\"active\""));
    assert!(recovery_list_body.contains("mfa_recovery_code_"));
    for recovery_code in &recovery_codes {
        assert!(!recovery_list_body.contains(recovery_code));
    }

    let login_without_mfa = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/login")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({"identifier": "owner@example.com", "password": "change-me-123"})
                        .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(login_without_mfa.status(), StatusCode::UNAUTHORIZED);
    assert!(body_text(login_without_mfa).await.contains("MFA_REQUIRED"));

    let first_recovery_code = recovery_codes.first().unwrap().clone();
    let login_with_recovery_code = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/login")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({
                        "identifier": "owner@example.com",
                        "password": "change-me-123",
                        "mfaCode": first_recovery_code
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(login_with_recovery_code.status(), StatusCode::OK);
    let recovery_login_cookie = cookie_header_from_set_cookie(login_with_recovery_code.headers());
    let recovery_login_body = body_text(login_with_recovery_code).await;
    assert!(recovery_login_body.contains("\"mfa_enabled\":true"));
    assert!(!recovery_login_body.contains(recovery_codes.first().unwrap()));

    let reuse_recovery_code = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/login")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({
                        "identifier": "owner@example.com",
                        "password": "change-me-123",
                        "mfaCode": recovery_codes.first().unwrap()
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(reuse_recovery_code.status(), StatusCode::UNAUTHORIZED);

    let recovery_list_after_use = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/auth/mfa/recovery-codes")
                .header(COOKIE, &recovery_login_cookie)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(recovery_list_after_use.status(), StatusCode::OK);
    assert!(
        body_text(recovery_list_after_use)
            .await
            .contains("\"status\":\"used\"")
    );

    let rotate_recovery_codes = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/mfa/recovery-codes")
                .header(COOKIE, &recovery_login_cookie)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(rotate_recovery_codes.status(), StatusCode::OK);
    let rotate_body = body_text(rotate_recovery_codes).await;
    let rotate_value: serde_json::Value = serde_json::from_str(&rotate_body).unwrap();
    assert_eq!(rotate_value["items"].as_array().unwrap().len(), 8);
    let rotated_codes: Vec<String> = rotate_value["recovery_codes"]
        .as_array()
        .expect("rotated recovery codes")
        .iter()
        .map(|code| code.as_str().expect("rotated code").to_string())
        .collect();
    assert_eq!(rotated_codes.len(), 8);
    for old_code in &recovery_codes {
        assert!(!rotate_body.contains(old_code));
        assert!(!rotated_codes.contains(old_code));
    }
    let rotated_active_count: i64 =
        sqlx::query_scalar("select count(*) from iam_mfa_recovery_codes where status = 'active'")
            .fetch_one(&pool)
            .await
            .unwrap();
    assert_eq!(rotated_active_count, 8);

    let old_recovery_after_rotation = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/login")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({
                        "identifier": "owner@example.com",
                        "password": "change-me-123",
                        "mfaCode": recovery_codes.get(1).unwrap()
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(
        old_recovery_after_rotation.status(),
        StatusCode::UNAUTHORIZED
    );

    let code = crypto::totp_code_for_now(secret).expect("totp code");
    let login_with_mfa = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/login")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({
                        "identifier": "owner@example.com",
                        "password": "change-me-123",
                        "mfaCode": code
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(login_with_mfa.status(), StatusCode::OK);
    let login_cookie = cookie_header_from_set_cookie(login_with_mfa.headers());
    let login_body = body_text(login_with_mfa).await;
    assert!(login_body.contains("\"mfa_enabled\":true"));
    assert!(!login_body.contains(secret));
    for rotated_code in &rotated_codes {
        assert!(!login_body.contains(rotated_code));
    }

    let revoke = router
        .clone()
        .oneshot(
            Request::builder()
                .method("DELETE")
                .uri(format!("/api/v1/auth/mfa/factors/{factor_id}"))
                .header(COOKIE, &login_cookie)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(revoke.status(), StatusCode::OK);
    assert!(body_text(revoke).await.contains("\"revoked\":true"));

    let revoked_list = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/auth/mfa/factors")
                .header(COOKIE, &login_cookie)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(revoked_list.status(), StatusCode::OK);
    assert!(
        body_text(revoked_list)
            .await
            .contains("\"status\":\"revoked\"")
    );
    let active_recovery_after_revoke: i64 =
        sqlx::query_scalar("select count(*) from iam_mfa_recovery_codes where status = 'active'")
            .fetch_one(&pool)
            .await
            .unwrap();
    assert_eq!(active_recovery_after_revoke, 0);

    let raw_secret_hits: i64 =
        sqlx::query_scalar("select count(*) from iam_audit_logs where detail like '%' || ? || '%'")
            .bind(secret)
            .fetch_one(&pool)
            .await
            .unwrap();
    assert_eq!(raw_secret_hits, 0);
    for recovery_code in recovery_codes.iter().chain(rotated_codes.iter()) {
        let raw_recovery_audit_hits: i64 = sqlx::query_scalar(
            "select count(*) from iam_audit_logs where detail like '%' || ? || '%'",
        )
        .bind(recovery_code)
        .fetch_one(&pool)
        .await
        .unwrap();
        assert_eq!(raw_recovery_audit_hits, 0);
    }
}

#[tokio::test]
async fn permission_routes_enforce_cookie_and_api_token_scope() {
    let (router, db_url) = test_router().await;
    let (cookie_pair, org_id) = create_initial_admin(router.clone()).await;

    let anonymous_catalog = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/apis")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(anonymous_catalog.status(), StatusCode::UNAUTHORIZED);

    let cookie_catalog = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/apis")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(cookie_catalog.status(), StatusCode::OK);
    assert!(body_text(cookie_catalog).await.contains("permission:read"));

    let orgs = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/iam/orgs")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(orgs.status(), StatusCode::OK);
    let orgs_body = body_text(orgs).await;
    assert!(orgs_body.contains("\"code\":\"main\""));
    assert!(orgs_body.contains("\"scope\":\"tenant\""));
    assert!(!orgs_body.contains("password_hash"));

    let org_users = router
        .clone()
        .oneshot(
            Request::builder()
                .uri(format!("/api/v1/iam/orgs/{org_id}/users"))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(org_users.status(), StatusCode::OK);
    let org_users_body = body_text(org_users).await;
    assert!(org_users_body.contains("owner@example.com"));
    assert!(org_users_body.contains("\"owner\""));
    assert!(!org_users_body.contains("password_hash"));

    let org_roles = router
        .clone()
        .oneshot(
            Request::builder()
                .uri(format!("/api/v1/iam/orgs/{org_id}/roles"))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(org_roles.status(), StatusCode::OK);
    let org_roles_body = body_text(org_roles).await;
    assert!(org_roles_body.contains("\"code\":\"owner\""));
    assert!(org_roles_body.contains("api_token:read"));

    let iam_permissions = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/iam/permissions")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(iam_permissions.status(), StatusCode::OK);
    let iam_permissions_body = body_text(iam_permissions).await;
    assert!(iam_permissions_body.contains("org:read"));
    assert!(iam_permissions_body.contains("user:read"));
    assert!(iam_permissions_body.contains("role:read"));

    let token = create_api_token(router.clone(), &cookie_pair, org_id).await;
    let bearer = format!("Bearer {token}");

    let token_list = router
        .clone()
        .oneshot(
            Request::builder()
                .uri(format!("/api/v1/orgs/{org_id}/api-tokens"))
                .header(AUTHORIZATION, &bearer)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(token_list.status(), StatusCode::OK);

    let token_orgs = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/iam/orgs")
                .header(AUTHORIZATION, &bearer)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(token_orgs.status(), StatusCode::FORBIDDEN);

    let token_catalog = router
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/apis")
                .header(AUTHORIZATION, &bearer)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(token_catalog.status(), StatusCode::FORBIDDEN);

    let pool = sqlx::SqlitePool::connect(&db_url).await.unwrap();
    let detail: String = sqlx::query_scalar(
        "select detail from iam_audit_logs where action = 'iam.api_token.created'",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert!(!detail.contains(&token));
}

#[tokio::test]
async fn tenant_role_write_management_enforces_scope_and_role_usage() {
    let (router, db_url) = test_router().await;
    let (cookie_pair, org_id) = create_initial_admin(router.clone()).await;
    let pool = sqlx::SqlitePool::connect(&db_url).await.unwrap();

    let create = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!("/api/v1/iam/orgs/{org_id}/roles"))
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({
                        "code": "operator",
                        "name": "运营角色",
                        "permission_codes": ["user:read", "role:read", "user:invite"]
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(create.status(), StatusCode::OK);
    let created: serde_json::Value = serde_json::from_str(&body_text(create).await).unwrap();
    let operator_role_id = created["id"].as_i64().expect("operator role id");
    assert_eq!(created["code"], "operator");
    assert_eq!(created["system_builtin"], false);
    assert!(
        created["permissions"]
            .as_array()
            .unwrap()
            .contains(&json!("user:invite"))
    );

    let platform_permission = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!("/api/v1/iam/orgs/{org_id}/roles"))
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({
                        "code": "platform-shadow",
                        "name": "错误平台权限",
                        "permission_codes": ["permission:read"]
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    let platform_permission_status = platform_permission.status();
    let platform_permission_body = body_text(platform_permission).await;
    assert_eq!(
        platform_permission_status,
        StatusCode::BAD_REQUEST,
        "{platform_permission_body}"
    );

    let update = router
        .clone()
        .oneshot(
            Request::builder()
                .method("PUT")
                .uri(format!(
                    "/api/v1/iam/orgs/{org_id}/roles/{operator_role_id}"
                ))
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({
                        "name": "只读运营",
                        "permission_codes": ["user:read"]
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(update.status(), StatusCode::OK);
    let updated: serde_json::Value = serde_json::from_str(&body_text(update).await).unwrap();
    assert_eq!(updated["permissions"], json!(["user:read"]));

    let invite = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!("/api/v1/orgs/{org_id}/users/invitations"))
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({
                        "email": "operator@example.com",
                        "role_code": "operator"
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(invite.status(), StatusCode::OK);

    let raw_token = "invitation_token_operator_fixture_token";
    insert_invitation_fixture(
        &pool,
        org_id,
        "operator-accepted@example.com",
        raw_token,
        "operator",
    )
    .await;
    let accept = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/invitations/accept")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({
                        "token": raw_token,
                        "password": "change-me-123",
                        "display_name": "Accepted Operator"
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(accept.status(), StatusCode::OK);
    let accepted: serde_json::Value = serde_json::from_str(&body_text(accept).await).unwrap();
    assert_eq!(accepted["permissions"], json!(["user:read"]));

    let delete_used_role = router
        .clone()
        .oneshot(
            Request::builder()
                .method("DELETE")
                .uri(format!(
                    "/api/v1/iam/orgs/{org_id}/roles/{operator_role_id}"
                ))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(delete_used_role.status(), StatusCode::CONFLICT);

    let temp_role = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!("/api/v1/iam/orgs/{org_id}/roles"))
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({
                        "code": "temporary",
                        "name": "临时角色",
                        "permission_codes": ["user:read"]
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(temp_role.status(), StatusCode::OK);
    let temporary: serde_json::Value = serde_json::from_str(&body_text(temp_role).await).unwrap();
    let temporary_role_id = temporary["id"].as_i64().expect("temporary role id");

    let delete_temp_role = router
        .clone()
        .oneshot(
            Request::builder()
                .method("DELETE")
                .uri(format!(
                    "/api/v1/iam/orgs/{org_id}/roles/{temporary_role_id}"
                ))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(delete_temp_role.status(), StatusCode::OK);
    let deleted: serde_json::Value =
        serde_json::from_str(&body_text(delete_temp_role).await).unwrap();
    assert_eq!(deleted["deleted"], true);

    let owner_role_id: i64 =
        sqlx::query_scalar("select id from iam_roles where org_id = ? and code = 'owner'")
            .bind(org_id)
            .fetch_one(&pool)
            .await
            .unwrap();
    let update_builtin = router
        .oneshot(
            Request::builder()
                .method("PUT")
                .uri(format!("/api/v1/iam/orgs/{org_id}/roles/{owner_role_id}"))
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({
                        "name": "不能修改内置角色",
                        "permission_codes": ["user:read"]
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(update_builtin.status(), StatusCode::CONFLICT);
}

#[tokio::test]
async fn tenant_user_update_assigns_roles_and_disables_sessions_safely() {
    let (router, db_url) = test_router().await;
    let (cookie_pair, org_id) = create_initial_admin(router.clone()).await;
    let pool = sqlx::SqlitePool::connect(&db_url).await.unwrap();

    let create_role = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!("/api/v1/iam/orgs/{org_id}/roles"))
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({
                        "code": "operator",
                        "name": "运营角色",
                        "permission_codes": ["user:read"]
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(create_role.status(), StatusCode::OK);

    let raw_token = "invitation_token_user_update_fixture_token";
    insert_invitation_fixture(&pool, org_id, "managed@example.com", raw_token, "operator").await;
    let accept = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/invitations/accept")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({
                        "token": raw_token,
                        "password": "change-me-123",
                        "display_name": "Managed User"
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(accept.status(), StatusCode::OK);
    let managed_cookie = cookie_header_from_set_cookie(accept.headers());
    let managed_user_id: i64 = sqlx::query_scalar("select id from iam_users where email = ?")
        .bind("managed@example.com")
        .fetch_one(&pool)
        .await
        .unwrap();

    let promote = router
        .clone()
        .oneshot(
            Request::builder()
                .method("PUT")
                .uri(format!("/api/v1/iam/orgs/{org_id}/users/{managed_user_id}"))
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({
                        "display_name": "Managed Owner",
                        "status": "active",
                        "role_codes": ["owner"]
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(promote.status(), StatusCode::OK);
    let promoted: serde_json::Value = serde_json::from_str(&body_text(promote).await).unwrap();
    assert_eq!(promoted["display_name"], "Managed Owner");
    assert_eq!(promoted["role_codes"], json!(["owner"]));

    let disable = router
        .clone()
        .oneshot(
            Request::builder()
                .method("PUT")
                .uri(format!("/api/v1/iam/orgs/{org_id}/users/{managed_user_id}"))
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({
                        "display_name": "Disabled Operator",
                        "status": "disabled",
                        "role_codes": ["operator"]
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(disable.status(), StatusCode::OK);
    let disabled: serde_json::Value = serde_json::from_str(&body_text(disable).await).unwrap();
    assert_eq!(disabled["status"], "disabled");
    assert_eq!(disabled["role_codes"], json!(["operator"]));

    let disabled_session = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/me/session")
                .header(COOKIE, managed_cookie)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(disabled_session.status(), StatusCode::UNAUTHORIZED);

    let disabled_login = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/login")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({
                        "identifier": "managed@example.com",
                        "password": "change-me-123"
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(disabled_login.status(), StatusCode::UNAUTHORIZED);

    let owner_user_id: i64 = sqlx::query_scalar("select id from iam_users where email = ?")
        .bind("owner@example.com")
        .fetch_one(&pool)
        .await
        .unwrap();
    let disable_last_owner = router
        .oneshot(
            Request::builder()
                .method("PUT")
                .uri(format!("/api/v1/iam/orgs/{org_id}/users/{owner_user_id}"))
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({
                        "display_name": "Owner",
                        "status": "disabled",
                        "role_codes": ["owner"]
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(disable_last_owner.status(), StatusCode::CONFLICT);

    let membership_roles: Vec<String> = sqlx::query_scalar(
        "select role_code
         from iam_memberships
         where org_id = ? and user_id = ?
         order by role_code asc",
    )
    .bind(org_id)
    .bind(managed_user_id)
    .fetch_all(&pool)
    .await
    .unwrap();
    assert_eq!(membership_roles, vec!["operator".to_string()]);
}

#[tokio::test]
async fn system_menus_status_and_operation_records_are_protected_and_real() {
    let (router, db_url) = test_router().await;
    let (cookie_pair, org_id) = create_initial_admin(router.clone()).await;

    let anonymous_menus = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/menus")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(anonymous_menus.status(), StatusCode::UNAUTHORIZED);

    let menus = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/menus")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(menus.status(), StatusCode::OK);
    let menu_body = body_text(menus).await;
    assert!(menu_body.contains("system.server-status"));
    assert!(menu_body.contains("iam.api-tokens"));

    let token = create_api_token(router.clone(), &cookie_pair, org_id).await;
    let token_menus = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/menus")
                .header(AUTHORIZATION, format!("Bearer {token}"))
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(token_menus.status(), StatusCode::FORBIDDEN);

    let status = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/server-status")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(status.status(), StatusCode::OK);
    let status_value: serde_json::Value = serde_json::from_str(&body_text(status).await).unwrap();
    assert_eq!(status_value["source"], "runtime-process");
    assert!(status_value["process_id"].as_u64().unwrap() > 0);
    assert!(status_value["available_parallelism"].as_u64().unwrap() >= 1);
    assert_eq!(status_value["product_code"], "console");
    assert_eq!(status_value["metrics"]["source"], "sysinfo");
    assert!(
        status_value["metrics"]["cpu_usage_percent"]
            .as_f64()
            .unwrap()
            >= 0.0
    );
    assert!(
        status_value["metrics"]["process_cpu_usage_percent"]
            .as_f64()
            .unwrap()
            >= 0.0
    );
    assert!(
        status_value["metrics"]["total_memory_bytes"]
            .as_u64()
            .unwrap()
            > 0
    );
    assert!(
        status_value["metrics"]["process_memory_bytes"]
            .as_u64()
            .unwrap()
            <= status_value["metrics"]["total_memory_bytes"]
                .as_u64()
                .unwrap()
    );
    assert!(
        status_value["metrics"]["process_virtual_memory_bytes"]
            .as_u64()
            .unwrap()
            >= status_value["metrics"]["process_memory_bytes"]
                .as_u64()
                .unwrap()
    );
    assert!(
        status_value["metrics"]["total_disk_bytes"]
            .as_u64()
            .unwrap()
            >= status_value["metrics"]["available_disk_bytes"]
                .as_u64()
                .unwrap()
    );
    assert_eq!(
        status_value["metrics"]["used_disk_bytes"].as_u64().unwrap(),
        status_value["metrics"]["total_disk_bytes"]
            .as_u64()
            .unwrap()
            .saturating_sub(
                status_value["metrics"]["available_disk_bytes"]
                    .as_u64()
                    .unwrap()
            )
    );
    assert!(
        status_value["metrics"]["system_uptime_seconds"]
            .as_u64()
            .unwrap()
            > 0
    );
    assert!(
        status_value["metrics"]["system_boot_time_seconds"]
            .as_u64()
            .unwrap()
            > 0
    );
    assert!(
        status_value["metrics"]["load_average_one"]
            .as_f64()
            .unwrap()
            >= 0.0
    );
    assert!(
        status_value["metrics"]["network_interface_count"]
            .as_u64()
            .is_some()
    );
    assert!(
        status_value["metrics"]["network_received_bytes"]
            .as_u64()
            .is_some()
    );
    assert!(
        status_value["metrics"]["network_transmitted_bytes"]
            .as_u64()
            .is_some()
    );

    let anonymous_metrics = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/metrics/prometheus")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(anonymous_metrics.status(), StatusCode::UNAUTHORIZED);

    let token_metrics = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/metrics/prometheus")
                .header(AUTHORIZATION, format!("Bearer {token}"))
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(token_metrics.status(), StatusCode::FORBIDDEN);

    let prometheus = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/metrics/prometheus")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(prometheus.status(), StatusCode::OK);
    assert_eq!(
        prometheus
            .headers()
            .get(CONTENT_TYPE)
            .and_then(|value| value.to_str().ok()),
        Some("text/plain; version=0.0.4; charset=utf-8")
    );
    let prometheus_body = body_text(prometheus).await;
    assert!(prometheus_body.contains("# TYPE console_cpu_usage_percent gauge"));
    assert!(prometheus_body.contains("console_memory_bytes{"));
    assert!(prometheus_body.contains("# TYPE console_network_interface_count gauge"));
    assert!(prometheus_body.contains("# TYPE console_network_bytes counter"));
    assert!(prometheus_body.contains("direction=\"received\""));
    assert!(prometheus_body.contains("direction=\"transmitted\""));
    assert!(prometheus_body.contains("product_code=\"console\""));
    assert!(prometheus_body.contains("metrics_source=\"sysinfo\""));
    assert!(!prometheus_body.contains("session_token_"));
    assert!(!prometheus_body.contains("secret"));

    let openapi = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/openapi.yaml")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(openapi.status(), StatusCode::OK);

    let records = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/operation-records")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(records.status(), StatusCode::OK);
    let records_body = body_text(records).await;
    assert!(records_body.contains("/api/v1/system/server-status"));
    assert!(!records_body.contains("/openapi.yaml"));

    let pool = sqlx::SqlitePool::connect(&db_url).await.unwrap();
    let openapi_records: i64 = sqlx::query_scalar(
        "select count(*) from system_operation_records where path = '/openapi.yaml'",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert_eq!(openapi_records, 0);
}

#[tokio::test]
async fn prometheus_metrics_accepts_configured_scrape_token_hash_without_raw_token_leak() {
    let raw_scrape_token = "prometheus_scrape_token_for_http_smoke";
    let mut settings = test_settings();
    settings.observability.prometheus_scrape_token_hash =
        crypto::hash_secret(raw_scrape_token, &settings.auth.session_secret);
    let app = App::boot(settings).await.expect("boot app");
    let router = app.router();

    let wrong_token = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/metrics/prometheus")
                .header(AUTHORIZATION, "Bearer wrong-prometheus-token")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(wrong_token.status(), StatusCode::UNAUTHORIZED);

    let prometheus = router
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/metrics/prometheus")
                .header(AUTHORIZATION, format!("Bearer {raw_scrape_token}"))
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(prometheus.status(), StatusCode::OK);
    let prometheus_body = body_text(prometheus).await;
    assert!(prometheus_body.contains("# TYPE console_cpu_usage_percent gauge"));
    assert!(prometheus_body.contains("console_memory_bytes{"));
    assert!(!prometheus_body.contains(raw_scrape_token));
    assert!(!prometheus_body.contains("secret"));
}

#[tokio::test]
async fn system_configs_dictionaries_and_parameters_are_protected_and_persisted() {
    let (router, db_url) = test_router().await;
    let (cookie_pair, org_id) = create_initial_admin(router.clone()).await;

    let anonymous_configs = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/configs")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(anonymous_configs.status(), StatusCode::UNAUTHORIZED);

    let token = create_api_token(router.clone(), &cookie_pair, org_id).await;
    let token_configs = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/configs")
                .header(AUTHORIZATION, format!("Bearer {token}"))
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(token_configs.status(), StatusCode::FORBIDDEN);

    let config = router
        .clone()
        .oneshot(
            Request::builder()
                .method("PUT")
                .uri("/api/v1/system/configs/feature_flags")
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(json!({"value": {"setup_v2": true}}).to_string()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(config.status(), StatusCode::OK);
    assert!(body_text(config).await.contains("setup_v2"));

    let secret_config = router
        .clone()
        .oneshot(
            Request::builder()
                .method("PUT")
                .uri("/api/v1/system/configs/session_token_secret")
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(json!({"value": "leak"}).to_string()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(secret_config.status(), StatusCode::BAD_REQUEST);

    for key in [
        "auth_private_key",
        "smtp_password",
        "oauth_credential",
        "api_token_ttl",
    ] {
        let response = router
            .clone()
            .oneshot(
                Request::builder()
                    .method("PUT")
                    .uri(format!("/api/v1/system/configs/{key}"))
                    .header(CONTENT_TYPE, "application/json")
                    .header(COOKIE, &cookie_pair)
                    .body(Body::from(json!({"value": "blocked"}).to_string()))
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(response.status(), StatusCode::BAD_REQUEST);
    }

    let dictionary = router
        .clone()
        .oneshot(
            Request::builder()
                .method("PUT")
                .uri("/api/v1/system/dictionaries/locales")
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(json!({"name": "语言"}).to_string()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(dictionary.status(), StatusCode::OK);
    assert!(body_text(dictionary).await.contains("locales"));

    let parameter = router
        .clone()
        .oneshot(
            Request::builder()
                .method("PUT")
                .uri("/api/v1/system/parameters/page_size")
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({"name": "默认分页大小", "value": "20"}).to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(parameter.status(), StatusCode::OK);
    assert!(body_text(parameter).await.contains("page_size"));

    for key in ["smtp_password", "private_storage_key", "token_window"] {
        let response = router
            .clone()
            .oneshot(
                Request::builder()
                    .method("PUT")
                    .uri(format!("/api/v1/system/parameters/{key}"))
                    .header(CONTENT_TYPE, "application/json")
                    .header(COOKIE, &cookie_pair)
                    .body(Body::from(
                        json!({"name": "敏感参数", "value": "blocked"}).to_string(),
                    ))
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(response.status(), StatusCode::BAD_REQUEST);
    }

    let list = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/configs")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(list.status(), StatusCode::OK);
    let list_body = body_text(list).await;
    assert!(list_body.contains("feature_flags"));
    assert!(!list_body.contains("session_token_secret"));

    let delete_dictionary = router
        .clone()
        .oneshot(
            Request::builder()
                .method("DELETE")
                .uri("/api/v1/system/dictionaries/locales")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(delete_dictionary.status(), StatusCode::OK);
    assert!(
        body_text(delete_dictionary)
            .await
            .contains("\"deleted\":true")
    );

    let delete_parameter = router
        .clone()
        .oneshot(
            Request::builder()
                .method("DELETE")
                .uri("/api/v1/system/parameters/page_size")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(delete_parameter.status(), StatusCode::OK);
    assert!(
        body_text(delete_parameter)
            .await
            .contains("\"deleted\":true")
    );

    let pool = sqlx::SqlitePool::connect(&db_url).await.unwrap();
    let secret_count: i64 = sqlx::query_scalar(
        "select count(*) from system_configs
         where key in (
            'session_token_secret',
            'auth_private_key',
            'smtp_password',
            'oauth_credential',
            'api_token_ttl'
         )",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert_eq!(secret_count, 0);
    let secret_parameter_count: i64 = sqlx::query_scalar(
        "select count(*) from system_parameters
         where key in ('smtp_password', 'private_storage_key', 'token_window')",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert_eq!(secret_parameter_count, 0);
    let operation_count: i64 = sqlx::query_scalar(
        "select count(*) from system_operation_records
         where path = '/api/v1/system/configs/{key}' and actor_user_id is not null",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert!(operation_count >= 1);
    let raw_operation_path_count: i64 = sqlx::query_scalar(
        "select count(*) from system_operation_records
         where path = '/api/v1/system/configs/feature_flags'",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert_eq!(raw_operation_path_count, 0);

    let owner_user_id: i64 = sqlx::query_scalar("select id from iam_users where email = ?")
        .bind("owner@example.com")
        .fetch_one(&pool)
        .await
        .unwrap();
    let filtered_records = router
        .clone()
        .oneshot(
            Request::builder()
                .uri(format!(
                    "/api/v1/system/operation-records?method=PUT&path=%2Fapi%2Fv1%2Fsystem%2Fconfigs%2F%7Bkey%7D&status=200&actor_user_id={owner_user_id}&created_from=2000-01-01T00:00:00Z&created_to=2999-01-01T00:00:00Z&limit=1&offset=0"
                ))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(filtered_records.status(), StatusCode::OK);
    let filtered_value: serde_json::Value =
        serde_json::from_str(&body_text(filtered_records).await).unwrap();
    let filtered_items = filtered_value.as_array().expect("operation records");
    assert_eq!(filtered_items.len(), 1);
    assert_eq!(filtered_items[0]["method"], "PUT");
    assert_eq!(filtered_items[0]["path"], "/api/v1/system/configs/{key}");
    assert_eq!(filtered_items[0]["status"].as_i64(), Some(200));
    assert_eq!(
        filtered_items[0]["actor_user_id"].as_i64(),
        Some(owner_user_id)
    );

    let paged_records = router
        .clone()
        .oneshot(
            Request::builder()
                .uri(format!(
                    "/api/v1/system/operation-records?method=PUT&path=%2Fapi%2Fv1%2Fsystem%2Fconfigs%2F%7Bkey%7D&status=200&actor_user_id={owner_user_id}&created_from=2000-01-01T00:00:00Z&created_to=2999-01-01T00:00:00Z&limit=1&offset=1"
                ))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(paged_records.status(), StatusCode::OK);
    let paged_value: serde_json::Value =
        serde_json::from_str(&body_text(paged_records).await).unwrap();
    assert_eq!(paged_value.as_array().expect("operation records").len(), 0);

    let exported_records = router
        .clone()
        .oneshot(
            Request::builder()
                .uri(format!(
                    "/api/v1/system/operation-records/export.csv?method=PUT&path=%2Fapi%2Fv1%2Fsystem%2Fconfigs%2F%7Bkey%7D&status=200&actor_user_id={owner_user_id}&created_from=2000-01-01T00:00:00Z&created_to=2999-01-01T00:00:00Z&limit=1&offset=0"
                ))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(exported_records.status(), StatusCode::OK);
    assert!(
        exported_records
            .headers()
            .get(CONTENT_TYPE)
            .and_then(|value| value.to_str().ok())
            .is_some_and(|value| value.starts_with("text/csv"))
    );
    assert!(
        exported_records
            .headers()
            .get(CONTENT_DISPOSITION)
            .and_then(|value| value.to_str().ok())
            .is_some_and(|value| value.contains("operation-records.csv"))
    );
    let exported_body = body_text(exported_records).await;
    assert!(exported_body.starts_with("id,actor_user_id,method,path,status,created_at\n"));
    assert!(exported_body.contains(",PUT,/api/v1/system/configs/{key},200,"));
    assert!(!exported_body.contains("token=raw"));

    let summarized_records = router
        .clone()
        .oneshot(
            Request::builder()
                .uri(format!(
                    "/api/v1/system/operation-records/summary?method=PUT&path=%2Fapi%2Fv1%2Fsystem%2Fconfigs%2F%7Bkey%7D&status=200&actor_user_id={owner_user_id}&created_from=2000-01-01T00:00:00Z&created_to=2999-01-01T00:00:00Z&top_limit=3"
                ))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(summarized_records.status(), StatusCode::OK);
    let summary_value: serde_json::Value =
        serde_json::from_str(&body_text(summarized_records).await).unwrap();
    assert!(
        summary_value["generated_at"]
            .as_str()
            .is_some_and(|value| value.contains('T'))
    );
    assert_eq!(summary_value["top_limit"].as_i64(), Some(3));
    assert!(summary_value["total_count"].as_i64().unwrap_or_default() >= 1);
    assert!(summary_value["success_count"].as_i64().unwrap_or_default() >= 1);
    assert_eq!(summary_value["server_error_count"].as_i64(), Some(0));
    let by_method = summary_value["by_method"].as_array().unwrap();
    assert!(
        by_method
            .iter()
            .any(|item| item["key"] == "PUT" && item["count"].as_i64().unwrap_or_default() >= 1)
    );
    let by_status_class = summary_value["by_status_class"].as_array().unwrap();
    assert!(
        by_status_class
            .iter()
            .any(|item| item["key"] == "2xx" && item["count"].as_i64().unwrap_or_default() >= 1)
    );
    let top_paths = summary_value["top_paths"].as_array().unwrap();
    assert!(top_paths.iter().any(|item| {
        item["path"] == "/api/v1/system/configs/{key}"
            && item["count"].as_i64().unwrap_or_default() >= 1
            && item["error_count"].as_i64() == Some(0)
            && item["last_seen_at"]
                .as_str()
                .is_some_and(|value| value.contains('T'))
    }));
    let invalid_summary = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/operation-records/summary?top_limit=100")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(invalid_summary.status(), StatusCode::BAD_REQUEST);

    sqlx::query(
        "insert into system_operation_records(actor_user_id, method, path, status, created_at)
         values (?, 'GET', '/api/v1/system/operation-records', 200, '2000-01-01T00:00:00Z')",
    )
    .bind(owner_user_id)
    .execute(&pool)
    .await
    .unwrap();
    let prune_records = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/system/operation-records/prune")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(prune_records.status(), StatusCode::OK);
    let prune_value: serde_json::Value =
        serde_json::from_str(&body_text(prune_records).await).unwrap();
    assert_eq!(prune_value["retention_days"].as_i64(), Some(180));
    assert_eq!(prune_value["prune_batch_size"].as_i64(), Some(1000));
    assert!(
        prune_value["cutoff"]
            .as_str()
            .is_some_and(|value| value.contains('T'))
    );
    assert!(prune_value["deleted"].as_i64().unwrap_or_default() >= 1);
    let stale_operation_count: i64 = sqlx::query_scalar(
        "select count(*) from system_operation_records
         where created_at = '2000-01-01T00:00:00Z'",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert_eq!(stale_operation_count, 0);
    let retained_operation_count: i64 = sqlx::query_scalar(
        "select count(*) from system_operation_records
         where path = '/api/v1/system/configs/{key}'",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert!(retained_operation_count >= 1);

    let unsafe_filter = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/operation-records?path=%2Fapi%2Fv1%2Fsystem%2Fconfigs%2F%7Bkey%7D%3Ftoken%3Draw")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(unsafe_filter.status(), StatusCode::BAD_REQUEST);

    let invalid_time_range = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/operation-records?created_from=2999-01-01T00:00:00Z&created_to=2000-01-01T00:00:00Z")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(invalid_time_range.status(), StatusCode::BAD_REQUEST);
}

#[tokio::test]
async fn version_packages_and_media_assets_are_protected_and_metadata_only() {
    let (router, db_url) = test_router().await;
    let (cookie_pair, org_id) = create_initial_admin(router.clone()).await;

    let anonymous_versions = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/version-packages")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(anonymous_versions.status(), StatusCode::UNAUTHORIZED);

    let token = create_api_token(router.clone(), &cookie_pair, org_id).await;
    let token_media = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/media-assets")
                .header(AUTHORIZATION, format!("Bearer {token}"))
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(token_media.status(), StatusCode::FORBIDDEN);

    let version = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/system/version-packages")
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({
                        "version_name": "Aoi 0.1.0",
                        "version_code": "0.1.0",
                        "manifest": {"channel": "local", "notes": ["init"]}
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(version.status(), StatusCode::OK);
    let version_value: serde_json::Value = serde_json::from_str(&body_text(version).await).unwrap();
    let version_id = version_value["id"].as_i64().expect("version id");
    assert_eq!(version_value["manifest"]["channel"], "local");
    assert_eq!(version_value["status"], "draft");

    let next_version = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/system/version-packages")
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({
                        "version_name": "Aoi 0.2.0",
                        "version_code": "0.2.0",
                        "manifest": {"channel": "stable", "notes": ["release"]}
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(next_version.status(), StatusCode::OK);
    let next_version_value: serde_json::Value =
        serde_json::from_str(&body_text(next_version).await).unwrap();
    let next_version_id = next_version_value["id"].as_i64().expect("next version id");

    let publish_version = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!(
                    "/api/v1/system/version-packages/{version_id}/publish"
                ))
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(json!({"reason": "initial release"}).to_string()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(publish_version.status(), StatusCode::OK);
    let publish_value: serde_json::Value =
        serde_json::from_str(&body_text(publish_version).await).unwrap();
    assert_eq!(publish_value["package"]["status"], "active");
    assert!(publish_value["previous_active_id"].is_null());

    let delete_active_version = router
        .clone()
        .oneshot(
            Request::builder()
                .method("DELETE")
                .uri(format!("/api/v1/system/version-packages/{version_id}"))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(delete_active_version.status(), StatusCode::CONFLICT);

    let publish_next_version = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!(
                    "/api/v1/system/version-packages/{next_version_id}/publish"
                ))
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(json!({"reason": "stable release"}).to_string()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(publish_next_version.status(), StatusCode::OK);
    let publish_next_value: serde_json::Value =
        serde_json::from_str(&body_text(publish_next_version).await).unwrap();
    assert_eq!(publish_next_value["previous_active_id"], version_id);
    assert_eq!(publish_next_value["package"]["status"], "active");

    let rollback_version = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!(
                    "/api/v1/system/version-packages/{version_id}/rollback"
                ))
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(json!({"reason": "rollback smoke"}).to_string()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(rollback_version.status(), StatusCode::OK);
    let rollback_value: serde_json::Value =
        serde_json::from_str(&body_text(rollback_version).await).unwrap();
    assert_eq!(rollback_value["previous_active_id"], next_version_id);
    assert_eq!(rollback_value["package"]["status"], "active");

    let release_events = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/version-packages/releases")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(release_events.status(), StatusCode::OK);
    let release_events_text = body_text(release_events).await;
    assert!(release_events_text.contains("\"action\":\"publish\""));
    assert!(release_events_text.contains("\"action\":\"rollback\""));

    let media = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/system/media-assets")
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({
                        "category": "avatars",
                        "display_name": "Logo",
                        "storage_key": "media/logos/aoi.png",
                        "mime_type": "IMAGE/PNG",
                        "size_bytes": 1024
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(media.status(), StatusCode::OK);
    let media_value: serde_json::Value = serde_json::from_str(&body_text(media).await).unwrap();
    let media_id = media_value["id"].as_i64().expect("media id");
    assert_eq!(media_value["mime_type"], "image/png");

    let unsafe_media = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/system/media-assets")
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({
                        "display_name": "Secret",
                        "storage_key": "../secret-token.txt",
                        "mime_type": "text/plain",
                        "size_bytes": 1
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(unsafe_media.status(), StatusCode::BAD_REQUEST);

    let versions = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/version-packages")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(versions.status(), StatusCode::OK);
    assert!(body_text(versions).await.contains("Aoi 0.1.0"));

    let delete_version = router
        .clone()
        .oneshot(
            Request::builder()
                .method("DELETE")
                .uri(format!("/api/v1/system/version-packages/{next_version_id}"))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(delete_version.status(), StatusCode::OK);
    assert!(body_text(delete_version).await.contains("\"deleted\":true"));

    let delete_media = router
        .clone()
        .oneshot(
            Request::builder()
                .method("DELETE")
                .uri(format!("/api/v1/system/media-assets/{media_id}"))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(delete_media.status(), StatusCode::OK);
    assert!(body_text(delete_media).await.contains("\"deleted\":true"));

    let pool = sqlx::SqlitePool::connect(&db_url).await.unwrap();
    let unsafe_count: i64 = sqlx::query_scalar(
        "select count(*) from system_media_assets where storage_key = '../secret-token.txt'",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert_eq!(unsafe_count, 0);
    let operation_count: i64 = sqlx::query_scalar(
        "select count(*) from system_operation_records
         where path = '/api/v1/system/media-assets' and actor_user_id is not null",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert!(operation_count >= 1);
}

#[tokio::test]
async fn media_upload_persists_local_file_and_rejects_unsafe_filename() {
    let settings = test_settings();
    let media_dir = settings.storage.local_dir.clone();
    let db_url = settings.database.url.clone();
    let app = App::boot(settings).await.expect("boot app");
    let router = app.router();
    let (cookie_pair, org_id) = create_initial_admin(router.clone()).await;

    let boundary = "----console-media-upload-boundary";
    let upload_body = multipart_body(
        boundary,
        Some(("category", "avatars")),
        Some(("display_name", "Uploaded Logo")),
        "file",
        "logo.txt",
        "text/plain",
        b"console-media-file",
    );
    let upload = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/system/media-assets/upload")
                .header(
                    CONTENT_TYPE,
                    format!("multipart/form-data; boundary={boundary}"),
                )
                .header(COOKIE, &cookie_pair)
                .body(Body::from(upload_body))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(upload.status(), StatusCode::OK);
    let upload_value: serde_json::Value = serde_json::from_str(&body_text(upload).await).unwrap();
    assert_eq!(upload_value["display_name"], "Uploaded Logo");
    assert_eq!(upload_value["mime_type"], "text/plain");
    assert_eq!(upload_value["size_bytes"], 18);
    let storage_key = upload_value["storage_key"]
        .as_str()
        .expect("storage key")
        .to_string();
    assert!(storage_key.starts_with("local/"));
    assert!(!storage_key.contains("logo"));

    let stored_path = storage_key
        .split('/')
        .fold(PathBuf::from(&media_dir), |path, part| path.join(part));
    assert_eq!(
        fs::read(&stored_path).expect("stored media file"),
        b"console-media-file"
    );

    let anonymous_objects = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/storage-objects")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(anonymous_objects.status(), StatusCode::UNAUTHORIZED);

    let token = create_api_token(router.clone(), &cookie_pair, org_id).await;
    let token_objects = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/storage-objects")
                .header(AUTHORIZATION, format!("Bearer {token}"))
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(token_objects.status(), StatusCode::FORBIDDEN);

    let storage_objects = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/storage-objects?prefix=local/&limit=10")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(storage_objects.status(), StatusCode::OK);
    let object_values: Vec<serde_json::Value> =
        serde_json::from_str(&body_text(storage_objects).await).unwrap();
    assert!(object_values.iter().any(|item| {
        item["storage_key"] == storage_key
            && item["size_bytes"] == 18
            && item["updated_at"].as_str().is_some()
    }));

    let delete_storage_object = router
        .clone()
        .oneshot(
            Request::builder()
                .method("DELETE")
                .uri("/api/v1/system/storage-objects")
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({ "storage_key": storage_key.clone() }).to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(delete_storage_object.status(), StatusCode::OK);
    assert!(
        body_text(delete_storage_object)
            .await
            .contains("\"deleted\":true")
    );
    assert!(!stored_path.exists());

    let unsafe_boundary = "----console-media-unsafe-boundary";
    let unsafe_body = multipart_body(
        unsafe_boundary,
        None,
        Some(("display_name", "Unsafe Upload")),
        "file",
        "../secret-token.txt",
        "text/plain",
        b"unsafe",
    );
    let unsafe_upload = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/system/media-assets/upload")
                .header(
                    CONTENT_TYPE,
                    format!("multipart/form-data; boundary={unsafe_boundary}"),
                )
                .header(COOKIE, &cookie_pair)
                .body(Body::from(unsafe_body))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(unsafe_upload.status(), StatusCode::BAD_REQUEST);

    let pool = sqlx::SqlitePool::connect(&db_url).await.unwrap();
    let unsafe_count: i64 = sqlx::query_scalar(
        "select count(*) from system_media_assets
         where display_name = 'Unsafe Upload' or storage_key like '%secret%'",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert_eq!(unsafe_count, 0);
}

#[tokio::test]
async fn traffic_probes_use_real_http_collection_and_platform_permissions() {
    let (router, db_url) = test_router().await;
    let (cookie_pair, org_id) = create_initial_admin(router.clone()).await;
    let probe_url = spawn_probe_http_target("204 No Content").await;
    let warning_probe_url = spawn_probe_http_target("503 Service Unavailable").await;
    let recovering_probe_url =
        spawn_probe_http_target_sequence(vec!["503 Service Unavailable", "200 OK"]).await;

    let anonymous_targets = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/traffic-probes/targets")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(anonymous_targets.status(), StatusCode::UNAUTHORIZED);

    let token = create_api_token(router.clone(), &cookie_pair, org_id).await;
    let token_targets = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/traffic-probes/targets")
                .header(AUTHORIZATION, format!("Bearer {token}"))
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(token_targets.status(), StatusCode::FORBIDDEN);

    let target = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/system/traffic-probes/targets")
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({
                        "name": "本地健康端点",
                        "url": probe_url,
                        "expected_status": 204
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(target.status(), StatusCode::OK);
    let target_value: serde_json::Value = serde_json::from_str(&body_text(target).await).unwrap();
    let target_id = target_value["id"].as_i64().expect("target id");
    assert_eq!(target_value["status"], "pending");

    let unsafe_target = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/system/traffic-probes/targets")
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({
                        "name": "泄露风险",
                        "url": "https://example.com/health?token=raw-secret"
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(unsafe_target.status(), StatusCode::BAD_REQUEST);

    let run = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!(
                    "/api/v1/system/traffic-probes/targets/{target_id}/run"
                ))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(run.status(), StatusCode::OK);
    let run_value: serde_json::Value = serde_json::from_str(&body_text(run).await).unwrap();
    assert_eq!(run_value["status"], "healthy");
    assert_eq!(run_value["detail"]["status_code"], 204);
    assert_eq!(run_value["detail"]["reason"], "expected_status_matched");
    assert!(run_value["detail"]["duration_ms"].as_i64().unwrap() >= 0);

    let results = router
        .clone()
        .oneshot(
            Request::builder()
                .uri(format!(
                    "/api/v1/system/traffic-probes/results?target_id={target_id}&limit=5"
                ))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(results.status(), StatusCode::OK);
    let results_body = body_text(results).await;
    assert!(results_body.contains("\"healthy\""));
    assert!(results_body.contains("\"reqwest-http\""));

    let warning_target = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/system/traffic-probes/targets")
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({
                        "name": "本地异常端点",
                        "url": warning_probe_url,
                        "expected_status": 200
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(warning_target.status(), StatusCode::OK);
    let warning_target_value: serde_json::Value =
        serde_json::from_str(&body_text(warning_target).await).unwrap();
    let warning_target_id = warning_target_value["id"]
        .as_i64()
        .expect("warning target id");

    let warning_run = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!(
                    "/api/v1/system/traffic-probes/targets/{warning_target_id}/run"
                ))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(warning_run.status(), StatusCode::OK);
    let warning_run_value: serde_json::Value =
        serde_json::from_str(&body_text(warning_run).await).unwrap();
    assert_eq!(warning_run_value["status"], "warning");
    assert_eq!(warning_run_value["detail"]["status_code"], 503);
    assert_eq!(warning_run_value["detail"]["reason"], "status_mismatch");

    let token_alerts = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/traffic-probes/alerts")
                .header(AUTHORIZATION, format!("Bearer {token}"))
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(token_alerts.status(), StatusCode::FORBIDDEN);

    let alerts = router
        .clone()
        .oneshot(
            Request::builder()
                .uri(format!(
                    "/api/v1/system/traffic-probes/alerts?target_id={warning_target_id}&status=open&limit=5"
                ))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(alerts.status(), StatusCode::OK);
    let alert_values: Vec<serde_json::Value> =
        serde_json::from_str(&body_text(alerts).await).unwrap();
    assert_eq!(alert_values.len(), 1);
    assert_eq!(alert_values[0]["severity"], "warning");
    assert_eq!(alert_values[0]["status"], "open");
    assert_eq!(alert_values[0]["reason"], "status_mismatch");
    assert_eq!(alert_values[0]["detail"]["status_code"], 503);
    let alert_id = alert_values[0]["id"].as_i64().expect("alert id");

    let unauthenticated_events = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/traffic-probes/events")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(unauthenticated_events.status(), StatusCode::UNAUTHORIZED);

    let probe_events = router
        .clone()
        .oneshot(
            Request::builder()
                .uri(format!(
                    "/api/v1/system/traffic-probes/events?target_id={warning_target_id}&status=open&limit=5"
                ))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(probe_events.status(), StatusCode::OK);
    assert!(
        probe_events
            .headers()
            .get(CONTENT_TYPE)
            .and_then(|value| value.to_str().ok())
            .is_some_and(|value| value.starts_with("text/event-stream"))
    );
    let mut event_stream = probe_events.into_body().into_data_stream();
    let first_event = tokio::time::timeout(std::time::Duration::from_secs(2), event_stream.next())
        .await
        .expect("traffic probe event stream should yield immediately")
        .expect("traffic probe event stream should have a first item")
        .expect("traffic probe event chunk should be readable");
    let first_event_text = String::from_utf8(first_event.to_vec()).unwrap();
    assert!(first_event_text.contains("event: traffic_probe.alerts.snapshot"));
    assert!(first_event_text.contains("retry: 30000"));
    assert!(first_event_text.contains("\"event_type\":\"traffic_probe.alerts.snapshot\""));
    assert!(first_event_text.contains("\"reconnect_after_millis\":30000"));
    assert!(first_event_text.contains("\"reason\":\"status_mismatch\""));

    let ack_alert = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!(
                    "/api/v1/system/traffic-probes/alerts/{alert_id}/ack"
                ))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(ack_alert.status(), StatusCode::OK);
    assert!(body_text(ack_alert).await.contains("\"updated\":true"));

    let resolve_alert = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!(
                    "/api/v1/system/traffic-probes/alerts/{alert_id}/resolve"
                ))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(resolve_alert.status(), StatusCode::OK);
    assert!(body_text(resolve_alert).await.contains("\"updated\":true"));

    let resolved_alerts = router
        .clone()
        .oneshot(
            Request::builder()
                .uri(format!(
                    "/api/v1/system/traffic-probes/alerts?target_id={warning_target_id}&status=resolved&limit=5"
                ))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(resolved_alerts.status(), StatusCode::OK);
    let resolved_alert_values: Vec<serde_json::Value> =
        serde_json::from_str(&body_text(resolved_alerts).await).unwrap();
    assert_eq!(resolved_alert_values[0]["status"], "resolved");

    let recovering_target = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/system/traffic-probes/targets")
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({
                        "name": "本地恢复端点",
                        "url": recovering_probe_url,
                        "expected_status": 200
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(recovering_target.status(), StatusCode::OK);
    let recovering_target_value: serde_json::Value =
        serde_json::from_str(&body_text(recovering_target).await).unwrap();
    let recovering_target_id = recovering_target_value["id"]
        .as_i64()
        .expect("recovering target id");

    let first_recovery_run = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!(
                    "/api/v1/system/traffic-probes/targets/{recovering_target_id}/run"
                ))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(first_recovery_run.status(), StatusCode::OK);
    let first_recovery_value: serde_json::Value =
        serde_json::from_str(&body_text(first_recovery_run).await).unwrap();
    assert_eq!(first_recovery_value["status"], "warning");

    let second_recovery_run = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!(
                    "/api/v1/system/traffic-probes/targets/{recovering_target_id}/run"
                ))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(second_recovery_run.status(), StatusCode::OK);
    let second_recovery_value: serde_json::Value =
        serde_json::from_str(&body_text(second_recovery_run).await).unwrap();
    assert_eq!(second_recovery_value["status"], "healthy");

    let auto_resolved_alerts = router
        .clone()
        .oneshot(
            Request::builder()
                .uri(format!(
                    "/api/v1/system/traffic-probes/alerts?target_id={recovering_target_id}&status=resolved&limit=5"
                ))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(auto_resolved_alerts.status(), StatusCode::OK);
    let auto_resolved_values: Vec<serde_json::Value> =
        serde_json::from_str(&body_text(auto_resolved_alerts).await).unwrap();
    assert_eq!(auto_resolved_values.len(), 1);
    assert_eq!(auto_resolved_values[0]["status"], "resolved");
    assert!(auto_resolved_values[0]["resolved_at"].is_string());

    let targets = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/system/traffic-probes/targets")
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(targets.status(), StatusCode::OK);
    assert!(body_text(targets).await.contains("\"healthy\""));

    let delete_target = router
        .clone()
        .oneshot(
            Request::builder()
                .method("DELETE")
                .uri(format!("/api/v1/system/traffic-probes/targets/{target_id}"))
                .header(COOKIE, &cookie_pair)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(delete_target.status(), StatusCode::OK);
    assert!(body_text(delete_target).await.contains("\"deleted\":true"));

    let pool = sqlx::SqlitePool::connect(&db_url).await.unwrap();
    let unsafe_count: i64 = sqlx::query_scalar(
        "select count(*) from system_traffic_probe_targets where url like '%token=%'",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert_eq!(unsafe_count, 0);
    let status: String = sqlx::query_scalar(
        "select status from system_traffic_probe_results where target_id = ? limit 1",
    )
    .bind(target_id)
    .fetch_one(&pool)
    .await
    .unwrap();
    assert_eq!(status, "healthy");
    let alert_count: i64 = sqlx::query_scalar(
        "select count(*) from system_traffic_probe_alerts
         where target_id = ? and result_id is not null and status = 'resolved'",
    )
    .bind(warning_target_id)
    .fetch_one(&pool)
    .await
    .unwrap();
    assert_eq!(alert_count, 1);
    let operation_count: i64 = sqlx::query_scalar(
        "select count(*) from system_operation_records
         where path = '/api/v1/system/traffic-probes/targets' and actor_user_id is not null",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert!(operation_count >= 1);
}

#[tokio::test]
async fn scheduler_run_once_collects_real_traffic_probe_results() {
    let (app, db_url, _) = test_app().await;
    let router = app.router();
    let (cookie_pair, _) = create_initial_admin(router.clone()).await;
    let probe_url = spawn_probe_http_target("200 OK").await;

    let target = router
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/system/traffic-probes/targets")
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({
                        "name": "scheduler 本地端点",
                        "url": probe_url,
                        "expected_status": 200
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(target.status(), StatusCode::OK);

    let report = app
        .run_scheduled_tasks_once()
        .await
        .expect("run scheduler once");
    assert!(report.traffic_probe.enabled);
    assert_eq!(report.traffic_probe.scanned_targets, 1);
    assert_eq!(report.traffic_probe.recorded_results, 1);
    assert_eq!(report.traffic_probe.failed_targets, 0);
    assert!(report.operation_record_retention.enabled);
    assert_eq!(report.operation_record_retention.retention_days, 180);
    assert_eq!(report.operation_record_retention.deleted, 0);

    let pool = sqlx::SqlitePool::connect(&db_url).await.unwrap();
    let scheduler_result_count: i64 = sqlx::query_scalar(
        "select count(*)
         from system_traffic_probe_results
         where status = 'healthy'
           and detail_json like '%reqwest-http%'",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert_eq!(scheduler_result_count, 1);
}

#[tokio::test]
async fn pending_notification_flows_do_not_leak_raw_tokens_or_pollute_unknown_email() {
    let (app, db_url, notification_dir) = test_app().await;
    let router = app.router();
    let (cookie_pair, org_id) = create_initial_admin(router.clone()).await;

    let invite = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!("/api/v1/orgs/{org_id}/users/invitations"))
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({"email": "invitee@example.com", "role_code": "owner"}).to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(invite.status(), StatusCode::OK);
    let invitation_token_body = body_text(invite).await;
    assert!(!invitation_token_body.contains("invitation_token_"));
    assert!(invitation_token_body.contains("notification-outbox"));

    let forgot = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/password/forgot")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({"email": "owner@example.com"}).to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(forgot.status(), StatusCode::OK);
    let forgot_body = body_text(forgot).await;
    assert!(!forgot_body.contains("password_reset_token_"));
    assert!(forgot_body.contains("notification-outbox"));

    let verify = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/email-verifications")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({"email": "owner@example.com"}).to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(verify.status(), StatusCode::OK);
    let verify_body = body_text(verify).await;
    assert!(!verify_body.contains("email_verify_token_"));
    assert!(verify_body.contains("notification-outbox"));

    let pool = sqlx::SqlitePool::connect(&db_url).await.unwrap();
    let invitation_token_hash: String =
        sqlx::query_scalar("select token_hash from iam_invitations limit 1")
            .fetch_one(&pool)
            .await
            .unwrap();
    assert!(!invitation_token_hash.starts_with("invitation_token_"));
    let reset_count_before: i64 = sqlx::query_scalar("select count(*) from iam_password_resets")
        .fetch_one(&pool)
        .await
        .unwrap();
    let outbox_count_before_unknown: i64 =
        sqlx::query_scalar("select count(*) from iam_notification_outbox")
            .fetch_one(&pool)
            .await
            .unwrap();
    assert_eq!(outbox_count_before_unknown, 3);
    let unsafe_outbox_count: i64 = sqlx::query_scalar(
        "select count(*) from iam_notification_outbox
         where payload_json like '%invitation_token_%'
            or payload_json like '%password_reset_token_%'
            or payload_json like '%email_verify_token_%'",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert_eq!(unsafe_outbox_count, 0);
    let outbox_kinds: Vec<String> =
        sqlx::query_scalar("select related_kind from iam_notification_outbox order by id asc")
            .fetch_all(&pool)
            .await
            .unwrap();
    assert_eq!(
        outbox_kinds,
        vec![
            "iam_invitation".to_string(),
            "iam_password_reset".to_string(),
            "iam_email_verification".to_string()
        ]
    );
    let pending_secret_count: i64 = sqlx::query_scalar(
        "select count(*)
         from iam_notification_delivery_secrets
         where status = 'pending' and secret_ciphertext is not null",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert_eq!(pending_secret_count, 3);
    let unsafe_secret_count: i64 = sqlx::query_scalar(
        "select count(*)
         from iam_notification_delivery_secrets
         where secret_ciphertext like '%invitation_token_%'
            or secret_ciphertext like '%password_reset_token_%'
            or secret_ciphertext like '%email_verify_token_%'",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert_eq!(unsafe_secret_count, 0);

    let unknown = router
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/password/forgot")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({"email": "missing@example.com"}).to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(unknown.status(), StatusCode::OK);
    let reset_count_after: i64 = sqlx::query_scalar("select count(*) from iam_password_resets")
        .fetch_one(&pool)
        .await
        .unwrap();
    assert_eq!(reset_count_before, reset_count_after);
    let outbox_count_after_unknown: i64 =
        sqlx::query_scalar("select count(*) from iam_notification_outbox")
            .fetch_one(&pool)
            .await
            .unwrap();
    assert_eq!(outbox_count_before_unknown, outbox_count_after_unknown);

    let report = app
        .drain_notifications_once(Some(10))
        .await
        .expect("drain notifications");
    assert_eq!(report.driver, "file");
    assert_eq!(report.claimed, 3);
    assert_eq!(report.delivered, 3);
    assert_eq!(report.failed, 0);
    let delivered_file_count = fs::read_dir(&notification_dir)
        .expect("notification delivery dir")
        .count();
    assert_eq!(delivered_file_count, 3);

    let delivered_count: i64 = sqlx::query_scalar(
        "select count(*)
         from iam_notification_outbox
         where status = 'delivered'
           and locked_at is not null
           and delivered_at is not null
           and failed_at is null
           and failure_reason is null",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert_eq!(delivered_count, 3);
    let purged_secret_count: i64 = sqlx::query_scalar(
        "select count(*)
         from iam_notification_delivery_secrets
         where status = 'purged' and secret_ciphertext is null and purged_at is not null",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert_eq!(purged_secret_count, 3);

    let second_report = app
        .drain_notifications_once(Some(10))
        .await
        .expect("drain notifications again");
    assert_eq!(second_report.claimed, 0);
}

#[tokio::test]
async fn queue_notification_driver_writes_encrypted_envelope_without_raw_token() {
    let mut settings = test_settings();
    settings.auth.self_signup_enabled = true;
    settings.notification.driver = NotificationDriver::Queue;
    settings.notification.queue.dir = test_temp_dir("notification-queue")
        .to_string_lossy()
        .into_owned();
    settings.notification.queue.secret_key = "queue-smoke-secret-at-least-32-bytes".into();
    let db_url = settings.database.url.clone();
    let queue_dir = PathBuf::from(&settings.notification.queue.dir);
    let queue_secret = settings.notification.queue.secret_key.clone();
    let app = App::boot(settings).await.expect("boot queue app");
    let router = app.router();

    let register = router
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/register")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({
                        "email": "queue-signup@example.com",
                        "password": "change-me-123",
                        "display_name": "Queue Signup",
                        "organization_code": "queue-org",
                        "organization_name": "Queue Org"
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(register.status(), StatusCode::OK);
    let register_body = body_text(register).await;
    assert!(register_body.contains("notification-outbox"));
    assert!(!register_body.contains("email_verify_token_"));

    let report = app
        .drain_notifications_once(Some(10))
        .await
        .expect("drain queue notifications");
    assert_eq!(report.driver, "queue");
    assert_eq!(report.claimed, 1);
    assert_eq!(report.delivered, 1);
    assert_eq!(report.failed, 0);

    let queue_files: Vec<PathBuf> = fs::read_dir(&queue_dir)
        .expect("queue dir")
        .map(|entry| entry.expect("queue entry").path())
        .collect();
    assert_eq!(queue_files.len(), 1);
    let envelope = fs::read_to_string(&queue_files[0]).expect("queue envelope");
    assert!(envelope.contains("\"secret_ciphertext\""));
    assert!(!envelope.contains("email_verify_token_"));
    let value: serde_json::Value = serde_json::from_str(&envelope).expect("queue json");
    assert_eq!(
        value["template_code"],
        "iam.registration.email_verification"
    );
    let ciphertext = value["secret_ciphertext"].as_str().expect("ciphertext");
    let decrypted =
        crypto::decrypt_secret(ciphertext, &queue_secret).expect("decrypt queue secret");
    assert!(decrypted.starts_with("email_verify_token_"));

    let pool = sqlx::SqlitePool::connect(&db_url).await.unwrap();
    let purged_secret_count: i64 = sqlx::query_scalar(
        "select count(*)
         from iam_notification_delivery_secrets
         where status = 'purged' and secret_ciphertext is null and purged_at is not null",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert_eq!(purged_secret_count, 1);
}

#[tokio::test]
async fn self_signup_is_config_gated_transactional_and_requires_email_verification() {
    let disabled_app = App::boot(test_settings()).await.expect("boot disabled app");
    let disabled = disabled_app
        .router()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/register")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({
                        "email": "signup@example.com",
                        "password": "change-me-123",
                        "display_name": "Signup User",
                        "organization_code": "signup-main",
                        "organization_name": "Signup Main"
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(disabled.status(), StatusCode::FORBIDDEN);

    let mut settings = test_settings();
    settings.auth.self_signup_enabled = true;
    let db_url = settings.database.url.clone();
    let delivery_secret = settings.notification.delivery_secret_key.clone();
    let app = App::boot(settings).await.expect("boot signup app");
    let router = app.router();

    let register = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/register")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({
                        "email": "signup@example.com",
                        "password": "change-me-123",
                        "display_name": "Signup User",
                        "organization_code": "signup-main",
                        "organization_name": "Signup Main"
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(register.status(), StatusCode::OK);
    let register_body = body_text(register).await;
    assert!(register_body.contains("notification-outbox"));
    assert!(!register_body.contains("email_verify_token_"));

    let pool = sqlx::SqlitePool::connect(&db_url).await.unwrap();
    let user_row = sqlx::query(
        "select id, status, email_verified_at from iam_users where email = 'signup@example.com'",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    let user_id: i64 = user_row.get("id");
    let user_status: String = user_row.get("status");
    let verified_at: Option<String> = user_row.get("email_verified_at");
    assert_eq!(user_status, "pending_verification");
    assert!(verified_at.is_none());

    let membership_count: i64 = sqlx::query_scalar(
        "select count(*)
         from iam_memberships m
         join iam_organizations o on o.id = m.org_id
         where m.user_id = ? and m.role_code = 'owner' and o.code = 'signup-main'",
    )
    .bind(user_id)
    .fetch_one(&pool)
    .await
    .unwrap();
    assert_eq!(membership_count, 1);

    let unsafe_outbox_count: i64 = sqlx::query_scalar(
        "select count(*)
         from iam_notification_outbox
         where payload_json like '%email_verify_token_%'",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert_eq!(unsafe_outbox_count, 0);

    let secret_ciphertext: String = sqlx::query_scalar(
        "select s.secret_ciphertext
         from iam_notification_delivery_secrets s
         join iam_notification_outbox o on o.id = s.outbox_id
         where o.template_code = 'iam.registration.email_verification'
         limit 1",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert!(!secret_ciphertext.contains("email_verify_token_"));
    let raw_token =
        crypto::decrypt_secret(&secret_ciphertext, &delivery_secret).expect("decrypt signup token");
    assert!(raw_token.starts_with("email_verify_token_"));

    let login_before_verify = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/login")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({"identifier": "signup@example.com", "password": "change-me-123"})
                        .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(login_before_verify.status(), StatusCode::UNAUTHORIZED);

    let old_path_verify = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!(
                    "/api/v1/auth/email-verifications/{raw_token}/confirm"
                ))
                .header(CONTENT_TYPE, "application/json")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(old_path_verify.status(), StatusCode::NOT_FOUND);

    let verify = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/email-verifications/confirm")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(json!({ "token": raw_token }).to_string()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(verify.status(), StatusCode::OK);
    let verify_body = body_text(verify).await;
    assert!(verify_body.contains("\"verified\":true"));
    assert!(!verify_body.contains("email_verify_token_"));

    let leaked_operation_paths: i64 =
        sqlx::query_scalar("select count(*) from system_operation_records where path like ?")
            .bind(format!("%{raw_token}%"))
            .fetch_one(&pool)
            .await
            .unwrap();
    assert_eq!(leaked_operation_paths, 0);
    let templated_verify_path_count: i64 = sqlx::query_scalar(
        "select count(*)
         from system_operation_records
         where path = '/api/v1/auth/email-verifications/confirm'",
    )
    .fetch_one(&pool)
    .await
    .unwrap();
    assert!(templated_verify_path_count >= 1);

    let user_status_after: String = sqlx::query_scalar("select status from iam_users where id = ?")
        .bind(user_id)
        .fetch_one(&pool)
        .await
        .unwrap();
    assert_eq!(user_status_after, "active");

    let login_after_verify = router
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/login")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({"identifier": "signup@example.com", "password": "change-me-123"})
                        .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(login_after_verify.status(), StatusCode::OK);
    let login_body = body_text(login_after_verify).await;
    assert!(login_body.contains("\"authenticated\":true"));
    assert!(!login_body.contains("session_token_"));
    assert!(!login_body.contains("refresh_token_"));
}

#[tokio::test]
async fn invitation_acceptance_creates_user_membership_and_session_token_without_token_leak() {
    let (router, db_url) = test_router().await;
    let (cookie_pair, org_id) = create_initial_admin(router.clone()).await;

    let create_invite = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!("/api/v1/orgs/{org_id}/users/invitations"))
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, &cookie_pair)
                .body(Body::from(
                    json!({"email": "listed@example.com", "role_code": "owner"}).to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(create_invite.status(), StatusCode::OK);
    let create_invitation_token_body = body_text(create_invite).await;
    assert!(create_invitation_token_body.contains("\"role_code\":\"owner\""));
    assert!(!create_invitation_token_body.contains("invitation_token_"));

    let pool = sqlx::SqlitePool::connect(&db_url).await.unwrap();
    let listed_role: String =
        sqlx::query_scalar("select role_code from iam_invitations where email = ?")
            .bind("listed@example.com")
            .fetch_one(&pool)
            .await
            .unwrap();
    assert_eq!(listed_role, "owner");

    let raw_token = "invitation_token_accept_fixture_token";
    insert_invitation_fixture(&pool, org_id, "accepted@example.com", raw_token, "owner").await;

    let accept = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/invitations/accept")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({
                        "token": raw_token,
                        "password": "change-me-123",
                        "display_name": "Accepted User"
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(accept.status(), StatusCode::OK);
    let accepted_cookie = cookie_header_from_set_cookie(accept.headers());
    let accept_body = body_text(accept).await;
    assert!(!accept_body.contains(raw_token));
    let accept_value: serde_json::Value = serde_json::from_str(&accept_body).unwrap();
    assert_eq!(accept_value["user"]["email"], "accepted@example.com");
    assert_eq!(accept_value["organization"]["id"], org_id);
    assert!(
        accept_value["permissions"]
            .as_array()
            .expect("permissions")
            .iter()
            .any(|item| item == "user:read")
    );

    let session = router
        .clone()
        .oneshot(
            Request::builder()
                .uri("/api/v1/me/session")
                .header(COOKIE, accepted_cookie)
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(session.status(), StatusCode::OK);
    assert!(body_text(session).await.contains("accepted@example.com"));

    let accepted_status: String =
        sqlx::query_scalar("select status from iam_invitations where email = ?")
            .bind("accepted@example.com")
            .fetch_one(&pool)
            .await
            .unwrap();
    assert_eq!(accepted_status, "accepted");
    let membership_role: String = sqlx::query_scalar(
        "select m.role_code
         from iam_memberships m
         join iam_users u on u.id = m.user_id
         where u.email = ? and m.org_id = ?",
    )
    .bind("accepted@example.com")
    .bind(org_id)
    .fetch_one(&pool)
    .await
    .unwrap();
    assert_eq!(membership_role, "owner");

    let repeat = router
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/invitations/accept")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({
                        "token": raw_token,
                        "password": "change-me-123",
                        "display_name": "Accepted Again"
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(repeat.status(), StatusCode::UNAUTHORIZED);

    let conflict_token = "invitation_token_conflict_fixture_token";
    insert_invitation_fixture(&pool, org_id, "owner@example.com", conflict_token, "owner").await;
    let conflict = router
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/invitations/accept")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(
                    json!({
                        "token": conflict_token,
                        "password": "change-me-123",
                        "display_name": "Duplicate Owner"
                    })
                    .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(conflict.status(), StatusCode::CONFLICT);
    let conflict_status: String = sqlx::query_scalar(
        "select status from iam_invitations where email = ? order by id desc limit 1",
    )
    .bind("owner@example.com")
    .fetch_one(&pool)
    .await
    .unwrap();
    assert_eq!(conflict_status, "pending");
}

async fn body_text(response: http::Response<Body>) -> String {
    let bytes = to_bytes(response.into_body(), usize::MAX).await.unwrap();
    String::from_utf8(bytes.to_vec()).unwrap()
}

fn setup_step_status<'a>(value: &'a serde_json::Value, key: &str) -> &'a str {
    value["required_steps"]
        .as_array()
        .expect("setup steps")
        .iter()
        .find(|step| step["key"].as_str() == Some(key))
        .and_then(|step| step["status"].as_str())
        .expect("setup step status")
}

async fn insert_invitation_fixture(
    pool: &sqlx::SqlitePool,
    org_id: i64,
    email: &str,
    raw_token: &str,
    role_code: &str,
) {
    let settings = Settings::default();
    let token_hash = crypto::hash_secret(raw_token, &settings.auth.session_secret);
    let now = Utc::now().to_rfc3339();
    let expires_at = (Utc::now() + Duration::hours(1)).to_rfc3339();
    sqlx::query(
        "insert into iam_invitations(org_id, email, role_code, token_hash, status, expires_at, created_at)
         values (?, ?, ?, ?, 'pending', ?, ?)",
    )
    .bind(org_id)
    .bind(email)
    .bind(role_code)
    .bind(token_hash)
    .bind(expires_at)
    .bind(now)
    .execute(pool)
    .await
    .unwrap();
}

async fn create_initial_admin(router: axum::Router) -> (String, i64) {
    let payload = json!({
        "email": "owner@example.com",
        "password": "change-me-123",
        "display_name": "Owner",
        "organization_code": "main",
        "organization_name": "Main"
    });
    let response = router
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/auth/setup/initial-admin")
                .header(CONTENT_TYPE, "application/json")
                .body(Body::from(payload.to_string()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(response.status(), StatusCode::OK);
    let cookie = cookie_header_from_set_cookie(response.headers());
    let body = body_text(response).await;
    let value: serde_json::Value = serde_json::from_str(&body).unwrap();
    let org_id = value["organization"]["id"].as_i64().expect("org id");
    (cookie, org_id)
}

fn multipart_body(
    boundary: &str,
    category: Option<(&str, &str)>,
    display_name: Option<(&str, &str)>,
    file_field: &str,
    file_name: &str,
    mime_type: &str,
    file_bytes: &[u8],
) -> Vec<u8> {
    let mut body = Vec::new();
    for (name, value) in [category, display_name].into_iter().flatten() {
        body.extend_from_slice(format!("--{boundary}\r\n").as_bytes());
        body.extend_from_slice(
            format!("Content-Disposition: form-data; name=\"{name}\"\r\n\r\n").as_bytes(),
        );
        body.extend_from_slice(value.as_bytes());
        body.extend_from_slice(b"\r\n");
    }
    body.extend_from_slice(format!("--{boundary}\r\n").as_bytes());
    body.extend_from_slice(
        format!(
            "Content-Disposition: form-data; name=\"{file_field}\"; filename=\"{file_name}\"\r\n"
        )
        .as_bytes(),
    );
    body.extend_from_slice(format!("Content-Type: {mime_type}\r\n\r\n").as_bytes());
    body.extend_from_slice(file_bytes);
    body.extend_from_slice(b"\r\n");
    body.extend_from_slice(format!("--{boundary}--\r\n").as_bytes());
    body
}

async fn create_api_token(router: axum::Router, cookie_pair: &str, org_id: i64) -> String {
    let response = router
        .oneshot(
            Request::builder()
                .method("POST")
                .uri(format!("/api/v1/orgs/{org_id}/api-tokens"))
                .header(CONTENT_TYPE, "application/json")
                .header(COOKIE, cookie_pair)
                .body(Body::from(json!({"expires_in_days": 7}).to_string()))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(response.status(), StatusCode::OK);
    let body = body_text(response).await;
    let value: serde_json::Value = serde_json::from_str(&body).unwrap();
    value["token"].as_str().expect("api token").to_string()
}
