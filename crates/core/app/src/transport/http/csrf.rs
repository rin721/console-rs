use std::sync::Arc;

use axum::extract::{Request, State};
use axum::middleware::Next;
use axum::response::{IntoResponse, Response};
use http::Method;

use crate::app::{AppError, AppState};
use crate::handler::http::error::HttpError;
use crate::transport::http::request_context::cookie_value;

pub async fn require_csrf(
    State(state): State<Arc<AppState>>,
    request: Request,
    next: Next,
) -> Response {
    if !state.settings.auth.csrf.enabled || csrf_exempt(request.method(), request.uri().path()) {
        return next.run(request).await;
    }

    let settings = &state.settings;
    let headers = request.headers();
    let cookie_token = cookie_value(headers, &settings.auth.csrf.cookie_name);
    let header_token = headers
        .get(&settings.auth.csrf.header_name)
        .and_then(|value| value.to_str().ok())
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(ToOwned::to_owned);

    if cookie_token.is_none() || cookie_token != header_token {
        return HttpError::from(AppError::Forbidden).into_response();
    }

    next.run(request).await
}

fn csrf_exempt(method: &Method, path: &str) -> bool {
    matches!(method, &Method::GET | &Method::HEAD | &Method::OPTIONS) || !path.starts_with("/api/")
}
