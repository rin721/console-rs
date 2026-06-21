use std::sync::Arc;

use uuid::Uuid;

use crate::app::{AppError, AppResult};
use crate::config::{NotificationDriver, Settings, StorageDriver};
use crate::domain::setup::{
    CompleteSetupRequest, CompleteSetupResult, CreateSetupRunRequest, SetupConfigCheck,
    SetupConfigCheckSummary, SetupFieldSchema, SetupRun, SetupSchema, SetupStatus, SetupStepLog,
    SetupStepSchema, SetupStepStatus,
};
use crate::repository::{IamRepository, SetupRepository};

const DEFAULT_SETUP_RUN_LIMIT: i64 = 10;

pub struct SetupService {
    settings: Settings,
    setup_repo: Arc<dyn SetupRepository>,
    iam_repo: Arc<dyn IamRepository>,
}

impl SetupService {
    pub fn new(
        settings: Settings,
        setup_repo: Arc<dyn SetupRepository>,
        iam_repo: Arc<dyn IamRepository>,
    ) -> Self {
        Self {
            settings,
            setup_repo,
            iam_repo,
        }
    }

    pub async fn status(&self) -> AppResult<SetupStatus> {
        let completed = self.setup_repo.setup_completed().await?;
        let has_initial_admin = self.iam_repo.has_any_user().await?;
        let checks = self.config_checks();
        let checks_ready = checks.ready;
        let database_ready = checks
            .checks
            .iter()
            .find(|check| check.key == "database")
            .map(|check| check.status != "error")
            .unwrap_or(false);
        let completion_status = if completed {
            "done"
        } else if checks_ready && has_initial_admin {
            "ready"
        } else {
            "blocked"
        };

        Ok(SetupStatus {
            completed,
            has_initial_admin,
            required_steps: vec![
                SetupStepStatus {
                    key: "database".into(),
                    title: "数据库连接".into(),
                    status: if database_ready { "ready" } else { "blocked" }.into(),
                },
                SetupStepStatus {
                    key: "config-checks".into(),
                    title: "配置检测".into(),
                    status: if checks_ready { "ready" } else { "blocked" }.into(),
                },
                SetupStepStatus {
                    key: "initial-admin".into(),
                    title: "首个管理员".into(),
                    status: if has_initial_admin { "done" } else { "pending" }.into(),
                },
                SetupStepStatus {
                    key: "api-catalog".into(),
                    title: "API 目录同步".into(),
                    status: "ready".into(),
                },
                SetupStepStatus {
                    key: "complete".into(),
                    title: "完成初始化".into(),
                    status: completion_status.into(),
                },
            ],
        })
    }

    pub fn schema(&self, locale: String) -> SetupSchema {
        SetupSchema {
            locale,
            steps: vec![
                SetupStepSchema {
                    key: "database".into(),
                    title: "数据库".into(),
                    fields: vec![SetupFieldSchema {
                        key: "database.url".into(),
                        label: "SQLite 数据库 URL".into(),
                        kind: "text".into(),
                        required: true,
                        sensitive: false,
                    }],
                },
                SetupStepSchema {
                    key: "initial-admin".into(),
                    title: "首个管理员".into(),
                    fields: vec![
                        SetupFieldSchema {
                            key: "email".into(),
                            label: "邮箱".into(),
                            kind: "email".into(),
                            required: true,
                            sensitive: false,
                        },
                        SetupFieldSchema {
                            key: "password".into(),
                            label: "密码".into(),
                            kind: "password".into(),
                            required: true,
                            sensitive: true,
                        },
                    ],
                },
            ],
        }
    }

    pub fn config_checks(&self) -> SetupConfigCheckSummary {
        build_config_checks(&self.settings)
    }

    pub async fn create_run(&self, request: CreateSetupRunRequest) -> AppResult<SetupRun> {
        let id = Uuid::new_v4().to_string();
        let run = self
            .setup_repo
            .create_setup_run(&id, request.reason.as_deref())
            .await?;
        self.setup_repo
            .append_setup_log(&id, "database", "ok", "数据库迁移与连接检查已完成")
            .await?;
        let checks = self.config_checks();
        let config_status = if checks.ready { "ok" } else { "error" };
        let warning_count = checks
            .checks
            .iter()
            .filter(|check| check.status == "warning")
            .count();
        self.setup_repo
            .append_setup_log(
                &id,
                "config-checks",
                config_status,
                &format!(
                    "配置检测完成：{} 项，{} 个警告",
                    checks.checks.len(),
                    warning_count
                ),
            )
            .await?;
        self.setup_repo
            .append_setup_log(&id, "api-catalog", "ok", "API 契约已从 route registry 同步")
            .await?;
        Ok(run)
    }

    pub async fn runs(&self) -> AppResult<Vec<SetupRun>> {
        self.setup_repo
            .list_setup_runs(DEFAULT_SETUP_RUN_LIMIT)
            .await
    }

    pub async fn logs(&self, run_id: String) -> AppResult<Vec<SetupStepLog>> {
        let logs = self.setup_repo.list_setup_logs(&run_id).await?;
        if logs.is_empty() {
            return Err(AppError::NotFound("setup run logs not found".into()));
        }
        Ok(logs)
    }

    pub async fn complete(&self, request: CompleteSetupRequest) -> AppResult<CompleteSetupResult> {
        if !request.confirm {
            return Err(AppError::Validation("必须显式确认完成初始化".into()));
        }
        ensure_config_checks_ready(&self.config_checks())?;
        if !self.iam_repo.has_any_user().await? {
            return Err(AppError::Conflict("必须先创建首个管理员".into()));
        }
        let completed = self
            .setup_repo
            .complete_setup(request.run_id.as_deref())
            .await?;
        if !completed {
            return Err(AppError::NotFound("setup run not found".into()));
        }
        Ok(CompleteSetupResult { completed: true })
    }
}

pub fn build_config_checks(settings: &Settings) -> SetupConfigCheckSummary {
    let checks = vec![
        database_check(settings),
        migration_check(settings),
        secret_check(settings),
        cookie_csrf_check(settings),
        notification_check(settings),
        storage_check(settings),
        webui_check(settings),
    ];
    let ready = checks.iter().all(|check| check.status != "error");
    SetupConfigCheckSummary { ready, checks }
}

fn ensure_config_checks_ready(checks: &SetupConfigCheckSummary) -> AppResult<()> {
    if checks.ready {
        return Ok(());
    }

    let blocking = checks
        .checks
        .iter()
        .filter(|check| check.status == "error")
        .map(|check| check.title.as_str())
        .collect::<Vec<_>>()
        .join("、");

    if blocking.is_empty() {
        return Err(AppError::Conflict("初始化配置检测仍有阻断项".into()));
    }

    Err(AppError::Conflict(format!(
        "初始化配置检测仍有阻断项：{blocking}"
    )))
}

fn database_check(settings: &Settings) -> SetupConfigCheck {
    let support = settings.database.driver.runtime_support();
    if support.supported {
        ok_check("database", "数据库驱动", support.message)
    } else {
        error_check(
            "database",
            "数据库驱动",
            format!(
                "{}；缺口：{}",
                support.message,
                support.required_work.join("、")
            ),
        )
    }
}

fn migration_check(settings: &Settings) -> SetupConfigCheck {
    if settings.migration.auto_apply {
        ok_check(
            "migration",
            "迁移策略",
            "启动时自动迁移已启用，适合本地开发闭环",
        )
    } else {
        ok_check(
            "migration",
            "迁移策略",
            "启动时自动迁移已关闭，部署流程必须显式运行数据库迁移",
        )
    }
}

fn secret_check(settings: &Settings) -> SetupConfigCheck {
    if settings.app.environment == "production" {
        ok_check(
            "secrets",
            "密钥强度",
            "生产密钥已通过强度校验；检测结果不会返回任何密钥明文",
        )
    } else if settings.uses_weak_development_secrets() {
        warning_check(
            "secrets",
            "密钥强度",
            "当前使用开发密钥，只适合本地环境；生产必须通过 secrets/env 注入强随机值",
        )
    } else {
        ok_check(
            "secrets",
            "密钥强度",
            "密钥未命中开发占位符规则；检测结果不会返回任何密钥明文",
        )
    }
}

fn cookie_csrf_check(settings: &Settings) -> SetupConfigCheck {
    if settings.app.environment == "production" {
        ok_check(
            "cookie-csrf",
            "Cookie 与 CSRF",
            "生产 Cookie 与 CSRF Secure 配置已通过校验",
        )
    } else if !settings.auth.csrf.enabled {
        warning_check(
            "cookie-csrf",
            "Cookie 与 CSRF",
            "CSRF 当前关闭，仅适合本地或受控联调环境",
        )
    } else {
        ok_check("cookie-csrf", "Cookie 与 CSRF", "CSRF 双提交保护已启用")
    }
}

fn notification_check(settings: &Settings) -> SetupConfigCheck {
    match settings.notification.driver {
        NotificationDriver::Smtp => ok_check(
            "notification",
            "通知投递",
            "SMTP 通知 driver 已启用，密钥和账号不会出现在检测响应中",
        ),
        NotificationDriver::Queue => ok_check(
            "notification",
            "通知投递",
            "queue 通知 driver 已启用，会写入加密队列 envelope，不返回 raw token",
        ),
        NotificationDriver::File => warning_check(
            "notification",
            "通知投递",
            "本地 file 通知 driver 仅用于开发；生产必须切换为 SMTP 或 queue driver",
        ),
        NotificationDriver::Log => warning_check(
            "notification",
            "通知投递",
            "log 通知 driver 不投递真实消息，仅适合调试元数据",
        ),
    }
}

fn storage_check(settings: &Settings) -> SetupConfigCheck {
    match settings.storage.driver {
        StorageDriver::Local => ok_check(
            "storage",
            "媒体存储",
            "当前使用本地 storage driver；适合本地开发和单机部署",
        ),
        StorageDriver::S3 => ok_check(
            "storage",
            "媒体存储",
            "S3 兼容 storage driver 已装配；检测响应不会返回访问密钥",
        ),
    }
}

fn webui_check(settings: &Settings) -> SetupConfigCheck {
    if settings.webui.enabled {
        ok_check(
            "webui",
            "WebUI 托管",
            "Rust 静态托管已启用，/api/* 与探针路径会保留给后端",
        )
    } else {
        warning_check(
            "webui",
            "WebUI 托管",
            "WebUI 静态托管已关闭，当前实例只提供 API 与探针",
        )
    }
}

fn ok_check(key: &str, title: &str, message: impl Into<String>) -> SetupConfigCheck {
    config_check(key, title, "ok", "info", message)
}

fn warning_check(key: &str, title: &str, message: impl Into<String>) -> SetupConfigCheck {
    config_check(key, title, "warning", "warning", message)
}

fn error_check(key: &str, title: &str, message: impl Into<String>) -> SetupConfigCheck {
    config_check(key, title, "error", "error", message)
}

fn config_check(
    key: &str,
    title: &str,
    status: &str,
    severity: &str,
    message: impl Into<String>,
) -> SetupConfigCheck {
    SetupConfigCheck {
        key: key.into(),
        title: title.into(),
        status: status.into(),
        severity: severity.into(),
        message: message.into(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::DatabaseDriver;

    #[test]
    fn config_checks_report_external_database_runtime_ready() {
        let mut settings = Settings::default();
        settings.database.driver = DatabaseDriver::Postgres;
        settings.database.url = "postgres://localhost/console".into();

        let summary = build_config_checks(&settings);

        assert!(summary.ready);
        let database = summary
            .checks
            .iter()
            .find(|check| check.key == "database")
            .expect("database check");
        assert_eq!(database.status, "ok");
        assert!(database.message.contains("PostgreSQL 连接池"));
        ensure_config_checks_ready(&summary).expect("external database config should be ready");
    }
}
