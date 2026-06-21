use crate::app::{AppError, AppResult};

pub trait CryptoResultExt<T> {
    fn into_app(self) -> AppResult<T>;
}

impl<T> CryptoResultExt<T> for crypto::CryptoResult<T> {
    fn into_app(self) -> AppResult<T> {
        self.map_err(map_crypto_error)
    }
}

fn map_crypto_error(err: crypto::CryptoError) -> AppError {
    match err {
        crypto::CryptoError::InvalidSecret => AppError::Unauthorized,
        _ => AppError::Internal(format!("安全工具错误：{err}")),
    }
}
