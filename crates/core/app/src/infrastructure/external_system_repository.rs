use chrono::Utc;
use sqlx::{MySql, Pool, Postgres};

use crate::app::{AppError, AppResult};
use crate::domain::system::{
    ApiCatalogEntry, MediaAssetEntry, OperationRecord, OperationRecordCountBucket,
    OperationRecordPathBucket, OperationRecordSummary, SystemConfigEntry, SystemDictionaryEntry,
    SystemMenuEntry, SystemParameterEntry, TrafficProbeAlertEntry, TrafficProbeResultEntry,
    TrafficProbeTargetEntry, VersionPackageActionResult, VersionPackageEntry,
    VersionReleaseEventEntry,
};
use crate::infrastructure::sql_templates::SqlDialect;
use crate::repository::{
    CreateMediaAssetRecord, CreateOperationRecord, CreateTrafficProbeAlertRecord,
    CreateTrafficProbeResultRecord, CreateTrafficProbeTargetRecord, CreateVersionPackageRecord,
    OperationRecordListQuery, OperationRecordSummaryFilter, SystemRepository,
    UpsertSystemConfigRecord, UpsertSystemDictionaryRecord, UpsertSystemParameterRecord,
    VersionPackageActionRecord,
};

#[derive(Clone)]
pub enum ExternalSystemRepository {
    Postgres(Pool<Postgres>),
    Mysql(Pool<MySql>),
}

impl ExternalSystemRepository {
    pub fn postgres(pool: Pool<Postgres>) -> Self {
        Self::Postgres(pool)
    }

    pub fn mysql(pool: Pool<MySql>) -> Self {
        Self::Mysql(pool)
    }

    fn dialect(&self) -> SqlDialect {
        match self {
            Self::Postgres(_) => SqlDialect::Postgres,
            Self::Mysql(_) => SqlDialect::Mysql,
        }
    }

    fn now() -> String {
        Utc::now().to_rfc3339()
    }

    async fn mysql_last_insert_id(tx: &mut sqlx::Transaction<'_, MySql>) -> AppResult<i64> {
        let id = sqlx::query_scalar::<_, i64>("select cast(last_insert_id() as signed)")
            .fetch_one(&mut **tx)
            .await?;
        Ok(id)
    }

    async fn activate_version_package(
        &self,
        input: VersionPackageActionRecord,
        action: &str,
    ) -> AppResult<VersionPackageActionResult> {
        let now = Self::now();
        match self {
            Self::Postgres(pool) => {
                let mut tx = pool.begin().await?;
                let target =
                    sqlx::query_as::<_, VersionPackageRow>(self.dialect().version_package_by_id())
                        .bind(input.id)
                        .fetch_optional(&mut *tx)
                        .await?
                        .ok_or_else(|| AppError::NotFound("版本包不存在".into()))?;
                if target.status == "active" {
                    return Err(AppError::Conflict("版本包已经是当前 active 版本".into()));
                }

                let previous_active_id =
                    sqlx::query_scalar::<_, i64>(self.dialect().active_version_package_id())
                        .fetch_optional(&mut *tx)
                        .await?;
                if let Some(active_id) = previous_active_id {
                    sqlx::query(self.dialect().retire_version_package())
                        .bind(&now)
                        .bind(active_id)
                        .execute(&mut *tx)
                        .await?;
                }

                sqlx::query(self.dialect().activate_version_package())
                    .bind(&now)
                    .bind(input.id)
                    .execute(&mut *tx)
                    .await?;
                let event_id =
                    sqlx::query_scalar::<_, i64>(self.dialect().create_version_release_event())
                        .bind(input.id)
                        .bind(previous_active_id)
                        .bind(action)
                        .bind(&input.reason)
                        .bind(&now)
                        .fetch_one(&mut *tx)
                        .await?;
                tx.commit().await?;

                Ok(version_action_result(
                    event_id,
                    previous_active_id,
                    target,
                    now,
                )?)
            }
            Self::Mysql(pool) => {
                let mut tx = pool.begin().await?;
                let target =
                    sqlx::query_as::<_, VersionPackageRow>(self.dialect().version_package_by_id())
                        .bind(input.id)
                        .fetch_optional(&mut *tx)
                        .await?
                        .ok_or_else(|| AppError::NotFound("版本包不存在".into()))?;
                if target.status == "active" {
                    return Err(AppError::Conflict("版本包已经是当前 active 版本".into()));
                }

                let previous_active_id =
                    sqlx::query_scalar::<_, i64>(self.dialect().active_version_package_id())
                        .fetch_optional(&mut *tx)
                        .await?;
                if let Some(active_id) = previous_active_id {
                    sqlx::query(self.dialect().retire_version_package())
                        .bind(&now)
                        .bind(active_id)
                        .execute(&mut *tx)
                        .await?;
                }

                sqlx::query(self.dialect().activate_version_package())
                    .bind(&now)
                    .bind(input.id)
                    .execute(&mut *tx)
                    .await?;
                sqlx::query(self.dialect().create_version_release_event())
                    .bind(input.id)
                    .bind(previous_active_id)
                    .bind(action)
                    .bind(&input.reason)
                    .bind(&now)
                    .execute(&mut *tx)
                    .await?;
                let event_id = Self::mysql_last_insert_id(&mut tx).await?;
                tx.commit().await?;

                Ok(version_action_result(
                    event_id,
                    previous_active_id,
                    target,
                    now,
                )?)
            }
        }
    }
}

#[async_trait::async_trait]
impl SystemRepository for ExternalSystemRepository {
    async fn sync_api_catalog(&self, entries: &[ApiCatalogEntry]) -> AppResult<()> {
        let now = Self::now();
        match self {
            Self::Postgres(pool) => {
                let mut tx = pool.begin().await?;
                for entry in entries {
                    sqlx::query(self.dialect().system_api_upsert())
                        .bind(&entry.id)
                        .bind(&entry.method)
                        .bind(&entry.path)
                        .bind(&entry.tag)
                        .bind(&entry.summary)
                        .bind(&entry.access)
                        .bind(&entry.permission)
                        .bind(&entry.scope)
                        .bind(&entry.product_code)
                        .bind(&now)
                        .bind(&now)
                        .execute(&mut *tx)
                        .await?;
                    if let Some(permission) = &entry.permission {
                        sqlx::query(self.dialect().permission_upsert())
                            .bind(&entry.product_code)
                            .bind(&entry.scope)
                            .bind(permission)
                            .bind(&entry.summary)
                            .bind(&now)
                            .execute(&mut *tx)
                            .await?;
                    }
                }
                sqlx::query(self.dialect().platform_builtin_role_permissions())
                    .execute(&mut *tx)
                    .await?;
                sqlx::query(self.dialect().tenant_builtin_role_permissions())
                    .execute(&mut *tx)
                    .await?;
                tx.commit().await?;
            }
            Self::Mysql(pool) => {
                let mut tx = pool.begin().await?;
                for entry in entries {
                    sqlx::query(self.dialect().system_api_upsert())
                        .bind(&entry.id)
                        .bind(&entry.method)
                        .bind(&entry.path)
                        .bind(&entry.tag)
                        .bind(&entry.summary)
                        .bind(&entry.access)
                        .bind(&entry.permission)
                        .bind(&entry.scope)
                        .bind(&entry.product_code)
                        .bind(&now)
                        .bind(&now)
                        .execute(&mut *tx)
                        .await?;
                    if let Some(permission) = &entry.permission {
                        sqlx::query(self.dialect().permission_upsert())
                            .bind(&entry.product_code)
                            .bind(&entry.scope)
                            .bind(permission)
                            .bind(&entry.summary)
                            .bind(&now)
                            .execute(&mut *tx)
                            .await?;
                    }
                }
                sqlx::query(self.dialect().platform_builtin_role_permissions())
                    .execute(&mut *tx)
                    .await?;
                sqlx::query(self.dialect().tenant_builtin_role_permissions())
                    .execute(&mut *tx)
                    .await?;
                tx.commit().await?;
            }
        }
        Ok(())
    }

    async fn sync_system_menus(&self, menus: &[SystemMenuEntry]) -> AppResult<()> {
        let now = Self::now();
        match self {
            Self::Postgres(pool) => {
                let mut tx = pool.begin().await?;
                for menu in menus {
                    sqlx::query(self.dialect().system_menu_upsert())
                        .bind(&menu.code)
                        .bind(&menu.title)
                        .bind(&menu.path)
                        .bind(&menu.permission)
                        .bind(&menu.scope)
                        .bind(menu.sort_order)
                        .bind(&now)
                        .execute(&mut *tx)
                        .await?;
                }
                tx.commit().await?;
            }
            Self::Mysql(pool) => {
                let mut tx = pool.begin().await?;
                for menu in menus {
                    sqlx::query(self.dialect().system_menu_upsert())
                        .bind(&menu.code)
                        .bind(&menu.title)
                        .bind(&menu.path)
                        .bind(&menu.permission)
                        .bind(&menu.scope)
                        .bind(menu.sort_order)
                        .bind(&now)
                        .execute(&mut *tx)
                        .await?;
                }
                tx.commit().await?;
            }
        }
        Ok(())
    }

    async fn list_api_catalog(&self) -> AppResult<Vec<ApiCatalogEntry>> {
        let rows = match self {
            Self::Postgres(pool) => {
                sqlx::query_as::<_, ApiCatalogRow>(self.dialect().system_apis_list())
                    .fetch_all(pool)
                    .await?
            }
            Self::Mysql(pool) => {
                sqlx::query_as::<_, ApiCatalogRow>(self.dialect().system_apis_list())
                    .fetch_all(pool)
                    .await?
            }
        };
        Ok(rows.into_iter().map(api_catalog_from_row).collect())
    }

    async fn list_system_menus(&self) -> AppResult<Vec<SystemMenuEntry>> {
        let rows = match self {
            Self::Postgres(pool) => {
                sqlx::query_as::<_, SystemMenuRow>(self.dialect().system_menus_list())
                    .fetch_all(pool)
                    .await?
            }
            Self::Mysql(pool) => {
                sqlx::query_as::<_, SystemMenuRow>(self.dialect().system_menus_list())
                    .fetch_all(pool)
                    .await?
            }
        };
        Ok(rows.into_iter().map(system_menu_from_row).collect())
    }

    async fn create_operation_record(&self, input: CreateOperationRecord) -> AppResult<()> {
        match self {
            Self::Postgres(pool) => {
                sqlx::query(self.dialect().create_operation_record())
                    .bind(input.actor_user_id)
                    .bind(input.method)
                    .bind(input.path)
                    .bind(input.status)
                    .bind(Self::now())
                    .execute(pool)
                    .await?;
            }
            Self::Mysql(pool) => {
                sqlx::query(self.dialect().create_operation_record())
                    .bind(input.actor_user_id)
                    .bind(input.method)
                    .bind(input.path)
                    .bind(input.status)
                    .bind(Self::now())
                    .execute(pool)
                    .await?;
            }
        }
        Ok(())
    }

    async fn list_operation_records(
        &self,
        query: OperationRecordListQuery,
    ) -> AppResult<Vec<OperationRecord>> {
        let rows = match self {
            Self::Postgres(pool) => {
                sqlx::query_as::<_, OperationRecordRow>(self.dialect().operation_records_list())
                    .bind(query.method.as_deref())
                    .bind(query.method.as_deref())
                    .bind(query.path.as_deref())
                    .bind(query.path.as_deref())
                    .bind(query.status)
                    .bind(query.status)
                    .bind(query.actor_user_id)
                    .bind(query.actor_user_id)
                    .bind(query.created_from.as_deref())
                    .bind(query.created_from.as_deref())
                    .bind(query.created_to.as_deref())
                    .bind(query.created_to.as_deref())
                    .bind(query.limit)
                    .bind(query.offset)
                    .fetch_all(pool)
                    .await?
            }
            Self::Mysql(pool) => {
                sqlx::query_as::<_, OperationRecordRow>(self.dialect().operation_records_list())
                    .bind(query.method.as_deref())
                    .bind(query.method.as_deref())
                    .bind(query.path.as_deref())
                    .bind(query.path.as_deref())
                    .bind(query.status)
                    .bind(query.status)
                    .bind(query.actor_user_id)
                    .bind(query.actor_user_id)
                    .bind(query.created_from.as_deref())
                    .bind(query.created_from.as_deref())
                    .bind(query.created_to.as_deref())
                    .bind(query.created_to.as_deref())
                    .bind(query.limit)
                    .bind(query.offset)
                    .fetch_all(pool)
                    .await?
            }
        };
        Ok(rows.into_iter().map(operation_record_from_row).collect())
    }

    async fn summarize_operation_records(
        &self,
        query: OperationRecordSummaryFilter,
    ) -> AppResult<OperationRecordSummary> {
        let (counts, by_method, by_status_class, top_paths) = match self {
            Self::Postgres(pool) => {
                let counts = sqlx::query_as::<_, OperationRecordSummaryCountsRow>(
                    self.dialect().operation_records_summary_counts(),
                )
                .bind(query.method.as_deref())
                .bind(query.method.as_deref())
                .bind(query.path.as_deref())
                .bind(query.path.as_deref())
                .bind(query.status)
                .bind(query.status)
                .bind(query.actor_user_id)
                .bind(query.actor_user_id)
                .bind(query.created_from.as_deref())
                .bind(query.created_from.as_deref())
                .bind(query.created_to.as_deref())
                .bind(query.created_to.as_deref())
                .fetch_one(pool)
                .await?;
                let by_method = sqlx::query_as::<_, OperationRecordCountBucketRow>(
                    self.dialect().operation_records_summary_methods(),
                )
                .bind(query.method.as_deref())
                .bind(query.method.as_deref())
                .bind(query.path.as_deref())
                .bind(query.path.as_deref())
                .bind(query.status)
                .bind(query.status)
                .bind(query.actor_user_id)
                .bind(query.actor_user_id)
                .bind(query.created_from.as_deref())
                .bind(query.created_from.as_deref())
                .bind(query.created_to.as_deref())
                .bind(query.created_to.as_deref())
                .fetch_all(pool)
                .await?;
                let by_status_class = sqlx::query_as::<_, OperationRecordCountBucketRow>(
                    self.dialect().operation_records_summary_status_classes(),
                )
                .bind(query.method.as_deref())
                .bind(query.method.as_deref())
                .bind(query.path.as_deref())
                .bind(query.path.as_deref())
                .bind(query.status)
                .bind(query.status)
                .bind(query.actor_user_id)
                .bind(query.actor_user_id)
                .bind(query.created_from.as_deref())
                .bind(query.created_from.as_deref())
                .bind(query.created_to.as_deref())
                .bind(query.created_to.as_deref())
                .fetch_all(pool)
                .await?;
                let top_paths = sqlx::query_as::<_, OperationRecordPathBucketRow>(
                    self.dialect().operation_records_summary_paths(),
                )
                .bind(query.method.as_deref())
                .bind(query.method.as_deref())
                .bind(query.path.as_deref())
                .bind(query.path.as_deref())
                .bind(query.status)
                .bind(query.status)
                .bind(query.actor_user_id)
                .bind(query.actor_user_id)
                .bind(query.created_from.as_deref())
                .bind(query.created_from.as_deref())
                .bind(query.created_to.as_deref())
                .bind(query.created_to.as_deref())
                .bind(query.top_limit)
                .fetch_all(pool)
                .await?;
                (counts, by_method, by_status_class, top_paths)
            }
            Self::Mysql(pool) => {
                let counts = sqlx::query_as::<_, OperationRecordSummaryCountsRow>(
                    self.dialect().operation_records_summary_counts(),
                )
                .bind(query.method.as_deref())
                .bind(query.method.as_deref())
                .bind(query.path.as_deref())
                .bind(query.path.as_deref())
                .bind(query.status)
                .bind(query.status)
                .bind(query.actor_user_id)
                .bind(query.actor_user_id)
                .bind(query.created_from.as_deref())
                .bind(query.created_from.as_deref())
                .bind(query.created_to.as_deref())
                .bind(query.created_to.as_deref())
                .fetch_one(pool)
                .await?;
                let by_method = sqlx::query_as::<_, OperationRecordCountBucketRow>(
                    self.dialect().operation_records_summary_methods(),
                )
                .bind(query.method.as_deref())
                .bind(query.method.as_deref())
                .bind(query.path.as_deref())
                .bind(query.path.as_deref())
                .bind(query.status)
                .bind(query.status)
                .bind(query.actor_user_id)
                .bind(query.actor_user_id)
                .bind(query.created_from.as_deref())
                .bind(query.created_from.as_deref())
                .bind(query.created_to.as_deref())
                .bind(query.created_to.as_deref())
                .fetch_all(pool)
                .await?;
                let by_status_class = sqlx::query_as::<_, OperationRecordCountBucketRow>(
                    self.dialect().operation_records_summary_status_classes(),
                )
                .bind(query.method.as_deref())
                .bind(query.method.as_deref())
                .bind(query.path.as_deref())
                .bind(query.path.as_deref())
                .bind(query.status)
                .bind(query.status)
                .bind(query.actor_user_id)
                .bind(query.actor_user_id)
                .bind(query.created_from.as_deref())
                .bind(query.created_from.as_deref())
                .bind(query.created_to.as_deref())
                .bind(query.created_to.as_deref())
                .fetch_all(pool)
                .await?;
                let top_paths = sqlx::query_as::<_, OperationRecordPathBucketRow>(
                    self.dialect().operation_records_summary_paths(),
                )
                .bind(query.method.as_deref())
                .bind(query.method.as_deref())
                .bind(query.path.as_deref())
                .bind(query.path.as_deref())
                .bind(query.status)
                .bind(query.status)
                .bind(query.actor_user_id)
                .bind(query.actor_user_id)
                .bind(query.created_from.as_deref())
                .bind(query.created_from.as_deref())
                .bind(query.created_to.as_deref())
                .bind(query.created_to.as_deref())
                .bind(query.top_limit)
                .fetch_all(pool)
                .await?;
                (counts, by_method, by_status_class, top_paths)
            }
        };

        Ok(OperationRecordSummary {
            generated_at: String::new(),
            total_count: counts.total_count,
            success_count: counts.success_count,
            redirect_count: counts.redirect_count,
            client_error_count: counts.client_error_count,
            server_error_count: counts.server_error_count,
            other_count: counts.other_count,
            top_limit: query.top_limit,
            by_method: by_method.into_iter().map(count_bucket_from_row).collect(),
            by_status_class: by_status_class
                .into_iter()
                .map(count_bucket_from_row)
                .collect(),
            top_paths: top_paths.into_iter().map(path_bucket_from_row).collect(),
        })
    }

    async fn prune_operation_records(&self, cutoff: &str, limit: i64) -> AppResult<i64> {
        let rows_affected = match self {
            Self::Postgres(pool) => sqlx::query(self.dialect().prune_operation_records())
                .bind(cutoff)
                .bind(limit)
                .execute(pool)
                .await?
                .rows_affected(),
            Self::Mysql(pool) => sqlx::query(self.dialect().prune_operation_records())
                .bind(cutoff)
                .bind(limit)
                .execute(pool)
                .await?
                .rows_affected(),
        };
        Ok(rows_affected.min(i64::MAX as u64) as i64)
    }

    async fn list_system_configs(&self) -> AppResult<Vec<SystemConfigEntry>> {
        let rows = match self {
            Self::Postgres(pool) => {
                sqlx::query_as::<_, SystemConfigRow>(self.dialect().system_configs_list())
                    .fetch_all(pool)
                    .await?
            }
            Self::Mysql(pool) => {
                sqlx::query_as::<_, SystemConfigRow>(self.dialect().system_configs_list())
                    .fetch_all(pool)
                    .await?
            }
        };
        rows.into_iter().map(config_from_row).collect()
    }

    async fn upsert_system_config(
        &self,
        input: UpsertSystemConfigRecord,
    ) -> AppResult<SystemConfigEntry> {
        let now = Self::now();
        match self {
            Self::Postgres(pool) => {
                sqlx::query(self.dialect().system_config_upsert())
                    .bind(&input.key)
                    .bind(&input.value_json)
                    .bind(&now)
                    .execute(pool)
                    .await?;
            }
            Self::Mysql(pool) => {
                sqlx::query(self.dialect().system_config_upsert())
                    .bind(&input.key)
                    .bind(&input.value_json)
                    .bind(&now)
                    .execute(pool)
                    .await?;
            }
        }
        Ok(SystemConfigEntry {
            key: input.key,
            value: parse_json_value(&input.value_json)?,
            updated_at: now,
        })
    }

    async fn delete_system_config(&self, key: &str) -> AppResult<bool> {
        let rows = match self {
            Self::Postgres(pool) => sqlx::query(self.dialect().delete_system_config())
                .bind(key)
                .execute(pool)
                .await?
                .rows_affected(),
            Self::Mysql(pool) => sqlx::query(self.dialect().delete_system_config())
                .bind(key)
                .execute(pool)
                .await?
                .rows_affected(),
        };
        Ok(rows > 0)
    }

    async fn list_system_dictionaries(&self) -> AppResult<Vec<SystemDictionaryEntry>> {
        let rows = match self {
            Self::Postgres(pool) => {
                sqlx::query_as::<_, SystemDictionaryRow>(self.dialect().system_dictionaries_list())
                    .fetch_all(pool)
                    .await?
            }
            Self::Mysql(pool) => {
                sqlx::query_as::<_, SystemDictionaryRow>(self.dialect().system_dictionaries_list())
                    .fetch_all(pool)
                    .await?
            }
        };
        Ok(rows.into_iter().map(dictionary_from_row).collect())
    }

    async fn upsert_system_dictionary(
        &self,
        input: UpsertSystemDictionaryRecord,
    ) -> AppResult<SystemDictionaryEntry> {
        let now = Self::now();
        let row = match self {
            Self::Postgres(pool) => {
                sqlx::query(self.dialect().system_dictionary_upsert())
                    .bind(&input.code)
                    .bind(&input.name)
                    .bind(&now)
                    .execute(pool)
                    .await?;
                sqlx::query_as::<_, SystemDictionaryRow>(self.dialect().system_dictionary_by_code())
                    .bind(&input.code)
                    .fetch_one(pool)
                    .await?
            }
            Self::Mysql(pool) => {
                sqlx::query(self.dialect().system_dictionary_upsert())
                    .bind(&input.code)
                    .bind(&input.name)
                    .bind(&now)
                    .execute(pool)
                    .await?;
                sqlx::query_as::<_, SystemDictionaryRow>(self.dialect().system_dictionary_by_code())
                    .bind(&input.code)
                    .fetch_one(pool)
                    .await?
            }
        };
        Ok(dictionary_from_row(row))
    }

    async fn delete_system_dictionary(&self, code: &str) -> AppResult<bool> {
        let now = Self::now();
        let rows = match self {
            Self::Postgres(pool) => sqlx::query(self.dialect().delete_system_dictionary())
                .bind(&now)
                .bind(code)
                .execute(pool)
                .await?
                .rows_affected(),
            Self::Mysql(pool) => sqlx::query(self.dialect().delete_system_dictionary())
                .bind(&now)
                .bind(code)
                .execute(pool)
                .await?
                .rows_affected(),
        };
        Ok(rows > 0)
    }

    async fn list_system_parameters(&self) -> AppResult<Vec<SystemParameterEntry>> {
        let rows = match self {
            Self::Postgres(pool) => {
                sqlx::query_as::<_, SystemParameterRow>(self.dialect().system_parameters_list())
                    .fetch_all(pool)
                    .await?
            }
            Self::Mysql(pool) => {
                sqlx::query_as::<_, SystemParameterRow>(self.dialect().system_parameters_list())
                    .fetch_all(pool)
                    .await?
            }
        };
        Ok(rows.into_iter().map(parameter_from_row).collect())
    }

    async fn upsert_system_parameter(
        &self,
        input: UpsertSystemParameterRecord,
    ) -> AppResult<SystemParameterEntry> {
        let now = Self::now();
        let row = match self {
            Self::Postgres(pool) => {
                sqlx::query(self.dialect().system_parameter_upsert())
                    .bind(&input.key)
                    .bind(&input.name)
                    .bind(&input.value)
                    .bind(&now)
                    .bind(&now)
                    .execute(pool)
                    .await?;
                sqlx::query_as::<_, SystemParameterRow>(self.dialect().system_parameter_by_key())
                    .bind(&input.key)
                    .fetch_one(pool)
                    .await?
            }
            Self::Mysql(pool) => {
                sqlx::query(self.dialect().system_parameter_upsert())
                    .bind(&input.key)
                    .bind(&input.name)
                    .bind(&input.value)
                    .bind(&now)
                    .bind(&now)
                    .execute(pool)
                    .await?;
                sqlx::query_as::<_, SystemParameterRow>(self.dialect().system_parameter_by_key())
                    .bind(&input.key)
                    .fetch_one(pool)
                    .await?
            }
        };
        Ok(parameter_from_row(row))
    }

    async fn delete_system_parameter(&self, key: &str) -> AppResult<bool> {
        let now = Self::now();
        let rows = match self {
            Self::Postgres(pool) => sqlx::query(self.dialect().delete_system_parameter())
                .bind(&now)
                .bind(&now)
                .bind(key)
                .execute(pool)
                .await?
                .rows_affected(),
            Self::Mysql(pool) => sqlx::query(self.dialect().delete_system_parameter())
                .bind(&now)
                .bind(&now)
                .bind(key)
                .execute(pool)
                .await?
                .rows_affected(),
        };
        Ok(rows > 0)
    }

    async fn create_version_package(
        &self,
        input: CreateVersionPackageRecord,
    ) -> AppResult<VersionPackageEntry> {
        let now = Self::now();
        let id = match self {
            Self::Postgres(pool) => {
                sqlx::query_scalar::<_, i64>(self.dialect().create_version_package())
                    .bind(&input.version_name)
                    .bind(&input.version_code)
                    .bind(&input.manifest_json)
                    .bind(&now)
                    .fetch_one(pool)
                    .await?
            }
            Self::Mysql(pool) => {
                let mut tx = pool.begin().await?;
                sqlx::query(self.dialect().create_version_package())
                    .bind(&input.version_name)
                    .bind(&input.version_code)
                    .bind(&input.manifest_json)
                    .bind(&now)
                    .execute(&mut *tx)
                    .await?;
                let id = Self::mysql_last_insert_id(&mut tx).await?;
                tx.commit().await?;
                id
            }
        };
        Ok(VersionPackageEntry {
            id,
            version_name: input.version_name,
            version_code: input.version_code,
            manifest: parse_json_value(&input.manifest_json)?,
            status: "draft".into(),
            created_at: now,
            published_at: None,
            retired_at: None,
        })
    }

    async fn list_version_packages(&self) -> AppResult<Vec<VersionPackageEntry>> {
        let rows = match self {
            Self::Postgres(pool) => {
                sqlx::query_as::<_, VersionPackageRow>(self.dialect().version_packages_list())
                    .fetch_all(pool)
                    .await?
            }
            Self::Mysql(pool) => {
                sqlx::query_as::<_, VersionPackageRow>(self.dialect().version_packages_list())
                    .fetch_all(pool)
                    .await?
            }
        };
        rows.into_iter().map(version_package_from_row).collect()
    }

    async fn publish_version_package(
        &self,
        input: VersionPackageActionRecord,
    ) -> AppResult<VersionPackageActionResult> {
        self.activate_version_package(input, "publish").await
    }

    async fn rollback_version_package(
        &self,
        input: VersionPackageActionRecord,
    ) -> AppResult<VersionPackageActionResult> {
        self.activate_version_package(input, "rollback").await
    }

    async fn list_version_release_events(&self) -> AppResult<Vec<VersionReleaseEventEntry>> {
        let rows = match self {
            Self::Postgres(pool) => {
                sqlx::query_as::<_, VersionReleaseEventRow>(
                    self.dialect().version_release_events_list(),
                )
                .fetch_all(pool)
                .await?
            }
            Self::Mysql(pool) => {
                sqlx::query_as::<_, VersionReleaseEventRow>(
                    self.dialect().version_release_events_list(),
                )
                .fetch_all(pool)
                .await?
            }
        };
        Ok(rows
            .into_iter()
            .map(version_release_event_from_row)
            .collect())
    }

    async fn delete_version_package(&self, id: i64) -> AppResult<bool> {
        let now = Self::now();
        match self {
            Self::Postgres(pool) => {
                let result = sqlx::query(self.dialect().delete_version_package())
                    .bind(&now)
                    .bind(id)
                    .execute(pool)
                    .await?;
                if result.rows_affected() > 0 {
                    return Ok(true);
                }
                let status =
                    sqlx::query_scalar::<_, String>(self.dialect().version_package_status_by_id())
                        .bind(id)
                        .fetch_optional(pool)
                        .await?;
                if status.as_deref() == Some("active") {
                    return Err(AppError::Conflict("当前 active 版本包不能删除".into()));
                }
            }
            Self::Mysql(pool) => {
                let result = sqlx::query(self.dialect().delete_version_package())
                    .bind(&now)
                    .bind(id)
                    .execute(pool)
                    .await?;
                if result.rows_affected() > 0 {
                    return Ok(true);
                }
                let status =
                    sqlx::query_scalar::<_, String>(self.dialect().version_package_status_by_id())
                        .bind(id)
                        .fetch_optional(pool)
                        .await?;
                if status.as_deref() == Some("active") {
                    return Err(AppError::Conflict("当前 active 版本包不能删除".into()));
                }
            }
        }
        Ok(false)
    }

    async fn create_media_asset(
        &self,
        input: CreateMediaAssetRecord,
    ) -> AppResult<MediaAssetEntry> {
        let now = Self::now();
        let id = match self {
            Self::Postgres(pool) => {
                sqlx::query_scalar::<_, i64>(self.dialect().create_media_asset())
                    .bind(&input.category)
                    .bind(&input.display_name)
                    .bind(&input.storage_key)
                    .bind(&input.mime_type)
                    .bind(input.size_bytes)
                    .bind(&now)
                    .fetch_one(pool)
                    .await?
            }
            Self::Mysql(pool) => {
                let mut tx = pool.begin().await?;
                sqlx::query(self.dialect().create_media_asset())
                    .bind(&input.category)
                    .bind(&input.display_name)
                    .bind(&input.storage_key)
                    .bind(&input.mime_type)
                    .bind(input.size_bytes)
                    .bind(&now)
                    .execute(&mut *tx)
                    .await?;
                let id = Self::mysql_last_insert_id(&mut tx).await?;
                tx.commit().await?;
                id
            }
        };
        Ok(MediaAssetEntry {
            id,
            category: input.category,
            display_name: input.display_name,
            storage_key: input.storage_key,
            mime_type: input.mime_type,
            size_bytes: input.size_bytes,
            created_at: now,
        })
    }

    async fn list_media_assets(&self) -> AppResult<Vec<MediaAssetEntry>> {
        let rows = match self {
            Self::Postgres(pool) => {
                sqlx::query_as::<_, MediaAssetRow>(self.dialect().media_assets_list())
                    .fetch_all(pool)
                    .await?
            }
            Self::Mysql(pool) => {
                sqlx::query_as::<_, MediaAssetRow>(self.dialect().media_assets_list())
                    .fetch_all(pool)
                    .await?
            }
        };
        Ok(rows.into_iter().map(media_asset_from_row).collect())
    }

    async fn delete_media_asset(&self, id: i64) -> AppResult<bool> {
        let now = Self::now();
        let rows = match self {
            Self::Postgres(pool) => sqlx::query(self.dialect().delete_media_asset())
                .bind(&now)
                .bind(id)
                .execute(pool)
                .await?
                .rows_affected(),
            Self::Mysql(pool) => sqlx::query(self.dialect().delete_media_asset())
                .bind(&now)
                .bind(id)
                .execute(pool)
                .await?
                .rows_affected(),
        };
        Ok(rows > 0)
    }

    async fn create_traffic_probe_target(
        &self,
        input: CreateTrafficProbeTargetRecord,
    ) -> AppResult<TrafficProbeTargetEntry> {
        let now = Self::now();
        let id = match self {
            Self::Postgres(pool) => {
                sqlx::query_scalar::<_, i64>(self.dialect().create_traffic_probe_target())
                    .bind(&input.name)
                    .bind(&input.url)
                    .bind(input.expected_status)
                    .bind(&now)
                    .fetch_one(pool)
                    .await?
            }
            Self::Mysql(pool) => {
                let mut tx = pool.begin().await?;
                sqlx::query(self.dialect().create_traffic_probe_target())
                    .bind(&input.name)
                    .bind(&input.url)
                    .bind(input.expected_status)
                    .bind(&now)
                    .execute(&mut *tx)
                    .await?;
                let id = Self::mysql_last_insert_id(&mut tx).await?;
                tx.commit().await?;
                id
            }
        };
        Ok(TrafficProbeTargetEntry {
            id,
            name: input.name,
            url: input.url,
            expected_status: input.expected_status,
            status: "pending".into(),
            created_at: now,
        })
    }

    async fn list_traffic_probe_targets(&self) -> AppResult<Vec<TrafficProbeTargetEntry>> {
        let rows = match self {
            Self::Postgres(pool) => {
                sqlx::query_as::<_, TrafficProbeTargetRow>(
                    self.dialect().traffic_probe_targets_list(),
                )
                .fetch_all(pool)
                .await?
            }
            Self::Mysql(pool) => {
                sqlx::query_as::<_, TrafficProbeTargetRow>(
                    self.dialect().traffic_probe_targets_list(),
                )
                .fetch_all(pool)
                .await?
            }
        };
        Ok(rows
            .into_iter()
            .map(traffic_probe_target_from_row)
            .collect())
    }

    async fn find_traffic_probe_target(
        &self,
        id: i64,
    ) -> AppResult<Option<TrafficProbeTargetEntry>> {
        let row = match self {
            Self::Postgres(pool) => {
                sqlx::query_as::<_, TrafficProbeTargetRow>(
                    self.dialect().traffic_probe_target_by_id(),
                )
                .bind(id)
                .fetch_optional(pool)
                .await?
            }
            Self::Mysql(pool) => {
                sqlx::query_as::<_, TrafficProbeTargetRow>(
                    self.dialect().traffic_probe_target_by_id(),
                )
                .bind(id)
                .fetch_optional(pool)
                .await?
            }
        };
        Ok(row.map(traffic_probe_target_from_row))
    }

    async fn delete_traffic_probe_target(&self, id: i64) -> AppResult<bool> {
        let now = Self::now();
        let rows = match self {
            Self::Postgres(pool) => sqlx::query(self.dialect().delete_traffic_probe_target())
                .bind(&now)
                .bind(id)
                .execute(pool)
                .await?
                .rows_affected(),
            Self::Mysql(pool) => sqlx::query(self.dialect().delete_traffic_probe_target())
                .bind(&now)
                .bind(id)
                .execute(pool)
                .await?
                .rows_affected(),
        };
        Ok(rows > 0)
    }

    async fn create_traffic_probe_result(
        &self,
        input: CreateTrafficProbeResultRecord,
    ) -> AppResult<TrafficProbeResultEntry> {
        let now = Self::now();
        let id = match self {
            Self::Postgres(pool) => {
                let mut tx = pool.begin().await?;
                let id = sqlx::query_scalar::<_, i64>(self.dialect().create_traffic_probe_result())
                    .bind(input.target_id)
                    .bind(&input.status)
                    .bind(&input.detail_json)
                    .bind(&now)
                    .fetch_one(&mut *tx)
                    .await?;
                sqlx::query(self.dialect().update_traffic_probe_target_status())
                    .bind(&input.status)
                    .bind(input.target_id)
                    .execute(&mut *tx)
                    .await?;
                tx.commit().await?;
                id
            }
            Self::Mysql(pool) => {
                let mut tx = pool.begin().await?;
                sqlx::query(self.dialect().create_traffic_probe_result())
                    .bind(input.target_id)
                    .bind(&input.status)
                    .bind(&input.detail_json)
                    .bind(&now)
                    .execute(&mut *tx)
                    .await?;
                let id = Self::mysql_last_insert_id(&mut tx).await?;
                sqlx::query(self.dialect().update_traffic_probe_target_status())
                    .bind(&input.status)
                    .bind(input.target_id)
                    .execute(&mut *tx)
                    .await?;
                tx.commit().await?;
                id
            }
        };
        Ok(TrafficProbeResultEntry {
            id,
            target_id: input.target_id,
            status: input.status,
            detail: parse_json_value(&input.detail_json)?,
            probed_at: now,
        })
    }

    async fn list_traffic_probe_results(
        &self,
        target_id: Option<i64>,
        limit: i64,
    ) -> AppResult<Vec<TrafficProbeResultEntry>> {
        let limit = limit.clamp(1, 200);
        let rows = match (self, target_id) {
            (Self::Postgres(pool), Some(target_id)) => {
                sqlx::query_as::<_, TrafficProbeResultRow>(
                    self.dialect().traffic_probe_results_for_target(),
                )
                .bind(target_id)
                .bind(limit)
                .fetch_all(pool)
                .await?
            }
            (Self::Mysql(pool), Some(target_id)) => {
                sqlx::query_as::<_, TrafficProbeResultRow>(
                    self.dialect().traffic_probe_results_for_target(),
                )
                .bind(target_id)
                .bind(limit)
                .fetch_all(pool)
                .await?
            }
            (Self::Postgres(pool), None) => {
                sqlx::query_as::<_, TrafficProbeResultRow>(
                    self.dialect().traffic_probe_results_all(),
                )
                .bind(limit)
                .fetch_all(pool)
                .await?
            }
            (Self::Mysql(pool), None) => {
                sqlx::query_as::<_, TrafficProbeResultRow>(
                    self.dialect().traffic_probe_results_all(),
                )
                .bind(limit)
                .fetch_all(pool)
                .await?
            }
        };
        rows.into_iter()
            .map(traffic_probe_result_from_row)
            .collect()
    }

    async fn create_traffic_probe_alert(
        &self,
        input: CreateTrafficProbeAlertRecord,
    ) -> AppResult<TrafficProbeAlertEntry> {
        let now = Self::now();
        let id = match self {
            Self::Postgres(pool) => {
                sqlx::query_scalar::<_, i64>(self.dialect().create_traffic_probe_alert())
                    .bind(input.target_id)
                    .bind(input.result_id)
                    .bind(&input.severity)
                    .bind(&input.reason)
                    .bind(&input.detail_json)
                    .bind(&now)
                    .fetch_one(pool)
                    .await?
            }
            Self::Mysql(pool) => {
                let mut tx = pool.begin().await?;
                sqlx::query(self.dialect().create_traffic_probe_alert())
                    .bind(input.target_id)
                    .bind(input.result_id)
                    .bind(&input.severity)
                    .bind(&input.reason)
                    .bind(&input.detail_json)
                    .bind(&now)
                    .execute(&mut *tx)
                    .await?;
                let id = Self::mysql_last_insert_id(&mut tx).await?;
                tx.commit().await?;
                id
            }
        };
        Ok(TrafficProbeAlertEntry {
            id,
            target_id: input.target_id,
            result_id: input.result_id,
            severity: input.severity,
            status: "open".into(),
            reason: input.reason,
            detail: parse_json_value(&input.detail_json)?,
            opened_at: now,
            acknowledged_at: None,
            resolved_at: None,
        })
    }

    async fn list_traffic_probe_alerts(
        &self,
        target_id: Option<i64>,
        status: Option<String>,
        limit: i64,
    ) -> AppResult<Vec<TrafficProbeAlertEntry>> {
        let limit = limit.clamp(1, 200);
        let rows = match (self, target_id, status) {
            (Self::Postgres(pool), Some(target_id), Some(status)) => {
                sqlx::query_as::<_, TrafficProbeAlertRow>(
                    self.dialect().traffic_probe_alerts_target_status(),
                )
                .bind(target_id)
                .bind(status)
                .bind(limit)
                .fetch_all(pool)
                .await?
            }
            (Self::Mysql(pool), Some(target_id), Some(status)) => {
                sqlx::query_as::<_, TrafficProbeAlertRow>(
                    self.dialect().traffic_probe_alerts_target_status(),
                )
                .bind(target_id)
                .bind(status)
                .bind(limit)
                .fetch_all(pool)
                .await?
            }
            (Self::Postgres(pool), Some(target_id), None) => {
                sqlx::query_as::<_, TrafficProbeAlertRow>(
                    self.dialect().traffic_probe_alerts_target(),
                )
                .bind(target_id)
                .bind(limit)
                .fetch_all(pool)
                .await?
            }
            (Self::Mysql(pool), Some(target_id), None) => {
                sqlx::query_as::<_, TrafficProbeAlertRow>(
                    self.dialect().traffic_probe_alerts_target(),
                )
                .bind(target_id)
                .bind(limit)
                .fetch_all(pool)
                .await?
            }
            (Self::Postgres(pool), None, Some(status)) => {
                sqlx::query_as::<_, TrafficProbeAlertRow>(
                    self.dialect().traffic_probe_alerts_status(),
                )
                .bind(status)
                .bind(limit)
                .fetch_all(pool)
                .await?
            }
            (Self::Mysql(pool), None, Some(status)) => {
                sqlx::query_as::<_, TrafficProbeAlertRow>(
                    self.dialect().traffic_probe_alerts_status(),
                )
                .bind(status)
                .bind(limit)
                .fetch_all(pool)
                .await?
            }
            (Self::Postgres(pool), None, None) => {
                sqlx::query_as::<_, TrafficProbeAlertRow>(self.dialect().traffic_probe_alerts_all())
                    .bind(limit)
                    .fetch_all(pool)
                    .await?
            }
            (Self::Mysql(pool), None, None) => {
                sqlx::query_as::<_, TrafficProbeAlertRow>(self.dialect().traffic_probe_alerts_all())
                    .bind(limit)
                    .fetch_all(pool)
                    .await?
            }
        };
        rows.into_iter().map(traffic_probe_alert_from_row).collect()
    }

    async fn acknowledge_traffic_probe_alert(&self, id: i64) -> AppResult<bool> {
        let now = Self::now();
        let rows = match self {
            Self::Postgres(pool) => sqlx::query(self.dialect().acknowledge_traffic_probe_alert())
                .bind(&now)
                .bind(id)
                .execute(pool)
                .await?
                .rows_affected(),
            Self::Mysql(pool) => sqlx::query(self.dialect().acknowledge_traffic_probe_alert())
                .bind(&now)
                .bind(id)
                .execute(pool)
                .await?
                .rows_affected(),
        };
        Ok(rows > 0)
    }

    async fn resolve_traffic_probe_alert(&self, id: i64) -> AppResult<bool> {
        let now = Self::now();
        let rows = match self {
            Self::Postgres(pool) => sqlx::query(self.dialect().resolve_traffic_probe_alert())
                .bind(&now)
                .bind(id)
                .execute(pool)
                .await?
                .rows_affected(),
            Self::Mysql(pool) => sqlx::query(self.dialect().resolve_traffic_probe_alert())
                .bind(&now)
                .bind(id)
                .execute(pool)
                .await?
                .rows_affected(),
        };
        Ok(rows > 0)
    }

    async fn resolve_traffic_probe_alerts_for_target(&self, target_id: i64) -> AppResult<i64> {
        let now = Self::now();
        let rows = match self {
            Self::Postgres(pool) => {
                sqlx::query(self.dialect().resolve_traffic_probe_alerts_for_target())
                    .bind(&now)
                    .bind(target_id)
                    .execute(pool)
                    .await?
                    .rows_affected()
            }
            Self::Mysql(pool) => {
                sqlx::query(self.dialect().resolve_traffic_probe_alerts_for_target())
                    .bind(&now)
                    .bind(target_id)
                    .execute(pool)
                    .await?
                    .rows_affected()
            }
        };
        Ok(rows.min(i64::MAX as u64) as i64)
    }
}

#[derive(sqlx::FromRow)]
struct ApiCatalogRow {
    id: String,
    method: String,
    path: String,
    tag: String,
    summary: String,
    access: String,
    permission: Option<String>,
    scope: String,
    product_code: String,
}

fn api_catalog_from_row(row: ApiCatalogRow) -> ApiCatalogEntry {
    ApiCatalogEntry {
        id: row.id,
        method: row.method,
        path: row.path,
        tag: row.tag,
        summary: row.summary,
        access: row.access,
        permission: row.permission,
        scope: row.scope,
        product_code: row.product_code,
    }
}

#[derive(sqlx::FromRow)]
struct SystemMenuRow {
    code: String,
    title: String,
    path: String,
    permission: Option<String>,
    scope: String,
    sort_order: i64,
}

fn system_menu_from_row(row: SystemMenuRow) -> SystemMenuEntry {
    SystemMenuEntry {
        code: row.code,
        title: row.title,
        path: row.path,
        permission: row.permission,
        scope: row.scope,
        sort_order: row.sort_order,
    }
}

#[derive(sqlx::FromRow)]
struct OperationRecordRow {
    id: i64,
    actor_user_id: Option<i64>,
    method: String,
    path: String,
    status: i64,
    created_at: String,
}

fn operation_record_from_row(row: OperationRecordRow) -> OperationRecord {
    OperationRecord {
        id: row.id,
        actor_user_id: row.actor_user_id,
        method: row.method,
        path: row.path,
        status: row.status,
        created_at: row.created_at,
    }
}

#[derive(sqlx::FromRow)]
struct OperationRecordSummaryCountsRow {
    total_count: i64,
    success_count: i64,
    redirect_count: i64,
    client_error_count: i64,
    server_error_count: i64,
    other_count: i64,
}

#[derive(sqlx::FromRow)]
struct OperationRecordCountBucketRow {
    key: String,
    count: i64,
}

#[derive(sqlx::FromRow)]
struct OperationRecordPathBucketRow {
    path: String,
    count: i64,
    error_count: i64,
    last_seen_at: Option<String>,
}

fn count_bucket_from_row(row: OperationRecordCountBucketRow) -> OperationRecordCountBucket {
    OperationRecordCountBucket {
        key: row.key,
        count: row.count,
    }
}

fn path_bucket_from_row(row: OperationRecordPathBucketRow) -> OperationRecordPathBucket {
    OperationRecordPathBucket {
        path: row.path,
        count: row.count,
        error_count: row.error_count,
        last_seen_at: row.last_seen_at,
    }
}

#[derive(sqlx::FromRow)]
struct SystemConfigRow {
    key: String,
    value_json: String,
    updated_at: String,
}

fn config_from_row(row: SystemConfigRow) -> AppResult<SystemConfigEntry> {
    Ok(SystemConfigEntry {
        key: row.key,
        value: parse_json_value(&row.value_json)?,
        updated_at: row.updated_at,
    })
}

fn parse_json_value(value_json: &str) -> AppResult<serde_json::Value> {
    serde_json::from_str(value_json)
        .map_err(|err| AppError::Internal(format!("系统 JSON 解析失败：{err}")))
}

#[derive(sqlx::FromRow)]
struct SystemDictionaryRow {
    id: i64,
    code: String,
    name: String,
    created_at: String,
}

fn dictionary_from_row(row: SystemDictionaryRow) -> SystemDictionaryEntry {
    SystemDictionaryEntry {
        id: row.id,
        code: row.code,
        name: row.name,
        created_at: row.created_at,
    }
}

#[derive(sqlx::FromRow)]
struct SystemParameterRow {
    id: i64,
    key: String,
    name: String,
    value: String,
    created_at: String,
    updated_at: String,
}

fn parameter_from_row(row: SystemParameterRow) -> SystemParameterEntry {
    SystemParameterEntry {
        id: row.id,
        key: row.key,
        name: row.name,
        value: row.value,
        created_at: row.created_at,
        updated_at: row.updated_at,
    }
}

#[derive(sqlx::FromRow)]
struct VersionPackageRow {
    id: i64,
    version_name: String,
    version_code: String,
    manifest_json: String,
    status: String,
    created_at: String,
    published_at: Option<String>,
    retired_at: Option<String>,
}

fn version_package_from_row(row: VersionPackageRow) -> AppResult<VersionPackageEntry> {
    Ok(VersionPackageEntry {
        id: row.id,
        version_name: row.version_name,
        version_code: row.version_code,
        manifest: parse_json_value(&row.manifest_json)?,
        status: row.status,
        created_at: row.created_at,
        published_at: row.published_at,
        retired_at: row.retired_at,
    })
}

fn version_action_result(
    event_id: i64,
    previous_active_id: Option<i64>,
    target: VersionPackageRow,
    now: String,
) -> AppResult<VersionPackageActionResult> {
    Ok(VersionPackageActionResult {
        event_id,
        previous_active_id,
        package: VersionPackageEntry {
            id: target.id,
            version_name: target.version_name,
            version_code: target.version_code,
            manifest: parse_json_value(&target.manifest_json)?,
            status: "active".into(),
            created_at: target.created_at,
            published_at: Some(now),
            retired_at: None,
        },
    })
}

#[derive(sqlx::FromRow)]
struct VersionReleaseEventRow {
    id: i64,
    package_id: i64,
    previous_active_id: Option<i64>,
    action: String,
    status: String,
    reason: Option<String>,
    created_at: String,
}

fn version_release_event_from_row(row: VersionReleaseEventRow) -> VersionReleaseEventEntry {
    VersionReleaseEventEntry {
        id: row.id,
        package_id: row.package_id,
        previous_active_id: row.previous_active_id,
        action: row.action,
        status: row.status,
        reason: row.reason,
        created_at: row.created_at,
    }
}

#[derive(sqlx::FromRow)]
struct MediaAssetRow {
    id: i64,
    category: Option<String>,
    display_name: String,
    storage_key: String,
    mime_type: String,
    size_bytes: i64,
    created_at: String,
}

fn media_asset_from_row(row: MediaAssetRow) -> MediaAssetEntry {
    MediaAssetEntry {
        id: row.id,
        category: row.category,
        display_name: row.display_name,
        storage_key: row.storage_key,
        mime_type: row.mime_type,
        size_bytes: row.size_bytes,
        created_at: row.created_at,
    }
}

#[derive(sqlx::FromRow)]
struct TrafficProbeTargetRow {
    id: i64,
    name: String,
    url: String,
    expected_status: i64,
    status: String,
    created_at: String,
}

fn traffic_probe_target_from_row(row: TrafficProbeTargetRow) -> TrafficProbeTargetEntry {
    TrafficProbeTargetEntry {
        id: row.id,
        name: row.name,
        url: row.url,
        expected_status: row.expected_status,
        status: row.status,
        created_at: row.created_at,
    }
}

#[derive(sqlx::FromRow)]
struct TrafficProbeResultRow {
    id: i64,
    target_id: i64,
    status: String,
    detail_json: String,
    probed_at: String,
}

fn traffic_probe_result_from_row(row: TrafficProbeResultRow) -> AppResult<TrafficProbeResultEntry> {
    Ok(TrafficProbeResultEntry {
        id: row.id,
        target_id: row.target_id,
        status: row.status,
        detail: parse_json_value(&row.detail_json)?,
        probed_at: row.probed_at,
    })
}

#[derive(sqlx::FromRow)]
struct TrafficProbeAlertRow {
    id: i64,
    target_id: i64,
    result_id: i64,
    severity: String,
    status: String,
    reason: String,
    detail_json: String,
    opened_at: String,
    acknowledged_at: Option<String>,
    resolved_at: Option<String>,
}

fn traffic_probe_alert_from_row(row: TrafficProbeAlertRow) -> AppResult<TrafficProbeAlertEntry> {
    Ok(TrafficProbeAlertEntry {
        id: row.id,
        target_id: row.target_id,
        result_id: row.result_id,
        severity: row.severity,
        status: row.status,
        reason: row.reason,
        detail: parse_json_value(&row.detail_json)?,
        opened_at: row.opened_at,
        acknowledged_at: row.acknowledged_at,
        resolved_at: row.resolved_at,
    })
}
