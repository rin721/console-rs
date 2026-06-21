pub type AppResult<T> = Result<T, AppError>;

#[derive(Debug, thiserror::Error)]
pub enum AppError {
    #[error("参数校验失败：{0}")]
    Validation(String),
    #[error("未登录或会话已失效")]
    Unauthorized,
    #[error("需要输入多因素认证验证码")]
    MfaRequired,
    #[error("无权访问该资源")]
    Forbidden,
    #[error("资源不存在：{0}")]
    NotFound(String),
    #[error("业务冲突：{0}")]
    Conflict(String),
    #[error("基础设施错误：{0}")]
    Infrastructure(String),
    #[error("存储基础设施错误：{0}")]
    Storage(String),
    #[error("内部错误：{0}")]
    Internal(String),
}

impl AppError {
    pub fn infrastructure(error: impl std::fmt::Display) -> Self {
        Self::Infrastructure(error.to_string())
    }

    pub fn storage(error: impl std::fmt::Display) -> Self {
        Self::Storage(error.to_string())
    }
}

impl From<sqlx::Error> for AppError {
    fn from(error: sqlx::Error) -> Self {
        Self::infrastructure(error)
    }
}

impl From<std::io::Error> for AppError {
    fn from(error: std::io::Error) -> Self {
        Self::storage(error)
    }
}
