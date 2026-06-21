use aes_gcm::aead::{Aead, KeyInit};
use aes_gcm::{Aes256Gcm, Nonce};
use argon2::Argon2;
use argon2::password_hash::{PasswordHash, PasswordHasher, PasswordVerifier, SaltString};
use base64::Engine;
use base64::engine::general_purpose::STANDARD_NO_PAD;
use data_encoding::BASE32_NOPAD;
use hmac::{Hmac, Mac};
use percent_encoding::{NON_ALPHANUMERIC, utf8_percent_encode};
use rand_core::{OsRng, RngCore};
use sha1::Sha1;
use sha2::{Digest, Sha256};
use uuid::Uuid;

pub type CryptoResult<T> = Result<T, CryptoError>;

#[derive(Debug, thiserror::Error)]
pub enum CryptoError {
    #[error("密码哈希失败：{0}")]
    PasswordHash(String),
    #[error("密码哈希格式无效：{0}")]
    PasswordHashFormat(String),
    #[error("加密器初始化失败：{0}")]
    CipherInit(String),
    #[error("secret 加密失败：{0}")]
    SecretEncrypt(String),
    #[error("secret 密文编码无效：{0}")]
    SecretDecode(String),
    #[error("secret 密文无效")]
    InvalidSecret,
    #[error("secret 非 UTF-8：{0}")]
    SecretUtf8(String),
    #[error("TOTP secret 编码无效：{0}")]
    TotpSecretDecode(String),
    #[error("TOTP HMAC 初始化失败：{0}")]
    TotpMac(String),
}

type HmacSha1 = Hmac<Sha1>;

pub fn hash_password(password: &str) -> CryptoResult<String> {
    let salt = SaltString::generate(&mut OsRng);
    Argon2::default()
        .hash_password(password.as_bytes(), &salt)
        .map(|hash| hash.to_string())
        .map_err(|err| CryptoError::PasswordHash(err.to_string()))
}

pub fn verify_password(password: &str, hash: &str) -> CryptoResult<bool> {
    let parsed =
        PasswordHash::new(hash).map_err(|err| CryptoError::PasswordHashFormat(err.to_string()))?;
    Ok(Argon2::default()
        .verify_password(password.as_bytes(), &parsed)
        .is_ok())
}

pub fn new_secret(prefix: &str) -> String {
    format!(
        "{prefix}_{}{}",
        Uuid::new_v4().simple(),
        Uuid::new_v4().simple()
    )
}

pub fn hash_secret(secret: &str, pepper: &str) -> String {
    let mut hasher = Sha256::new();
    hasher.update(pepper.as_bytes());
    hasher.update(b":");
    hasher.update(secret.as_bytes());
    hex::encode(hasher.finalize())
}

pub fn sha256_hex(bytes: impl AsRef<[u8]>) -> String {
    let mut hasher = Sha256::new();
    hasher.update(bytes.as_ref());
    hex::encode(hasher.finalize())
}

pub fn encrypt_secret(plaintext: &str, key_material: &str) -> CryptoResult<String> {
    let cipher = Aes256Gcm::new_from_slice(&secret_key(key_material))
        .map_err(|err| CryptoError::CipherInit(err.to_string()))?;
    let mut nonce_bytes = [0_u8; 12];
    OsRng.fill_bytes(&mut nonce_bytes);
    let nonce = Nonce::from_slice(&nonce_bytes);
    let ciphertext = cipher
        .encrypt(nonce, plaintext.as_bytes())
        .map_err(|err| CryptoError::SecretEncrypt(err.to_string()))?;
    let mut sealed = Vec::with_capacity(nonce_bytes.len() + ciphertext.len());
    sealed.extend_from_slice(&nonce_bytes);
    sealed.extend_from_slice(&ciphertext);
    Ok(STANDARD_NO_PAD.encode(sealed))
}

pub fn decrypt_secret(ciphertext: &str, key_material: &str) -> CryptoResult<String> {
    let raw = STANDARD_NO_PAD
        .decode(ciphertext.as_bytes())
        .map_err(|err| CryptoError::SecretDecode(err.to_string()))?;
    if raw.len() < 12 {
        return Err(CryptoError::InvalidSecret);
    }
    let cipher = Aes256Gcm::new_from_slice(&secret_key(key_material))
        .map_err(|err| CryptoError::CipherInit(err.to_string()))?;
    let (nonce_bytes, encrypted) = raw.split_at(12);
    let plaintext = cipher
        .decrypt(Nonce::from_slice(nonce_bytes), encrypted)
        .map_err(|_| CryptoError::InvalidSecret)?;
    String::from_utf8(plaintext).map_err(|err| CryptoError::SecretUtf8(err.to_string()))
}

pub fn new_totp_secret() -> String {
    let mut raw = [0_u8; 20];
    OsRng.fill_bytes(&mut raw);
    BASE32_NOPAD.encode(&raw)
}

pub fn totp_otpauth_url(issuer: &str, account: &str, secret: &str) -> String {
    let label = format!(
        "{}:{}",
        utf8_percent_encode(issuer, NON_ALPHANUMERIC),
        utf8_percent_encode(account, NON_ALPHANUMERIC)
    );
    format!(
        "otpauth://totp/{label}?secret={secret}&issuer={}&algorithm=SHA1&digits=6&period=30",
        utf8_percent_encode(issuer, NON_ALPHANUMERIC)
    )
}

pub fn verify_totp_code(secret: &str, code: &str) -> CryptoResult<bool> {
    let code = code.trim();
    if code.len() != 6 || !code.chars().all(|ch| ch.is_ascii_digit()) {
        return Ok(false);
    }
    let now = chrono::Utc::now().timestamp();
    for offset in -1..=1 {
        let step = (now / 30).saturating_add(offset);
        if step >= 0 && totp_code_at_step(secret, step as u64)? == code {
            return Ok(true);
        }
    }
    Ok(false)
}

pub fn totp_code_for_now(secret: &str) -> CryptoResult<String> {
    let step = chrono::Utc::now().timestamp() as u64 / 30;
    totp_code_at_step(secret, step)
}

fn secret_key(key_material: &str) -> [u8; 32] {
    let mut hasher = Sha256::new();
    hasher.update(key_material.as_bytes());
    hasher.finalize().into()
}

fn totp_code_at_step(secret: &str, step: u64) -> CryptoResult<String> {
    let key = BASE32_NOPAD
        .decode(secret.trim().as_bytes())
        .map_err(|err| CryptoError::TotpSecretDecode(err.to_string()))?;
    let mut mac = <HmacSha1 as Mac>::new_from_slice(&key)
        .map_err(|err| CryptoError::TotpMac(err.to_string()))?;
    mac.update(&step.to_be_bytes());
    let hash = mac.finalize().into_bytes();
    let offset = usize::from(hash[19] & 0x0f);
    let binary = (u32::from(hash[offset] & 0x7f) << 24)
        | (u32::from(hash[offset + 1]) << 16)
        | (u32::from(hash[offset + 2]) << 8)
        | u32::from(hash[offset + 3]);
    Ok(format!("{:06}", binary % 1_000_000))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn sha256_hex_matches_known_vector() {
        assert_eq!(
            sha256_hex("abc"),
            "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
        );
    }
}
