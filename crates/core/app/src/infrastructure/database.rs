use std::collections::HashSet;
use std::fs;
use std::path::{Path, PathBuf};
use std::str::FromStr;
use std::sync::Arc;

use anyhow::Context;
use chrono::{Duration, Utc};
use serde::Serialize;
use sqlx::any::{AnyPoolOptions, install_default_drivers};
use sqlx::migrate::Migrator;
use sqlx::mysql::MySqlPoolOptions;
use sqlx::postgres::PgPoolOptions;
use sqlx::sqlite::{SqliteConnectOptions, SqliteJournalMode, SqlitePoolOptions};
use sqlx::{Any, MySql, Pool, Postgres, Row, Sqlite};

use crate::app::AppResult;
use crate::config::{DatabaseDriver, Settings};
use crate::domain::system::{ApiCatalogEntry, SystemMenuEntry};
use crate::infrastructure::external_iam_repository::ExternalIamRepository;
use crate::infrastructure::external_notification_repository::ExternalNotificationRepository;
use crate::infrastructure::external_setup_repository::ExternalSetupRepository;
use crate::infrastructure::external_system_repository::ExternalSystemRepository;
use crate::infrastructure::sql_templates::{InsertIdRead, SqlDialect};
use crate::infrastructure::sqlite_repository::SqliteRepository;
use crate::repository::{
    AcceptInvitationRecord, CreateAPITokenRecord, CreateAuditLogRecord, CreateInitialAdminRecord,
    CreateInvitationRecord, CreateMediaAssetRecord, CreateMfaFactorRecord,
    CreateMfaRecoveryCodeRecord, CreateNotificationOutboxRecord, CreateOperationRecord,
    CreatePasswordResetRecord, CreateRegistrationRecord, CreateSessionRecord,
    CreateTrafficProbeAlertRecord, CreateTrafficProbeResultRecord, CreateTrafficProbeTargetRecord,
    CreateVersionPackageRecord, IamRepository, NotificationRepository, OperationRecordListQuery,
    OperationRecordSummaryFilter, SetupRepository, SystemRepository, UpsertSystemConfigRecord,
    UpsertSystemDictionaryRecord, UpsertSystemParameterRecord, VersionPackageActionRecord,
};
use uuid::Uuid;

static SQLITE_MIGRATOR: Migrator = sqlx::migrate!("../../../migrations");

const REQUIRED_RUNTIME_TABLES: &[&str] = &[
    "setup_state",
    "setup_runs",
    "setup_step_logs",
    "iam_organizations",
    "iam_users",
    "iam_roles",
    "iam_permissions",
    "iam_role_permissions",
    "iam_memberships",
    "iam_sessions",
    "iam_api_tokens",
    "iam_invitations",
    "iam_password_resets",
    "iam_email_verifications",
    "iam_mfa_factors",
    "iam_mfa_recovery_codes",
    "iam_audit_logs",
    "iam_notification_outbox",
    "iam_notification_delivery_secrets",
    "system_apis",
    "system_menus",
    "system_configs",
    "system_dictionaries",
    "system_parameters",
    "system_operation_records",
    "system_server_metrics",
    "system_version_packages",
    "system_version_release_events",
    "system_media_assets",
    "system_traffic_probe_targets",
    "system_traffic_probe_results",
    "system_traffic_probe_alerts",
];

#[derive(Debug, Serialize)]
pub struct DatabaseMigrationPlan {
    pub driver: DatabaseDriver,
    pub runtime_supported: bool,
    pub runtime_status: String,
    pub runtime_message: String,
    pub required_work: Vec<String>,
    pub auto_apply: bool,
    pub migration_dir: String,
    pub migration_files: Vec<DatabaseMigrationFile>,
}

#[derive(Clone, Debug, Serialize)]
pub struct DatabaseMigrationFile {
    pub name: String,
    pub path: String,
    pub bytes: u64,
    pub sha256: String,
}

#[derive(Debug, Serialize)]
pub struct DatabasePingReport {
    pub driver: DatabaseDriver,
    pub connection_ok: bool,
    pub runtime_supported: bool,
    pub runtime_status: String,
    pub repository_runtime: RepositoryRuntimeReport,
    pub repository_ready: bool,
    pub migration_runtime_ready: bool,
    pub message: String,
}

#[derive(Debug, Serialize)]
pub struct DatabaseMigrateReport {
    pub driver: DatabaseDriver,
    pub migration_dir: String,
    pub applied_files: Vec<String>,
    pub skipped_files: Vec<String>,
    pub repository_runtime: RepositoryRuntimeReport,
    pub repository_ready: bool,
    pub message: String,
}

#[derive(Debug, Serialize)]
pub struct DatabaseMigrationHistoryReport {
    pub driver: DatabaseDriver,
    pub checksum_source: String,
    pub records: Vec<DatabaseMigrationHistoryRecord>,
    pub message: String,
}

#[derive(Debug, Serialize)]
pub struct DatabaseMigrationHistoryRecord {
    pub version: i64,
    pub name: String,
    pub checksum: String,
    pub success: bool,
}

#[derive(Debug, Serialize)]
pub struct DatabaseInsertIdProbeReport {
    pub driver: DatabaseDriver,
    pub insert_id_strategy: String,
    pub insert_id_read: String,
    pub inserted_id: i64,
    pub same_connection_required: bool,
    pub temporary_table: String,
    pub message: String,
}

#[derive(Debug, Serialize)]
pub struct DatabaseSetupRepositoryProbeReport {
    pub driver: DatabaseDriver,
    pub implementation: String,
    pub run_id: String,
    pub completed_before: bool,
    pub missing_complete_result: bool,
    pub run_listed: bool,
    pub log_count: usize,
    pub message: String,
}

#[derive(Debug, Serialize)]
pub struct DatabaseIamRepositoryProbeReport {
    pub driver: DatabaseDriver,
    pub implementation: String,
    pub initial_admin_created: bool,
    pub permissions_synced: bool,
    pub organization_roundtrip: bool,
    pub org_user_roundtrip: bool,
    pub role_roundtrip: bool,
    pub session_roundtrip: bool,
    pub refresh_rotation_roundtrip: bool,
    pub api_token_roundtrip: bool,
    pub registration_pending_roundtrip: bool,
    pub invitation_roundtrip: bool,
    pub password_reset_roundtrip: bool,
    pub email_verification_roundtrip: bool,
    pub mfa_roundtrip: bool,
    pub audit_record_written: bool,
    pub message: String,
}

#[derive(Debug, Serialize)]
pub struct DatabaseNotificationRepositoryProbeReport {
    pub driver: DatabaseDriver,
    pub implementation: String,
    pub claimed_probe_items: usize,
    pub delivered_result: bool,
    pub retry_result: bool,
    pub final_failure_result: bool,
    pub dead_letter_reported: bool,
    pub dead_letter_secret_state: String,
    pub delivered_secret_purged: bool,
    pub failed_secret_purged: bool,
    pub purged_requeue_skipped: bool,
    pub pending_secret_requeue_result: bool,
    pub message: String,
}

#[derive(Debug, Serialize)]
pub struct DatabaseSystemRepositoryProbeReport {
    pub driver: DatabaseDriver,
    pub implementation: String,
    pub api_catalog_synced: bool,
    pub menu_synced: bool,
    pub config_roundtrip: bool,
    pub dictionary_roundtrip: bool,
    pub parameter_roundtrip: bool,
    pub operation_record_written: bool,
    pub operation_record_summary_reported: bool,
    pub operation_record_retention_prune: bool,
    pub version_package_roundtrip: bool,
    pub media_asset_roundtrip: bool,
    pub traffic_probe_roundtrip: bool,
    pub message: String,
}

#[derive(Debug, Serialize)]
pub struct DatabasePreflightReport {
    pub driver: DatabaseDriver,
    pub runtime_supported: bool,
    pub runtime_status: String,
    pub connection_ok: bool,
    pub migration_plan_ok: bool,
    pub migration_history_ok: bool,
    pub schema_ready: bool,
    pub repository_runtime: RepositoryRuntimeReport,
    pub repository_ready: bool,
    pub serve_ready: bool,
    pub checks: Vec<DatabasePreflightCheck>,
    pub message: String,
}

#[derive(Debug, Serialize)]
pub struct DatabasePreflightCheck {
    pub key: String,
    pub title: String,
    pub status: String,
    pub severity: String,
    pub message: String,
}

#[derive(Debug, Serialize)]
pub struct DatabaseSchemaCheckReport {
    pub driver: DatabaseDriver,
    pub runtime_supported: bool,
    pub runtime_status: String,
    pub checked_tables: Vec<String>,
    pub missing_tables: Vec<String>,
    pub schema_ready: bool,
    pub repository_runtime: RepositoryRuntimeReport,
    pub repository_ready: bool,
    pub message: String,
}

#[derive(Clone, Debug, Serialize)]
pub struct RepositoryRuntimeReport {
    pub driver: DatabaseDriver,
    pub ready: bool,
    pub setup: RepositoryCapability,
    pub iam: RepositoryCapability,
    pub notification: RepositoryCapability,
    pub system: RepositoryCapability,
    pub blockers: Vec<String>,
}

#[derive(Clone, Debug, Serialize)]
pub struct RepositoryCapability {
    pub trait_name: String,
    pub ready: bool,
    pub implementation: Option<String>,
    pub blocker: Option<String>,
}

#[derive(Clone, Debug)]
pub enum DatabaseConnection {
    Sqlite(Pool<Sqlite>),
    Postgres {
        pool: Pool<Postgres>,
        iam_pool: Pool<Any>,
    },
    Mysql {
        pool: Pool<MySql>,
        iam_pool: Pool<Any>,
    },
}

pub struct RepositorySet {
    pub setup: Arc<dyn SetupRepository>,
    pub iam: Arc<dyn IamRepository>,
    pub notification: Arc<dyn NotificationRepository>,
    pub system: Arc<dyn SystemRepository>,
}

impl DatabaseConnection {
    pub fn driver(&self) -> DatabaseDriver {
        match self {
            Self::Sqlite(_) => DatabaseDriver::Sqlite,
            Self::Postgres { .. } => DatabaseDriver::Postgres,
            Self::Mysql { .. } => DatabaseDriver::Mysql,
        }
    }

    pub async fn ping(&self) -> AppResult<()> {
        match self {
            Self::Sqlite(pool) => {
                sqlx::query("select 1").execute(pool).await?;
            }
            Self::Postgres { pool, .. } => {
                sqlx::query("select 1").execute(pool).await?;
            }
            Self::Mysql { pool, .. } => {
                sqlx::query("select 1").execute(pool).await?;
            }
        }
        Ok(())
    }
}

pub fn repositories(connection: &DatabaseConnection) -> RepositorySet {
    match connection {
        DatabaseConnection::Sqlite(pool) => {
            let repo = Arc::new(SqliteRepository::new(pool.clone()));
            RepositorySet {
                setup: repo.clone(),
                iam: repo.clone(),
                notification: repo.clone(),
                system: repo,
            }
        }
        DatabaseConnection::Postgres { pool, iam_pool } => RepositorySet {
            setup: Arc::new(ExternalSetupRepository::postgres(pool.clone())),
            iam: Arc::new(ExternalIamRepository::new(
                SqlDialect::Postgres,
                iam_pool.clone(),
            )),
            notification: Arc::new(ExternalNotificationRepository::postgres(pool.clone())),
            system: Arc::new(ExternalSystemRepository::postgres(pool.clone())),
        },
        DatabaseConnection::Mysql { pool, iam_pool } => RepositorySet {
            setup: Arc::new(ExternalSetupRepository::mysql(pool.clone())),
            iam: Arc::new(ExternalIamRepository::new(
                SqlDialect::Mysql,
                iam_pool.clone(),
            )),
            notification: Arc::new(ExternalNotificationRepository::mysql(pool.clone())),
            system: Arc::new(ExternalSystemRepository::mysql(pool.clone())),
        },
    }
}

pub fn repository_runtime_report(driver: DatabaseDriver) -> RepositoryRuntimeReport {
    match driver {
        DatabaseDriver::Sqlite => {
            let implementation = "SqliteRepository";
            RepositoryRuntimeReport {
                driver,
                ready: true,
                setup: repository_capability_ready("SetupRepository", implementation),
                iam: repository_capability_ready("IamRepository", implementation),
                notification: repository_capability_ready("NotificationRepository", implementation),
                system: repository_capability_ready("SystemRepository", implementation),
                blockers: Vec::new(),
            }
        }
        DatabaseDriver::Postgres | DatabaseDriver::Mysql => {
            repository_runtime_external_ready(driver)
        }
    }
}

fn repository_runtime_external_ready(driver: DatabaseDriver) -> RepositoryRuntimeReport {
    let setup = repository_capability_ready("SetupRepository", "ExternalSetupRepository");
    let iam = repository_capability_ready("IamRepository", "ExternalIamRepository");
    let notification =
        repository_capability_ready("NotificationRepository", "ExternalNotificationRepository");
    let system = repository_capability_ready("SystemRepository", "ExternalSystemRepository");
    let blockers = [&setup, &iam, &notification, &system]
        .into_iter()
        .filter_map(|capability| capability.blocker.clone())
        .collect::<Vec<_>>();

    RepositoryRuntimeReport {
        driver,
        ready: blockers.is_empty(),
        setup,
        iam,
        notification,
        system,
        blockers,
    }
}

fn repository_capability_ready(
    trait_name: &'static str,
    implementation: &'static str,
) -> RepositoryCapability {
    RepositoryCapability {
        trait_name: trait_name.into(),
        ready: true,
        implementation: Some(implementation.into()),
        blocker: None,
    }
}

pub async fn connect(settings: &Settings) -> anyhow::Result<DatabaseConnection> {
    match settings.database.driver {
        DatabaseDriver::Sqlite => connect_sqlite(settings)
            .await
            .map(DatabaseConnection::Sqlite),
        DatabaseDriver::Postgres => connect_postgres(settings).await,
        DatabaseDriver::Mysql => connect_mysql(settings).await,
    }
}

pub async fn ping_database(settings: &Settings) -> anyhow::Result<DatabasePingReport> {
    let support = settings.database.driver.runtime_support();
    let repository_runtime = repository_runtime_report(settings.database.driver);
    match settings.database.driver {
        DatabaseDriver::Sqlite => {
            let pool = connect_sqlite_pool(settings, false).await?;
            sqlx::query("select 1").execute(&pool).await?;
            Ok(DatabasePingReport {
                driver: DatabaseDriver::Sqlite,
                connection_ok: true,
                runtime_supported: support.supported,
                runtime_status: support.status,
                repository_ready: repository_runtime.ready,
                repository_runtime,
                migration_runtime_ready: true,
                message: "SQLite 连接探针成功；该 driver 已接入 app runtime repository 和迁移"
                    .into(),
            })
        }
        DatabaseDriver::Postgres => {
            let pool = PgPoolOptions::new()
                .max_connections(1)
                .connect(&settings.database.url)
                .await?;
            sqlx::query("select 1").execute(&pool).await?;
            Ok(DatabasePingReport {
                driver: DatabaseDriver::Postgres,
                connection_ok: true,
                runtime_supported: support.supported,
                runtime_status: support.status,
                repository_ready: repository_runtime.ready,
                repository_runtime,
                migration_runtime_ready: true,
                message:
                    "PostgreSQL 连接探针成功；该 driver 已接入 app runtime repository 和迁移方言"
                        .into(),
            })
        }
        DatabaseDriver::Mysql => {
            let pool = MySqlPoolOptions::new()
                .max_connections(1)
                .connect(&settings.database.url)
                .await?;
            sqlx::query("select 1").execute(&pool).await?;
            Ok(DatabasePingReport {
                driver: DatabaseDriver::Mysql,
                connection_ok: true,
                runtime_supported: support.supported,
                runtime_status: support.status,
                repository_ready: repository_runtime.ready,
                repository_runtime,
                migration_runtime_ready: true,
                message: "MySQL 连接探针成功；该 driver 已接入 app runtime repository 和迁移方言"
                    .into(),
            })
        }
    }
}

pub async fn migrate_database(settings: &Settings) -> anyhow::Result<DatabaseMigrateReport> {
    let plan = migration_plan(settings)?;
    match settings.database.driver {
        DatabaseDriver::Sqlite => migrate_sqlite(settings, plan).await,
        DatabaseDriver::Postgres => migrate_postgres(settings, plan).await,
        DatabaseDriver::Mysql => migrate_mysql(settings, plan).await,
    }
}

pub async fn migration_history(
    settings: &Settings,
) -> anyhow::Result<DatabaseMigrationHistoryReport> {
    match settings.database.driver {
        DatabaseDriver::Sqlite => {
            let pool = connect_sqlite_pool(settings, false).await?;
            let records = sqlite_migration_history(&pool).await?;
            Ok(DatabaseMigrationHistoryReport {
                driver: DatabaseDriver::Sqlite,
                checksum_source: "sqlx_migrator".into(),
                records,
                message: "SQLite 迁移历史来自 sqlx _sqlx_migrations 表".into(),
            })
        }
        DatabaseDriver::Postgres => {
            let pool = PgPoolOptions::new()
                .max_connections(1)
                .connect(&settings.database.url)
                .await?;
            let records = postgres_migration_history(&pool).await?;
            Ok(DatabaseMigrationHistoryReport {
                driver: DatabaseDriver::Postgres,
                checksum_source: "sha256".into(),
                records,
                message:
                    "PostgreSQL 迁移历史来自 schema_migrations 表，checksum 为迁移文件 SHA-256"
                        .into(),
            })
        }
        DatabaseDriver::Mysql => {
            let pool = MySqlPoolOptions::new()
                .max_connections(1)
                .connect(&settings.database.url)
                .await?;
            let records = mysql_migration_history(&pool).await?;
            Ok(DatabaseMigrationHistoryReport {
                driver: DatabaseDriver::Mysql,
                checksum_source: "sha256".into(),
                records,
                message: "MySQL 迁移历史来自 schema_migrations 表，checksum 为迁移文件 SHA-256"
                    .into(),
            })
        }
    }
}

pub async fn probe_insert_id(settings: &Settings) -> anyhow::Result<DatabaseInsertIdProbeReport> {
    let driver = settings.database.driver;
    let dialect = sql_dialect_for_driver(driver);
    let inserted_id = match driver {
        DatabaseDriver::Sqlite => probe_sqlite_insert_id(settings).await?,
        DatabaseDriver::Postgres => probe_postgres_insert_id(settings).await?,
        DatabaseDriver::Mysql => probe_mysql_insert_id(settings).await?,
    };

    Ok(DatabaseInsertIdProbeReport {
        driver,
        insert_id_strategy: format!("{:?}", dialect.insert_id_strategy()),
        insert_id_read: insert_id_read_label(dialect.insert_id_read()),
        inserted_id,
        same_connection_required: matches!(
            dialect.insert_id_read(),
            InsertIdRead::PostInsertQuery(_)
        ),
        temporary_table: "console_insert_id_probe".into(),
        message: match driver {
            DatabaseDriver::Sqlite | DatabaseDriver::Postgres => {
                "插入 ID 探针通过：当前方言通过 insert ... returning id 读取生成 ID".into()
            }
            DatabaseDriver::Mysql => {
                "插入 ID 探针通过：MySQL 在同一连接内通过 select last_insert_id() 读取生成 ID"
                    .into()
            }
        },
    })
}

pub async fn probe_setup_repository(
    settings: &Settings,
) -> anyhow::Result<DatabaseSetupRepositoryProbeReport> {
    let driver = settings.database.driver;
    let run_id = format!("setup-repository-probe-{}", uuid::Uuid::new_v4());
    let repository: Box<dyn SetupRepository> = match driver {
        DatabaseDriver::Sqlite => {
            let pool = connect_sqlite_pool(settings, false).await?;
            Box::new(SqliteRepository::new(pool))
        }
        DatabaseDriver::Postgres => {
            let pool = PgPoolOptions::new()
                .max_connections(settings.database.max_connections)
                .connect(&settings.database.url)
                .await?;
            Box::new(ExternalSetupRepository::postgres(pool))
        }
        DatabaseDriver::Mysql => {
            let pool = MySqlPoolOptions::new()
                .max_connections(settings.database.max_connections)
                .connect(&settings.database.url)
                .await?;
            Box::new(ExternalSetupRepository::mysql(pool))
        }
    };

    let completed_before = repository.setup_completed().await?;
    let missing_complete_result = repository
        .complete_setup(Some("missing-setup-repository-probe-run"))
        .await?;
    let run = repository
        .create_setup_run(&run_id, Some("database setup repository probe"))
        .await?;
    repository
        .append_setup_log(
            &run.id,
            "repository-probe",
            "ok",
            "SetupRepository 读写探针已完成",
        )
        .await?;
    let runs = repository.list_setup_runs(20).await?;
    let logs = repository.list_setup_logs(&run.id).await?;
    let run_listed = runs.iter().any(|item| item.id == run.id);

    Ok(DatabaseSetupRepositoryProbeReport {
        driver,
        implementation: match driver {
            DatabaseDriver::Sqlite => "SqliteRepository".into(),
            DatabaseDriver::Postgres | DatabaseDriver::Mysql => "ExternalSetupRepository".into(),
        },
        run_id: run.id,
        completed_before,
        missing_complete_result,
        run_listed,
        log_count: logs.len(),
        message: match driver {
            DatabaseDriver::Sqlite => {
                "SQLite SetupRepository 探针通过；当前 driver 已接入 app runtime".into()
            }
            DatabaseDriver::Postgres | DatabaseDriver::Mysql => {
                "外部 SetupRepository 探针通过；该 driver 已接入 app runtime repository".into()
            }
        },
    })
}

pub async fn probe_iam_repository(
    settings: &Settings,
) -> anyhow::Result<DatabaseIamRepositoryProbeReport> {
    let suffix = Uuid::new_v4().simple().to_string();
    let product_code = settings.app.product_code.as_str();
    match settings.database.driver {
        DatabaseDriver::Sqlite => {
            let pool = connect_sqlite_pool(settings, false).await?;
            let repository = SqliteRepository::new(pool);
            sync_iam_probe_permissions(&repository, product_code, &suffix).await?;
            probe_iam_repository_with(
                DatabaseDriver::Sqlite,
                "SqliteRepository",
                &repository,
                product_code,
                &settings.auth.session_secret,
                &suffix,
            )
            .await
        }
        DatabaseDriver::Postgres => {
            let pool = PgPoolOptions::new()
                .max_connections(settings.database.max_connections)
                .connect(&settings.database.url)
                .await?;
            let system_repository = ExternalSystemRepository::postgres(pool);
            sync_iam_probe_permissions(&system_repository, product_code, &suffix).await?;
            let iam_repository = ExternalIamRepository::connect(
                SqlDialect::Postgres,
                &settings.database.url,
                settings.database.max_connections,
            )
            .await?;
            probe_iam_repository_with(
                DatabaseDriver::Postgres,
                "ExternalIamRepository",
                &iam_repository,
                product_code,
                &settings.auth.session_secret,
                &suffix,
            )
            .await
        }
        DatabaseDriver::Mysql => {
            let pool = MySqlPoolOptions::new()
                .max_connections(settings.database.max_connections)
                .connect(&settings.database.url)
                .await?;
            let system_repository = ExternalSystemRepository::mysql(pool);
            sync_iam_probe_permissions(&system_repository, product_code, &suffix).await?;
            let iam_repository = ExternalIamRepository::connect(
                SqlDialect::Mysql,
                &settings.database.url,
                settings.database.max_connections,
            )
            .await?;
            probe_iam_repository_with(
                DatabaseDriver::Mysql,
                "ExternalIamRepository",
                &iam_repository,
                product_code,
                &settings.auth.session_secret,
                &suffix,
            )
            .await
        }
    }
}

async fn sync_iam_probe_permissions(
    repository: &dyn SystemRepository,
    product_code: &str,
    suffix: &str,
) -> anyhow::Result<()> {
    let platform_permission = iam_probe_platform_permission(suffix);
    let tenant_permission = iam_probe_tenant_permission(suffix);
    repository
        .sync_api_catalog(&[
            ApiCatalogEntry {
                id: format!("probe.iam.platform.{suffix}"),
                method: "GET".into(),
                path: format!("/__probe/iam/{suffix}/platform"),
                tag: "probe".into(),
                summary: "IAM platform probe".into(),
                access: "authenticated".into(),
                permission: Some(platform_permission),
                scope: "platform".into(),
                product_code: product_code.into(),
            },
            ApiCatalogEntry {
                id: format!("probe.iam.tenant.{suffix}"),
                method: "POST".into(),
                path: format!("/__probe/iam/{suffix}/tenant"),
                tag: "probe".into(),
                summary: "IAM tenant probe".into(),
                access: "authenticated".into(),
                permission: Some(tenant_permission),
                scope: "tenant".into(),
                product_code: product_code.into(),
            },
        ])
        .await?;
    Ok(())
}

async fn probe_iam_repository_with(
    driver: DatabaseDriver,
    implementation: &str,
    repository: &dyn IamRepository,
    product_code: &str,
    session_secret: &str,
    suffix: &str,
) -> anyhow::Result<DatabaseIamRepositoryProbeReport> {
    if repository.has_any_user().await? {
        anyhow::bail!("IAM repository probe requires an empty smoke database");
    }

    let platform_permission = iam_probe_platform_permission(suffix);
    let tenant_permission = iam_probe_tenant_permission(suffix);
    let owner_email = format!("owner-{suffix}@example.invalid");
    let owner_org_code = format!("probe-iam-owner-{suffix}");
    let (owner, org) = repository
        .create_initial_admin(CreateInitialAdminRecord {
            email: owner_email.clone(),
            password_hash: format!("hash-owner-{suffix}"),
            display_name: "IAM Probe Owner".into(),
            organization_code: owner_org_code.clone(),
            organization_name: "IAM Probe Owner Org".into(),
            product_code: product_code.into(),
        })
        .await?;
    let initial_admin_created = owner.id > 0 && org.id > 0;

    let permissions = repository.list_permissions(product_code).await?;
    let resolved_permissions = repository
        .list_permissions_for_user(owner.id, org.id, product_code, true)
        .await?;
    let permissions_synced = permissions
        .iter()
        .any(|permission| permission.code == platform_permission && permission.scope == "platform")
        && permissions
            .iter()
            .any(|permission| permission.code == tenant_permission && permission.scope == "tenant")
        && resolved_permissions.contains(&platform_permission)
        && resolved_permissions.contains(&tenant_permission);

    let found_owner = repository
        .find_user_by_identifier(&owner_email)
        .await?
        .filter(|item| item.id == owner.id && item.password_hash == format!("hash-owner-{suffix}"))
        .is_some();
    let primary_org = repository
        .primary_organization_for_user(owner.id)
        .await?
        .filter(|item| item.id == org.id && item.code == owner_org_code)
        .is_some();
    let org_listed = repository
        .list_organizations()
        .await?
        .iter()
        .any(|item| item.id == org.id && item.code == owner_org_code);
    let organization_roundtrip = found_owner && primary_org && org_listed;

    let updated_owner = repository
        .update_org_user(crate::repository::UpdateOrgUserRecord {
            org_id: org.id,
            user_id: owner.id,
            display_name: "IAM Probe Owner Updated".into(),
            status: "active".into(),
            role_codes: vec!["owner".into()],
        })
        .await?;
    let org_user_listed = repository.list_org_users(org.id).await?.iter().any(|item| {
        item.id == owner.id
            && item.display_name == "IAM Probe Owner Updated"
            && item.role_codes.iter().any(|role| role == "owner")
    });
    let org_user_roundtrip = updated_owner.id == owner.id
        && updated_owner.display_name == "IAM Probe Owner Updated"
        && org_user_listed;

    let role_code = format!("probe_role_{suffix}");
    let role = repository
        .create_org_role(crate::repository::CreateRoleRecord {
            org_id: org.id,
            code: role_code.clone(),
            name: "IAM Probe Role".into(),
            product_code: product_code.into(),
            permission_codes: vec![tenant_permission.clone()],
        })
        .await?;
    let role_updated = repository
        .update_org_role(crate::repository::UpdateRoleRecord {
            org_id: org.id,
            role_id: role.id,
            name: "IAM Probe Role Updated".into(),
            product_code: product_code.into(),
            permission_codes: vec![tenant_permission.clone()],
        })
        .await?;
    let role_exists = repository.org_role_exists(org.id, &role_code).await?;
    let role_deleted = repository.delete_org_role(org.id, role.id).await?;
    let role_roundtrip =
        role.id > 0 && role_updated.name == "IAM Probe Role Updated" && role_exists && role_deleted;

    let expires_at = (Utc::now() + Duration::hours(2)).to_rfc3339();
    let refresh_expires_at = (Utc::now() + Duration::hours(4)).to_rfc3339();
    let session_hash = crypto::hash_secret(&format!("probe-session-{suffix}"), session_secret);
    let refresh_hash = crypto::hash_secret(&format!("probe-refresh-{suffix}"), session_secret);
    let session_id = format!("probe-session-{suffix}");
    repository
        .create_session(CreateSessionRecord {
            id: session_id.clone(),
            token_hash: session_hash.clone(),
            refresh_token_hash: refresh_hash.clone(),
            user_id: owner.id,
            org_id: org.id,
            product_code: product_code.into(),
            client_type: "probe".into(),
            expires_at: expires_at.clone(),
            refresh_expires_at: refresh_expires_at.clone(),
        })
        .await?;
    let session_found = repository
        .find_session_by_hash(&session_hash)
        .await?
        .filter(|item| item.id == session_id && item.user.id == owner.id)
        .is_some();
    let refresh_found = repository
        .find_session_by_refresh_hash(&refresh_hash)
        .await?
        .filter(|item| item.id == session_id && item.organization.id == org.id)
        .is_some();
    let session_roundtrip = session_found && refresh_found;

    let rotated_session_hash =
        crypto::hash_secret(&format!("probe-session-rotated-{suffix}"), session_secret);
    let rotated_refresh_hash =
        crypto::hash_secret(&format!("probe-refresh-rotated-{suffix}"), session_secret);
    let rotated = repository
        .rotate_session_tokens(
            &session_id,
            &refresh_hash,
            rotated_session_hash.clone(),
            rotated_refresh_hash.clone(),
            expires_at.clone(),
            refresh_expires_at.clone(),
        )
        .await?;
    let old_refresh_missing = repository
        .find_session_by_refresh_hash(&refresh_hash)
        .await?
        .is_none();
    let rotated_refresh_found = repository
        .find_session_by_refresh_hash(&rotated_refresh_hash)
        .await?
        .filter(|item| item.id == session_id)
        .is_some();
    repository
        .revoke_session_by_hash(&rotated_session_hash)
        .await?;
    let refresh_rotation_roundtrip = rotated && old_refresh_missing && rotated_refresh_found;

    let api_token_hash = crypto::hash_secret(&format!("probe-api-token-{suffix}"), session_secret);
    let api_token = repository
        .create_api_token(CreateAPITokenRecord {
            org_id: org.id,
            user_id: owner.id,
            token_hash: api_token_hash.clone(),
            token_prefix: format!("prb{}", &suffix[..8]),
            expires_at: Some(expires_at.clone()),
        })
        .await?;
    let api_token_listed = repository
        .list_api_tokens(org.id)
        .await?
        .iter()
        .any(|item| item.id == api_token.id && item.token_prefix == api_token.token_prefix);
    let api_token_auth = repository
        .find_api_token_by_hash(&api_token_hash)
        .await?
        .filter(|item| item.id == api_token.id && item.user.id == owner.id)
        .is_some();
    let api_token_revoked = repository.revoke_api_token(org.id, api_token.id).await?;
    let api_token_roundtrip =
        api_token.id > 0 && api_token_listed && api_token_auth && api_token_revoked;

    let registration_email = format!("register-{suffix}@example.invalid");
    let registration_token_hash =
        crypto::hash_secret(&format!("probe-register-email-{suffix}"), session_secret);
    let (registered_user, registered_org) = repository
        .create_registration_with_email_verification(
            CreateRegistrationRecord {
                email: registration_email.clone(),
                password_hash: format!("hash-register-{suffix}"),
                display_name: "IAM Probe Register".into(),
                organization_code: format!("probe-iam-register-{suffix}"),
                organization_name: "IAM Probe Register Org".into(),
                product_code: product_code.into(),
                email_verification_token_hash: registration_token_hash.clone(),
                email_verification_expires_at: expires_at.clone(),
            },
            probe_notification(
                Some(org.id),
                Some(owner.id),
                product_code,
                "iam_email_verification",
                &registration_email,
                suffix,
                &expires_at,
            ),
        )
        .await?;
    let registration_verification = repository
        .find_email_verification_by_hash(&registration_token_hash)
        .await?;
    let registration_pending_roundtrip = registered_user.id > 0
        && registered_org.id > 0
        && registered_user.status == "pending_verification"
        && registration_verification
            .as_ref()
            .is_some_and(|item| item.user_id == registered_user.id);

    let revoked_invitation = repository
        .create_invitation_with_notification(
            CreateInvitationRecord {
                org_id: org.id,
                email: format!("revoke-invite-{suffix}@example.invalid"),
                role_code: "owner".into(),
                token_hash: crypto::hash_secret(
                    &format!("probe-invitation-revoke-{suffix}"),
                    session_secret,
                ),
                expires_at: expires_at.clone(),
            },
            probe_notification(
                Some(org.id),
                Some(owner.id),
                product_code,
                "iam_invitation",
                "revoke-invite@example.invalid",
                suffix,
                &expires_at,
            ),
        )
        .await?;
    let invitation_revoked = repository
        .revoke_invitation(org.id, revoked_invitation.id)
        .await?;
    let accept_invitation_hash =
        crypto::hash_secret(&format!("probe-invitation-accept-{suffix}"), session_secret);
    let accepted_invitation = repository
        .create_invitation_with_notification(
            CreateInvitationRecord {
                org_id: org.id,
                email: format!("accept-invite-{suffix}@example.invalid"),
                role_code: "owner".into(),
                token_hash: accept_invitation_hash.clone(),
                expires_at: expires_at.clone(),
            },
            probe_notification(
                Some(org.id),
                Some(owner.id),
                product_code,
                "iam_invitation",
                "accept-invite@example.invalid",
                suffix,
                &expires_at,
            ),
        )
        .await?;
    let invitation_found = repository
        .find_invitation_by_hash(&accept_invitation_hash)
        .await?
        .filter(|item| item.id == accepted_invitation.id && item.org_id == org.id)
        .is_some();
    let (invited_user, invited_org) = repository
        .accept_invitation_with_user(AcceptInvitationRecord {
            invitation_id: accepted_invitation.id,
            org_id: org.id,
            email: format!("accept-invite-{suffix}@example.invalid"),
            role_code: "owner".into(),
            display_name: "IAM Probe Invited".into(),
            password_hash: format!("hash-invited-{suffix}"),
        })
        .await?;
    let invitation_roundtrip =
        invitation_revoked && invitation_found && invited_user.id > 0 && invited_org.id == org.id;

    let reset_hash = crypto::hash_secret(&format!("probe-password-reset-{suffix}"), session_secret);
    repository
        .create_password_reset_with_notification(
            CreatePasswordResetRecord {
                user_id: owner.id,
                token_hash: reset_hash.clone(),
                expires_at: expires_at.clone(),
            },
            probe_notification(
                Some(org.id),
                Some(owner.id),
                product_code,
                "iam_password_reset",
                &owner_email,
                suffix,
                &expires_at,
            ),
        )
        .await?;
    let reset_record = repository.find_password_reset_by_hash(&reset_hash).await?;
    let reset_done = if let Some(record) = reset_record.as_ref() {
        repository
            .reset_password_with_token(record.id, owner.id, format!("hash-owner-reset-{suffix}"))
            .await?
    } else {
        false
    };
    let password_reset_roundtrip =
        reset_record.is_some_and(|item| item.user_id == owner.id) && reset_done;

    let email_verification_hash = crypto::hash_secret(
        &format!("probe-email-verification-{suffix}"),
        session_secret,
    );
    repository
        .create_email_verification_with_notification(
            crate::repository::CreateEmailVerificationRecord {
                user_id: owner.id,
                email: owner_email.clone(),
                token_hash: email_verification_hash.clone(),
                expires_at: expires_at.clone(),
            },
            probe_notification(
                Some(org.id),
                Some(owner.id),
                product_code,
                "iam_email_verification",
                &owner_email,
                suffix,
                &expires_at,
            ),
        )
        .await?;
    let email_verification_record = repository
        .find_email_verification_by_hash(&email_verification_hash)
        .await?;
    let email_verified = if let Some(record) = email_verification_record.as_ref() {
        repository
            .confirm_email_verification(record.id, owner.id)
            .await?
    } else {
        false
    };
    let email_verification_roundtrip =
        email_verification_record.is_some_and(|item| item.user_id == owner.id) && email_verified;

    let factor = repository
        .create_pending_mfa_factor(CreateMfaFactorRecord {
            user_id: owner.id,
            kind: "totp".into(),
            secret_ciphertext: format!("encrypted-mfa-secret-{suffix}"),
        })
        .await?;
    let pending_factor_found = repository
        .find_pending_mfa_factor(owner.id)
        .await?
        .filter(|item| item.id == factor.id && item.secret_ciphertext.ends_with(suffix))
        .is_some();
    let recovery_records = vec![
        CreateMfaRecoveryCodeRecord {
            code_hash: crypto::hash_secret(&format!("probe-recovery-a-{suffix}"), session_secret),
            code_prefix: format!("ra{}", &suffix[..6]),
        },
        CreateMfaRecoveryCodeRecord {
            code_hash: crypto::hash_secret(&format!("probe-recovery-b-{suffix}"), session_secret),
            code_prefix: format!("rb{}", &suffix[..6]),
        },
    ];
    let activated = repository
        .activate_mfa_factor_with_recovery_codes(owner.id, factor.id, recovery_records)
        .await?;
    let verified_factor_found = repository
        .find_verified_mfa_factor(owner.id)
        .await?
        .filter(|item| item.id == factor.id)
        .is_some();
    let recovery_codes = repository.list_mfa_recovery_codes(owner.id).await?;
    let consumed_recovery = if recovery_codes.is_empty() {
        false
    } else {
        let recovery_hash =
            crypto::hash_secret(&format!("probe-recovery-a-{suffix}"), session_secret);
        repository
            .consume_mfa_recovery_code(owner.id, &recovery_hash)
            .await?
    };
    let rotated_recovery = repository
        .replace_mfa_recovery_codes(
            owner.id,
            vec![CreateMfaRecoveryCodeRecord {
                code_hash: crypto::hash_secret(
                    &format!("probe-recovery-c-{suffix}"),
                    session_secret,
                ),
                code_prefix: format!("rc{}", &suffix[..6]),
            }],
        )
        .await?;
    let revoked_factor = repository.revoke_mfa_factor(owner.id, factor.id).await?;
    let mfa_roundtrip = factor.id > 0
        && pending_factor_found
        && activated
        && verified_factor_found
        && recovery_codes.len() >= 2
        && consumed_recovery
        && rotated_recovery.len() == 1
        && revoked_factor;

    repository
        .record_audit(CreateAuditLogRecord {
            org_id: Some(org.id),
            user_id: Some(owner.id),
            action: format!("iam.repository.probe.{suffix}"),
            scope: "platform".into(),
            product_code: product_code.into(),
            detail: "IAM repository probe completed".into(),
        })
        .await?;
    let audit_record_written = true;

    Ok(DatabaseIamRepositoryProbeReport {
        driver,
        implementation: implementation.into(),
        initial_admin_created,
        permissions_synced,
        organization_roundtrip,
        org_user_roundtrip,
        role_roundtrip,
        session_roundtrip,
        refresh_rotation_roundtrip,
        api_token_roundtrip,
        registration_pending_roundtrip,
        invitation_roundtrip,
        password_reset_roundtrip,
        email_verification_roundtrip,
        mfa_roundtrip,
        audit_record_written,
        message: match driver {
            DatabaseDriver::Sqlite => {
                "SQLite IamRepository probe passed; current driver is app-runtime ready".into()
            }
            DatabaseDriver::Postgres | DatabaseDriver::Mysql => {
                "External IamRepository probe passed; current driver is app-runtime ready".into()
            }
        },
    })
}

fn iam_probe_platform_permission(suffix: &str) -> String {
    format!("probe.iam.platform.{suffix}:read")
}

fn iam_probe_tenant_permission(suffix: &str) -> String {
    format!("probe.iam.tenant.{suffix}:write")
}

fn probe_notification(
    org_id: Option<i64>,
    user_id: Option<i64>,
    product_code: &str,
    related_kind: &str,
    recipient: &str,
    suffix: &str,
    available_at: &str,
) -> CreateNotificationOutboxRecord {
    CreateNotificationOutboxRecord {
        org_id,
        user_id,
        product_code: product_code.into(),
        channel: "email".into(),
        template_code: format!("iam.repository.probe.{related_kind}"),
        recipient: recipient.into(),
        related_kind: related_kind.into(),
        payload_json: format!(r#"{{"probe":"iam-repository","case":"{suffix}"}}"#),
        available_at: available_at.into(),
        delivery_secret_ciphertext: Some(format!("ciphertext-for-iam-probe-{suffix}")),
    }
}

pub async fn probe_notification_repository(
    settings: &Settings,
) -> anyhow::Result<DatabaseNotificationRepositoryProbeReport> {
    match settings.database.driver {
        DatabaseDriver::Sqlite => probe_sqlite_notification_repository(settings).await,
        DatabaseDriver::Postgres => probe_postgres_notification_repository(settings).await,
        DatabaseDriver::Mysql => probe_mysql_notification_repository(settings).await,
    }
}

async fn probe_sqlite_notification_repository(
    settings: &Settings,
) -> anyhow::Result<DatabaseNotificationRepositoryProbeReport> {
    let pool = connect_sqlite_pool(settings, false).await?;
    let repository = SqliteRepository::new(pool.clone());
    let deliver_id =
        seed_sqlite_probe_notification(&pool, &settings.app.product_code, "deliver").await?;
    let retry_id =
        seed_sqlite_probe_notification(&pool, &settings.app.product_code, "retry").await?;
    let fail_id = seed_sqlite_probe_notification(&pool, &settings.app.product_code, "fail").await?;

    let claimed = repository.claim_due_notifications(1000, 1).await?;
    let claimed_probe_items = claimed_probe_items(&claimed, &[deliver_id, retry_id, fail_id]);
    let delivered_result = repository.mark_notification_delivered(deliver_id).await?;
    let retry = repository
        .mark_notification_failed(retry_id, "notification repository probe retry", 60, 3)
        .await?;
    let final_failure = repository
        .mark_notification_failed(
            fail_id,
            "notification repository probe final failure",
            60,
            1,
        )
        .await?;
    let delivered_secret_purged = sqlite_notification_secret_purged(&pool, deliver_id).await?;
    let failed_secret_purged = sqlite_notification_secret_purged(&pool, fail_id).await?;
    let (dead_letter_reported, dead_letter_secret_state) =
        notification_dead_letter_probe(&repository, fail_id).await?;
    let purged_requeue_skipped = !repository.requeue_failed_notification(fail_id).await?;
    let requeue_id =
        seed_sqlite_probe_notification(&pool, &settings.app.product_code, "requeue").await?;
    mark_sqlite_probe_notification_failed_without_purge(&pool, requeue_id).await?;
    let pending_secret_requeue_result = repository.requeue_failed_notification(requeue_id).await?;

    Ok(notification_probe_report(
        DatabaseDriver::Sqlite,
        "SqliteRepository",
        NotificationProbeOutcome {
            claimed_probe_items,
            delivered_result,
            retry_result: retry.retried && !retry.failed,
            final_failure_result: !final_failure.retried && final_failure.failed,
            dead_letter_reported,
            dead_letter_secret_state,
            delivered_secret_purged,
            failed_secret_purged,
            purged_requeue_skipped,
            pending_secret_requeue_result,
        },
    ))
}

async fn probe_postgres_notification_repository(
    settings: &Settings,
) -> anyhow::Result<DatabaseNotificationRepositoryProbeReport> {
    let pool = PgPoolOptions::new()
        .max_connections(settings.database.max_connections)
        .connect(&settings.database.url)
        .await?;
    let repository = ExternalNotificationRepository::postgres(pool.clone());
    let deliver_id =
        seed_postgres_probe_notification(&pool, &settings.app.product_code, "deliver").await?;
    let retry_id =
        seed_postgres_probe_notification(&pool, &settings.app.product_code, "retry").await?;
    let fail_id =
        seed_postgres_probe_notification(&pool, &settings.app.product_code, "fail").await?;

    let claimed = repository.claim_due_notifications(1000, 1).await?;
    let claimed_probe_items = claimed_probe_items(&claimed, &[deliver_id, retry_id, fail_id]);
    let delivered_result = repository.mark_notification_delivered(deliver_id).await?;
    let retry = repository
        .mark_notification_failed(retry_id, "notification repository probe retry", 60, 3)
        .await?;
    let final_failure = repository
        .mark_notification_failed(
            fail_id,
            "notification repository probe final failure",
            60,
            1,
        )
        .await?;
    let delivered_secret_purged = postgres_notification_secret_purged(&pool, deliver_id).await?;
    let failed_secret_purged = postgres_notification_secret_purged(&pool, fail_id).await?;
    let (dead_letter_reported, dead_letter_secret_state) =
        notification_dead_letter_probe(&repository, fail_id).await?;
    let purged_requeue_skipped = !repository.requeue_failed_notification(fail_id).await?;
    let requeue_id =
        seed_postgres_probe_notification(&pool, &settings.app.product_code, "requeue").await?;
    mark_postgres_probe_notification_failed_without_purge(&pool, requeue_id).await?;
    let pending_secret_requeue_result = repository.requeue_failed_notification(requeue_id).await?;

    Ok(notification_probe_report(
        DatabaseDriver::Postgres,
        "ExternalNotificationRepository",
        NotificationProbeOutcome {
            claimed_probe_items,
            delivered_result,
            retry_result: retry.retried && !retry.failed,
            final_failure_result: !final_failure.retried && final_failure.failed,
            dead_letter_reported,
            dead_letter_secret_state,
            delivered_secret_purged,
            failed_secret_purged,
            purged_requeue_skipped,
            pending_secret_requeue_result,
        },
    ))
}

async fn probe_mysql_notification_repository(
    settings: &Settings,
) -> anyhow::Result<DatabaseNotificationRepositoryProbeReport> {
    let pool = MySqlPoolOptions::new()
        .max_connections(settings.database.max_connections)
        .connect(&settings.database.url)
        .await?;
    let repository = ExternalNotificationRepository::mysql(pool.clone());
    let deliver_id =
        seed_mysql_probe_notification(&pool, &settings.app.product_code, "deliver").await?;
    let retry_id =
        seed_mysql_probe_notification(&pool, &settings.app.product_code, "retry").await?;
    let fail_id = seed_mysql_probe_notification(&pool, &settings.app.product_code, "fail").await?;

    let claimed = repository.claim_due_notifications(1000, 1).await?;
    let claimed_probe_items = claimed_probe_items(&claimed, &[deliver_id, retry_id, fail_id]);
    let delivered_result = repository.mark_notification_delivered(deliver_id).await?;
    let retry = repository
        .mark_notification_failed(retry_id, "notification repository probe retry", 60, 3)
        .await?;
    let final_failure = repository
        .mark_notification_failed(
            fail_id,
            "notification repository probe final failure",
            60,
            1,
        )
        .await?;
    let delivered_secret_purged = mysql_notification_secret_purged(&pool, deliver_id).await?;
    let failed_secret_purged = mysql_notification_secret_purged(&pool, fail_id).await?;
    let (dead_letter_reported, dead_letter_secret_state) =
        notification_dead_letter_probe(&repository, fail_id).await?;
    let purged_requeue_skipped = !repository.requeue_failed_notification(fail_id).await?;
    let requeue_id =
        seed_mysql_probe_notification(&pool, &settings.app.product_code, "requeue").await?;
    mark_mysql_probe_notification_failed_without_purge(&pool, requeue_id).await?;
    let pending_secret_requeue_result = repository.requeue_failed_notification(requeue_id).await?;

    Ok(notification_probe_report(
        DatabaseDriver::Mysql,
        "ExternalNotificationRepository",
        NotificationProbeOutcome {
            claimed_probe_items,
            delivered_result,
            retry_result: retry.retried && !retry.failed,
            final_failure_result: !final_failure.retried && final_failure.failed,
            dead_letter_reported,
            dead_letter_secret_state,
            delivered_secret_purged,
            failed_secret_purged,
            purged_requeue_skipped,
            pending_secret_requeue_result,
        },
    ))
}

struct NotificationProbeOutcome {
    claimed_probe_items: usize,
    delivered_result: bool,
    retry_result: bool,
    final_failure_result: bool,
    dead_letter_reported: bool,
    dead_letter_secret_state: String,
    delivered_secret_purged: bool,
    failed_secret_purged: bool,
    purged_requeue_skipped: bool,
    pending_secret_requeue_result: bool,
}

fn notification_probe_report(
    driver: DatabaseDriver,
    implementation: &str,
    outcome: NotificationProbeOutcome,
) -> DatabaseNotificationRepositoryProbeReport {
    DatabaseNotificationRepositoryProbeReport {
        driver,
        implementation: implementation.into(),
        claimed_probe_items: outcome.claimed_probe_items,
        delivered_result: outcome.delivered_result,
        retry_result: outcome.retry_result,
        final_failure_result: outcome.final_failure_result,
        dead_letter_reported: outcome.dead_letter_reported,
        dead_letter_secret_state: outcome.dead_letter_secret_state,
        delivered_secret_purged: outcome.delivered_secret_purged,
        failed_secret_purged: outcome.failed_secret_purged,
        purged_requeue_skipped: outcome.purged_requeue_skipped,
        pending_secret_requeue_result: outcome.pending_secret_requeue_result,
        message: match driver {
            DatabaseDriver::Sqlite => {
                "SQLite NotificationRepository probe passed; current driver is app-runtime ready"
                    .into()
            }
            DatabaseDriver::Postgres | DatabaseDriver::Mysql => {
                "External NotificationRepository probe passed; current driver is app-runtime ready"
                    .into()
            }
        },
    }
}

async fn notification_dead_letter_probe(
    repository: &dyn NotificationRepository,
    failed_id: i64,
) -> anyhow::Result<(bool, String)> {
    let dead_letters = repository.list_failed_notifications(1000).await?;
    let Some(dead_letter) = dead_letters.into_iter().find(|item| item.id == failed_id) else {
        return Ok((false, "missing".into()));
    };
    let secret_state = if dead_letter.delivery_secret_ciphertext.is_some() {
        "pending_secret_present".into()
    } else if dead_letter.delivery_secret_status.as_deref() == Some("purged")
        && dead_letter.delivery_secret_purged_at.is_some()
    {
        "purged".into()
    } else {
        dead_letter
            .delivery_secret_status
            .unwrap_or_else(|| "missing_secret_record".into())
    };
    Ok((true, secret_state))
}

fn claimed_probe_items(
    claimed: &[crate::domain::notification::NotificationOutboxItem],
    expected_ids: &[i64],
) -> usize {
    let claimed_ids = claimed.iter().map(|item| item.id).collect::<HashSet<_>>();
    expected_ids
        .iter()
        .filter(|id| claimed_ids.contains(id))
        .count()
}

async fn seed_sqlite_probe_notification(
    pool: &Pool<Sqlite>,
    product_code: &str,
    label: &str,
) -> anyhow::Result<i64> {
    let dialect = SqlDialect::Sqlite;
    let now = chrono::Utc::now().to_rfc3339();
    let mut tx = pool.begin().await?;
    let outbox_id = sqlx::query_scalar::<_, i64>(dialect.create_notification_outbox())
        .bind(Option::<i64>::None)
        .bind(Option::<i64>::None)
        .bind(product_code)
        .bind("email")
        .bind("repository_probe")
        .bind("probe@example.invalid")
        .bind(format!("notification_repository_probe_{label}"))
        .bind(0_i64)
        .bind(format!(
            r#"{{"probe":"notification-repository","case":"{label}"}}"#
        ))
        .bind(&now)
        .bind(&now)
        .fetch_one(&mut *tx)
        .await?;
    sqlx::query(dialect.create_notification_delivery_secret())
        .bind(outbox_id)
        .bind("ciphertext-for-probe")
        .bind(&now)
        .execute(&mut *tx)
        .await?;
    tx.commit().await?;
    Ok(outbox_id)
}

async fn seed_postgres_probe_notification(
    pool: &Pool<Postgres>,
    product_code: &str,
    label: &str,
) -> anyhow::Result<i64> {
    let dialect = SqlDialect::Postgres;
    let now = chrono::Utc::now().to_rfc3339();
    let mut tx = pool.begin().await?;
    let outbox_id = sqlx::query_scalar::<_, i64>(dialect.create_notification_outbox())
        .bind(Option::<i64>::None)
        .bind(Option::<i64>::None)
        .bind(product_code)
        .bind("email")
        .bind("repository_probe")
        .bind("probe@example.invalid")
        .bind(format!("notification_repository_probe_{label}"))
        .bind(0_i64)
        .bind(format!(
            r#"{{"probe":"notification-repository","case":"{label}"}}"#
        ))
        .bind(&now)
        .bind(&now)
        .fetch_one(&mut *tx)
        .await?;
    sqlx::query(dialect.create_notification_delivery_secret())
        .bind(outbox_id)
        .bind("ciphertext-for-probe")
        .bind(&now)
        .execute(&mut *tx)
        .await?;
    tx.commit().await?;
    Ok(outbox_id)
}

async fn seed_mysql_probe_notification(
    pool: &Pool<MySql>,
    product_code: &str,
    label: &str,
) -> anyhow::Result<i64> {
    let dialect = SqlDialect::Mysql;
    let now = chrono::Utc::now().to_rfc3339();
    let mut tx = pool.begin().await?;
    sqlx::query(dialect.create_notification_outbox())
        .bind(Option::<i64>::None)
        .bind(Option::<i64>::None)
        .bind(product_code)
        .bind("email")
        .bind("repository_probe")
        .bind("probe@example.invalid")
        .bind(format!("notification_repository_probe_{label}"))
        .bind(0_i64)
        .bind(format!(
            r#"{{"probe":"notification-repository","case":"{label}"}}"#
        ))
        .bind(&now)
        .bind(&now)
        .execute(&mut *tx)
        .await?;
    let outbox_id = sqlx::query_scalar::<_, i64>("select cast(last_insert_id() as signed)")
        .fetch_one(&mut *tx)
        .await?;
    sqlx::query(dialect.create_notification_delivery_secret())
        .bind(outbox_id)
        .bind("ciphertext-for-probe")
        .bind(&now)
        .execute(&mut *tx)
        .await?;
    tx.commit().await?;
    Ok(outbox_id)
}

async fn mark_sqlite_probe_notification_failed_without_purge(
    pool: &Pool<Sqlite>,
    outbox_id: i64,
) -> anyhow::Result<()> {
    let now = chrono::Utc::now().to_rfc3339();
    sqlx::query(
        "update iam_notification_outbox
         set status = 'failed',
             attempt_count = 2,
             failed_at = ?,
             failure_reason = ?
         where id = ?",
    )
    .bind(&now)
    .bind("notification repository probe requeue")
    .bind(outbox_id)
    .execute(pool)
    .await?;
    Ok(())
}

async fn mark_postgres_probe_notification_failed_without_purge(
    pool: &Pool<Postgres>,
    outbox_id: i64,
) -> anyhow::Result<()> {
    let now = chrono::Utc::now().to_rfc3339();
    sqlx::query(
        "update iam_notification_outbox
         set status = 'failed',
             attempt_count = 2,
             failed_at = $1,
             failure_reason = $2
         where id = $3",
    )
    .bind(&now)
    .bind("notification repository probe requeue")
    .bind(outbox_id)
    .execute(pool)
    .await?;
    Ok(())
}

async fn mark_mysql_probe_notification_failed_without_purge(
    pool: &Pool<MySql>,
    outbox_id: i64,
) -> anyhow::Result<()> {
    let now = chrono::Utc::now().to_rfc3339();
    sqlx::query(
        "update iam_notification_outbox
         set status = 'failed',
             attempt_count = 2,
             failed_at = ?,
             failure_reason = ?
         where id = ?",
    )
    .bind(&now)
    .bind("notification repository probe requeue")
    .bind(outbox_id)
    .execute(pool)
    .await?;
    Ok(())
}

async fn sqlite_notification_secret_purged(
    pool: &Pool<Sqlite>,
    outbox_id: i64,
) -> anyhow::Result<bool> {
    let count = sqlx::query_scalar::<_, i64>(
        "select count(*)
         from iam_notification_delivery_secrets
         where outbox_id = ? and status = 'purged' and secret_ciphertext is null",
    )
    .bind(outbox_id)
    .fetch_one(pool)
    .await?;
    Ok(count > 0)
}

async fn postgres_notification_secret_purged(
    pool: &Pool<Postgres>,
    outbox_id: i64,
) -> anyhow::Result<bool> {
    let count = sqlx::query_scalar::<_, i64>(
        "select count(*)::bigint
         from iam_notification_delivery_secrets
         where outbox_id = $1 and status = 'purged' and secret_ciphertext is null",
    )
    .bind(outbox_id)
    .fetch_one(pool)
    .await?;
    Ok(count > 0)
}

async fn mysql_notification_secret_purged(
    pool: &Pool<MySql>,
    outbox_id: i64,
) -> anyhow::Result<bool> {
    let count = sqlx::query_scalar::<_, i64>(
        "select cast(count(*) as signed)
         from iam_notification_delivery_secrets
         where outbox_id = ? and status = 'purged' and secret_ciphertext is null",
    )
    .bind(outbox_id)
    .fetch_one(pool)
    .await?;
    Ok(count > 0)
}

pub async fn probe_system_repository(
    settings: &Settings,
) -> anyhow::Result<DatabaseSystemRepositoryProbeReport> {
    match settings.database.driver {
        DatabaseDriver::Sqlite => {
            let pool = connect_sqlite_pool(settings, false).await?;
            let repository = SqliteRepository::new(pool);
            probe_system_repository_with(
                DatabaseDriver::Sqlite,
                "SqliteRepository",
                &repository,
                &settings.app.product_code,
            )
            .await
        }
        DatabaseDriver::Postgres => {
            let pool = PgPoolOptions::new()
                .max_connections(settings.database.max_connections)
                .connect(&settings.database.url)
                .await?;
            let repository = ExternalSystemRepository::postgres(pool);
            probe_system_repository_with(
                DatabaseDriver::Postgres,
                "ExternalSystemRepository",
                &repository,
                &settings.app.product_code,
            )
            .await
        }
        DatabaseDriver::Mysql => {
            let pool = MySqlPoolOptions::new()
                .max_connections(settings.database.max_connections)
                .connect(&settings.database.url)
                .await?;
            let repository = ExternalSystemRepository::mysql(pool);
            probe_system_repository_with(
                DatabaseDriver::Mysql,
                "ExternalSystemRepository",
                &repository,
                &settings.app.product_code,
            )
            .await
        }
    }
}

async fn probe_system_repository_with(
    driver: DatabaseDriver,
    implementation: &str,
    repository: &dyn SystemRepository,
    product_code: &str,
) -> anyhow::Result<DatabaseSystemRepositoryProbeReport> {
    let suffix = Uuid::new_v4().simple().to_string();
    let permission = format!("system.probe.{suffix}:read");
    let api_path = format!("/__probe/system/{suffix}");
    let api_entry = ApiCatalogEntry {
        id: format!("probe.system.{suffix}"),
        method: "GET".into(),
        path: api_path.clone(),
        tag: "probe".into(),
        summary: "SystemRepository probe".into(),
        access: "authenticated".into(),
        permission: Some(permission.clone()),
        scope: "platform".into(),
        product_code: product_code.into(),
    };
    repository.sync_api_catalog(&[api_entry]).await?;
    let api_catalog_synced =
        repository.list_api_catalog().await?.iter().any(|entry| {
            entry.path == api_path && entry.permission.as_deref() == Some(&permission)
        });

    let menu_code = format!("probe.system.{suffix}");
    repository
        .sync_system_menus(&[SystemMenuEntry {
            code: menu_code.clone(),
            title: "System Probe".into(),
            path: api_path.clone(),
            permission: Some(permission),
            scope: "platform".into(),
            sort_order: 9000,
        }])
        .await?;
    let menu_synced = repository
        .list_system_menus()
        .await?
        .iter()
        .any(|menu| menu.code == menu_code && menu.path == api_path);

    let config_key = format!("probe.system.config.{suffix}");
    let config_value = serde_json::json!({
        "probe": "system-repository",
        "case": suffix,
    });
    let config = repository
        .upsert_system_config(UpsertSystemConfigRecord {
            key: config_key.clone(),
            value_json: config_value.to_string(),
        })
        .await?;
    let config_listed = repository
        .list_system_configs()
        .await?
        .iter()
        .any(|item| item.key == config_key);
    let config_roundtrip =
        config.key == config_key && config.value == config_value && config_listed;
    let config_deleted = repository.delete_system_config(&config_key).await?;

    let dictionary_code = format!("probe_system_dictionary_{suffix}");
    let dictionary = repository
        .upsert_system_dictionary(UpsertSystemDictionaryRecord {
            code: dictionary_code.clone(),
            name: "System Probe Dictionary".into(),
        })
        .await?;
    let dictionary_listed = repository
        .list_system_dictionaries()
        .await?
        .iter()
        .any(|item| item.code == dictionary_code);
    let dictionary_deleted = repository
        .delete_system_dictionary(&dictionary_code)
        .await?;
    let dictionary_roundtrip =
        dictionary.code == dictionary_code && dictionary_listed && dictionary_deleted;

    let parameter_key = format!("probe.system.parameter.{suffix}");
    let parameter = repository
        .upsert_system_parameter(UpsertSystemParameterRecord {
            key: parameter_key.clone(),
            name: "System Probe Parameter".into(),
            value: "enabled".into(),
        })
        .await?;
    let parameter_listed = repository
        .list_system_parameters()
        .await?
        .iter()
        .any(|item| item.key == parameter_key);
    let parameter_deleted = repository.delete_system_parameter(&parameter_key).await?;
    let parameter_roundtrip =
        parameter.key == parameter_key && parameter_listed && parameter_deleted;

    repository
        .create_operation_record(CreateOperationRecord {
            actor_user_id: None,
            method: "PROBE".into(),
            path: api_path.clone(),
            status: 204,
        })
        .await?;
    let operation_record_written = repository
        .list_operation_records(OperationRecordListQuery {
            method: None,
            path: None,
            status: None,
            actor_user_id: None,
            created_from: None,
            created_to: None,
            limit: 50,
            offset: 0,
        })
        .await?
        .iter()
        .any(|record| record.path == api_path && record.status == 204);
    let operation_summary = repository
        .summarize_operation_records(OperationRecordSummaryFilter {
            method: Some("PROBE".into()),
            path: Some(api_path.clone()),
            status: None,
            actor_user_id: None,
            created_from: None,
            created_to: None,
            top_limit: 5,
        })
        .await?;
    let operation_record_summary_reported = operation_summary.total_count >= 1
        && operation_summary.success_count >= 1
        && operation_summary
            .top_paths
            .iter()
            .any(|item| item.path == api_path && item.count >= 1 && item.last_seen_at.is_some())
        && operation_summary
            .by_method
            .iter()
            .any(|item| item.key == "PROBE" && item.count >= 1)
        && operation_summary
            .by_status_class
            .iter()
            .any(|item| item.key == "2xx" && item.count >= 1);
    let operation_record_retention_prune = repository
        .prune_operation_records("1970-01-01T00:00:00Z", 1)
        .await?
        == 0;

    let version = repository
        .create_version_package(CreateVersionPackageRecord {
            version_name: format!("probe-{suffix}"),
            version_code: format!("probe-{suffix}"),
            manifest_json: serde_json::json!({"probe": "system-repository"}).to_string(),
        })
        .await?;
    let published = repository
        .publish_version_package(VersionPackageActionRecord {
            id: version.id,
            reason: Some("system repository probe".into()),
        })
        .await?;
    let release_event_listed = repository
        .list_version_release_events()
        .await?
        .iter()
        .any(|event| event.id == published.event_id && event.package_id == version.id);
    let version_package_roundtrip = version.id > 0
        && published.package.id == version.id
        && published.package.status == "active"
        && release_event_listed;

    let media_storage_key = format!("probe/system/{suffix}.txt");
    let media = repository
        .create_media_asset(CreateMediaAssetRecord {
            category: Some("probe".into()),
            display_name: "System Probe Media".into(),
            storage_key: media_storage_key.clone(),
            mime_type: "text/plain".into(),
            size_bytes: 16,
        })
        .await?;
    let media_listed = repository
        .list_media_assets()
        .await?
        .iter()
        .any(|asset| asset.id == media.id && asset.storage_key == media_storage_key);
    let media_deleted = repository.delete_media_asset(media.id).await?;
    let media_asset_roundtrip = media.id > 0 && media_listed && media_deleted;

    let target = repository
        .create_traffic_probe_target(CreateTrafficProbeTargetRecord {
            name: format!("System Probe {suffix}"),
            url: format!("https://example.invalid/probe/{suffix}"),
            expected_status: 204,
        })
        .await?;
    let target_found = repository
        .find_traffic_probe_target(target.id)
        .await?
        .is_some();
    let result = repository
        .create_traffic_probe_result(CreateTrafficProbeResultRecord {
            target_id: target.id,
            status: "ok".into(),
            detail_json: serde_json::json!({"probe": "system-repository"}).to_string(),
        })
        .await?;
    let result_listed = repository
        .list_traffic_probe_results(Some(target.id), 10)
        .await?
        .iter()
        .any(|item| item.id == result.id && item.target_id == target.id);
    let alert = repository
        .create_traffic_probe_alert(CreateTrafficProbeAlertRecord {
            target_id: target.id,
            result_id: result.id,
            severity: "warning".into(),
            reason: "system repository probe".into(),
            detail_json: serde_json::json!({"probe": "system-repository"}).to_string(),
        })
        .await?;
    let alert_listed = repository
        .list_traffic_probe_alerts(Some(target.id), Some("open".into()), 10)
        .await?
        .iter()
        .any(|item| item.id == alert.id);
    let alert_acknowledged = repository.acknowledge_traffic_probe_alert(alert.id).await?;
    let alerts_resolved = repository
        .resolve_traffic_probe_alerts_for_target(target.id)
        .await?;
    let target_deleted = repository.delete_traffic_probe_target(target.id).await?;
    let traffic_probe_roundtrip = target.id > 0
        && target_found
        && result_listed
        && alert_listed
        && alert_acknowledged
        && alerts_resolved >= 1
        && target_deleted;

    Ok(DatabaseSystemRepositoryProbeReport {
        driver,
        implementation: implementation.into(),
        api_catalog_synced,
        menu_synced,
        config_roundtrip: config_roundtrip && config_deleted,
        dictionary_roundtrip,
        parameter_roundtrip,
        operation_record_written,
        operation_record_summary_reported,
        operation_record_retention_prune,
        version_package_roundtrip,
        media_asset_roundtrip,
        traffic_probe_roundtrip,
        message: match driver {
            DatabaseDriver::Sqlite => {
                "SQLite SystemRepository probe passed; current driver is app-runtime ready".into()
            }
            DatabaseDriver::Postgres | DatabaseDriver::Mysql => {
                "External SystemRepository probe passed; current driver is app-runtime ready".into()
            }
        },
    })
}

pub async fn preflight_database(settings: &Settings) -> anyhow::Result<DatabasePreflightReport> {
    let support = settings.database.driver.runtime_support();
    let repository_runtime = repository_runtime_report(settings.database.driver);
    let mut checks = Vec::new();
    let mut migration_plan_ok = false;
    let mut migration_history_ok = false;
    let mut connection_ok = false;
    let mut schema_ready = false;
    let mut repository_ready = false;
    let mut plan_files = Vec::new();

    match migration_plan(settings) {
        Ok(plan) => {
            migration_plan_ok = true;
            plan_files = plan.migration_files.clone();
            checks.push(preflight_ok(
                "migration-plan",
                "迁移计划",
                format!(
                    "已找到 {} 个迁移文件；runtime_status={}",
                    plan_files.len(),
                    plan.runtime_status
                ),
            ));
        }
        Err(err) => checks.push(preflight_error(
            "migration-plan",
            "迁移计划",
            format!("迁移计划不可用：{err}"),
        )),
    }

    match ping_database(settings).await {
        Ok(report) => {
            connection_ok = report.connection_ok;
            checks.push(preflight_ok("database-ping", "连接探针", report.message));
        }
        Err(err) => checks.push(preflight_error(
            "database-ping",
            "连接探针",
            format!("数据库连接探针失败：{err}"),
        )),
    }

    match migration_history(settings).await {
        Ok(history) => {
            let history_check = migration_history_preflight_check(&history, &plan_files);
            migration_history_ok = history_check.status == "ok";
            checks.push(history_check);
        }
        Err(err) => checks.push(preflight_error(
            "migration-history",
            "迁移历史",
            format!("迁移历史不可用：{err}"),
        )),
    }

    match check_database_schema(settings).await {
        Ok(report) => {
            schema_ready = report.schema_ready;
            repository_ready = report.repository_ready;
            let check = if report.schema_ready {
                preflight_ok("schema-check", "核心表", report.message)
            } else {
                preflight_error("schema-check", "核心表", report.message)
            };
            checks.push(check);
        }
        Err(err) => checks.push(preflight_error(
            "schema-check",
            "核心表",
            format!("核心表检查失败：{err}"),
        )),
    }

    checks.push(repository_runtime_preflight_check(&repository_runtime));

    let serve_ready = support.supported
        && connection_ok
        && migration_plan_ok
        && migration_history_ok
        && schema_ready
        && repository_ready;
    let message = if serve_ready {
        "数据库预检通过；当前 driver 可用于 serve runtime".into()
    } else if schema_ready && repository_ready && !support.supported {
        format!(
            "数据库 schema 与 repository traits 已就绪，但 database.driver={} 的 runtime_support 仍为未支持，不能用于 serve",
            settings.database.driver
        )
    } else if schema_ready && !repository_ready {
        format!(
            "数据库 schema 已就绪，但 database.driver={} 的 repository runtime 未就绪，不能用于 serve",
            settings.database.driver
        )
    } else {
        "数据库预检未完全通过；请查看 checks 中的阻断项".into()
    };

    Ok(DatabasePreflightReport {
        driver: settings.database.driver,
        runtime_supported: support.supported,
        runtime_status: support.status,
        connection_ok,
        migration_plan_ok,
        migration_history_ok,
        schema_ready,
        repository_runtime,
        repository_ready,
        serve_ready,
        checks,
        message,
    })
}

pub async fn check_database_schema(
    settings: &Settings,
) -> anyhow::Result<DatabaseSchemaCheckReport> {
    let support = settings.database.driver.runtime_support();
    let repository_runtime = repository_runtime_report(settings.database.driver);
    let existing_tables = match settings.database.driver {
        DatabaseDriver::Sqlite => {
            let pool = connect_sqlite_pool(settings, false).await?;
            sqlite_tables(&pool).await?
        }
        DatabaseDriver::Postgres => {
            let pool = PgPoolOptions::new()
                .max_connections(1)
                .connect(&settings.database.url)
                .await?;
            postgres_tables(&pool).await?
        }
        DatabaseDriver::Mysql => {
            let pool = MySqlPoolOptions::new()
                .max_connections(1)
                .connect(&settings.database.url)
                .await?;
            mysql_tables(&pool).await?
        }
    };
    let checked_tables = REQUIRED_RUNTIME_TABLES
        .iter()
        .map(|table| (*table).to_string())
        .collect::<Vec<_>>();
    let missing_tables = checked_tables
        .iter()
        .filter(|table| !existing_tables.contains(table.as_str()))
        .cloned()
        .collect::<Vec<_>>();
    let schema_ready = missing_tables.is_empty();
    let repository_ready = repository_runtime.ready && schema_ready;
    let message = if schema_ready {
        match settings.database.driver {
            DatabaseDriver::Sqlite => {
                "SQLite 核心表检查通过；该 driver 已接入 app runtime repository".into()
            }
            DatabaseDriver::Postgres => {
                "PostgreSQL 核心表检查通过；schema 与 repository traits 已接入 app runtime".into()
            }
            DatabaseDriver::Mysql => {
                "MySQL 核心表检查通过；schema 与 repository traits 已接入 app runtime".into()
            }
        }
    } else {
        format!("数据库核心表缺失：{}", missing_tables.join("、"))
    };

    Ok(DatabaseSchemaCheckReport {
        driver: settings.database.driver,
        runtime_supported: support.supported,
        runtime_status: support.status,
        checked_tables,
        missing_tables,
        schema_ready,
        repository_runtime,
        repository_ready,
        message,
    })
}

pub fn migration_plan(settings: &Settings) -> anyhow::Result<DatabaseMigrationPlan> {
    let driver = settings.database.driver;
    let support = driver.runtime_support();
    let migration_dir = migration_dir_for_driver(driver);
    let migration_files = collect_migration_files(&migration_dir)?;
    if migration_files.is_empty() {
        anyhow::bail!(
            "database.driver={driver} 没有可用迁移文件：{}",
            display_path(&migration_dir)
        );
    }

    Ok(DatabaseMigrationPlan {
        driver,
        runtime_supported: support.supported,
        runtime_status: support.status,
        runtime_message: support.message,
        required_work: support.required_work,
        auto_apply: settings.migration.auto_apply,
        migration_dir: display_path(&migration_dir),
        migration_files,
    })
}

async fn sqlite_tables(pool: &Pool<Sqlite>) -> anyhow::Result<HashSet<String>> {
    let tables =
        sqlx::query_scalar::<_, String>("select name from sqlite_master where type = 'table'")
            .fetch_all(pool)
            .await?;
    Ok(normalize_table_names(tables))
}

async fn postgres_tables(pool: &Pool<Postgres>) -> anyhow::Result<HashSet<String>> {
    let tables = sqlx::query_scalar::<_, String>(
        "select table_name from information_schema.tables where table_schema = 'public'",
    )
    .fetch_all(pool)
    .await?;
    Ok(normalize_table_names(tables))
}

async fn mysql_tables(pool: &Pool<MySql>) -> anyhow::Result<HashSet<String>> {
    let tables = sqlx::query_scalar::<_, String>(
        "select table_name from information_schema.tables where table_schema = database()",
    )
    .fetch_all(pool)
    .await?;
    Ok(normalize_table_names(tables))
}

fn normalize_table_names(tables: Vec<String>) -> HashSet<String> {
    tables
        .into_iter()
        .map(|table| table.to_lowercase())
        .collect()
}

async fn connect_sqlite(settings: &Settings) -> anyhow::Result<Pool<Sqlite>> {
    connect_sqlite_pool(settings, settings.migration.auto_apply).await
}

async fn connect_postgres(settings: &Settings) -> anyhow::Result<DatabaseConnection> {
    if settings.migration.auto_apply {
        let plan = migration_plan(settings)?;
        migrate_postgres(settings, plan).await?;
    }
    let pool = PgPoolOptions::new()
        .max_connections(settings.database.max_connections)
        .connect(&settings.database.url)
        .await?;
    let iam_pool = connect_any_pool(settings).await?;
    Ok(DatabaseConnection::Postgres { pool, iam_pool })
}

async fn connect_mysql(settings: &Settings) -> anyhow::Result<DatabaseConnection> {
    if settings.migration.auto_apply {
        let plan = migration_plan(settings)?;
        migrate_mysql(settings, plan).await?;
    }
    let pool = MySqlPoolOptions::new()
        .max_connections(settings.database.max_connections)
        .connect(&settings.database.url)
        .await?;
    let iam_pool = connect_any_pool(settings).await?;
    Ok(DatabaseConnection::Mysql { pool, iam_pool })
}

async fn connect_any_pool(settings: &Settings) -> anyhow::Result<Pool<Any>> {
    install_default_drivers();
    AnyPoolOptions::new()
        .max_connections(settings.database.max_connections)
        .connect(&settings.database.url)
        .await
        .map_err(Into::into)
}

async fn connect_sqlite_pool(
    settings: &Settings,
    apply_migrations: bool,
) -> anyhow::Result<Pool<Sqlite>> {
    ensure_sqlite_parent(&settings.database.url)?;
    let options = SqliteConnectOptions::from_str(&settings.database.url)?
        .create_if_missing(true)
        .journal_mode(SqliteJournalMode::Wal)
        .foreign_keys(true);

    let pool = SqlitePoolOptions::new()
        .max_connections(settings.database.max_connections)
        .connect_with(options)
        .await?;

    if apply_migrations {
        SQLITE_MIGRATOR.run(&pool).await?;
    }

    Ok(pool)
}

async fn migrate_sqlite(
    settings: &Settings,
    plan: DatabaseMigrationPlan,
) -> anyhow::Result<DatabaseMigrateReport> {
    let pool = connect_sqlite_pool(settings, false).await?;
    let before_versions = sqlite_applied_migration_versions(&pool).await;
    SQLITE_MIGRATOR.run(&pool).await?;
    let after_versions = sqlite_applied_migration_versions(&pool).await;
    let mut applied_files = Vec::new();
    let mut skipped_files = Vec::new();
    for file in &plan.migration_files {
        let version = migration_version(&file.name)?;
        if before_versions.contains(&version) {
            skipped_files.push(file.name.clone());
        } else if after_versions.contains(&version) {
            applied_files.push(file.name.clone());
        }
    }

    Ok(DatabaseMigrateReport {
        driver: DatabaseDriver::Sqlite,
        migration_dir: plan.migration_dir,
        applied_files,
        skipped_files,
        repository_runtime: repository_runtime_report(DatabaseDriver::Sqlite),
        repository_ready: true,
        message: "SQLite 迁移执行完成；该 driver 已接入 app runtime repository".into(),
    })
}

async fn sqlite_applied_migration_versions(pool: &Pool<Sqlite>) -> HashSet<i64> {
    sqlx::query_scalar::<_, i64>("select version from _sqlx_migrations where success = true")
        .fetch_all(pool)
        .await
        .unwrap_or_default()
        .into_iter()
        .collect()
}

async fn sqlite_migration_history(
    pool: &Pool<Sqlite>,
) -> anyhow::Result<Vec<DatabaseMigrationHistoryRecord>> {
    let rows = sqlx::query(
        "select version, description, lower(hex(checksum)) as checksum, success
         from _sqlx_migrations
         order by version",
    )
    .fetch_all(pool)
    .await?;
    Ok(rows
        .into_iter()
        .map(|row| DatabaseMigrationHistoryRecord {
            version: row.get("version"),
            name: row.get("description"),
            checksum: row.get("checksum"),
            success: row.get::<bool, _>("success"),
        })
        .collect())
}

async fn migrate_postgres(
    settings: &Settings,
    plan: DatabaseMigrationPlan,
) -> anyhow::Result<DatabaseMigrateReport> {
    let pool = PgPoolOptions::new()
        .max_connections(settings.database.max_connections)
        .connect(&settings.database.url)
        .await?;
    sqlx::query(
        "create table if not exists schema_migrations (
            version bigint primary key,
            name text not null,
            checksum text not null,
            applied_at timestamptz not null default now()
        )",
    )
    .execute(&pool)
    .await?;

    let mut applied_files = Vec::new();
    let mut skipped_files = Vec::new();
    for file in &plan.migration_files {
        let version = migration_version(&file.name)?;
        let sql = fs::read_to_string(&file.path)?;
        let checksum = crypto::sha256_hex(sql.as_bytes());
        let recorded_checksum: Option<String> =
            sqlx::query_scalar("select checksum from schema_migrations where version = $1")
                .bind(version)
                .fetch_optional(&pool)
                .await?;
        if let Some(recorded_checksum) = recorded_checksum {
            ensure_migration_checksum_matches(
                DatabaseDriver::Postgres,
                &file.name,
                &recorded_checksum,
                &checksum,
            )?;
            skipped_files.push(file.name.clone());
            continue;
        }

        sqlx::raw_sql(&sql).execute(&pool).await?;
        sqlx::query("insert into schema_migrations (version, name, checksum) values ($1, $2, $3)")
            .bind(version)
            .bind(&file.name)
            .bind(checksum)
            .execute(&pool)
            .await?;
        applied_files.push(file.name.clone());
    }

    Ok(DatabaseMigrateReport {
        driver: DatabaseDriver::Postgres,
        migration_dir: plan.migration_dir,
        applied_files,
        skipped_files,
        repository_runtime: repository_runtime_report(DatabaseDriver::Postgres),
        repository_ready: true,
        message: "PostgreSQL 迁移执行完成；该 driver 已接入 app runtime repository".into(),
    })
}

async fn postgres_migration_history(
    pool: &Pool<Postgres>,
) -> anyhow::Result<Vec<DatabaseMigrationHistoryRecord>> {
    let rows = sqlx::query(
        "select version, name, checksum
         from schema_migrations
         order by version",
    )
    .fetch_all(pool)
    .await?;
    Ok(rows
        .into_iter()
        .map(|row| DatabaseMigrationHistoryRecord {
            version: row.get("version"),
            name: row.get("name"),
            checksum: row.get("checksum"),
            success: true,
        })
        .collect())
}

async fn migrate_mysql(
    settings: &Settings,
    plan: DatabaseMigrationPlan,
) -> anyhow::Result<DatabaseMigrateReport> {
    let pool = MySqlPoolOptions::new()
        .max_connections(settings.database.max_connections)
        .connect(&settings.database.url)
        .await?;
    sqlx::query(
        "create table if not exists schema_migrations (
            version bigint primary key,
            name varchar(255) not null,
            checksum varchar(255) not null,
            applied_at timestamp not null default current_timestamp
        ) engine=innodb",
    )
    .execute(&pool)
    .await?;

    let mut applied_files = Vec::new();
    let mut skipped_files = Vec::new();
    for file in &plan.migration_files {
        let version = migration_version(&file.name)?;
        let sql = fs::read_to_string(&file.path)?;
        let checksum = crypto::sha256_hex(sql.as_bytes());
        let recorded_checksum: Option<String> =
            sqlx::query_scalar("select checksum from schema_migrations where version = ?")
                .bind(version)
                .fetch_optional(&pool)
                .await?;
        if let Some(recorded_checksum) = recorded_checksum {
            ensure_migration_checksum_matches(
                DatabaseDriver::Mysql,
                &file.name,
                &recorded_checksum,
                &checksum,
            )?;
            skipped_files.push(file.name.clone());
            continue;
        }

        sqlx::raw_sql(&sql).execute(&pool).await?;
        sqlx::query("insert into schema_migrations (version, name, checksum) values (?, ?, ?)")
            .bind(version)
            .bind(&file.name)
            .bind(checksum)
            .execute(&pool)
            .await?;
        applied_files.push(file.name.clone());
    }

    Ok(DatabaseMigrateReport {
        driver: DatabaseDriver::Mysql,
        migration_dir: plan.migration_dir,
        applied_files,
        skipped_files,
        repository_runtime: repository_runtime_report(DatabaseDriver::Mysql),
        repository_ready: true,
        message: "MySQL 迁移执行完成；该 driver 已接入 app runtime repository".into(),
    })
}

async fn mysql_migration_history(
    pool: &Pool<MySql>,
) -> anyhow::Result<Vec<DatabaseMigrationHistoryRecord>> {
    let rows = sqlx::query(
        "select version, name, checksum
         from schema_migrations
         order by version",
    )
    .fetch_all(pool)
    .await?;
    Ok(rows
        .into_iter()
        .map(|row| DatabaseMigrationHistoryRecord {
            version: row.get("version"),
            name: row.get("name"),
            checksum: row.get("checksum"),
            success: true,
        })
        .collect())
}

fn sql_dialect_for_driver(driver: DatabaseDriver) -> SqlDialect {
    match driver {
        DatabaseDriver::Sqlite => SqlDialect::Sqlite,
        DatabaseDriver::Postgres => SqlDialect::Postgres,
        DatabaseDriver::Mysql => SqlDialect::Mysql,
    }
}

fn insert_id_read_label(read: InsertIdRead) -> String {
    match read {
        InsertIdRead::ReturningIdInStatement => "ReturningIdInStatement".into(),
        InsertIdRead::PostInsertQuery(query) => format!("PostInsertQuery({query})"),
    }
}

async fn probe_sqlite_insert_id(settings: &Settings) -> anyhow::Result<i64> {
    let pool = connect_sqlite_pool(settings, false).await?;
    let mut tx = pool.begin().await?;
    sqlx::query("drop table if exists console_insert_id_probe")
        .execute(&mut *tx)
        .await?;
    sqlx::query(
        "create temporary table console_insert_id_probe (
            id integer primary key autoincrement,
            label text not null
        )",
    )
    .execute(&mut *tx)
    .await?;
    let inserted_id = sqlx::query_scalar::<_, i64>(
        "insert into console_insert_id_probe (label) values (?) returning id",
    )
    .bind("probe")
    .fetch_one(&mut *tx)
    .await?;
    sqlx::query("drop table console_insert_id_probe")
        .execute(&mut *tx)
        .await?;
    tx.commit().await?;
    Ok(inserted_id)
}

async fn probe_postgres_insert_id(settings: &Settings) -> anyhow::Result<i64> {
    let pool = PgPoolOptions::new()
        .max_connections(settings.database.max_connections)
        .connect(&settings.database.url)
        .await?;
    let mut tx = pool.begin().await?;
    sqlx::query(
        "create temporary table console_insert_id_probe (
            id bigint generated by default as identity primary key,
            label text not null
        ) on commit drop",
    )
    .execute(&mut *tx)
    .await?;
    let inserted_id = sqlx::query_scalar::<_, i64>(
        "insert into console_insert_id_probe (label) values ($1) returning id",
    )
    .bind("probe")
    .fetch_one(&mut *tx)
    .await?;
    tx.commit().await?;
    Ok(inserted_id)
}

async fn probe_mysql_insert_id(settings: &Settings) -> anyhow::Result<i64> {
    let pool = MySqlPoolOptions::new()
        .max_connections(settings.database.max_connections)
        .connect(&settings.database.url)
        .await?;
    let mut connection = pool.acquire().await?;
    sqlx::query("drop temporary table if exists console_insert_id_probe")
        .execute(&mut *connection)
        .await?;
    sqlx::query(
        "create temporary table console_insert_id_probe (
            id bigint not null auto_increment primary key,
            label varchar(64) not null
        ) engine=innodb",
    )
    .execute(&mut *connection)
    .await?;
    sqlx::query("insert into console_insert_id_probe (label) values (?)")
        .bind("probe")
        .execute(&mut *connection)
        .await?;
    let inserted_id = sqlx::query_scalar::<_, i64>("select cast(last_insert_id() as signed)")
        .fetch_one(&mut *connection)
        .await?;
    sqlx::query("drop temporary table console_insert_id_probe")
        .execute(&mut *connection)
        .await?;
    Ok(inserted_id)
}

fn migration_history_preflight_check(
    history: &DatabaseMigrationHistoryReport,
    plan_files: &[DatabaseMigrationFile],
) -> DatabasePreflightCheck {
    if history.records.is_empty() {
        return preflight_error(
            "migration-history",
            "迁移历史",
            "当前数据库没有已应用迁移记录",
        );
    }
    if let Some(record) = history.records.iter().find(|record| !record.success) {
        return preflight_error(
            "migration-history",
            "迁移历史",
            format!("迁移记录 {} 未成功应用", record.name),
        );
    }

    if history.checksum_source == "sha256" && !plan_files.is_empty() {
        for file in plan_files {
            let matches = history
                .records
                .iter()
                .filter(|record| record.name == file.name)
                .collect::<Vec<_>>();
            if matches.len() != 1 {
                return preflight_error(
                    "migration-history",
                    "迁移历史",
                    format!(
                        "迁移历史中找到 {} 条 {} 记录，预期为 1 条",
                        matches.len(),
                        file.name
                    ),
                );
            }
            if matches[0].checksum != file.sha256 {
                return preflight_error(
                    "migration-history",
                    "迁移历史",
                    format!(
                        "迁移 {} 的 SHA-256 与计划不一致；请确认没有改写已发布迁移",
                        file.name
                    ),
                );
            }
        }
    }

    preflight_ok(
        "migration-history",
        "迁移历史",
        format!(
            "已读取 {} 条迁移历史记录；checksum_source={}",
            history.records.len(),
            history.checksum_source
        ),
    )
}

fn repository_runtime_preflight_check(report: &RepositoryRuntimeReport) -> DatabasePreflightCheck {
    if report.ready {
        return preflight_ok(
            "repository-runtime",
            "Repository 覆盖",
            "Setup/IAM/Notification/System repository trait 已接入当前 driver",
        );
    }

    preflight_error(
        "repository-runtime",
        "Repository 覆盖",
        format!(
            "当前 driver={} 尚未完成 app runtime repository 覆盖：{}",
            report.driver,
            report.blockers.join("、")
        ),
    )
}

fn preflight_ok(key: &str, title: &str, message: impl Into<String>) -> DatabasePreflightCheck {
    preflight_check(key, title, "ok", "info", message)
}

fn preflight_error(key: &str, title: &str, message: impl Into<String>) -> DatabasePreflightCheck {
    preflight_check(key, title, "error", "error", message)
}

fn preflight_check(
    key: &str,
    title: &str,
    status: &str,
    severity: &str,
    message: impl Into<String>,
) -> DatabasePreflightCheck {
    DatabasePreflightCheck {
        key: key.into(),
        title: title.into(),
        status: status.into(),
        severity: severity.into(),
        message: message.into(),
    }
}

fn ensure_migration_checksum_matches(
    driver: DatabaseDriver,
    file_name: &str,
    recorded: &str,
    current: &str,
) -> anyhow::Result<()> {
    if recorded == current {
        return Ok(());
    }

    anyhow::bail!(
        "database.driver={driver} 的迁移 {file_name} 已应用，但当前文件 SHA-256 摘要与 schema_migrations 记录不一致；请新建增量迁移，不要改写已发布迁移"
    )
}

fn ensure_sqlite_parent(url: &str) -> anyhow::Result<()> {
    let Some(path) = url.strip_prefix("sqlite://") else {
        return Ok(());
    };
    if path == ":memory:" || path.contains("mode=memory") {
        return Ok(());
    }
    let path = path.split('?').next().unwrap_or(path);
    if let Some(parent) = Path::new(path).parent()
        && !parent.as_os_str().is_empty()
    {
        std::fs::create_dir_all(parent)?;
    }
    Ok(())
}

fn migration_version(name: &str) -> anyhow::Result<i64> {
    let version = name
        .split_once('_')
        .map(|(version, _)| version)
        .unwrap_or(name)
        .parse::<i64>()
        .with_context(|| format!("迁移文件名缺少数字版本前缀：{name}"))?;
    Ok(version)
}

fn migration_dir_for_driver(driver: DatabaseDriver) -> PathBuf {
    let root = migration_root();
    match driver {
        DatabaseDriver::Sqlite => root,
        DatabaseDriver::Postgres => root.join("postgres"),
        DatabaseDriver::Mysql => root.join("mysql"),
    }
}

fn migration_root() -> PathBuf {
    let source_root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../migrations");
    if source_root.exists() {
        return source_root;
    }
    PathBuf::from("migrations")
}

fn collect_migration_files(dir: &Path) -> anyhow::Result<Vec<DatabaseMigrationFile>> {
    let entries = std::fs::read_dir(dir)
        .with_context(|| format!("读取迁移目录失败：{}", display_path(dir)))?;
    let mut files = Vec::new();
    for entry in entries {
        let entry = entry?;
        let path = entry.path();
        if path.extension().and_then(|value| value.to_str()) != Some("sql") {
            continue;
        }
        let metadata = entry.metadata()?;
        let bytes = fs::read(&path)?;
        files.push(DatabaseMigrationFile {
            name: entry.file_name().to_string_lossy().into_owned(),
            path: display_path(&path),
            bytes: metadata.len(),
            sha256: crypto::sha256_hex(bytes),
        });
    }
    files.sort_by(|left, right| left.name.cmp(&right.name));
    Ok(files)
}

fn display_path(path: &Path) -> String {
    let raw = path
        .canonicalize()
        .unwrap_or_else(|_| path.to_path_buf())
        .to_string_lossy()
        .into_owned();
    raw.strip_prefix(r"\\?\").unwrap_or(&raw).replace('\\', "/")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn external_database_drivers_use_their_own_runtime_without_sqlite_fallback() {
        for (driver, url, dir_segment) in [
            (
                DatabaseDriver::Postgres,
                "postgres://localhost/console",
                "migrations/postgres",
            ),
            (
                DatabaseDriver::Mysql,
                "mysql://localhost/console",
                "migrations/mysql",
            ),
        ] {
            let mut settings = Settings::default();
            settings.database.driver = driver;
            settings.database.url = url.into();

            let plan = migration_plan(&settings).expect("external database migration plan");

            assert_eq!(plan.driver, driver);
            assert!(plan.runtime_supported);
            assert_eq!(plan.runtime_status, "ready");
            assert!(plan.required_work.is_empty());
            assert!(plan.migration_dir.ends_with(dir_segment));
        }
    }

    #[test]
    fn migration_plan_lists_sqlite_runtime_migrations() {
        let settings = Settings::default();

        let plan = migration_plan(&settings).expect("sqlite migration plan");

        assert_eq!(plan.driver, DatabaseDriver::Sqlite);
        assert!(plan.runtime_supported);
        assert_eq!(plan.runtime_status, "ready");
        assert!(plan.migration_dir.ends_with("migrations"));
        assert!(
            plan.migration_files
                .iter()
                .any(|file| file.name == "20260621000100_init_core.sql")
        );
        assert!(
            plan.migration_files
                .iter()
                .all(|file| file.sha256.len() == 64)
        );
    }

    #[test]
    fn repository_runtime_report_keeps_external_trait_coverage_explicit() {
        let sqlite = repository_runtime_report(DatabaseDriver::Sqlite);
        assert!(sqlite.ready);
        assert!(sqlite.setup.ready);
        assert!(sqlite.iam.ready);
        assert!(sqlite.notification.ready);
        assert!(sqlite.system.ready);
        assert!(sqlite.blockers.is_empty());
        assert_eq!(
            sqlite.setup.implementation.as_deref(),
            Some("SqliteRepository")
        );

        for driver in [DatabaseDriver::Postgres, DatabaseDriver::Mysql] {
            let report = repository_runtime_report(driver);

            assert_eq!(report.driver, driver);
            assert!(report.ready);
            assert!(report.setup.ready);
            assert!(report.iam.ready);
            assert!(report.notification.ready);
            assert!(report.system.ready);
            assert_eq!(
                report.setup.implementation.as_deref(),
                Some("ExternalSetupRepository")
            );
            assert_eq!(
                report.iam.implementation.as_deref(),
                Some("ExternalIamRepository")
            );
            assert_eq!(
                report.notification.implementation.as_deref(),
                Some("ExternalNotificationRepository")
            );
            assert_eq!(
                report.system.implementation.as_deref(),
                Some("ExternalSystemRepository")
            );
            assert!(report.blockers.is_empty());
        }
    }

    #[test]
    fn migration_plan_lists_external_dialects_as_runtime_ready() {
        for (driver, url, dir_segment) in [
            (
                DatabaseDriver::Postgres,
                "postgres://localhost/console",
                "migrations/postgres",
            ),
            (
                DatabaseDriver::Mysql,
                "mysql://localhost/console",
                "migrations/mysql",
            ),
        ] {
            let mut settings = Settings::default();
            settings.database.driver = driver;
            settings.database.url = url.into();

            let plan = migration_plan(&settings).expect("external database migration plan");

            assert_eq!(plan.driver, driver);
            assert!(plan.runtime_supported);
            assert_eq!(plan.runtime_status, "ready");
            assert!(plan.migration_dir.ends_with(dir_segment));
            assert_eq!(plan.migration_files.len(), 1);
            assert!(plan.required_work.is_empty());
        }
    }

    #[test]
    fn external_migration_checksum_mismatch_requires_new_migration() {
        let error = ensure_migration_checksum_matches(
            DatabaseDriver::Postgres,
            "20260621000100_init_core.sql",
            "old-checksum",
            "new-checksum",
        )
        .expect_err("changed migration checksum must fail");

        let message = error.to_string();
        assert!(message.contains("SHA-256"));
        assert!(message.contains("新建增量迁移"));
    }

    #[tokio::test]
    async fn sqlite_ping_reports_runtime_ready_without_running_migrations() {
        let mut settings = Settings::default();
        let dir = std::env::temp_dir().join(format!("console-db-ping-{}", uuid::Uuid::new_v4()));
        settings.database.url = format!("sqlite://{}", dir.join("probe.sqlite").to_string_lossy());

        let report = ping_database(&settings).await.expect("sqlite ping");

        assert_eq!(report.driver, DatabaseDriver::Sqlite);
        assert!(report.connection_ok);
        assert!(report.runtime_supported);
        assert_eq!(report.runtime_status, "ready");
        assert!(report.repository_ready);
        assert!(report.repository_runtime.ready);
        assert!(report.repository_runtime.setup.ready);
        assert!(report.migration_runtime_ready);
    }

    #[tokio::test]
    async fn sqlite_insert_id_probe_reports_returning_strategy() {
        let mut settings = Settings::default();
        let dir =
            std::env::temp_dir().join(format!("console-db-insert-id-{}", uuid::Uuid::new_v4()));
        settings.database.url = format!("sqlite://{}", dir.join("probe.sqlite").to_string_lossy());

        let report = probe_insert_id(&settings)
            .await
            .expect("sqlite insert id probe");

        assert_eq!(report.driver, DatabaseDriver::Sqlite);
        assert_eq!(report.inserted_id, 1);
        assert_eq!(report.insert_id_strategy, "ReturningId");
        assert_eq!(report.insert_id_read, "ReturningIdInStatement");
        assert!(!report.same_connection_required);
        assert_eq!(report.temporary_table, "console_insert_id_probe");
    }

    #[tokio::test]
    async fn sqlite_migrate_applies_runtime_migrations() {
        let mut settings = Settings::default();
        let dir = std::env::temp_dir().join(format!("console-db-migrate-{}", uuid::Uuid::new_v4()));
        settings.database.url =
            format!("sqlite://{}", dir.join("migrate.sqlite").to_string_lossy());

        let report = migrate_database(&settings).await.expect("sqlite migrate");

        assert_eq!(report.driver, DatabaseDriver::Sqlite);
        assert!(report.repository_ready);
        assert!(report.repository_runtime.ready);
        assert!(
            report
                .applied_files
                .iter()
                .any(|file| file == "20260621000100_init_core.sql")
        );

        let pool = connect_sqlite_pool(&settings, false)
            .await
            .expect("connect migrated sqlite");
        let table_count: i64 = sqlx::query_scalar(
            "select count(*) from sqlite_master where type = 'table' and name = 'iam_users'",
        )
        .fetch_one(&pool)
        .await
        .expect("query migrated table");
        assert_eq!(table_count, 1);

        let setup_probe = probe_setup_repository(&settings)
            .await
            .expect("sqlite setup repository probe");
        assert_eq!(setup_probe.driver, DatabaseDriver::Sqlite);
        assert_eq!(setup_probe.implementation, "SqliteRepository");
        assert!(!setup_probe.missing_complete_result);
        assert!(setup_probe.run_listed);
        assert_eq!(setup_probe.log_count, 1);

        let notification_probe = probe_notification_repository(&settings)
            .await
            .expect("sqlite notification repository probe");
        assert_eq!(notification_probe.driver, DatabaseDriver::Sqlite);
        assert_eq!(notification_probe.implementation, "SqliteRepository");
        assert_eq!(notification_probe.claimed_probe_items, 3);
        assert!(notification_probe.delivered_result);
        assert!(notification_probe.retry_result);
        assert!(notification_probe.final_failure_result);
        assert!(notification_probe.dead_letter_reported);
        assert_eq!(notification_probe.dead_letter_secret_state, "purged");
        assert!(notification_probe.delivered_secret_purged);
        assert!(notification_probe.failed_secret_purged);
        assert!(notification_probe.purged_requeue_skipped);
        assert!(notification_probe.pending_secret_requeue_result);

        let system_probe = probe_system_repository(&settings)
            .await
            .expect("sqlite system repository probe");
        assert_eq!(system_probe.driver, DatabaseDriver::Sqlite);
        assert_eq!(system_probe.implementation, "SqliteRepository");
        assert!(system_probe.api_catalog_synced);
        assert!(system_probe.menu_synced);
        assert!(system_probe.config_roundtrip);
        assert!(system_probe.dictionary_roundtrip);
        assert!(system_probe.parameter_roundtrip);
        assert!(system_probe.operation_record_written);
        assert!(system_probe.operation_record_summary_reported);
        assert!(system_probe.operation_record_retention_prune);
        assert!(system_probe.version_package_roundtrip);
        assert!(system_probe.media_asset_roundtrip);
        assert!(system_probe.traffic_probe_roundtrip);

        let second_report = migrate_database(&settings)
            .await
            .expect("sqlite migrate second run");
        assert!(second_report.applied_files.is_empty());
        assert_eq!(
            second_report.skipped_files.len(),
            report.applied_files.len(),
            "第二次迁移应只报告已跳过文件，不能把已存在迁移误报为新应用"
        );

        let history = migration_history(&settings)
            .await
            .expect("sqlite migration history");
        assert_eq!(history.driver, DatabaseDriver::Sqlite);
        assert_eq!(history.checksum_source, "sqlx_migrator");
        assert_eq!(history.records.len(), report.applied_files.len());
        assert!(history.records.iter().all(|record| record.success));
        assert!(
            history
                .records
                .iter()
                .all(|record| !record.checksum.is_empty())
        );

        let schema_report = check_database_schema(&settings)
            .await
            .expect("sqlite schema check");
        assert!(schema_report.schema_ready);
        assert!(schema_report.repository_ready);
        assert!(schema_report.repository_runtime.ready);
        assert!(schema_report.missing_tables.is_empty());
        assert!(
            schema_report
                .checked_tables
                .iter()
                .any(|table| table == "iam_users")
        );

        let preflight = preflight_database(&settings)
            .await
            .expect("sqlite preflight");
        assert!(preflight.runtime_supported);
        assert!(preflight.connection_ok);
        assert!(preflight.migration_plan_ok);
        assert!(preflight.migration_history_ok);
        assert!(preflight.schema_ready);
        assert!(preflight.repository_ready);
        assert!(preflight.repository_runtime.ready);
        assert!(preflight.serve_ready);
        assert!(
            preflight
                .checks
                .iter()
                .any(|check| check.key == "migration-history" && check.status == "ok")
        );
        assert!(
            preflight
                .checks
                .iter()
                .any(|check| check.key == "repository-runtime" && check.status == "ok")
        );
    }
}
