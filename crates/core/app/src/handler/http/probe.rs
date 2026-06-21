use std::sync::Arc;

use axum::Json;
use axum::extract::State;
use serde_json::json;

use crate::app::AppState;
use crate::handler::http::error::HttpResult;

pub async fn health(State(state): State<Arc<AppState>>) -> Json<serde_json::Value> {
    Json(json!({
        "status": "ok",
        "product": state.settings.app.product_code,
        "version": state.settings.app.version,
    }))
}

pub async fn ready(State(state): State<Arc<AppState>>) -> HttpResult<Json<serde_json::Value>> {
    state.database.ping().await?;
    let setup = state.setup.status().await?;
    Ok(Json(json!({
        "ready": true,
        "database": "ok",
        "setupCompleted": setup.completed,
        "hasInitialAdmin": setup.has_initial_admin,
    })))
}
