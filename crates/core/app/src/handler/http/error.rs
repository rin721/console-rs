use axum::Json;
use axum::http::StatusCode;
use axum::response::{IntoResponse, Response};
use serde::Serialize;
use tracing::error;

use crate::app::AppError;

pub type HttpResult<T> = Result<T, HttpError>;

#[derive(Debug)]
pub struct HttpError(AppError);

impl From<AppError> for HttpError {
    fn from(value: AppError) -> Self {
        Self(value)
    }
}

impl IntoResponse for HttpError {
    fn into_response(self) -> Response {
        let (status, code, message) = match &self.0 {
            AppError::Validation(message) => (
                StatusCode::BAD_REQUEST,
                "VALIDATION_FAILED",
                message.clone(),
            ),
            AppError::Unauthorized => {
                (StatusCode::UNAUTHORIZED, "UNAUTHORIZED", self.0.to_string())
            }
            AppError::MfaRequired => (StatusCode::UNAUTHORIZED, "MFA_REQUIRED", self.0.to_string()),
            AppError::Forbidden => (StatusCode::FORBIDDEN, "FORBIDDEN", self.0.to_string()),
            AppError::NotFound(message) => (StatusCode::NOT_FOUND, "NOT_FOUND", message.clone()),
            AppError::Conflict(message) => (StatusCode::CONFLICT, "CONFLICT", message.clone()),
            AppError::Infrastructure(err) => {
                error!(error = %err, "基础设施错误");
                (
                    StatusCode::INTERNAL_SERVER_ERROR,
                    "INFRASTRUCTURE_ERROR",
                    "基础设施错误".into(),
                )
            }
            AppError::Storage(err) => {
                error!(error = %err, "存储基础设施错误");
                (
                    StatusCode::INTERNAL_SERVER_ERROR,
                    "STORAGE_ERROR",
                    "存储基础设施错误".into(),
                )
            }
            AppError::Internal(message) => {
                error!(error = %message, "内部错误");
                (
                    StatusCode::INTERNAL_SERVER_ERROR,
                    "INTERNAL_ERROR",
                    "内部错误".into(),
                )
            }
        };

        (status, Json(ErrorBody { code, message })).into_response()
    }
}

#[derive(Debug, Serialize)]
struct ErrorBody {
    code: &'static str,
    message: String,
}
