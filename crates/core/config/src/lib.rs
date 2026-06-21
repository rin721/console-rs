use std::fmt;
use std::net::{IpAddr, Ipv4Addr, SocketAddr};
use std::path::PathBuf;

use config_rs::{Config, Environment, File};
use serde::{Deserialize, Serialize};
use thiserror::Error;

#[derive(Debug, Error)]
pub enum ConfigError {
    #[error("读取配置失败：{0}")]
    Load(#[from] config_rs::ConfigError),
    #[error("配置校验失败：{0}")]
    Invalid(String),
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
#[serde(default)]
pub struct Settings {
    pub app: AppConfig,
    pub server: ServerConfig,
    pub database: DatabaseConfig,
    pub migration: MigrationConfig,
    pub auth: AuthConfig,
    pub notification: NotificationConfig,
    pub scheduler: SchedulerConfig,
    pub i18n: I18nConfig,
    pub webui: WebUiConfig,
    pub storage: StorageConfig,
    pub audit: AuditConfig,
    pub observability: ObservabilityConfig,
}

impl Settings {
    pub fn load(path: Option<PathBuf>) -> Result<Self, ConfigError> {
        Self::load_with_secrets(path, None)
    }

    pub fn load_with_secrets(
        path: Option<PathBuf>,
        secrets_path: Option<PathBuf>,
    ) -> Result<Self, ConfigError> {
        Self::load_with_options(path, secrets_path, true)
    }

    fn load_with_options(
        path: Option<PathBuf>,
        secrets_path: Option<PathBuf>,
        include_env: bool,
    ) -> Result<Self, ConfigError> {
        let config_path = path.unwrap_or_else(|| PathBuf::from("configs/console.yaml"));
        let mut builder = Config::builder().add_source(File::from(config_path).required(false));

        // secrets 文件只承载部署密钥，优先级高于主配置，低于环境变量。
        if let Some(secrets_path) = secrets_path {
            builder = builder.add_source(File::from(secrets_path).required(true));
        }

        if include_env {
            builder = builder.add_source(Environment::with_prefix("CONSOLE").separator("__"));
        }

        let settings: Self = builder.build()?.try_deserialize()?;
        settings.validate()?;
        Ok(settings)
    }

    pub fn validate(&self) -> Result<(), ConfigError> {
        if self.app.product_code.trim().is_empty() {
            return Err(ConfigError::Invalid("app.product_code 不能为空".into()));
        }
        if self.app.product_code.contains("aoi-admin")
            || self.app.product_code.contains("go-scaffold")
            || self.app.product_code.contains("go-admin")
        {
            return Err(ConfigError::Invalid(
                "app.product_code 不能继续使用旧项目身份".into(),
            ));
        }
        self.database.validate()?;
        if self.auth.password_policy.min_length < 8 {
            return Err(ConfigError::Invalid(
                "auth.password_policy.min_length 不能小于 8".into(),
            ));
        }
        if !(4..=24).contains(&self.auth.mfa_recovery_code_count) {
            return Err(ConfigError::Invalid(
                "auth.mfa_recovery_code_count 必须在 4 到 24 之间".into(),
            ));
        }
        if self.app.environment == "production" {
            if weak_secret(&self.auth.session_secret) {
                return Err(ConfigError::Invalid(
                    "生产环境必须通过 CONSOLE__AUTH__SESSION_SECRET 提供强随机会话密钥".into(),
                ));
            }
            if weak_secret(&self.auth.mfa_secret_key) {
                return Err(ConfigError::Invalid(
                    "生产环境必须通过 CONSOLE__AUTH__MFA_SECRET_KEY 提供强随机 MFA 加密密钥".into(),
                ));
            }
            if !self.auth.cookie.secure {
                return Err(ConfigError::Invalid(
                    "生产环境 auth.cookie.secure 必须为 true".into(),
                ));
            }
            if !self.auth.refresh_cookie.secure {
                return Err(ConfigError::Invalid(
                    "生产环境 auth.refresh_cookie.secure 必须为 true".into(),
                ));
            }
            if !self.auth.csrf.enabled {
                return Err(ConfigError::Invalid(
                    "生产环境 auth.csrf.enabled 必须为 true".into(),
                ));
            }
            if !self.auth.csrf.secure {
                return Err(ConfigError::Invalid(
                    "生产环境 auth.csrf.secure 必须为 true".into(),
                ));
            }
        }
        self.auth.csrf.validate()?;
        self.storage.validate(self.app.environment.as_str())?;
        self.notification.validate(self.app.environment.as_str())?;
        if !(1..=1000).contains(&self.notification.batch_size) {
            return Err(ConfigError::Invalid(
                "notification.batch_size 必须在 1 到 1000 之间".into(),
            ));
        }
        if self.notification.lock_ttl_seconds < 1 {
            return Err(ConfigError::Invalid(
                "notification.lock_ttl_seconds 必须大于 0".into(),
            ));
        }
        if !(1..=20).contains(&self.notification.max_attempts) {
            return Err(ConfigError::Invalid(
                "notification.max_attempts 必须在 1 到 20 之间".into(),
            ));
        }
        if self.notification.retry_backoff_seconds < 1 {
            return Err(ConfigError::Invalid(
                "notification.retry_backoff_seconds 必须大于 0".into(),
            ));
        }
        if self.app.environment == "production"
            && weak_secret(&self.notification.delivery_secret_key)
        {
            return Err(ConfigError::Invalid(
                "生产环境必须通过 CONSOLE__NOTIFICATION__DELIVERY_SECRET_KEY 提供强随机通知投递密钥"
                    .into(),
            ));
        }
        if self.scheduler.traffic_probe_interval_seconds < 5 {
            return Err(ConfigError::Invalid(
                "scheduler.traffic_probe_interval_seconds 不能小于 5".into(),
            ));
        }
        if self.scheduler.event_stream_heartbeat_seconds < 5 {
            return Err(ConfigError::Invalid(
                "scheduler.event_stream_heartbeat_seconds 不能小于 5".into(),
            ));
        }
        self.audit.validate()?;
        self.observability.validate()?;
        Ok(())
    }

    pub fn socket_addr(&self) -> SocketAddr {
        SocketAddr::new(self.server.host, self.server.port)
    }

    pub fn redacted_summary(&self) -> RedactedSettings {
        RedactedSettings {
            product_name: self.app.product_name.clone(),
            product_code: self.app.product_code.clone(),
            environment: self.app.environment.clone(),
            listen: self.socket_addr().to_string(),
            database_driver: self.database.driver.to_string(),
            database_url: self.database.url.clone(),
            database_runtime: self.database.driver.runtime_support(),
            default_locale: self.i18n.default_locale.clone(),
            self_signup_enabled: self.auth.self_signup_enabled,
            cookie_name: self.auth.cookie.name.clone(),
            refresh_cookie_name: self.auth.refresh_cookie.name.clone(),
            notification_driver: self.notification.driver.to_string(),
            notification_batch_size: self.notification.batch_size,
            notification_max_attempts: self.notification.max_attempts,
            notification_retry_backoff_seconds: self.notification.retry_backoff_seconds,
            notification_local_dir: self.notification.local_dir.clone(),
            notification_queue_dir: self.notification.queue.dir.clone(),
            notification_smtp_host: non_empty_option(&self.notification.smtp.host),
            notification_smtp_from: non_empty_option(&self.notification.smtp.from),
            notification_smtp_tls: self.notification.smtp.tls.to_string(),
            scheduler_enabled: self.scheduler.enabled,
            scheduler_traffic_probe_interval_seconds: self.scheduler.traffic_probe_interval_seconds,
            scheduler_event_stream_heartbeat_seconds: self.scheduler.event_stream_heartbeat_seconds,
            webui_enabled: self.webui.enabled,
            webui_dist_dir: self.webui.dist_dir.clone(),
            storage_driver: self.storage.driver.to_string(),
            storage_local_dir: self.storage.local_dir.clone(),
            storage_max_upload_bytes: self.storage.max_upload_bytes,
            storage_s3_endpoint: non_empty_option(&self.storage.s3.endpoint),
            storage_s3_bucket: non_empty_option(&self.storage.s3.bucket),
            storage_s3_region: self.storage.s3.region.clone(),
            storage_s3_prefix: self.storage.s3.normalized_prefix(),
            audit_operation_record_retention_days: self.audit.operation_record_retention_days,
            audit_operation_record_prune_batch_size: self.audit.operation_record_prune_batch_size,
            csrf_enabled: self.auth.csrf.enabled,
            csrf_cookie_name: self.auth.csrf.cookie_name.clone(),
            csrf_header_name: self.auth.csrf.header_name.clone(),
            product_header: self.auth.context.product_header.clone(),
            client_type_header: self.auth.context.client_type_header.clone(),
            prometheus_scrape_token_configured: !self
                .observability
                .prometheus_scrape_token_hash
                .trim()
                .is_empty(),
        }
    }

    pub fn uses_weak_development_secrets(&self) -> bool {
        weak_secret(&self.auth.session_secret)
            || weak_secret(&self.auth.mfa_secret_key)
            || weak_secret(&self.notification.delivery_secret_key)
    }
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct AppConfig {
    pub product_name: String,
    pub product_code: String,
    pub version: String,
    pub environment: String,
}

impl Default for AppConfig {
    fn default() -> Self {
        Self {
            product_name: "Aoi[葵]".into(),
            product_code: "console".into(),
            version: env!("CARGO_PKG_VERSION").into(),
            environment: "local".into(),
        }
    }
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct ServerConfig {
    pub host: IpAddr,
    pub port: u16,
}

impl Default for ServerConfig {
    fn default() -> Self {
        Self {
            host: IpAddr::V4(Ipv4Addr::LOCALHOST),
            port: 8080,
        }
    }
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct DatabaseConfig {
    pub driver: DatabaseDriver,
    pub url: String,
    pub max_connections: u32,
}

impl Default for DatabaseConfig {
    fn default() -> Self {
        Self {
            driver: DatabaseDriver::Sqlite,
            url: "sqlite://data/console.sqlite".into(),
            max_connections: 5,
        }
    }
}

impl DatabaseConfig {
    fn validate(&self) -> Result<(), ConfigError> {
        if self.url.trim().is_empty() {
            return Err(ConfigError::Invalid("database.url 不能为空".into()));
        }
        if self.max_connections == 0 {
            return Err(ConfigError::Invalid(
                "database.max_connections 必须大于 0".into(),
            ));
        }
        self.driver.validate_url(&self.url)
    }
}

#[derive(Clone, Copy, Debug, Default, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum DatabaseDriver {
    #[default]
    Sqlite,
    Postgres,
    Mysql,
}

impl DatabaseDriver {
    pub fn as_str(self) -> &'static str {
        match self {
            Self::Sqlite => "sqlite",
            Self::Postgres => "postgres",
            Self::Mysql => "mysql",
        }
    }

    pub fn runtime_support(self) -> DatabaseRuntimeSupport {
        match self {
            Self::Sqlite => DatabaseRuntimeSupport {
                supported: true,
                status: "ready".into(),
                message: "当前运行时已装配 SQLite 连接池、repository 实现和迁移方言".into(),
                required_work: vec![],
            },
            Self::Postgres => DatabaseRuntimeSupport {
                supported: true,
                status: "ready".into(),
                message:
                    "当前运行时已装配 PostgreSQL 连接池、外部 repository set 和迁移方言"
                        .into(),
                required_work: vec![],
            },
            Self::Mysql => DatabaseRuntimeSupport {
                supported: true,
                status: "ready".into(),
                message: "当前运行时已装配 MySQL 连接池、外部 repository set 和迁移方言；业务写路径使用同连接 last_insert_id() 读取生成 ID".into(),
                required_work: vec![],
            },
        }
    }

    fn validate_url(self, url: &str) -> Result<(), ConfigError> {
        let trimmed = url.trim();
        let valid = match self {
            Self::Sqlite => trimmed.starts_with("sqlite:"),
            Self::Postgres => {
                trimmed.starts_with("postgres://") || trimmed.starts_with("postgresql://")
            }
            Self::Mysql => trimmed.starts_with("mysql://"),
        };
        if valid {
            return Ok(());
        }

        Err(ConfigError::Invalid(format!(
            "database.driver={} 与 database.url 协议不匹配",
            self
        )))
    }
}

impl fmt::Display for DatabaseDriver {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(self.as_str())
    }
}

#[derive(Clone, Debug, Serialize)]
pub struct DatabaseRuntimeSupport {
    pub supported: bool,
    pub status: String,
    pub message: String,
    pub required_work: Vec<String>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct MigrationConfig {
    pub auto_apply: bool,
}

impl Default for MigrationConfig {
    fn default() -> Self {
        Self { auto_apply: true }
    }
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct AuthConfig {
    pub self_signup_enabled: bool,
    pub session_secret: String,
    pub mfa_issuer: String,
    pub mfa_secret_key: String,
    pub mfa_recovery_code_count: usize,
    pub session_ttl_seconds: i64,
    pub refresh_ttl_seconds: i64,
    pub api_token_default_days: i64,
    pub invitation_ttl_seconds: i64,
    pub password_reset_ttl_seconds: i64,
    pub email_verification_ttl_seconds: i64,
    pub cookie: CookieConfig,
    pub refresh_cookie: CookieConfig,
    pub csrf: CsrfConfig,
    pub context: RequestContextConfig,
    pub password_policy: PasswordPolicyConfig,
}

impl Default for AuthConfig {
    fn default() -> Self {
        Self {
            self_signup_enabled: false,
            session_secret: "dev-session-secret-change-me-32-bytes".into(),
            mfa_issuer: "Aoi[葵]".into(),
            mfa_secret_key: "dev-mfa-secret-change-me-32-bytes".into(),
            mfa_recovery_code_count: 8,
            session_ttl_seconds: 86_400,
            refresh_ttl_seconds: 604_800,
            api_token_default_days: 30,
            invitation_ttl_seconds: 86_400,
            password_reset_ttl_seconds: 1_800,
            email_verification_ttl_seconds: 86_400,
            cookie: CookieConfig::default(),
            refresh_cookie: CookieConfig::refresh_default(),
            csrf: CsrfConfig::default(),
            context: RequestContextConfig::default(),
            password_policy: PasswordPolicyConfig::default(),
        }
    }
}

fn weak_secret(value: &str) -> bool {
    let lowered = value.to_ascii_lowercase();
    lowered.starts_with("dev-")
        || lowered.contains("${")
        || lowered.contains("change-me")
        || lowered.contains("changeme")
        || lowered.contains("replace")
        || lowered.contains("example")
        || value.len() < 32
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct CookieConfig {
    pub name: String,
    pub path: String,
    pub same_site: String,
    pub secure: bool,
}

impl Default for CookieConfig {
    fn default() -> Self {
        Self {
            name: "console_session".into(),
            path: "/".into(),
            same_site: "Lax".into(),
            secure: false,
        }
    }
}

impl CookieConfig {
    pub fn refresh_default() -> Self {
        Self {
            name: "console_refresh".into(),
            path: "/".into(),
            same_site: "Lax".into(),
            secure: false,
        }
    }
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct CsrfConfig {
    pub enabled: bool,
    pub cookie_name: String,
    pub header_name: String,
    pub path: String,
    pub same_site: String,
    pub secure: bool,
    pub ttl_seconds: i64,
}

impl Default for CsrfConfig {
    fn default() -> Self {
        Self {
            enabled: false,
            cookie_name: "console_csrf".into(),
            header_name: "X-CSRF-Token".into(),
            path: "/".into(),
            same_site: "Lax".into(),
            secure: false,
            ttl_seconds: 86_400,
        }
    }
}

impl CsrfConfig {
    fn validate(&self) -> Result<(), ConfigError> {
        if self.cookie_name.trim().is_empty() {
            return Err(ConfigError::Invalid(
                "auth.csrf.cookie_name 不能为空".into(),
            ));
        }
        if self.header_name.trim().is_empty() {
            return Err(ConfigError::Invalid(
                "auth.csrf.header_name 不能为空".into(),
            ));
        }
        if self.path.trim().is_empty() {
            return Err(ConfigError::Invalid("auth.csrf.path 不能为空".into()));
        }
        if self.ttl_seconds <= 0 {
            return Err(ConfigError::Invalid(
                "auth.csrf.ttl_seconds 必须大于 0".into(),
            ));
        }
        Ok(())
    }
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct RequestContextConfig {
    pub product_header: String,
    pub client_type_header: String,
    pub default_product_code: String,
    pub default_client_type: String,
}

impl Default for RequestContextConfig {
    fn default() -> Self {
        Self {
            product_header: "X-Console-Product-Code".into(),
            client_type_header: "X-Console-Client-Type".into(),
            default_product_code: "console".into(),
            default_client_type: "pc_web".into(),
        }
    }
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct PasswordPolicyConfig {
    pub min_length: usize,
}

impl Default for PasswordPolicyConfig {
    fn default() -> Self {
        Self { min_length: 8 }
    }
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct NotificationConfig {
    pub driver: NotificationDriver,
    pub local_dir: String,
    pub queue: QueueNotificationConfig,
    pub batch_size: i64,
    pub lock_ttl_seconds: i64,
    pub max_attempts: i64,
    pub retry_backoff_seconds: i64,
    pub delivery_secret_key: String,
    pub smtp: SmtpNotificationConfig,
}

impl Default for NotificationConfig {
    fn default() -> Self {
        Self {
            driver: NotificationDriver::File,
            local_dir: "data/notifications".into(),
            queue: QueueNotificationConfig::default(),
            batch_size: 50,
            lock_ttl_seconds: 300,
            max_attempts: 3,
            retry_backoff_seconds: 60,
            delivery_secret_key: "dev-notification-secret-change-me-32-bytes".into(),
            smtp: SmtpNotificationConfig::default(),
        }
    }
}

impl NotificationConfig {
    fn validate(&self, environment: &str) -> Result<(), ConfigError> {
        match self.driver {
            NotificationDriver::File => {
                if self.local_dir.trim().is_empty() {
                    return Err(ConfigError::Invalid(
                        "notification.local_dir 不能为空".into(),
                    ));
                }
            }
            NotificationDriver::Log => {}
            NotificationDriver::Smtp => self.smtp.validate(environment)?,
            NotificationDriver::Queue => self.queue.validate(environment)?,
        }
        if environment == "production"
            && !matches!(
                self.driver,
                NotificationDriver::Smtp | NotificationDriver::Queue
            )
        {
            return Err(ConfigError::Invalid(
                "生产环境不能使用 file/log notification driver，请配置 notification.driver=smtp 或 queue"
                    .into(),
            ));
        }
        Ok(())
    }
}

#[derive(Clone, Copy, Debug, Default, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum NotificationDriver {
    #[default]
    File,
    Log,
    Smtp,
    Queue,
}

impl NotificationDriver {
    pub fn as_str(self) -> &'static str {
        match self {
            Self::File => "file",
            Self::Log => "log",
            Self::Smtp => "smtp",
            Self::Queue => "queue",
        }
    }
}

impl fmt::Display for NotificationDriver {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(self.as_str())
    }
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct QueueNotificationConfig {
    pub dir: String,
    pub secret_key: String,
}

impl Default for QueueNotificationConfig {
    fn default() -> Self {
        Self {
            dir: "data/notification-queue".into(),
            secret_key: "dev-notification-queue-secret-change-me-32-bytes".into(),
        }
    }
}

impl QueueNotificationConfig {
    fn validate(&self, environment: &str) -> Result<(), ConfigError> {
        if self.dir.trim().is_empty() {
            return Err(ConfigError::Invalid(
                "notification.queue.dir 不能为空".into(),
            ));
        }
        if self.secret_key.trim().is_empty() {
            return Err(ConfigError::Invalid(
                "notification.queue.secret_key 不能为空".into(),
            ));
        }
        if environment == "production" && weak_secret(&self.secret_key) {
            return Err(ConfigError::Invalid(
                "生产环境必须通过 secrets/env 设置强随机 notification.queue.secret_key".into(),
            ));
        }
        Ok(())
    }
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct SmtpNotificationConfig {
    pub host: String,
    pub port: u16,
    pub username: String,
    pub password: String,
    pub from: String,
    pub tls: SmtpTlsMode,
}

impl Default for SmtpNotificationConfig {
    fn default() -> Self {
        Self {
            host: String::new(),
            port: 587,
            username: String::new(),
            password: String::new(),
            from: String::new(),
            tls: SmtpTlsMode::StartTls,
        }
    }
}

impl SmtpNotificationConfig {
    fn validate(&self, environment: &str) -> Result<(), ConfigError> {
        if self.host.trim().is_empty() {
            return Err(ConfigError::Invalid(
                "notification.smtp.host 不能为空".into(),
            ));
        }
        if self.port == 0 {
            return Err(ConfigError::Invalid(
                "notification.smtp.port 必须大于 0".into(),
            ));
        }
        if self.from.trim().is_empty() || !self.from.contains('@') {
            return Err(ConfigError::Invalid(
                "notification.smtp.from 必须是有效邮箱地址".into(),
            ));
        }
        let has_username = !self.username.trim().is_empty();
        let has_password = !self.password.trim().is_empty();
        if has_username != has_password {
            return Err(ConfigError::Invalid(
                "notification.smtp.username 与 password 必须同时配置或同时为空".into(),
            ));
        }
        if environment == "production" && self.tls == SmtpTlsMode::None {
            return Err(ConfigError::Invalid(
                "生产环境 notification.smtp.tls 不能为 none".into(),
            ));
        }
        Ok(())
    }
}

#[derive(Clone, Copy, Debug, Default, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum SmtpTlsMode {
    #[default]
    StartTls,
    Tls,
    None,
}

impl SmtpTlsMode {
    pub fn as_str(self) -> &'static str {
        match self {
            Self::StartTls => "start_tls",
            Self::Tls => "tls",
            Self::None => "none",
        }
    }
}

impl fmt::Display for SmtpTlsMode {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(self.as_str())
    }
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct SchedulerConfig {
    pub enabled: bool,
    pub run_on_start: bool,
    pub traffic_probe_interval_seconds: u64,
    pub event_stream_heartbeat_seconds: u64,
}

impl Default for SchedulerConfig {
    fn default() -> Self {
        Self {
            enabled: false,
            run_on_start: false,
            traffic_probe_interval_seconds: 300,
            event_stream_heartbeat_seconds: 15,
        }
    }
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct I18nConfig {
    pub default_locale: String,
    pub supported_locales: Vec<String>,
}

impl Default for I18nConfig {
    fn default() -> Self {
        Self {
            default_locale: "zh-CN".into(),
            supported_locales: vec!["zh-CN".into(), "en".into()],
        }
    }
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct WebUiConfig {
    pub enabled: bool,
    pub dist_dir: String,
}

impl Default for WebUiConfig {
    fn default() -> Self {
        Self {
            enabled: true,
            dist_dir: "web/app/dist".into(),
        }
    }
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct StorageConfig {
    pub driver: StorageDriver,
    pub local_dir: String,
    pub max_upload_bytes: usize,
    pub s3: S3StorageConfig,
}

impl Default for StorageConfig {
    fn default() -> Self {
        Self {
            driver: StorageDriver::Local,
            local_dir: "data/media".into(),
            max_upload_bytes: 8 * 1024 * 1024,
            s3: S3StorageConfig::default(),
        }
    }
}

impl StorageConfig {
    fn validate(&self, environment: &str) -> Result<(), ConfigError> {
        if self.max_upload_bytes == 0 {
            return Err(ConfigError::Invalid(
                "storage.max_upload_bytes 必须大于 0".into(),
            ));
        }
        match self.driver {
            StorageDriver::Local => {
                if self.local_dir.trim().is_empty() {
                    return Err(ConfigError::Invalid("storage.local_dir 不能为空".into()));
                }
            }
            StorageDriver::S3 => self.s3.validate(environment)?,
        }
        Ok(())
    }
}

#[derive(Clone, Copy, Debug, Default, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum StorageDriver {
    #[default]
    Local,
    S3,
}

impl StorageDriver {
    pub fn as_str(self) -> &'static str {
        match self {
            Self::Local => "local",
            Self::S3 => "s3",
        }
    }
}

impl fmt::Display for StorageDriver {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(self.as_str())
    }
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct S3StorageConfig {
    pub endpoint: String,
    pub bucket: String,
    pub region: String,
    pub access_key_id: String,
    pub secret_access_key: String,
    pub force_path_style: bool,
    pub allow_http: bool,
    pub prefix: String,
}

impl Default for S3StorageConfig {
    fn default() -> Self {
        Self {
            endpoint: String::new(),
            bucket: String::new(),
            region: "us-east-1".into(),
            access_key_id: String::new(),
            secret_access_key: String::new(),
            force_path_style: true,
            allow_http: false,
            prefix: "media".into(),
        }
    }
}

impl S3StorageConfig {
    fn validate(&self, environment: &str) -> Result<(), ConfigError> {
        let endpoint = self.endpoint.trim();
        if endpoint.is_empty() {
            return Err(ConfigError::Invalid("storage.s3.endpoint 不能为空".into()));
        }
        if !(endpoint.starts_with("https://") || endpoint.starts_with("http://")) {
            return Err(ConfigError::Invalid(
                "storage.s3.endpoint 必须是 http 或 https 地址".into(),
            ));
        }
        if endpoint.contains('?') || endpoint.contains('#') {
            return Err(ConfigError::Invalid(
                "storage.s3.endpoint 不能包含 query 或 fragment".into(),
            ));
        }
        let authority = endpoint_authority(endpoint);
        if authority.is_empty() || authority.contains('@') {
            return Err(ConfigError::Invalid(
                "storage.s3.endpoint 必须包含主机名且不能包含用户名或密码".into(),
            ));
        }
        if endpoint.starts_with("http://") && !self.allow_http {
            return Err(ConfigError::Invalid(
                "storage.s3.endpoint 使用 http 时必须显式设置 storage.s3.allow_http=true".into(),
            ));
        }
        if environment == "production" && self.allow_http {
            return Err(ConfigError::Invalid(
                "生产环境 storage.s3.allow_http 必须为 false".into(),
            ));
        }
        if self.bucket.trim().is_empty() {
            return Err(ConfigError::Invalid("storage.s3.bucket 不能为空".into()));
        }
        if self.region.trim().is_empty() {
            return Err(ConfigError::Invalid("storage.s3.region 不能为空".into()));
        }
        if self.access_key_id.trim().is_empty() {
            return Err(ConfigError::Invalid(
                "storage.s3.access_key_id 不能为空".into(),
            ));
        }
        if self.secret_access_key.trim().is_empty() {
            return Err(ConfigError::Invalid(
                "storage.s3.secret_access_key 不能为空".into(),
            ));
        }
        if environment == "production" && weak_secret(&self.secret_access_key) {
            return Err(ConfigError::Invalid(
                "生产环境必须通过 secrets/env 注入强随机 storage.s3.secret_access_key".into(),
            ));
        }
        let prefix = self.normalized_prefix();
        if prefix.contains("..") || prefix.starts_with('/') || prefix.ends_with('/') {
            return Err(ConfigError::Invalid(
                "storage.s3.prefix 不能包含路径穿越、开头斜杠或结尾斜杠".into(),
            ));
        }
        Ok(())
    }

    pub fn normalized_prefix(&self) -> String {
        self.prefix
            .trim()
            .trim_matches('/')
            .replace('\\', "/")
            .split('/')
            .filter(|part| !part.is_empty())
            .collect::<Vec<_>>()
            .join("/")
    }
}

fn endpoint_authority(endpoint: &str) -> &str {
    endpoint
        .split_once("://")
        .map(|(_, rest)| rest)
        .unwrap_or(endpoint)
        .split('/')
        .next()
        .unwrap_or_default()
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct AuditConfig {
    pub operation_record_retention_days: i64,
    pub operation_record_prune_batch_size: i64,
}

impl Default for AuditConfig {
    fn default() -> Self {
        Self {
            operation_record_retention_days: 180,
            operation_record_prune_batch_size: 1000,
        }
    }
}

impl AuditConfig {
    fn validate(&self) -> Result<(), ConfigError> {
        if !(1..=3650).contains(&self.operation_record_retention_days) {
            return Err(ConfigError::Invalid(
                "audit.operation_record_retention_days 必须在 1 到 3650 天之间".into(),
            ));
        }
        if !(1..=10_000).contains(&self.operation_record_prune_batch_size) {
            return Err(ConfigError::Invalid(
                "audit.operation_record_prune_batch_size 必须在 1 到 10000 之间".into(),
            ));
        }
        Ok(())
    }
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct ObservabilityConfig {
    pub level: String,
    pub format: String,
    pub prometheus_scrape_token_hash: String,
}

impl Default for ObservabilityConfig {
    fn default() -> Self {
        Self {
            level: "info".into(),
            format: "console".into(),
            prometheus_scrape_token_hash: String::new(),
        }
    }
}

impl ObservabilityConfig {
    fn validate(&self) -> Result<(), ConfigError> {
        let token_hash = self.prometheus_scrape_token_hash.trim();
        if token_hash.is_empty() {
            return Ok(());
        }
        if token_hash.len() != 64 || !token_hash.chars().all(|ch| ch.is_ascii_hexdigit()) {
            return Err(ConfigError::Invalid(
                "observability.prometheus_scrape_token_hash 必须是 64 位 SHA-256 hex 哈希，不能写入原始 token"
                    .into(),
            ));
        }
        Ok(())
    }
}

#[derive(Debug, Serialize)]
pub struct RedactedSettings {
    pub product_name: String,
    pub product_code: String,
    pub environment: String,
    pub listen: String,
    pub database_driver: String,
    pub database_url: String,
    pub database_runtime: DatabaseRuntimeSupport,
    pub default_locale: String,
    pub self_signup_enabled: bool,
    pub cookie_name: String,
    pub refresh_cookie_name: String,
    pub notification_driver: String,
    pub notification_batch_size: i64,
    pub notification_max_attempts: i64,
    pub notification_retry_backoff_seconds: i64,
    pub notification_local_dir: String,
    pub notification_queue_dir: String,
    pub notification_smtp_host: Option<String>,
    pub notification_smtp_from: Option<String>,
    pub notification_smtp_tls: String,
    pub scheduler_enabled: bool,
    pub scheduler_traffic_probe_interval_seconds: u64,
    pub scheduler_event_stream_heartbeat_seconds: u64,
    pub webui_enabled: bool,
    pub webui_dist_dir: String,
    pub storage_driver: String,
    pub storage_local_dir: String,
    pub storage_max_upload_bytes: usize,
    pub storage_s3_endpoint: Option<String>,
    pub storage_s3_bucket: Option<String>,
    pub storage_s3_region: String,
    pub storage_s3_prefix: String,
    pub audit_operation_record_retention_days: i64,
    pub audit_operation_record_prune_batch_size: i64,
    pub csrf_enabled: bool,
    pub csrf_cookie_name: String,
    pub csrf_header_name: String,
    pub product_header: String,
    pub client_type_header: String,
    pub prometheus_scrape_token_configured: bool,
}

fn non_empty_option(value: &str) -> Option<String> {
    let trimmed = value.trim();
    if trimmed.is_empty() {
        None
    } else {
        Some(trimmed.to_string())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use std::time::{SystemTime, UNIX_EPOCH};

    #[test]
    fn database_driver_accepts_matching_url_schemes() {
        let cases = [
            (DatabaseDriver::Sqlite, "sqlite://data/console.sqlite"),
            (DatabaseDriver::Sqlite, "sqlite::memory:"),
            (DatabaseDriver::Postgres, "postgres://localhost/console"),
            (DatabaseDriver::Postgres, "postgresql://localhost/console"),
            (DatabaseDriver::Mysql, "mysql://localhost/console"),
        ];

        for (driver, url) in cases {
            let mut settings = Settings::default();
            settings.database.driver = driver;
            settings.database.url = url.into();
            assert!(
                settings.validate().is_ok(),
                "driver={driver} url={url} should validate"
            );
        }
    }

    #[test]
    fn database_driver_rejects_mismatched_url_scheme() {
        let mut settings = Settings::default();
        settings.database.driver = DatabaseDriver::Postgres;
        settings.database.url = "sqlite://data/console.sqlite".into();

        let error = settings
            .validate()
            .expect_err("mismatched scheme must fail");
        assert!(error.to_string().contains("database.driver=postgres"));
    }

    #[test]
    fn database_requires_positive_connection_limit() {
        let mut settings = Settings::default();
        settings.database.max_connections = 0;

        let error = settings.validate().expect_err("zero connections must fail");
        assert!(error.to_string().contains("database.max_connections"));
    }

    #[test]
    fn redacted_summary_exposes_database_driver_without_secrets() {
        let summary = Settings::default().redacted_summary();

        assert_eq!(summary.database_driver, "sqlite");
        assert_eq!(summary.database_url, "sqlite://data/console.sqlite");
        assert!(summary.database_runtime.supported);
        assert_eq!(summary.database_runtime.status, "ready");
        assert!(summary.database_runtime.required_work.is_empty());
        assert_eq!(summary.notification_queue_dir, "data/notification-queue");
        assert_eq!(summary.audit_operation_record_retention_days, 180);
        assert_eq!(summary.audit_operation_record_prune_batch_size, 1000);
        assert!(!summary.prometheus_scrape_token_configured);
    }

    #[test]
    fn audit_retention_policy_requires_safe_bounds() {
        let mut settings = Settings::default();
        settings.audit.operation_record_retention_days = 0;
        let error = settings
            .validate()
            .expect_err("zero retention days must fail");
        assert!(
            error
                .to_string()
                .contains("audit.operation_record_retention_days")
        );

        settings.audit.operation_record_retention_days = 180;
        settings.audit.operation_record_prune_batch_size = 0;
        let error = settings
            .validate()
            .expect_err("zero prune batch size must fail");
        assert!(
            error
                .to_string()
                .contains("audit.operation_record_prune_batch_size")
        );

        settings.audit.operation_record_prune_batch_size = 1000;
        assert!(settings.validate().is_ok());
    }

    #[test]
    fn observability_scrape_token_hash_must_be_hash_not_raw_token() {
        let mut settings = Settings::default();
        settings.observability.prometheus_scrape_token_hash = "raw-token-value".into();

        let error = settings.validate().expect_err("raw scrape token must fail");
        assert!(
            error
                .to_string()
                .contains("observability.prometheus_scrape_token_hash")
        );

        settings.observability.prometheus_scrape_token_hash =
            "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef".into();
        assert!(settings.validate().is_ok());
        assert!(
            settings
                .redacted_summary()
                .prometheus_scrape_token_configured
        );
    }

    #[test]
    fn database_runtime_support_reports_ready_external_drivers() {
        let postgres = DatabaseDriver::Postgres.runtime_support();
        assert!(postgres.supported);
        assert_eq!(postgres.status, "ready");
        assert!(postgres.required_work.is_empty());
        assert!(postgres.message.contains("PostgreSQL 连接池"));

        let mysql = DatabaseDriver::Mysql.runtime_support();
        assert!(mysql.supported);
        assert_eq!(mysql.status, "ready");
        assert!(mysql.required_work.is_empty());
        assert!(mysql.message.contains("last_insert_id()"));
    }

    #[test]
    fn storage_driver_accepts_local_and_s3_config() {
        let mut settings = Settings::default();
        assert!(settings.validate().is_ok());

        settings.storage.driver = StorageDriver::S3;
        settings.storage.s3.endpoint = "http://127.0.0.1:9000".into();
        settings.storage.s3.bucket = "console-media".into();
        settings.storage.s3.region = "local".into();
        settings.storage.s3.access_key_id = "minio-access".into();
        settings.storage.s3.secret_access_key = "minio-secret-for-local-development".into();
        settings.storage.s3.allow_http = true;
        settings.storage.s3.prefix = "/tenant-media/".into();

        assert!(settings.validate().is_ok());
        let summary = settings.redacted_summary();
        assert_eq!(summary.storage_driver, "s3");
        assert_eq!(
            summary.storage_s3_endpoint.as_deref(),
            Some("http://127.0.0.1:9000")
        );
        assert_eq!(summary.storage_s3_bucket.as_deref(), Some("console-media"));
        assert_eq!(summary.storage_s3_region, "local");
        assert_eq!(summary.storage_s3_prefix, "tenant-media");
    }

    #[test]
    fn s3_storage_rejects_missing_or_unsafe_config() {
        let mut settings = Settings::default();
        settings.storage.driver = StorageDriver::S3;

        let error = settings.validate().expect_err("missing endpoint must fail");
        assert!(error.to_string().contains("storage.s3.endpoint"));

        settings.storage.s3.endpoint = "http://user:pass@127.0.0.1:9000".into();
        settings.storage.s3.allow_http = true;
        settings.storage.s3.bucket = "console-media".into();
        settings.storage.s3.access_key_id = "minio-access".into();
        settings.storage.s3.secret_access_key = "minio-secret-for-local-development".into();

        let error = settings
            .validate()
            .expect_err("endpoint credentials must fail");
        assert!(error.to_string().contains("用户名或密码"));

        settings.storage.s3.endpoint = "https://s3.example.com?token=secret".into();
        let error = settings.validate().expect_err("endpoint query must fail");
        assert!(error.to_string().contains("query"));

        settings.storage.s3.endpoint = "http://127.0.0.1:9000".into();
        settings.storage.s3.allow_http = false;

        let error = settings
            .validate()
            .expect_err("http endpoint without allow_http must fail");
        assert!(error.to_string().contains("allow_http=true"));
    }

    #[test]
    fn production_s3_storage_rejects_http_and_placeholder_secret() {
        let mut settings = Settings::default();
        settings.app.environment = "production".into();
        settings.auth.session_secret = "prod-session-secret-at-least-32-bytes".into();
        settings.auth.mfa_secret_key = "prod-mfa-secret-at-least-32-bytes".into();
        settings.auth.cookie.secure = true;
        settings.auth.refresh_cookie.secure = true;
        settings.auth.csrf.enabled = true;
        settings.auth.csrf.secure = true;
        settings.notification.driver = NotificationDriver::Smtp;
        settings.notification.delivery_secret_key =
            "prod-notification-secret-at-least-32-bytes".into();
        settings.notification.smtp.host = "smtp.example.com".into();
        settings.notification.smtp.from = "Console <noreply@example.com>".into();
        settings.storage.driver = StorageDriver::S3;
        settings.storage.s3.endpoint = "http://127.0.0.1:9000".into();
        settings.storage.s3.bucket = "console-media".into();
        settings.storage.s3.access_key_id = "prod-access".into();
        settings.storage.s3.secret_access_key = "replace-with-storage-secret".into();
        settings.storage.s3.allow_http = true;

        let error = settings
            .validate()
            .expect_err("production http endpoint must fail");
        assert!(error.to_string().contains("storage.s3.allow_http"));

        settings.storage.s3.endpoint = "https://s3.example.com".into();
        settings.storage.s3.allow_http = false;
        let error = settings
            .validate()
            .expect_err("placeholder storage secret must fail");
        assert!(error.to_string().contains("storage.s3.secret_access_key"));
    }

    #[test]
    fn smtp_notification_driver_requires_smtp_config() {
        let mut settings = Settings::default();
        settings.notification.driver = NotificationDriver::Smtp;

        let error = settings
            .validate()
            .expect_err("missing smtp host must fail");

        assert!(error.to_string().contains("notification.smtp.host"));
    }

    #[test]
    fn smtp_notification_driver_accepts_complete_local_config() {
        let mut settings = Settings::default();
        settings.notification.driver = NotificationDriver::Smtp;
        settings.notification.smtp.host = "smtp.example.com".into();
        settings.notification.smtp.from = "Console <noreply@example.com>".into();
        settings.notification.smtp.username = "smtp-user".into();
        settings.notification.smtp.password = "smtp-password".into();

        assert!(settings.validate().is_ok());
        let summary = settings.redacted_summary();
        assert_eq!(summary.notification_driver, "smtp");
        assert_eq!(
            summary.notification_smtp_host.as_deref(),
            Some("smtp.example.com")
        );
        assert_eq!(
            summary.notification_smtp_from.as_deref(),
            Some("Console <noreply@example.com>")
        );
        assert_eq!(summary.notification_smtp_tls, "start_tls");
    }

    #[test]
    fn queue_notification_driver_requires_dir_and_strong_production_secret() {
        let mut settings = Settings::default();
        settings.notification.driver = NotificationDriver::Queue;
        settings.notification.queue.dir = String::new();

        let error = settings.validate().expect_err("empty queue dir must fail");
        assert!(error.to_string().contains("notification.queue.dir"));

        settings.notification.queue.dir = "data/notification-queue".into();
        assert!(settings.validate().is_ok());
        let summary = settings.redacted_summary();
        assert_eq!(summary.notification_driver, "queue");
        assert_eq!(summary.notification_queue_dir, "data/notification-queue");

        settings.app.environment = "production".into();
        settings.auth.session_secret = "prod-session-secret-at-least-32-bytes".into();
        settings.auth.mfa_secret_key = "prod-mfa-secret-at-least-32-bytes".into();
        settings.auth.cookie.secure = true;
        settings.auth.refresh_cookie.secure = true;
        settings.auth.csrf.enabled = true;
        settings.auth.csrf.secure = true;
        settings.notification.delivery_secret_key =
            "prod-notification-secret-at-least-32-bytes".into();

        let error = settings
            .validate()
            .expect_err("placeholder queue secret must fail in production");
        assert!(error.to_string().contains("notification.queue.secret_key"));

        settings.notification.queue.secret_key =
            "prod-notification-queue-secret-at-least-32-bytes".into();
        assert!(settings.validate().is_ok());
    }

    #[test]
    fn notification_retry_policy_requires_positive_bounds() {
        let mut settings = Settings::default();
        settings.notification.max_attempts = 0;

        let error = settings
            .validate()
            .expect_err("zero max attempts must fail");
        assert!(error.to_string().contains("notification.max_attempts"));

        settings.notification.max_attempts = 3;
        settings.notification.retry_backoff_seconds = 0;
        let error = settings
            .validate()
            .expect_err("zero retry backoff must fail");
        assert!(
            error
                .to_string()
                .contains("notification.retry_backoff_seconds")
        );
    }

    #[test]
    fn production_rejects_local_notification_driver_and_plain_smtp() {
        let mut settings = Settings::default();
        settings.app.environment = "production".into();
        settings.auth.session_secret = "prod-session-secret-at-least-32-bytes".into();
        settings.auth.mfa_secret_key = "prod-mfa-secret-at-least-32-bytes".into();
        settings.auth.cookie.secure = true;
        settings.auth.refresh_cookie.secure = true;
        settings.auth.csrf.enabled = true;
        settings.auth.csrf.secure = true;
        settings.notification.delivery_secret_key =
            "prod-notification-secret-at-least-32-bytes".into();

        let error = settings
            .validate()
            .expect_err("file driver must fail in production");
        assert!(
            error
                .to_string()
                .contains("notification.driver=smtp 或 queue")
        );

        settings.notification.driver = NotificationDriver::Smtp;
        settings.notification.smtp.host = "smtp.example.com".into();
        settings.notification.smtp.from = "Console <noreply@example.com>".into();
        settings.notification.smtp.tls = SmtpTlsMode::None;

        let error = settings
            .validate()
            .expect_err("plain smtp must fail in production");
        assert!(error.to_string().contains("notification.smtp.tls"));
    }

    #[test]
    fn production_requires_enabled_secure_csrf() {
        let mut settings = Settings::default();
        settings.app.environment = "production".into();
        settings.auth.session_secret = "prod-session-secret-at-least-32-bytes".into();
        settings.auth.mfa_secret_key = "prod-mfa-secret-at-least-32-bytes".into();
        settings.auth.cookie.secure = true;
        settings.auth.refresh_cookie.secure = true;

        let error = settings
            .validate()
            .expect_err("disabled csrf must fail in production");
        assert!(error.to_string().contains("auth.csrf.enabled"));

        settings.auth.csrf.enabled = true;
        let error = settings
            .validate()
            .expect_err("insecure csrf cookie must fail in production");
        assert!(error.to_string().contains("auth.csrf.secure"));
    }

    #[test]
    fn secrets_file_overrides_production_secret_values() {
        let config_path = write_temp_config(
            "production-base",
            r#"
app:
  environment: production
database:
  driver: sqlite
  url: sqlite://data/test.sqlite
auth:
  cookie:
    secure: true
  refresh_cookie:
    secure: true
  csrf:
    enabled: true
    secure: true
notification:
  driver: smtp
  smtp:
    host: smtp.example.com
    from: Console <noreply@example.com>
"#,
        );

        let error = Settings::load_with_options(Some(config_path.clone()), None, false)
            .expect_err("production config without secrets must fail");
        assert!(error.to_string().contains("CONSOLE__AUTH__SESSION_SECRET"));

        let secrets_path = write_temp_config(
            "production-secrets",
            r#"
auth:
  session_secret: prod-session-secret-at-least-32-bytes-from-secrets
  mfa_secret_key: prod-mfa-secret-at-least-32-bytes-from-secrets
notification:
  delivery_secret_key: prod-notification-secret-at-least-32-bytes-from-secrets
  queue:
    secret_key: prod-notification-queue-secret-at-least-32-bytes-from-secrets
"#,
        );

        let settings = Settings::load_with_options(Some(config_path), Some(secrets_path), false)
            .expect("production config with secrets should validate");
        assert_eq!(
            settings.auth.session_secret,
            "prod-session-secret-at-least-32-bytes-from-secrets"
        );
        assert_eq!(
            settings.notification.delivery_secret_key,
            "prod-notification-secret-at-least-32-bytes-from-secrets"
        );
        assert_eq!(
            settings.notification.queue.secret_key,
            "prod-notification-queue-secret-at-least-32-bytes-from-secrets"
        );
    }

    #[test]
    fn production_example_rejects_missing_or_placeholder_secrets() {
        let production_path = workspace_file("configs/console.production.example.yaml");

        let error = Settings::load_with_options(Some(production_path.clone()), None, false)
            .expect_err("production example without secrets must fail");
        assert!(error.to_string().contains("CONSOLE__AUTH__SESSION_SECRET"));

        let secrets_template_path = workspace_file("configs/console.secrets.example.yaml");
        let error =
            Settings::load_with_options(Some(production_path), Some(secrets_template_path), false)
                .expect_err("placeholder secrets template must fail in production");
        assert!(error.to_string().contains("CONSOLE__AUTH__SESSION_SECRET"));
    }

    #[test]
    fn production_example_accepts_real_external_secrets() {
        let production_path = workspace_file("configs/console.production.example.yaml");
        let secrets_path = write_temp_config(
            "production-example-secrets",
            r#"
auth:
  session_secret: prod-session-key-0123456789abcdef0123456789
  mfa_secret_key: prod-mfa-key-0123456789abcdef0123456789
notification:
  delivery_secret_key: prod-notification-key-0123456789abcdef0123456789
  queue:
    secret_key: prod-notification-queue-key-0123456789abcdef
  smtp:
    host: smtp.example.net
    from: Console <noreply@example.net>
    username: smtp-user
    password: smtp-password-loaded-from-secret-store
"#,
        );

        let settings =
            Settings::load_with_options(Some(production_path), Some(secrets_path), false)
                .expect("production example should pass with real external secrets");

        assert_eq!(settings.app.environment, "production");
        assert!(settings.auth.cookie.secure);
        assert!(settings.auth.refresh_cookie.secure);
        assert!(settings.auth.csrf.enabled);
        assert!(settings.auth.csrf.secure);
        assert_eq!(settings.notification.driver, NotificationDriver::Smtp);
        assert_eq!(settings.notification.smtp.host, "smtp.example.net");
    }

    fn write_temp_config(name: &str, content: &str) -> PathBuf {
        let nanos = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("system time")
            .as_nanos();
        let dir = std::env::temp_dir()
            .join("console-config-tests")
            .join(format!("{name}-{nanos}"));
        fs::create_dir_all(&dir).expect("create config test dir");
        let path = dir.join("config.yaml");
        fs::write(&path, content).expect("write config test file");
        path
    }

    fn workspace_file(path: &str) -> PathBuf {
        PathBuf::from(env!("CARGO_MANIFEST_DIR"))
            .join("../../..")
            .join(path)
    }
}
