use std::{fs, path::PathBuf};

use app::app::App;
use app::config::Settings;
use app::infrastructure::database;
use app::observability;
use app::transport::http::route_registry;
use clap::{Parser, Subcommand};

#[derive(Debug, Parser)]
#[command(name = "console")]
#[command(about = "Aoi[葵] Rust 共享产品底座与管理控制台")]
struct Cli {
    #[arg(long, global = true, env = "CONSOLE_CONFIG")]
    config: Option<PathBuf>,

    #[arg(long, global = true, env = "CONSOLE_SECRETS")]
    secrets: Option<PathBuf>,

    #[command(subcommand)]
    command: Option<Command>,
}

#[derive(Debug, Subcommand)]
enum Command {
    #[command(about = "启动 HTTP 服务")]
    Serve,
    #[command(about = "检查配置并输出脱敏摘要")]
    CheckConfig,
    #[command(about = "输出 route registry 中的 API 契约")]
    Routes,
    #[command(name = "database-plan", about = "输出数据库运行态与迁移文件计划")]
    DatabasePlan,
    #[command(name = "database-ping", about = "执行数据库连接探针")]
    DatabasePing,
    #[command(name = "database-migrate", about = "显式执行当前数据库迁移")]
    DatabaseMigrate,
    #[command(name = "database-migration-history", about = "输出当前数据库迁移历史")]
    DatabaseMigrationHistory,
    #[command(
        name = "database-insert-id-probe",
        about = "验证当前数据库插入后 ID 读取策略"
    )]
    DatabaseInsertIdProbe,
    #[command(
        name = "database-setup-repository-probe",
        about = "验证当前数据库 SetupRepository 读写路径"
    )]
    DatabaseSetupRepositoryProbe,
    #[command(
        name = "database-iam-repository-probe",
        about = "验证当前数据库 IamRepository 读写路径"
    )]
    DatabaseIamRepositoryProbe,
    #[command(
        name = "database-notification-repository-probe",
        about = "验证当前数据库 NotificationRepository 读写路径"
    )]
    DatabaseNotificationRepositoryProbe,
    #[command(
        name = "database-system-repository-probe",
        about = "验证当前数据库 SystemRepository 读写路径"
    )]
    DatabaseSystemRepositoryProbe,
    #[command(name = "database-preflight", about = "执行数据库运行前预检")]
    DatabasePreflight,
    #[command(name = "database-schema-check", about = "检查当前数据库核心表是否齐备")]
    DatabaseSchemaCheck,
    #[command(
        name = "observability-token-hash",
        about = "从环境变量读取 Prometheus scrape token 并只输出配置用哈希"
    )]
    ObservabilityTokenHash {
        #[arg(
            long,
            env = "CONSOLE_OBSERVABILITY_SCRAPE_TOKEN",
            hide_env_values = true
        )]
        token: String,
    },
    #[command(about = "从 route registry 生成 OpenAPI YAML")]
    Openapi {
        #[arg(long)]
        output: Option<PathBuf>,
    },
    #[command(name = "notification-drain", about = "领取并投递一批通知 outbox")]
    NotificationDrain {
        #[arg(long)]
        limit: Option<i64>,
    },
    #[command(
        name = "notification-dead-letters",
        about = "输出最终失败通知的脱敏 dead-letter 报表"
    )]
    NotificationDeadLetters {
        #[arg(long)]
        limit: Option<i64>,
    },
    #[command(
        name = "notification-requeue-failed",
        about = "把仍有投递 secret 的失败通知安全恢复为 pending"
    )]
    NotificationRequeueFailed {
        #[arg(long)]
        limit: Option<i64>,
    },
    #[command(
        name = "operation-records-prune",
        about = "按 audit 配置的留存策略清理过期操作记录"
    )]
    OperationRecordsPrune,
    #[command(name = "operation-records-summary", about = "输出操作记录审计汇总报表")]
    OperationRecordsSummary {
        #[arg(long)]
        method: Option<String>,
        #[arg(long)]
        path: Option<String>,
        #[arg(long)]
        status: Option<i64>,
        #[arg(long)]
        actor_user_id: Option<i64>,
        #[arg(long)]
        created_from: Option<String>,
        #[arg(long)]
        created_to: Option<String>,
        #[arg(long)]
        top_limit: Option<i64>,
    },
    #[command(name = "scheduler-run-once", about = "立即执行一轮后端调度任务")]
    SchedulerRunOnce,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let cli = Cli::parse();
    let settings = Settings::load_with_secrets(cli.config, cli.secrets)?;
    observability::init(&settings.observability)?;

    match cli.command.unwrap_or(Command::Serve) {
        Command::Serve => {
            let app = App::boot(settings).await?;
            app.serve().await
        }
        Command::CheckConfig => {
            println!(
                "{}",
                serde_json::to_string_pretty(&settings.redacted_summary())?
            );
            Ok(())
        }
        Command::Routes => {
            let contracts = route_registry::contracts(&settings);
            println!("{}", serde_json::to_string_pretty(&contracts)?);
            Ok(())
        }
        Command::DatabasePlan => {
            let plan = database::migration_plan(&settings)?;
            println!("{}", serde_json::to_string_pretty(&plan)?);
            Ok(())
        }
        Command::DatabasePing => {
            let report = database::ping_database(&settings).await?;
            println!("{}", serde_json::to_string_pretty(&report)?);
            Ok(())
        }
        Command::DatabaseMigrate => {
            let report = database::migrate_database(&settings).await?;
            println!("{}", serde_json::to_string_pretty(&report)?);
            Ok(())
        }
        Command::DatabaseMigrationHistory => {
            let report = database::migration_history(&settings).await?;
            println!("{}", serde_json::to_string_pretty(&report)?);
            Ok(())
        }
        Command::DatabaseInsertIdProbe => {
            let report = database::probe_insert_id(&settings).await?;
            println!("{}", serde_json::to_string_pretty(&report)?);
            Ok(())
        }
        Command::DatabaseSetupRepositoryProbe => {
            let report = database::probe_setup_repository(&settings).await?;
            println!("{}", serde_json::to_string_pretty(&report)?);
            Ok(())
        }
        Command::DatabaseIamRepositoryProbe => {
            let report = database::probe_iam_repository(&settings).await?;
            println!("{}", serde_json::to_string_pretty(&report)?);
            Ok(())
        }
        Command::DatabaseNotificationRepositoryProbe => {
            let report = database::probe_notification_repository(&settings).await?;
            println!("{}", serde_json::to_string_pretty(&report)?);
            Ok(())
        }
        Command::DatabaseSystemRepositoryProbe => {
            let report = database::probe_system_repository(&settings).await?;
            println!("{}", serde_json::to_string_pretty(&report)?);
            Ok(())
        }
        Command::DatabasePreflight => {
            let report = database::preflight_database(&settings).await?;
            println!("{}", serde_json::to_string_pretty(&report)?);
            Ok(())
        }
        Command::DatabaseSchemaCheck => {
            let report = database::check_database_schema(&settings).await?;
            println!("{}", serde_json::to_string_pretty(&report)?);
            Ok(())
        }
        Command::ObservabilityTokenHash { token } => {
            println!(
                "{}",
                crypto::hash_secret(&token, &settings.auth.session_secret)
            );
            Ok(())
        }
        Command::Openapi { output } => {
            let yaml = route_registry::openapi_yaml(&settings)?;
            if let Some(output) = output {
                if let Some(parent) = output.parent() {
                    fs::create_dir_all(parent)?;
                }
                fs::write(output, yaml)?;
            } else {
                print!("{yaml}");
            }
            Ok(())
        }
        Command::NotificationDrain { limit } => {
            let app = App::boot(settings).await?;
            let report = app.drain_notifications_once(limit).await?;
            println!("{}", serde_json::to_string_pretty(&report)?);
            Ok(())
        }
        Command::NotificationDeadLetters { limit } => {
            let app = App::boot(settings).await?;
            let report = app.notification_dead_letters(limit).await?;
            println!("{}", serde_json::to_string_pretty(&report)?);
            Ok(())
        }
        Command::NotificationRequeueFailed { limit } => {
            let app = App::boot(settings).await?;
            let report = app.requeue_failed_notifications(limit).await?;
            println!("{}", serde_json::to_string_pretty(&report)?);
            Ok(())
        }
        Command::OperationRecordsPrune => {
            let app = App::boot(settings).await?;
            let report = app.prune_operation_records().await?;
            println!("{}", serde_json::to_string_pretty(&report)?);
            Ok(())
        }
        Command::OperationRecordsSummary {
            method,
            path,
            status,
            actor_user_id,
            created_from,
            created_to,
            top_limit,
        } => {
            let app = App::boot(settings).await?;
            let report = app
                .operation_record_summary(app::domain::system::OperationRecordSummaryQuery {
                    method,
                    path,
                    status,
                    actor_user_id,
                    created_from,
                    created_to,
                    top_limit,
                })
                .await?;
            println!("{}", serde_json::to_string_pretty(&report)?);
            Ok(())
        }
        Command::SchedulerRunOnce => {
            let app = App::boot(settings).await?;
            let report = app.run_scheduled_tasks_once().await?;
            println!("{}", serde_json::to_string_pretty(&report)?);
            Ok(())
        }
    }
}
