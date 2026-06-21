use std::path::{Component, Path, PathBuf};
use std::sync::Arc;

use axum::Router;
use axum::body::Body;
use axum::extract::{DefaultBodyLimit, Request, State};
use axum::http::{StatusCode, header};
use axum::response::{IntoResponse, Response};
use axum::routing::{delete, get, post, put};

use crate::app::AppState;
use crate::handler::http::{iam, probe, setup, system};
use crate::transport::http::route_registry;

pub fn build(state: Arc<AppState>) -> Router {
    let mut router = Router::new();
    for contract in route_registry::contracts(&state.settings) {
        let path = contract.axum_path();
        router = match contract.id.as_str() {
            "probe.health" => router.route(&path, get(probe::health)),
            "probe.ready" => router.route(&path, get(probe::ready)),
            "openapi.yaml" => router.route(&path, get(system::openapi)),
            "setup.status" => router.route(&path, get(setup::status)),
            "setup.schema" => router.route(&path, get(setup::schema)),
            "setup.config-checks" => router.route(&path, get(setup::config_checks)),
            "setup.run.list" => router.route(&path, get(setup::runs)),
            "setup.run.create" => router.route(&path, post(setup::create_run)),
            "setup.run.logs" => router.route(&path, get(setup::logs)),
            "setup.complete" => router.route(&path, post(setup::complete)),
            "iam.setup.status" => router.route(&path, get(iam::setup_status)),
            "iam.initial-admin" => router.route(&path, post(iam::initial_admin)),
            "iam.login" => router.route(&path, post(iam::login)),
            "iam.register" => router.route(&path, post(iam::register)),
            "iam.refresh" => router.route(&path, post(iam::refresh_session)),
            "iam.password.forgot" => router.route(&path, post(iam::forgot_password)),
            "iam.password.reset" => router.route(&path, post(iam::reset_password)),
            "iam.email-verification.request" => {
                router.route(&path, post(iam::request_email_verification))
            }
            "iam.email-verification.confirm" => {
                router.route(&path, post(iam::confirm_email_verification))
            }
            "iam.mfa.setup" => router.route(&path, post(iam::setup_mfa)),
            "iam.mfa.factors.list" => router.route(&path, get(iam::list_mfa_factors)),
            "iam.mfa.verify" => router.route(&path, post(iam::verify_mfa)),
            "iam.mfa.recovery-codes.list" => router.route(&path, get(iam::list_mfa_recovery_codes)),
            "iam.mfa.recovery-codes.rotate" => {
                router.route(&path, post(iam::rotate_mfa_recovery_codes))
            }
            "iam.mfa.revoke" => router.route(&path, delete(iam::revoke_mfa)),
            "iam.logout" => router.route(&path, post(iam::logout)),
            "iam.me.session" => router.route(&path, get(iam::me_session)),
            "iam.organizations.list" => router.route(&path, get(iam::list_organizations)),
            "iam.org-users.list" => router.route(&path, get(iam::list_org_users)),
            "iam.org-users.update" => router.route(&path, put(iam::update_org_user)),
            "iam.org-roles.list" => router.route(&path, get(iam::list_org_roles)),
            "iam.org-roles.create" => router.route(&path, post(iam::create_org_role)),
            "iam.org-roles.update" => router.route(&path, put(iam::update_org_role)),
            "iam.org-roles.delete" => router.route(&path, delete(iam::delete_org_role)),
            "iam.permissions.list" => router.route(&path, get(iam::list_permissions)),
            "iam.api-tokens.list" => router.route(&path, get(iam::list_api_tokens)),
            "iam.api-tokens.create" => router.route(&path, post(iam::create_api_token)),
            "iam.api-tokens.revoke" => router.route(&path, delete(iam::revoke_api_token)),
            "iam.invitations.list" => router.route(&path, get(iam::list_invitations)),
            "iam.invitations.create" => router.route(&path, post(iam::invite_user)),
            "iam.invitations.revoke" => router.route(&path, delete(iam::revoke_invitation)),
            "iam.invitations.accept" => router.route(&path, post(iam::accept_invitation)),
            "system.public-settings" => router.route(&path, get(system::public_settings)),
            "system.menus" => router.route(&path, get(system::menus)),
            "system.apis" => router.route(&path, get(system::apis)),
            "system.operation-records" => router.route(&path, get(system::operation_records)),
            "system.operation-records.export" => {
                router.route(&path, get(system::operation_records_export))
            }
            "system.operation-records.summary" => {
                router.route(&path, get(system::operation_record_summary))
            }
            "system.operation-records.prune" => {
                router.route(&path, post(system::prune_operation_records))
            }
            "system.server-status" => router.route(&path, get(system::server_status)),
            "system.metrics.prometheus" => router.route(&path, get(system::prometheus_metrics)),
            "system.configs.list" => router.route(&path, get(system::configs)),
            "system.configs.upsert" => router.route(&path, put(system::upsert_config)),
            "system.configs.delete" => router.route(&path, delete(system::delete_config)),
            "system.dictionaries.list" => router.route(&path, get(system::dictionaries)),
            "system.dictionaries.upsert" => router.route(&path, put(system::upsert_dictionary)),
            "system.dictionaries.delete" => router.route(&path, delete(system::delete_dictionary)),
            "system.parameters.list" => router.route(&path, get(system::parameters)),
            "system.parameters.upsert" => router.route(&path, put(system::upsert_parameter)),
            "system.parameters.delete" => router.route(&path, delete(system::delete_parameter)),
            "system.version-packages.list" => router.route(&path, get(system::version_packages)),
            "system.version-packages.create" => {
                router.route(&path, post(system::create_version_package))
            }
            "system.version-packages.releases" => {
                router.route(&path, get(system::version_release_events))
            }
            "system.version-packages.publish" => {
                router.route(&path, post(system::publish_version_package))
            }
            "system.version-packages.rollback" => {
                router.route(&path, post(system::rollback_version_package))
            }
            "system.version-packages.delete" => {
                router.route(&path, delete(system::delete_version_package))
            }
            "system.media-assets.list" => router.route(&path, get(system::media_assets)),
            "system.media-assets.create" => router.route(&path, post(system::create_media_asset)),
            "system.media-assets.upload" => router.route(
                &path,
                post(system::upload_media_asset).layer(DefaultBodyLimit::max(
                    state.settings.storage.max_upload_bytes,
                )),
            ),
            "system.media-assets.delete" => router.route(&path, delete(system::delete_media_asset)),
            "system.storage-objects.list" => router.route(&path, get(system::storage_objects)),
            "system.storage-objects.delete" => {
                router.route(&path, delete(system::delete_storage_object))
            }
            "system.traffic-probes.targets.list" => {
                router.route(&path, get(system::traffic_probe_targets))
            }
            "system.traffic-probes.targets.create" => {
                router.route(&path, post(system::create_traffic_probe_target))
            }
            "system.traffic-probes.targets.delete" => {
                router.route(&path, delete(system::delete_traffic_probe_target))
            }
            "system.traffic-probes.targets.run" => {
                router.route(&path, post(system::run_traffic_probe))
            }
            "system.traffic-probes.results" => {
                router.route(&path, get(system::traffic_probe_results))
            }
            "system.traffic-probes.alerts" => {
                router.route(&path, get(system::traffic_probe_alerts))
            }
            "system.traffic-probes.events" => {
                router.route(&path, get(system::traffic_probe_events))
            }
            "system.traffic-probes.alerts.ack" => {
                router.route(&path, post(system::acknowledge_traffic_probe_alert))
            }
            "system.traffic-probes.alerts.resolve" => {
                router.route(&path, post(system::resolve_traffic_probe_alert))
            }
            id => panic!("route registry contains unregistered handler: {id}"),
        };
    }
    router.fallback(webui_fallback).with_state(state)
}

async fn webui_fallback(State(state): State<Arc<AppState>>, request: Request) -> Response {
    let path = request.uri().path();
    if is_reserved_runtime_path(path) {
        return (StatusCode::NOT_FOUND, "Aoi[葵] API 路径不存在").into_response();
    }
    if !state.settings.webui.enabled {
        return (StatusCode::NOT_FOUND, "Aoi[葵] WebUI 未启用").into_response();
    }

    let dist_dir = PathBuf::from(&state.settings.webui.dist_dir);
    let Some(candidate) = static_candidate(&dist_dir, path) else {
        return (StatusCode::NOT_FOUND, "Aoi[葵] WebUI 路径无效").into_response();
    };
    let file_path = if candidate.exists() && candidate.is_file() {
        candidate
    } else if path_has_extension(path) {
        return (StatusCode::NOT_FOUND, "Aoi[葵] WebUI 静态资源不存在").into_response();
    } else {
        dist_dir.join("index.html")
    };

    match tokio::fs::read(&file_path).await {
        Ok(bytes) => {
            let content_type = content_type(&file_path);
            ([(header::CONTENT_TYPE, content_type)], Body::from(bytes)).into_response()
        }
        Err(_) => (StatusCode::NOT_FOUND, "Aoi[葵] WebUI 构建产物不存在").into_response(),
    }
}

fn is_reserved_runtime_path(path: &str) -> bool {
    path == "/api" || path.starts_with("/api/")
}

fn static_candidate(dist_dir: &Path, request_path: &str) -> Option<PathBuf> {
    let relative = request_path.trim_start_matches('/');
    if relative.is_empty() {
        return Some(dist_dir.join("index.html"));
    }
    let mut path = PathBuf::from(dist_dir);
    for component in Path::new(relative).components() {
        match component {
            Component::Normal(part) => path.push(part),
            _ => return None,
        }
    }
    Some(path)
}

fn path_has_extension(path: &str) -> bool {
    Path::new(path).extension().is_some()
}

fn content_type(path: &Path) -> &'static str {
    match path.extension().and_then(|extension| extension.to_str()) {
        Some("css") => "text/css; charset=utf-8",
        Some("html") => "text/html; charset=utf-8",
        Some("js") => "text/javascript; charset=utf-8",
        Some("json") => "application/json; charset=utf-8",
        Some("png") => "image/png",
        Some("svg") => "image/svg+xml",
        Some("webp") => "image/webp",
        _ => "application/octet-stream",
    }
}
