use std::sync::Arc;

use axum::extract::{MatchedPath, Request, State};
use axum::middleware::Next;
use axum::response::Response;
use tracing::error;

use crate::app::AppState;
use crate::transport::http::request_context::{
    auth_credential_from_headers, request_context_from_headers,
};
use crate::transport::http::route_registry;

pub async fn record_operation(
    State(state): State<Arc<AppState>>,
    request: Request,
    next: Next,
) -> Response {
    let method = request.method().clone();
    let raw_path = request.uri().path().to_string();
    let recorded_path = request
        .extensions()
        .get::<MatchedPath>()
        .map(|matched| matched.as_str().to_string());
    let headers = request.headers().clone();
    let response = next.run(request).await;
    let status = response.status().as_u16();

    if route_registry::should_record_operation(&raw_path)
        && let Some(recorded_path) = recorded_path
    {
        let ctx = request_context_from_headers(&headers, &state.settings);
        let credential = auth_credential_from_headers(&headers, &state.settings);
        // 操作记录不能反过来影响主请求；坏 Cookie 或已撤销 token 只会让 actor 为空。
        let actor_user_id = state
            .iam
            .actor_user_id(credential, ctx)
            .await
            .unwrap_or(None);
        if let Err(err) = state
            .system
            .record_operation(actor_user_id, method.as_str(), &recorded_path, status)
            .await
        {
            error!(error = %err, "写入系统操作记录失败");
        }
    }

    response
}
