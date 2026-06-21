use std::sync::Arc;

use axum::Json;
use axum::extract::{Path, State};

use crate::app::AppState;
use crate::domain::setup::{CompleteSetupRequest, CreateSetupRunRequest, SetupSchema};
use crate::handler::http::error::HttpResult;
use crate::transport::http::request_context::locale_from_headers;

pub async fn status(
    State(state): State<Arc<AppState>>,
) -> HttpResult<Json<crate::domain::setup::SetupStatus>> {
    Ok(Json(state.setup.status().await?))
}

pub async fn schema(
    State(state): State<Arc<AppState>>,
    headers: axum::http::HeaderMap,
) -> HttpResult<Json<SetupSchema>> {
    let locale = locale_from_headers(&headers, &state.settings);
    Ok(Json(state.setup.schema(locale)))
}

pub async fn config_checks(
    State(state): State<Arc<AppState>>,
) -> HttpResult<Json<crate::domain::setup::SetupConfigCheckSummary>> {
    Ok(Json(state.setup.config_checks()))
}

pub async fn create_run(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<CreateSetupRunRequest>,
) -> HttpResult<Json<crate::domain::setup::SetupRun>> {
    Ok(Json(state.setup.create_run(payload).await?))
}

pub async fn runs(
    State(state): State<Arc<AppState>>,
) -> HttpResult<Json<Vec<crate::domain::setup::SetupRun>>> {
    Ok(Json(state.setup.runs().await?))
}

pub async fn logs(
    State(state): State<Arc<AppState>>,
    Path(run_id): Path<String>,
) -> HttpResult<Json<Vec<crate::domain::setup::SetupStepLog>>> {
    Ok(Json(state.setup.logs(run_id).await?))
}

pub async fn complete(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<CompleteSetupRequest>,
) -> HttpResult<Json<crate::domain::setup::CompleteSetupResult>> {
    Ok(Json(state.setup.complete(payload).await?))
}
