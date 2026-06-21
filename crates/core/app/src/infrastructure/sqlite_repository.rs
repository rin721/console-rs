use chrono::{Duration, Utc};
use sqlx::{Pool, Row, Sqlite, SqliteConnection};

use crate::app::{AppError, AppResult};
use crate::domain::iam::{
    APITokenAuthRecord, APITokenSummary, EmailVerificationRecord, InvitationAcceptRecord,
    InvitationSummary, MfaFactorSummary, MfaRecoveryCodeSummary, Organization, OrganizationSummary,
    OrganizationUserSummary, PasswordResetRecord, PermissionSummary, RoleSummary, SessionRecord,
    StoredMfaFactor, StoredUser, User,
};
use crate::domain::notification::{
    NotificationDeadLetterRecord, NotificationFailureDisposition, NotificationOutboxItem,
};
use crate::domain::setup::{SetupRun, SetupStepLog};
use crate::domain::system::{
    ApiCatalogEntry, MediaAssetEntry, OperationRecord, OperationRecordCountBucket,
    OperationRecordPathBucket, OperationRecordSummary, SystemConfigEntry, SystemDictionaryEntry,
    SystemMenuEntry, SystemParameterEntry, TrafficProbeAlertEntry, TrafficProbeResultEntry,
    TrafficProbeTargetEntry, VersionPackageActionResult, VersionPackageEntry,
    VersionReleaseEventEntry,
};
use crate::infrastructure::sql_templates::SqlDialect;
use crate::repository::{
    AcceptInvitationRecord, CreateAPITokenRecord, CreateAuditLogRecord,
    CreateEmailVerificationRecord, CreateInitialAdminRecord, CreateInvitationRecord,
    CreateMediaAssetRecord, CreateMfaFactorRecord, CreateMfaRecoveryCodeRecord,
    CreateNotificationOutboxRecord, CreateOperationRecord, CreatePasswordResetRecord,
    CreateRegistrationRecord, CreateRoleRecord, CreateSessionRecord, CreateTrafficProbeAlertRecord,
    CreateTrafficProbeResultRecord, CreateTrafficProbeTargetRecord, CreateVersionPackageRecord,
    IamRepository, NotificationRepository, OperationRecordListQuery, OperationRecordSummaryFilter,
    SetupRepository, SystemRepository, UpdateOrgUserRecord, UpdateRoleRecord,
    UpsertSystemConfigRecord, UpsertSystemDictionaryRecord, UpsertSystemParameterRecord,
    VersionPackageActionRecord,
};

const DIALECT: SqlDialect = SqlDialect::Sqlite;

#[derive(sqlx::FromRow)]
struct SessionRecordRow {
    id: String,
    session_token_hash: String,
    refresh_token_hash: Option<String>,
    product_code: String,
    client_type: String,
    expires_at: String,
    refresh_expires_at: Option<String>,
    revoked_at: Option<String>,
    user_id: i64,
    email: String,
    display_name: String,
    user_status: String,
    org_id: i64,
    org_code: String,
    org_name: String,
    org_scope: String,
}

#[derive(Clone)]
pub struct SqliteRepository {
    pool: Pool<Sqlite>,
}

impl SqliteRepository {
    pub fn new(pool: Pool<Sqlite>) -> Self {
        Self { pool }
    }

    fn now() -> String {
        Utc::now().to_rfc3339()
    }

    async fn insert_notification_outbox(
        conn: &mut SqliteConnection,
        input: &CreateNotificationOutboxRecord,
        related_id: i64,
        created_at: &str,
    ) -> AppResult<()> {
        let outbox_id = sqlx::query_scalar::<_, i64>(DIALECT.create_notification_outbox())
            .bind(input.org_id)
            .bind(input.user_id)
            .bind(&input.product_code)
            .bind(&input.channel)
            .bind(&input.template_code)
            .bind(&input.recipient)
            .bind(&input.related_kind)
            .bind(related_id)
            .bind(&input.payload_json)
            .bind(&input.available_at)
            .bind(created_at)
            .fetch_one(&mut *conn)
            .await?;
        if let Some(secret_ciphertext) = &input.delivery_secret_ciphertext {
            sqlx::query(DIALECT.create_notification_delivery_secret())
                .bind(outbox_id)
                .bind(secret_ciphertext)
                .bind(created_at)
                .execute(&mut *conn)
                .await?;
        }
        Ok(())
    }

    async fn purge_notification_secret(
        conn: &mut SqliteConnection,
        outbox_id: i64,
        purged_at: &str,
    ) -> AppResult<()> {
        sqlx::query(DIALECT.purge_notification_delivery_secret())
            .bind(purged_at)
            .bind(outbox_id)
            .execute(&mut *conn)
            .await?;
        Ok(())
    }

    async fn insert_mfa_recovery_codes(
        conn: &mut SqliteConnection,
        user_id: i64,
        records: &[CreateMfaRecoveryCodeRecord],
        created_at: &str,
    ) -> AppResult<Vec<MfaRecoveryCodeSummary>> {
        let mut summaries = Vec::with_capacity(records.len());
        for record in records {
            let id = sqlx::query_scalar::<_, i64>(DIALECT.create_mfa_recovery_code())
                .bind(user_id)
                .bind(&record.code_hash)
                .bind(&record.code_prefix)
                .bind(created_at)
                .fetch_one(&mut *conn)
                .await?;
            summaries.push(MfaRecoveryCodeSummary {
                id,
                code_prefix: record.code_prefix.clone(),
                status: "active".into(),
                created_at: created_at.into(),
                used_at: None,
                revoked_at: None,
            });
        }
        Ok(summaries)
    }

    async fn role_permissions(&self, role_id: i64) -> AppResult<Vec<String>> {
        let rows = sqlx::query(DIALECT.role_permissions_for_role())
            .bind(role_id)
            .fetch_all(&self.pool)
            .await?;
        Ok(rows.into_iter().map(|row| row.get("code")).collect())
    }

    async fn activate_version_package(
        &self,
        input: VersionPackageActionRecord,
        action: &str,
    ) -> AppResult<VersionPackageActionResult> {
        let now = Self::now();
        let mut tx = self.pool.begin().await?;
        let target = sqlx::query_as::<_, VersionPackageRow>(DIALECT.version_package_by_id())
            .bind(input.id)
            .fetch_optional(&mut *tx)
            .await?
            .ok_or_else(|| AppError::NotFound("版本包不存在".into()))?;
        if target.status == "active" {
            return Err(AppError::Conflict("版本包已经是当前 active 版本".into()));
        }

        let previous_active_id = sqlx::query_scalar::<_, i64>(DIALECT.active_version_package_id())
            .fetch_optional(&mut *tx)
            .await?;

        if let Some(active_id) = previous_active_id {
            sqlx::query(DIALECT.retire_version_package())
                .bind(&now)
                .bind(active_id)
                .execute(&mut *tx)
                .await?;
        }

        sqlx::query(DIALECT.activate_version_package())
            .bind(&now)
            .bind(input.id)
            .execute(&mut *tx)
            .await?;

        let event_id = sqlx::query_scalar::<_, i64>(DIALECT.create_version_release_event())
            .bind(input.id)
            .bind(previous_active_id)
            .bind(action)
            .bind(&input.reason)
            .bind(&now)
            .fetch_one(&mut *tx)
            .await?;

        tx.commit().await?;

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

    async fn bind_tenant_role_permissions(
        conn: &mut SqliteConnection,
        role_id: i64,
        product_code: &str,
        permission_codes: &[String],
    ) -> AppResult<()> {
        let mut permission_ids = Vec::with_capacity(permission_codes.len());
        for code in permission_codes {
            let permission_id =
                sqlx::query_scalar::<_, i64>(DIALECT.tenant_permission_id_by_code())
                    .bind(product_code)
                    .bind(code)
                    .fetch_optional(&mut *conn)
                    .await?;
            let Some(permission_id) = permission_id else {
                return Err(AppError::Validation(format!(
                    "租户角色不能绑定不存在或平台级权限：{code}"
                )));
            };
            permission_ids.push(permission_id);
        }

        sqlx::query(DIALECT.delete_role_permissions())
            .bind(role_id)
            .execute(&mut *conn)
            .await?;
        for permission_id in permission_ids {
            sqlx::query(DIALECT.role_permission_values())
                .bind(role_id)
                .bind(permission_id)
                .execute(&mut *conn)
                .await?;
        }
        Ok(())
    }

    fn session_record_from_row(row: SessionRecordRow) -> SessionRecord {
        SessionRecord {
            id: row.id,
            token_hash: row.session_token_hash,
            refresh_token_hash: row.refresh_token_hash,
            product_code: row.product_code,
            client_type: row.client_type,
            expires_at: row.expires_at,
            refresh_expires_at: row.refresh_expires_at,
            revoked_at: row.revoked_at,
            user: User {
                id: row.user_id,
                email: row.email,
                display_name: row.display_name,
                status: row.user_status,
            },
            organization: Organization {
                id: row.org_id,
                code: row.org_code,
                name: row.org_name,
                scope: row.org_scope,
            },
        }
    }
}

#[async_trait::async_trait]
impl SetupRepository for SqliteRepository {
    async fn setup_completed(&self) -> AppResult<bool> {
        let completed = sqlx::query_scalar::<_, Option<i64>>(DIALECT.setup_completed_value())
            .fetch_optional(&self.pool)
            .await?
            .flatten()
            .unwrap_or_default();
        Ok(completed == 1)
    }

    async fn complete_setup(&self, run_id: Option<&str>) -> AppResult<bool> {
        let now = Self::now();
        let mut tx = self.pool.begin().await?;

        if let Some(run_id) = run_id {
            let run_update = sqlx::query(DIALECT.complete_setup_run())
                .bind(&now)
                .bind(run_id)
                .execute(&mut *tx)
                .await?;

            if run_update.rows_affected() == 0 {
                tx.rollback().await?;
                return Ok(false);
            }

            sqlx::query(DIALECT.append_setup_complete_log())
                .bind(run_id)
                .bind(&now)
                .execute(&mut *tx)
                .await?;
        }

        sqlx::query(DIALECT.setup_state_completed_upsert())
            .bind(&now)
            .execute(&mut *tx)
            .await?;
        tx.commit().await?;
        Ok(true)
    }

    async fn create_setup_run(&self, id: &str, reason: Option<&str>) -> AppResult<SetupRun> {
        let now = Self::now();
        sqlx::query(DIALECT.create_setup_run())
            .bind(id)
            .bind(reason)
            .bind(&now)
            .bind(&now)
            .execute(&self.pool)
            .await?;
        Ok(SetupRun {
            id: id.into(),
            status: "running".into(),
            reason: reason.map(str::to_owned),
            created_at: now.clone(),
            updated_at: now,
        })
    }

    async fn list_setup_runs(&self, limit: i64) -> AppResult<Vec<SetupRun>> {
        let rows = sqlx::query(DIALECT.setup_runs_list())
            .bind(limit)
            .fetch_all(&self.pool)
            .await?;
        Ok(rows
            .into_iter()
            .map(|row| SetupRun {
                id: row.get("id"),
                status: row.get("status"),
                reason: row.get("reason"),
                created_at: row.get("created_at"),
                updated_at: row.get("updated_at"),
            })
            .collect())
    }

    async fn append_setup_log(
        &self,
        run_id: &str,
        step_key: &str,
        status: &str,
        message: &str,
    ) -> AppResult<()> {
        sqlx::query(DIALECT.append_setup_step_log())
            .bind(run_id)
            .bind(step_key)
            .bind(status)
            .bind(message)
            .bind(Self::now())
            .execute(&self.pool)
            .await?;
        Ok(())
    }

    async fn list_setup_logs(&self, run_id: &str) -> AppResult<Vec<SetupStepLog>> {
        let rows = sqlx::query(DIALECT.setup_step_logs_for_run())
            .bind(run_id)
            .fetch_all(&self.pool)
            .await?;
        Ok(rows
            .into_iter()
            .map(|row| SetupStepLog {
                step_key: row.get("step_key"),
                status: row.get("status"),
                message: row.get("message"),
                created_at: row.get("created_at"),
            })
            .collect())
    }
}

#[async_trait::async_trait]
impl IamRepository for SqliteRepository {
    async fn has_any_user(&self) -> AppResult<bool> {
        let count = sqlx::query_scalar::<_, i64>(DIALECT.users_count())
            .fetch_one(&self.pool)
            .await?;
        Ok(count > 0)
    }

    async fn create_initial_admin(
        &self,
        input: CreateInitialAdminRecord,
    ) -> AppResult<(User, Organization)> {
        let mut tx = self.pool.begin().await?;
        let count = sqlx::query_scalar::<_, i64>(DIALECT.users_count())
            .fetch_one(&mut *tx)
            .await?;
        if count > 0 {
            return Err(AppError::Conflict("平台已存在初始化管理员".into()));
        }

        let now = Self::now();
        let org_id = sqlx::query_scalar::<_, i64>(DIALECT.create_tenant_organization())
            .bind(&input.organization_code)
            .bind(&input.organization_name)
            .bind(&now)
            .bind(&now)
            .fetch_one(&mut *tx)
            .await?;

        let user_id = sqlx::query_scalar::<_, i64>(DIALECT.create_active_user())
            .bind(&input.email)
            .bind(&input.display_name)
            .bind(&input.password_hash)
            .bind(&now)
            .bind(&now)
            .bind(&now)
            .fetch_one(&mut *tx)
            .await?;

        let tenant_role_id = sqlx::query_scalar::<_, i64>(DIALECT.create_tenant_owner_role())
            .bind(org_id)
            .bind(&now)
            .bind(&now)
            .fetch_one(&mut *tx)
            .await?;

        let platform_role_id = sqlx::query_scalar::<_, i64>(DIALECT.create_platform_owner_role())
            .bind(&now)
            .bind(&now)
            .fetch_one(&mut *tx)
            .await?;

        sqlx::query(DIALECT.create_active_membership())
            .bind(org_id)
            .bind(user_id)
            .bind("owner")
            .bind(&now)
            .bind(&now)
            .execute(&mut *tx)
            .await?;

        sqlx::query(DIALECT.create_active_membership())
            .bind(None::<i64>)
            .bind(user_id)
            .bind("platform_owner")
            .bind(&now)
            .bind(&now)
            .execute(&mut *tx)
            .await?;

        // 首个管理员必须同时拥有平台级和当前租户级能力；权限项来自 route registry 同步后的 iam_permissions。
        sqlx::query(DIALECT.role_permissions_for_tenant_scopes())
            .bind(tenant_role_id)
            .bind(&input.product_code)
            .execute(&mut *tx)
            .await?;

        sqlx::query(DIALECT.role_permissions_for_platform_scope())
            .bind(platform_role_id)
            .bind(&input.product_code)
            .execute(&mut *tx)
            .await?;

        sqlx::query(DIALECT.create_audit_log())
            .bind(org_id)
            .bind(user_id)
            .bind("iam.initial_admin.created")
            .bind("platform")
            .bind(&input.product_code)
            .bind("首次初始化管理员创建完成")
            .bind(&now)
            .execute(&mut *tx)
            .await?;

        tx.commit().await?;

        Ok((
            User {
                id: user_id,
                email: input.email,
                display_name: input.display_name,
                status: "active".into(),
            },
            Organization {
                id: org_id,
                code: input.organization_code,
                name: input.organization_name,
                scope: "tenant".into(),
            },
        ))
    }

    async fn create_registration_with_email_verification(
        &self,
        input: CreateRegistrationRecord,
        notification: CreateNotificationOutboxRecord,
    ) -> AppResult<(User, Organization)> {
        let mut tx = self.pool.begin().await?;
        let existing_user = sqlx::query_scalar::<_, i64>(DIALECT.user_email_count())
            .bind(&input.email)
            .fetch_one(&mut *tx)
            .await?;
        if existing_user > 0 {
            return Err(AppError::Conflict("注册邮箱已存在".into()));
        }
        let existing_org = sqlx::query_scalar::<_, i64>(DIALECT.organization_code_count())
            .bind(&input.organization_code)
            .fetch_one(&mut *tx)
            .await?;
        if existing_org > 0 {
            return Err(AppError::Conflict("组织编码已存在".into()));
        }

        let now = Self::now();
        let org_id = sqlx::query_scalar::<_, i64>(DIALECT.create_tenant_organization())
            .bind(&input.organization_code)
            .bind(&input.organization_name)
            .bind(&now)
            .bind(&now)
            .fetch_one(&mut *tx)
            .await?;

        let user_id = sqlx::query_scalar::<_, i64>(DIALECT.create_pending_verification_user())
            .bind(&input.email)
            .bind(&input.display_name)
            .bind(&input.password_hash)
            .bind(&now)
            .bind(&now)
            .fetch_one(&mut *tx)
            .await?;

        let owner_role_id = sqlx::query_scalar::<_, i64>(DIALECT.create_tenant_owner_role())
            .bind(org_id)
            .bind(&now)
            .bind(&now)
            .fetch_one(&mut *tx)
            .await?;

        sqlx::query(DIALECT.create_active_membership())
            .bind(org_id)
            .bind(user_id)
            .bind("owner")
            .bind(&now)
            .bind(&now)
            .execute(&mut *tx)
            .await?;

        sqlx::query(DIALECT.role_permissions_for_tenant_scopes())
            .bind(owner_role_id)
            .bind(&input.product_code)
            .execute(&mut *tx)
            .await?;

        let verification_id = sqlx::query_scalar::<_, i64>(DIALECT.create_email_verification())
            .bind(user_id)
            .bind(&input.email)
            .bind(&input.email_verification_token_hash)
            .bind(&input.email_verification_expires_at)
            .bind(&now)
            .fetch_one(&mut *tx)
            .await?;
        let notification = CreateNotificationOutboxRecord {
            org_id: Some(org_id),
            user_id: Some(user_id),
            ..notification
        };
        Self::insert_notification_outbox(&mut tx, &notification, verification_id, &now).await?;

        sqlx::query(DIALECT.create_audit_log())
            .bind(org_id)
            .bind(user_id)
            .bind("iam.registration.created")
            .bind("tenant")
            .bind(&input.product_code)
            .bind("自助注册已创建，等待邮箱验证")
            .bind(&now)
            .execute(&mut *tx)
            .await?;

        tx.commit().await?;

        Ok((
            User {
                id: user_id,
                email: input.email,
                display_name: input.display_name,
                status: "pending_verification".into(),
            },
            Organization {
                id: org_id,
                code: input.organization_code,
                name: input.organization_name,
                scope: "tenant".into(),
            },
        ))
    }

    async fn find_user_by_identifier(&self, identifier: &str) -> AppResult<Option<StoredUser>> {
        let row = sqlx::query(DIALECT.user_by_identifier())
            .bind(identifier)
            .fetch_optional(&self.pool)
            .await?;

        Ok(row.map(|row| StoredUser {
            id: row.get("id"),
            email: row.get("email"),
            display_name: row.get("display_name"),
            password_hash: row.get("password_hash"),
            status: row.get("status"),
        }))
    }

    async fn primary_organization_for_user(&self, user_id: i64) -> AppResult<Option<Organization>> {
        let row = sqlx::query(DIALECT.primary_organization_for_user())
            .bind(user_id)
            .fetch_optional(&self.pool)
            .await?;

        Ok(row.map(|row| Organization {
            id: row.get("id"),
            code: row.get("code"),
            name: row.get("name"),
            scope: row.get("scope"),
        }))
    }

    async fn list_organizations(&self) -> AppResult<Vec<OrganizationSummary>> {
        let rows = sqlx::query(DIALECT.organizations_list())
            .fetch_all(&self.pool)
            .await?;
        Ok(rows
            .into_iter()
            .map(|row| OrganizationSummary {
                id: row.get("id"),
                code: row.get("code"),
                name: row.get("name"),
                scope: row.get("scope"),
                status: row.get("status"),
                created_at: row.get("created_at"),
            })
            .collect())
    }

    async fn list_org_users(&self, org_id: i64) -> AppResult<Vec<OrganizationUserSummary>> {
        let rows = sqlx::query(DIALECT.org_users_list())
            .bind(org_id)
            .fetch_all(&self.pool)
            .await?;
        Ok(rows
            .into_iter()
            .map(|row| {
                let role_codes: Option<String> = row.get("role_codes");
                OrganizationUserSummary {
                    id: row.get("id"),
                    email: row.get("email"),
                    display_name: row.get("display_name"),
                    status: row.get("status"),
                    email_verified_at: row.get("email_verified_at"),
                    role_codes: split_csv(role_codes),
                }
            })
            .collect())
    }

    async fn update_org_user(
        &self,
        input: UpdateOrgUserRecord,
    ) -> AppResult<OrganizationUserSummary> {
        let now = Self::now();
        let mut tx = self.pool.begin().await?;
        let user_row = sqlx::query(DIALECT.org_user_membership_context())
            .bind(input.org_id)
            .bind(input.org_id)
            .bind(input.user_id)
            .fetch_optional(&mut *tx)
            .await?
            .ok_or_else(|| AppError::NotFound("组织用户不存在".into()))?;

        if user_row.get::<i64, _>("in_org") == 0 {
            return Err(AppError::NotFound("组织用户不存在".into()));
        }

        for role_code in &input.role_codes {
            let role_exists = sqlx::query_scalar::<_, i64>(DIALECT.tenant_org_role_code_count())
                .bind(input.org_id)
                .bind(role_code)
                .fetch_one(&mut *tx)
                .await?;
            if role_exists == 0 {
                return Err(AppError::Validation("组织用户角色必须来自当前租户".into()));
            }
        }

        let currently_owner = user_row.get::<i64, _>("is_owner") == 1;
        let keeps_owner =
            input.status == "active" && input.role_codes.iter().any(|code| code == "owner");
        if currently_owner && !keeps_owner {
            let other_owner_count =
                sqlx::query_scalar::<_, i64>(DIALECT.org_active_owner_count_except_user())
                    .bind(input.org_id)
                    .bind(input.user_id)
                    .fetch_one(&mut *tx)
                    .await?;
            if other_owner_count == 0 {
                return Err(AppError::Conflict(
                    "不能移除或禁用组织最后一个 active owner".into(),
                ));
            }
        }

        sqlx::query(DIALECT.update_org_user_profile_status())
            .bind(&input.display_name)
            .bind(&input.status)
            .bind(&now)
            .bind(input.user_id)
            .execute(&mut *tx)
            .await?;

        sqlx::query(DIALECT.delete_org_user_memberships())
            .bind(input.org_id)
            .bind(input.user_id)
            .execute(&mut *tx)
            .await?;

        for role_code in &input.role_codes {
            sqlx::query(DIALECT.create_active_membership())
                .bind(input.org_id)
                .bind(input.user_id)
                .bind(role_code)
                .bind(&now)
                .bind(&now)
                .execute(&mut *tx)
                .await?;
        }

        let summary = OrganizationUserSummary {
            id: input.user_id,
            email: user_row.get("email"),
            display_name: input.display_name,
            status: input.status,
            email_verified_at: user_row.get("email_verified_at"),
            role_codes: input.role_codes,
        };
        tx.commit().await?;
        Ok(summary)
    }

    async fn list_org_roles(&self, org_id: i64) -> AppResult<Vec<RoleSummary>> {
        let rows = sqlx::query(DIALECT.org_roles_list())
            .bind(org_id)
            .fetch_all(&self.pool)
            .await?;
        let mut roles = Vec::with_capacity(rows.len());
        for row in rows {
            let role_id = row.get("id");
            roles.push(RoleSummary {
                id: role_id,
                org_id: row.get("org_id"),
                code: row.get("code"),
                name: row.get("name"),
                scope: row.get("scope"),
                system_builtin: row.get::<i64, _>("system_builtin") == 1,
                permissions: self.role_permissions(role_id).await?,
            });
        }
        Ok(roles)
    }

    async fn create_org_role(&self, input: CreateRoleRecord) -> AppResult<RoleSummary> {
        let now = Self::now();
        let mut tx = self.pool.begin().await?;
        let existing_count = sqlx::query_scalar::<_, i64>(DIALECT.org_role_code_count())
            .bind(input.org_id)
            .bind(&input.code)
            .fetch_one(&mut *tx)
            .await?;
        if existing_count > 0 {
            return Err(AppError::Conflict("组织角色编码已存在".into()));
        }

        let role_id = sqlx::query_scalar::<_, i64>(DIALECT.create_org_role())
            .bind(input.org_id)
            .bind(&input.code)
            .bind(&input.name)
            .bind(&now)
            .bind(&now)
            .fetch_one(&mut *tx)
            .await?;
        Self::bind_tenant_role_permissions(
            &mut tx,
            role_id,
            &input.product_code,
            &input.permission_codes,
        )
        .await?;
        tx.commit().await?;
        Ok(RoleSummary {
            id: role_id,
            org_id: Some(input.org_id),
            code: input.code,
            name: input.name,
            scope: "tenant".into(),
            system_builtin: false,
            permissions: input.permission_codes,
        })
    }

    async fn update_org_role(&self, input: UpdateRoleRecord) -> AppResult<RoleSummary> {
        let now = Self::now();
        let mut tx = self.pool.begin().await?;
        let row = sqlx::query(DIALECT.tenant_org_role_by_id())
            .bind(input.role_id)
            .bind(input.org_id)
            .fetch_optional(&mut *tx)
            .await?
            .ok_or_else(|| AppError::NotFound("组织角色不存在".into()))?;
        if row.get::<i64, _>("system_builtin") == 1 {
            return Err(AppError::Conflict("内置角色不能修改".into()));
        }
        sqlx::query(DIALECT.update_org_role_name())
            .bind(&input.name)
            .bind(&now)
            .bind(input.role_id)
            .execute(&mut *tx)
            .await?;
        Self::bind_tenant_role_permissions(
            &mut tx,
            input.role_id,
            &input.product_code,
            &input.permission_codes,
        )
        .await?;
        tx.commit().await?;
        Ok(RoleSummary {
            id: input.role_id,
            org_id: Some(input.org_id),
            code: row.get("code"),
            name: input.name,
            scope: "tenant".into(),
            system_builtin: false,
            permissions: input.permission_codes,
        })
    }

    async fn delete_org_role(&self, org_id: i64, role_id: i64) -> AppResult<bool> {
        let mut tx = self.pool.begin().await?;
        let row = sqlx::query(DIALECT.tenant_org_role_by_id())
            .bind(role_id)
            .bind(org_id)
            .fetch_optional(&mut *tx)
            .await?;
        let Some(row) = row else {
            tx.rollback().await?;
            return Ok(false);
        };
        if row.get::<i64, _>("system_builtin") == 1 {
            return Err(AppError::Conflict("内置角色不能删除".into()));
        }
        let role_code: String = row.get("code");
        let member_count = sqlx::query_scalar::<_, i64>(DIALECT.org_role_active_member_count())
            .bind(org_id)
            .bind(&role_code)
            .fetch_one(&mut *tx)
            .await?;
        if member_count > 0 {
            return Err(AppError::Conflict("角色仍被组织成员使用，不能删除".into()));
        }
        let pending_invitation_count =
            sqlx::query_scalar::<_, i64>(DIALECT.org_role_pending_invitation_count())
                .bind(org_id)
                .bind(&role_code)
                .fetch_one(&mut *tx)
                .await?;
        if pending_invitation_count > 0 {
            return Err(AppError::Conflict(
                "角色仍被待处理邀请使用，不能删除".into(),
            ));
        }
        sqlx::query(DIALECT.delete_org_role())
            .bind(role_id)
            .bind(org_id)
            .execute(&mut *tx)
            .await?;
        tx.commit().await?;
        Ok(true)
    }

    async fn list_permissions(&self, product_code: &str) -> AppResult<Vec<PermissionSummary>> {
        let rows = sqlx::query(DIALECT.permissions_list())
            .bind(product_code)
            .fetch_all(&self.pool)
            .await?;
        Ok(rows
            .into_iter()
            .map(|row| PermissionSummary {
                id: row.get("id"),
                product_code: row.get("product_code"),
                scope: row.get("scope"),
                code: row.get("code"),
                name: row.get("name"),
            })
            .collect())
    }

    async fn create_session(&self, input: CreateSessionRecord) -> AppResult<()> {
        let now = Self::now();
        sqlx::query(DIALECT.create_session())
            .bind(input.id)
            .bind(input.token_hash)
            .bind(input.refresh_token_hash)
            .bind(input.user_id)
            .bind(input.org_id)
            .bind(input.product_code)
            .bind(input.client_type)
            .bind(input.expires_at)
            .bind(input.refresh_expires_at)
            .bind(&now)
            .bind(&now)
            .execute(&self.pool)
            .await?;
        Ok(())
    }

    async fn find_session_by_hash(&self, token_hash: &str) -> AppResult<Option<SessionRecord>> {
        let row = sqlx::query_as::<_, SessionRecordRow>(DIALECT.session_by_token_hash())
            .bind(token_hash)
            .bind(1_i64)
            .fetch_optional(&self.pool)
            .await?;

        Ok(row.map(Self::session_record_from_row))
    }

    async fn find_session_by_refresh_hash(
        &self,
        refresh_hash: &str,
    ) -> AppResult<Option<SessionRecord>> {
        let row = sqlx::query_as::<_, SessionRecordRow>(DIALECT.session_by_refresh_hash())
            .bind(refresh_hash)
            .bind(1_i64)
            .fetch_optional(&self.pool)
            .await?;

        Ok(row.map(Self::session_record_from_row))
    }

    async fn rotate_session_tokens(
        &self,
        session_id: &str,
        current_refresh_hash: &str,
        new_session_hash: String,
        new_refresh_hash: String,
        expires_at: String,
        refresh_expires_at: String,
    ) -> AppResult<bool> {
        let now = Self::now();
        let result = sqlx::query(DIALECT.rotate_session_tokens())
            .bind(new_session_hash)
            .bind(new_refresh_hash)
            .bind(expires_at)
            .bind(refresh_expires_at)
            .bind(&now)
            .bind(session_id)
            .bind(current_refresh_hash)
            .execute(&self.pool)
            .await?;
        Ok(result.rows_affected() == 1)
    }

    async fn revoke_session_by_hash(&self, token_hash: &str) -> AppResult<()> {
        let now = Self::now();
        sqlx::query(DIALECT.revoke_session_by_token_hash())
            .bind(&now)
            .bind(&now)
            .bind(token_hash)
            .execute(&self.pool)
            .await?;
        Ok(())
    }

    async fn revoke_session_by_refresh_hash(&self, refresh_hash: &str) -> AppResult<()> {
        let now = Self::now();
        sqlx::query(DIALECT.revoke_session_by_refresh_hash())
            .bind(&now)
            .bind(&now)
            .bind(refresh_hash)
            .execute(&self.pool)
            .await?;
        Ok(())
    }

    async fn list_permissions_for_user(
        &self,
        user_id: i64,
        org_id: i64,
        product_code: &str,
        include_platform: bool,
    ) -> AppResult<Vec<String>> {
        let rows = sqlx::query(DIALECT.permissions_for_user())
            .bind(user_id)
            .bind(product_code)
            .bind(org_id)
            .bind(if include_platform { 1 } else { 0 })
            .fetch_all(&self.pool)
            .await?;
        Ok(rows.into_iter().map(|row| row.get("code")).collect())
    }

    async fn create_api_token(&self, input: CreateAPITokenRecord) -> AppResult<APITokenSummary> {
        let now = Self::now();
        let id = sqlx::query_scalar::<_, i64>(DIALECT.create_api_token())
            .bind(input.org_id)
            .bind(input.user_id)
            .bind(input.token_hash)
            .bind(&input.token_prefix)
            .bind(&input.expires_at)
            .bind(&now)
            .fetch_one(&self.pool)
            .await?;
        Ok(APITokenSummary {
            id,
            org_id: input.org_id,
            user_id: input.user_id,
            token_prefix: input.token_prefix,
            status: "active".into(),
            expires_at: input.expires_at,
            created_at: now,
            revoked_at: None,
        })
    }

    async fn find_api_token_by_hash(
        &self,
        token_hash: &str,
    ) -> AppResult<Option<APITokenAuthRecord>> {
        let row = sqlx::query_as::<_, APITokenAuthRow>(DIALECT.api_token_by_hash())
            .bind(token_hash)
            .bind(1_i64)
            .fetch_optional(&self.pool)
            .await?;

        Ok(row.map(api_token_auth_from_row))
    }

    async fn list_api_tokens(&self, org_id: i64) -> AppResult<Vec<APITokenSummary>> {
        let rows = sqlx::query_as::<_, APITokenSummaryRow>(DIALECT.api_tokens_for_org())
            .bind(org_id)
            .fetch_all(&self.pool)
            .await?;
        Ok(rows.into_iter().map(api_token_from_row).collect())
    }

    async fn revoke_api_token(&self, org_id: i64, token_id: i64) -> AppResult<bool> {
        let now = Self::now();
        let result = sqlx::query(DIALECT.revoke_api_token())
            .bind(now)
            .bind(org_id)
            .bind(token_id)
            .execute(&self.pool)
            .await?;
        Ok(result.rows_affected() > 0)
    }

    async fn org_role_exists(&self, org_id: i64, role_code: &str) -> AppResult<bool> {
        let count = sqlx::query_scalar::<_, i64>(DIALECT.org_role_code_count())
            .bind(org_id)
            .bind(role_code)
            .fetch_one(&self.pool)
            .await?;
        Ok(count > 0)
    }

    async fn create_invitation_with_notification(
        &self,
        input: CreateInvitationRecord,
        notification: CreateNotificationOutboxRecord,
    ) -> AppResult<InvitationSummary> {
        let now = Self::now();
        let mut tx = self.pool.begin().await?;
        let invitation_id = sqlx::query_scalar::<_, i64>(DIALECT.create_invitation())
            .bind(input.org_id)
            .bind(&input.email)
            .bind(&input.role_code)
            .bind(input.token_hash)
            .bind(&input.expires_at)
            .bind(&now)
            .fetch_one(&mut *tx)
            .await?;
        Self::insert_notification_outbox(&mut tx, &notification, invitation_id, &now).await?;
        tx.commit().await?;
        Ok(InvitationSummary {
            id: invitation_id,
            org_id: input.org_id,
            email: input.email,
            role_code: input.role_code,
            status: "pending".into(),
            expires_at: input.expires_at,
            created_at: now,
            accepted_at: None,
            revoked_at: None,
        })
    }

    async fn list_invitations(&self, org_id: i64) -> AppResult<Vec<InvitationSummary>> {
        let rows = sqlx::query_as::<_, InvitationSummaryRow>(DIALECT.invitations_for_org())
            .bind(org_id)
            .fetch_all(&self.pool)
            .await?;
        Ok(rows.into_iter().map(invitation_from_row).collect())
    }

    async fn revoke_invitation(&self, org_id: i64, invitation_id: i64) -> AppResult<bool> {
        let now = Self::now();
        let result = sqlx::query(DIALECT.revoke_invitation())
            .bind(now)
            .bind(org_id)
            .bind(invitation_id)
            .execute(&self.pool)
            .await?;
        Ok(result.rows_affected() > 0)
    }

    async fn find_invitation_by_hash(
        &self,
        token_hash: &str,
    ) -> AppResult<Option<InvitationAcceptRecord>> {
        let row = sqlx::query(DIALECT.invitation_by_hash())
            .bind(token_hash)
            .fetch_optional(&self.pool)
            .await?;
        Ok(row.map(|row| InvitationAcceptRecord {
            id: row.get("id"),
            org_id: row.get("org_id"),
            email: row.get("email"),
            role_code: row.get("role_code"),
            status: row.get("status"),
            expires_at: row.get("expires_at"),
            accepted_at: row.get("accepted_at"),
            revoked_at: row.get("revoked_at"),
        }))
    }

    async fn accept_invitation_with_user(
        &self,
        input: AcceptInvitationRecord,
    ) -> AppResult<(User, Organization)> {
        let now = Self::now();
        let mut tx = self.pool.begin().await?;
        let org_row = sqlx::query(DIALECT.organization_by_id())
            .bind(input.org_id)
            .fetch_optional(&mut *tx)
            .await?
            .ok_or_else(|| AppError::NotFound("邀请组织不存在".into()))?;
        let org = Organization {
            id: org_row.get("id"),
            code: org_row.get("code"),
            name: org_row.get("name"),
            scope: org_row.get("scope"),
        };

        let user_id = sqlx::query_scalar::<_, i64>(DIALECT.create_active_user())
            .bind(&input.email)
            .bind(&input.display_name)
            .bind(&input.password_hash)
            .bind(&now)
            .bind(&now)
            .bind(&now)
            .fetch_one(&mut *tx)
            .await?;

        sqlx::query(DIALECT.create_active_membership())
            .bind(input.org_id)
            .bind(user_id)
            .bind(&input.role_code)
            .bind(&now)
            .bind(&now)
            .execute(&mut *tx)
            .await?;

        let accept_result = sqlx::query(DIALECT.accept_invitation())
            .bind(&now)
            .bind(input.invitation_id)
            .execute(&mut *tx)
            .await?;
        if accept_result.rows_affected() == 0 {
            tx.rollback().await?;
            return Err(AppError::Unauthorized);
        }
        tx.commit().await?;

        Ok((
            User {
                id: user_id,
                email: input.email,
                display_name: input.display_name,
                status: "active".into(),
            },
            org,
        ))
    }

    async fn create_password_reset_with_notification(
        &self,
        input: CreatePasswordResetRecord,
        notification: CreateNotificationOutboxRecord,
    ) -> AppResult<()> {
        let now = Self::now();
        let mut tx = self.pool.begin().await?;
        let reset_id = sqlx::query_scalar::<_, i64>(DIALECT.create_password_reset())
            .bind(input.user_id)
            .bind(input.token_hash)
            .bind(input.expires_at)
            .bind(&now)
            .fetch_one(&mut *tx)
            .await?;
        Self::insert_notification_outbox(&mut tx, &notification, reset_id, &now).await?;
        tx.commit().await?;
        Ok(())
    }

    async fn find_password_reset_by_hash(
        &self,
        token_hash: &str,
    ) -> AppResult<Option<PasswordResetRecord>> {
        let row = sqlx::query(DIALECT.password_reset_by_hash())
            .bind(token_hash)
            .fetch_optional(&self.pool)
            .await?;
        Ok(row.map(|row| PasswordResetRecord {
            id: row.get("id"),
            user_id: row.get("user_id"),
            status: row.get("status"),
            expires_at: row.get("expires_at"),
            used_at: row.get("used_at"),
        }))
    }

    async fn reset_password_with_token(
        &self,
        reset_id: i64,
        user_id: i64,
        password_hash: String,
    ) -> AppResult<bool> {
        let now = Self::now();
        let mut tx = self.pool.begin().await?;
        let reset_result = sqlx::query(DIALECT.mark_password_reset_used())
            .bind(&now)
            .bind(reset_id)
            .bind(user_id)
            .execute(&mut *tx)
            .await?;
        if reset_result.rows_affected() == 0 {
            tx.rollback().await?;
            return Ok(false);
        }
        sqlx::query(DIALECT.update_user_password_hash())
            .bind(password_hash)
            .bind(&now)
            .bind(user_id)
            .execute(&mut *tx)
            .await?;
        tx.commit().await?;
        Ok(true)
    }

    async fn create_email_verification_with_notification(
        &self,
        input: CreateEmailVerificationRecord,
        notification: CreateNotificationOutboxRecord,
    ) -> AppResult<()> {
        let now = Self::now();
        let mut tx = self.pool.begin().await?;
        let verification_id = sqlx::query_scalar::<_, i64>(DIALECT.create_email_verification())
            .bind(input.user_id)
            .bind(input.email)
            .bind(input.token_hash)
            .bind(input.expires_at)
            .bind(&now)
            .fetch_one(&mut *tx)
            .await?;
        Self::insert_notification_outbox(&mut tx, &notification, verification_id, &now).await?;
        tx.commit().await?;
        Ok(())
    }

    async fn find_email_verification_by_hash(
        &self,
        token_hash: &str,
    ) -> AppResult<Option<EmailVerificationRecord>> {
        let row = sqlx::query(DIALECT.email_verification_by_hash())
            .bind(token_hash)
            .fetch_optional(&self.pool)
            .await?;
        Ok(row.map(|row| EmailVerificationRecord {
            id: row.get("id"),
            user_id: row.get("user_id"),
            status: row.get("status"),
            expires_at: row.get("expires_at"),
            verified_at: row.get("verified_at"),
        }))
    }

    async fn confirm_email_verification(
        &self,
        verification_id: i64,
        user_id: i64,
    ) -> AppResult<bool> {
        let now = Self::now();
        let mut tx = self.pool.begin().await?;
        let verify_result = sqlx::query(DIALECT.mark_email_verification_verified())
            .bind(&now)
            .bind(verification_id)
            .bind(user_id)
            .execute(&mut *tx)
            .await?;
        if verify_result.rows_affected() == 0 {
            tx.rollback().await?;
            return Ok(false);
        }
        sqlx::query(DIALECT.mark_user_email_verified())
            .bind(&now)
            .bind(&now)
            .bind(user_id)
            .execute(&mut *tx)
            .await?;
        tx.commit().await?;
        Ok(true)
    }

    async fn create_pending_mfa_factor(
        &self,
        input: CreateMfaFactorRecord,
    ) -> AppResult<MfaFactorSummary> {
        let now = Self::now();
        let mut tx = self.pool.begin().await?;
        sqlx::query(DIALECT.revoke_pending_mfa_factors())
            .bind(&now)
            .bind(input.user_id)
            .bind(&input.kind)
            .execute(&mut *tx)
            .await?;
        let id = sqlx::query_scalar::<_, i64>(DIALECT.create_mfa_factor())
            .bind(input.user_id)
            .bind(&input.kind)
            .bind(&input.secret_ciphertext)
            .bind(&now)
            .fetch_one(&mut *tx)
            .await?;
        tx.commit().await?;
        Ok(MfaFactorSummary {
            id,
            kind: input.kind,
            status: "pending".into(),
            created_at: now,
            verified_at: None,
            revoked_at: None,
        })
    }

    async fn list_mfa_factors(&self, user_id: i64) -> AppResult<Vec<MfaFactorSummary>> {
        let rows = sqlx::query_as::<_, MfaFactorSummaryRow>(DIALECT.mfa_factors_for_user())
            .bind(user_id)
            .fetch_all(&self.pool)
            .await?;
        Ok(rows.into_iter().map(mfa_factor_summary_from_row).collect())
    }

    async fn find_pending_mfa_factor(&self, user_id: i64) -> AppResult<Option<StoredMfaFactor>> {
        let row = sqlx::query_as::<_, StoredMfaFactorRow>(DIALECT.pending_mfa_factor_for_user())
            .bind(user_id)
            .fetch_optional(&self.pool)
            .await?;
        Ok(row.map(stored_mfa_factor_from_row))
    }

    async fn find_verified_mfa_factor(&self, user_id: i64) -> AppResult<Option<StoredMfaFactor>> {
        let row = sqlx::query_as::<_, StoredMfaFactorRow>(DIALECT.verified_mfa_factor_for_user())
            .bind(user_id)
            .fetch_optional(&self.pool)
            .await?;
        Ok(row.map(stored_mfa_factor_from_row))
    }

    async fn activate_mfa_factor_with_recovery_codes(
        &self,
        user_id: i64,
        factor_id: i64,
        recovery_codes: Vec<CreateMfaRecoveryCodeRecord>,
    ) -> AppResult<bool> {
        let now = Self::now();
        let mut tx = self.pool.begin().await?;
        let activate = sqlx::query(DIALECT.activate_mfa_factor())
            .bind(&now)
            .bind(factor_id)
            .bind(user_id)
            .execute(&mut *tx)
            .await?;
        if activate.rows_affected() == 0 {
            tx.rollback().await?;
            return Ok(false);
        }
        sqlx::query(DIALECT.revoke_active_mfa_recovery_codes())
            .bind(&now)
            .bind(user_id)
            .execute(&mut *tx)
            .await?;
        Self::insert_mfa_recovery_codes(&mut tx, user_id, &recovery_codes, &now).await?;
        sqlx::query(DIALECT.revoke_other_mfa_factors())
            .bind(&now)
            .bind(user_id)
            .bind(factor_id)
            .execute(&mut *tx)
            .await?;
        tx.commit().await?;
        Ok(true)
    }

    async fn revoke_mfa_factor(&self, user_id: i64, factor_id: i64) -> AppResult<bool> {
        let now = Self::now();
        let mut tx = self.pool.begin().await?;
        let result = sqlx::query(DIALECT.revoke_mfa_factor())
            .bind(&now)
            .bind(factor_id)
            .bind(user_id)
            .execute(&mut *tx)
            .await?;
        let revoked = result.rows_affected() > 0;
        if revoked {
            sqlx::query(DIALECT.revoke_active_mfa_recovery_codes())
                .bind(&now)
                .bind(user_id)
                .execute(&mut *tx)
                .await?;
        }
        tx.commit().await?;
        Ok(revoked)
    }

    async fn list_mfa_recovery_codes(
        &self,
        user_id: i64,
    ) -> AppResult<Vec<MfaRecoveryCodeSummary>> {
        let rows =
            sqlx::query_as::<_, MfaRecoveryCodeSummaryRow>(DIALECT.mfa_recovery_codes_for_user())
                .bind(user_id)
                .fetch_all(&self.pool)
                .await?;
        Ok(rows
            .into_iter()
            .map(mfa_recovery_code_summary_from_row)
            .collect())
    }

    async fn replace_mfa_recovery_codes(
        &self,
        user_id: i64,
        recovery_codes: Vec<CreateMfaRecoveryCodeRecord>,
    ) -> AppResult<Vec<MfaRecoveryCodeSummary>> {
        let now = Self::now();
        let mut tx = self.pool.begin().await?;
        sqlx::query(DIALECT.revoke_active_mfa_recovery_codes())
            .bind(&now)
            .bind(user_id)
            .execute(&mut *tx)
            .await?;
        let summaries =
            Self::insert_mfa_recovery_codes(&mut tx, user_id, &recovery_codes, &now).await?;
        tx.commit().await?;
        Ok(summaries)
    }

    async fn consume_mfa_recovery_code(&self, user_id: i64, code_hash: &str) -> AppResult<bool> {
        let now = Self::now();
        let result = sqlx::query(DIALECT.consume_mfa_recovery_code())
            .bind(&now)
            .bind(user_id)
            .bind(code_hash)
            .execute(&self.pool)
            .await?;
        Ok(result.rows_affected() > 0)
    }

    async fn record_audit(&self, input: CreateAuditLogRecord) -> AppResult<()> {
        sqlx::query(DIALECT.create_audit_log())
            .bind(input.org_id)
            .bind(input.user_id)
            .bind(input.action)
            .bind(input.scope)
            .bind(input.product_code)
            .bind(input.detail)
            .bind(Self::now())
            .execute(&self.pool)
            .await?;
        Ok(())
    }
}

#[derive(sqlx::FromRow)]
struct APITokenSummaryRow {
    id: i64,
    org_id: i64,
    user_id: i64,
    token_prefix: String,
    status: String,
    expires_at: Option<String>,
    created_at: String,
    revoked_at: Option<String>,
}

fn api_token_from_row(row: APITokenSummaryRow) -> APITokenSummary {
    APITokenSummary {
        id: row.id,
        org_id: row.org_id,
        user_id: row.user_id,
        token_prefix: row.token_prefix,
        status: row.status,
        expires_at: row.expires_at,
        created_at: row.created_at,
        revoked_at: row.revoked_at,
    }
}

#[derive(sqlx::FromRow)]
struct APITokenAuthRow {
    id: i64,
    status: String,
    expires_at: Option<String>,
    revoked_at: Option<String>,
    user_id: i64,
    email: String,
    display_name: String,
    user_status: String,
    org_id: i64,
    org_code: String,
    org_name: String,
    org_scope: String,
}

fn api_token_auth_from_row(row: APITokenAuthRow) -> APITokenAuthRecord {
    APITokenAuthRecord {
        id: row.id,
        status: row.status,
        expires_at: row.expires_at,
        revoked_at: row.revoked_at,
        user: User {
            id: row.user_id,
            email: row.email,
            display_name: row.display_name,
            status: row.user_status,
        },
        organization: Organization {
            id: row.org_id,
            code: row.org_code,
            name: row.org_name,
            scope: row.org_scope,
        },
    }
}

#[derive(sqlx::FromRow)]
struct InvitationSummaryRow {
    id: i64,
    org_id: i64,
    email: String,
    role_code: String,
    status: String,
    expires_at: String,
    created_at: String,
    accepted_at: Option<String>,
    revoked_at: Option<String>,
}

fn invitation_from_row(row: InvitationSummaryRow) -> InvitationSummary {
    InvitationSummary {
        id: row.id,
        org_id: row.org_id,
        email: row.email,
        role_code: row.role_code,
        status: row.status,
        expires_at: row.expires_at,
        created_at: row.created_at,
        accepted_at: row.accepted_at,
        revoked_at: row.revoked_at,
    }
}

#[derive(sqlx::FromRow)]
struct MfaFactorSummaryRow {
    id: i64,
    kind: String,
    status: String,
    created_at: String,
    verified_at: Option<String>,
    revoked_at: Option<String>,
}

fn mfa_factor_summary_from_row(row: MfaFactorSummaryRow) -> MfaFactorSummary {
    MfaFactorSummary {
        id: row.id,
        kind: row.kind,
        status: row.status,
        created_at: row.created_at,
        verified_at: row.verified_at,
        revoked_at: row.revoked_at,
    }
}

#[derive(sqlx::FromRow)]
struct MfaRecoveryCodeSummaryRow {
    id: i64,
    code_prefix: String,
    status: String,
    created_at: String,
    used_at: Option<String>,
    revoked_at: Option<String>,
}

fn mfa_recovery_code_summary_from_row(row: MfaRecoveryCodeSummaryRow) -> MfaRecoveryCodeSummary {
    MfaRecoveryCodeSummary {
        id: row.id,
        code_prefix: row.code_prefix,
        status: row.status,
        created_at: row.created_at,
        used_at: row.used_at,
        revoked_at: row.revoked_at,
    }
}

#[derive(sqlx::FromRow)]
struct StoredMfaFactorRow {
    id: i64,
    user_id: i64,
    kind: String,
    secret_ciphertext: String,
    status: String,
    created_at: String,
    verified_at: Option<String>,
    revoked_at: Option<String>,
}

fn stored_mfa_factor_from_row(row: StoredMfaFactorRow) -> StoredMfaFactor {
    StoredMfaFactor {
        id: row.id,
        user_id: row.user_id,
        kind: row.kind,
        secret_ciphertext: row.secret_ciphertext,
        status: row.status,
        created_at: row.created_at,
        verified_at: row.verified_at,
        revoked_at: row.revoked_at,
    }
}

#[async_trait::async_trait]
impl NotificationRepository for SqliteRepository {
    async fn claim_due_notifications(
        &self,
        limit: i64,
        lock_ttl_seconds: i64,
    ) -> AppResult<Vec<NotificationOutboxItem>> {
        let now = Utc::now();
        let now_text = now.to_rfc3339();
        let stale_locked_before = (now - Duration::seconds(lock_ttl_seconds.max(1))).to_rfc3339();
        let mut tx = self.pool.begin().await?;
        let rows = sqlx::query_as::<_, NotificationOutboxRow>(DIALECT.due_notifications())
            .bind(&now_text)
            .bind(&stale_locked_before)
            .bind(limit)
            .fetch_all(&mut *tx)
            .await?;

        for row in &rows {
            sqlx::query(DIALECT.claim_notification())
                .bind(&now_text)
                .bind(row.id)
                .execute(&mut *tx)
                .await?;
        }

        tx.commit().await?;
        Ok(rows.into_iter().map(notification_outbox_from_row).collect())
    }

    async fn mark_notification_delivered(&self, notification_id: i64) -> AppResult<bool> {
        let now = Self::now();
        let mut tx = self.pool.begin().await?;
        let result = sqlx::query(DIALECT.mark_notification_delivered())
            .bind(&now)
            .bind(notification_id)
            .execute(&mut *tx)
            .await?;
        Self::purge_notification_secret(&mut tx, notification_id, &now).await?;
        tx.commit().await?;
        Ok(result.rows_affected() > 0)
    }

    async fn mark_notification_failed(
        &self,
        notification_id: i64,
        failure_reason: &str,
        retry_backoff_seconds: i64,
        max_attempts: i64,
    ) -> AppResult<NotificationFailureDisposition> {
        let now = Utc::now();
        let now_text = now.to_rfc3339();
        let retry_at = (now + Duration::seconds(retry_backoff_seconds.max(1))).to_rfc3339();
        let max_attempts = max_attempts.max(1);
        let mut tx = self.pool.begin().await?;
        let attempt_count =
            sqlx::query_scalar::<_, Option<i64>>(DIALECT.notification_attempt_count())
                .bind(notification_id)
                .fetch_optional(&mut *tx)
                .await?
                .flatten();
        let Some(attempt_count) = attempt_count else {
            tx.commit().await?;
            return Ok(NotificationFailureDisposition {
                retried: false,
                failed: false,
            });
        };

        let next_attempt_count = attempt_count + 1;
        if next_attempt_count < max_attempts {
            sqlx::query(DIALECT.mark_notification_retry())
                .bind(next_attempt_count)
                .bind(&retry_at)
                .bind(&now_text)
                .bind(failure_reason)
                .bind(notification_id)
                .execute(&mut *tx)
                .await?;
            tx.commit().await?;
            return Ok(NotificationFailureDisposition {
                retried: true,
                failed: false,
            });
        }

        sqlx::query(DIALECT.mark_notification_final_failed())
            .bind(next_attempt_count)
            .bind(&now_text)
            .bind(failure_reason)
            .bind(notification_id)
            .execute(&mut *tx)
            .await?;
        Self::purge_notification_secret(&mut tx, notification_id, &now_text).await?;
        tx.commit().await?;
        Ok(NotificationFailureDisposition {
            retried: false,
            failed: true,
        })
    }

    async fn list_failed_notifications(
        &self,
        limit: i64,
    ) -> AppResult<Vec<NotificationDeadLetterRecord>> {
        let rows = sqlx::query_as::<_, NotificationDeadLetterRow>(DIALECT.failed_notifications())
            .bind(limit)
            .fetch_all(&self.pool)
            .await?;
        Ok(rows
            .into_iter()
            .map(notification_dead_letter_from_row)
            .collect())
    }

    async fn requeue_failed_notification(&self, notification_id: i64) -> AppResult<bool> {
        let now = Self::now();
        let result = sqlx::query(DIALECT.requeue_failed_notification())
            .bind(&now)
            .bind(notification_id)
            .execute(&self.pool)
            .await?;
        Ok(result.rows_affected() > 0)
    }
}

#[async_trait::async_trait]
impl SystemRepository for SqliteRepository {
    async fn sync_api_catalog(&self, entries: &[ApiCatalogEntry]) -> AppResult<()> {
        let mut tx = self.pool.begin().await?;
        let now = Self::now();
        for entry in entries {
            sqlx::query(DIALECT.system_api_upsert())
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
                sqlx::query(DIALECT.permission_upsert())
                    .bind(&entry.product_code)
                    .bind(&entry.scope)
                    .bind(permission)
                    .bind(&entry.summary)
                    .bind(&now)
                    .execute(&mut *tx)
                    .await?;
            }
        }

        sqlx::query(DIALECT.platform_builtin_role_permissions())
            .execute(&mut *tx)
            .await?;

        sqlx::query(DIALECT.tenant_builtin_role_permissions())
            .execute(&mut *tx)
            .await?;

        tx.commit().await?;
        Ok(())
    }

    async fn sync_system_menus(&self, menus: &[SystemMenuEntry]) -> AppResult<()> {
        let mut tx = self.pool.begin().await?;
        let now = Self::now();
        for menu in menus {
            sqlx::query(DIALECT.system_menu_upsert())
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
        Ok(())
    }

    async fn list_api_catalog(&self) -> AppResult<Vec<ApiCatalogEntry>> {
        let rows = sqlx::query(DIALECT.system_apis_list())
            .fetch_all(&self.pool)
            .await?;
        Ok(rows
            .into_iter()
            .map(|row| ApiCatalogEntry {
                id: row.get("id"),
                method: row.get("method"),
                path: row.get("path"),
                tag: row.get("tag"),
                summary: row.get("summary"),
                access: row.get("access"),
                permission: row.get("permission"),
                scope: row.get("scope"),
                product_code: row.get("product_code"),
            })
            .collect())
    }

    async fn list_system_menus(&self) -> AppResult<Vec<SystemMenuEntry>> {
        let rows = sqlx::query(DIALECT.system_menus_list())
            .fetch_all(&self.pool)
            .await?;
        Ok(rows
            .into_iter()
            .map(|row| SystemMenuEntry {
                code: row.get("code"),
                title: row.get("title"),
                path: row.get("path"),
                permission: row.get("permission"),
                scope: row.get("scope"),
                sort_order: row.get("sort_order"),
            })
            .collect())
    }

    async fn create_operation_record(&self, input: CreateOperationRecord) -> AppResult<()> {
        sqlx::query(DIALECT.create_operation_record())
            .bind(input.actor_user_id)
            .bind(input.method)
            .bind(input.path)
            .bind(input.status)
            .bind(Self::now())
            .execute(&self.pool)
            .await?;
        Ok(())
    }

    async fn list_operation_records(
        &self,
        query: OperationRecordListQuery,
    ) -> AppResult<Vec<OperationRecord>> {
        let rows = sqlx::query(DIALECT.operation_records_list())
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
            .fetch_all(&self.pool)
            .await?;
        Ok(rows
            .into_iter()
            .map(|row| OperationRecord {
                id: row.get("id"),
                actor_user_id: row.get("actor_user_id"),
                method: row.get("method"),
                path: row.get("path"),
                status: row.get("status"),
                created_at: row.get("created_at"),
            })
            .collect())
    }

    async fn summarize_operation_records(
        &self,
        query: OperationRecordSummaryFilter,
    ) -> AppResult<OperationRecordSummary> {
        let counts = sqlx::query_as::<_, OperationRecordSummaryCountsRow>(
            DIALECT.operation_records_summary_counts(),
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
        .fetch_one(&self.pool)
        .await?;

        let by_method = sqlx::query_as::<_, OperationRecordCountBucketRow>(
            DIALECT.operation_records_summary_methods(),
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
        .fetch_all(&self.pool)
        .await?;

        let by_status_class = sqlx::query_as::<_, OperationRecordCountBucketRow>(
            DIALECT.operation_records_summary_status_classes(),
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
        .fetch_all(&self.pool)
        .await?;

        let top_paths = sqlx::query_as::<_, OperationRecordPathBucketRow>(
            DIALECT.operation_records_summary_paths(),
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
        .fetch_all(&self.pool)
        .await?;

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
        let result = sqlx::query(DIALECT.prune_operation_records())
            .bind(cutoff)
            .bind(limit)
            .execute(&self.pool)
            .await?;
        Ok(result.rows_affected().min(i64::MAX as u64) as i64)
    }

    async fn list_system_configs(&self) -> AppResult<Vec<SystemConfigEntry>> {
        let rows = sqlx::query_as::<_, SystemConfigRow>(DIALECT.system_configs_list())
            .fetch_all(&self.pool)
            .await?;
        rows.into_iter().map(config_from_row).collect()
    }

    async fn upsert_system_config(
        &self,
        input: UpsertSystemConfigRecord,
    ) -> AppResult<SystemConfigEntry> {
        let now = Self::now();
        sqlx::query(DIALECT.system_config_upsert())
            .bind(&input.key)
            .bind(&input.value_json)
            .bind(&now)
            .execute(&self.pool)
            .await?;
        Ok(SystemConfigEntry {
            key: input.key,
            value: parse_json_value(&input.value_json)?,
            updated_at: now,
        })
    }

    async fn delete_system_config(&self, key: &str) -> AppResult<bool> {
        let result = sqlx::query(DIALECT.delete_system_config())
            .bind(key)
            .execute(&self.pool)
            .await?;
        Ok(result.rows_affected() > 0)
    }

    async fn list_system_dictionaries(&self) -> AppResult<Vec<SystemDictionaryEntry>> {
        let rows = sqlx::query_as::<_, SystemDictionaryRow>(DIALECT.system_dictionaries_list())
            .fetch_all(&self.pool)
            .await?;
        Ok(rows.into_iter().map(dictionary_from_row).collect())
    }

    async fn upsert_system_dictionary(
        &self,
        input: UpsertSystemDictionaryRecord,
    ) -> AppResult<SystemDictionaryEntry> {
        let now = Self::now();
        sqlx::query(DIALECT.system_dictionary_upsert())
            .bind(&input.code)
            .bind(&input.name)
            .bind(&now)
            .execute(&self.pool)
            .await?;
        let row = sqlx::query_as::<_, SystemDictionaryRow>(DIALECT.system_dictionary_by_code())
            .bind(&input.code)
            .fetch_one(&self.pool)
            .await?;
        Ok(dictionary_from_row(row))
    }

    async fn delete_system_dictionary(&self, code: &str) -> AppResult<bool> {
        let result = sqlx::query(DIALECT.delete_system_dictionary())
            .bind(Self::now())
            .bind(code)
            .execute(&self.pool)
            .await?;
        Ok(result.rows_affected() > 0)
    }

    async fn list_system_parameters(&self) -> AppResult<Vec<SystemParameterEntry>> {
        let rows = sqlx::query_as::<_, SystemParameterRow>(DIALECT.system_parameters_list())
            .fetch_all(&self.pool)
            .await?;
        Ok(rows.into_iter().map(parameter_from_row).collect())
    }

    async fn upsert_system_parameter(
        &self,
        input: UpsertSystemParameterRecord,
    ) -> AppResult<SystemParameterEntry> {
        let now = Self::now();
        sqlx::query(DIALECT.system_parameter_upsert())
            .bind(&input.key)
            .bind(&input.name)
            .bind(&input.value)
            .bind(&now)
            .bind(&now)
            .execute(&self.pool)
            .await?;
        let row = sqlx::query_as::<_, SystemParameterRow>(DIALECT.system_parameter_by_key())
            .bind(&input.key)
            .fetch_one(&self.pool)
            .await?;
        Ok(parameter_from_row(row))
    }

    async fn delete_system_parameter(&self, key: &str) -> AppResult<bool> {
        let now = Self::now();
        let result = sqlx::query(DIALECT.delete_system_parameter())
            .bind(&now)
            .bind(&now)
            .bind(key)
            .execute(&self.pool)
            .await?;
        Ok(result.rows_affected() > 0)
    }

    async fn create_version_package(
        &self,
        input: CreateVersionPackageRecord,
    ) -> AppResult<VersionPackageEntry> {
        let now = Self::now();
        let id = sqlx::query_scalar::<_, i64>(DIALECT.create_version_package())
            .bind(&input.version_name)
            .bind(&input.version_code)
            .bind(&input.manifest_json)
            .bind(&now)
            .fetch_one(&self.pool)
            .await?;
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
        let rows = sqlx::query_as::<_, VersionPackageRow>(DIALECT.version_packages_list())
            .fetch_all(&self.pool)
            .await?;
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
        let rows =
            sqlx::query_as::<_, VersionReleaseEventRow>(DIALECT.version_release_events_list())
                .fetch_all(&self.pool)
                .await?;
        Ok(rows
            .into_iter()
            .map(version_release_event_from_row)
            .collect())
    }

    async fn delete_version_package(&self, id: i64) -> AppResult<bool> {
        let result = sqlx::query(DIALECT.delete_version_package())
            .bind(Self::now())
            .bind(id)
            .execute(&self.pool)
            .await?;
        if result.rows_affected() > 0 {
            return Ok(true);
        }
        let status = sqlx::query_scalar::<_, String>(DIALECT.version_package_status_by_id())
            .bind(id)
            .fetch_optional(&self.pool)
            .await?;
        if status.as_deref() == Some("active") {
            return Err(AppError::Conflict("当前 active 版本包不能删除".into()));
        }
        Ok(false)
    }

    async fn create_media_asset(
        &self,
        input: CreateMediaAssetRecord,
    ) -> AppResult<MediaAssetEntry> {
        let now = Self::now();
        let id = sqlx::query_scalar::<_, i64>(DIALECT.create_media_asset())
            .bind(&input.category)
            .bind(&input.display_name)
            .bind(&input.storage_key)
            .bind(&input.mime_type)
            .bind(input.size_bytes)
            .bind(&now)
            .fetch_one(&self.pool)
            .await?;
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
        let rows = sqlx::query_as::<_, MediaAssetRow>(DIALECT.media_assets_list())
            .fetch_all(&self.pool)
            .await?;
        Ok(rows.into_iter().map(media_asset_from_row).collect())
    }

    async fn delete_media_asset(&self, id: i64) -> AppResult<bool> {
        let result = sqlx::query(DIALECT.delete_media_asset())
            .bind(Self::now())
            .bind(id)
            .execute(&self.pool)
            .await?;
        Ok(result.rows_affected() > 0)
    }

    async fn create_traffic_probe_target(
        &self,
        input: CreateTrafficProbeTargetRecord,
    ) -> AppResult<TrafficProbeTargetEntry> {
        let now = Self::now();
        let id = sqlx::query_scalar::<_, i64>(DIALECT.create_traffic_probe_target())
            .bind(&input.name)
            .bind(&input.url)
            .bind(input.expected_status)
            .bind(&now)
            .fetch_one(&self.pool)
            .await?;
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
        let rows = sqlx::query_as::<_, TrafficProbeTargetRow>(DIALECT.traffic_probe_targets_list())
            .fetch_all(&self.pool)
            .await?;
        Ok(rows
            .into_iter()
            .map(traffic_probe_target_from_row)
            .collect())
    }

    async fn find_traffic_probe_target(
        &self,
        id: i64,
    ) -> AppResult<Option<TrafficProbeTargetEntry>> {
        let row = sqlx::query_as::<_, TrafficProbeTargetRow>(DIALECT.traffic_probe_target_by_id())
            .bind(id)
            .fetch_optional(&self.pool)
            .await?;
        Ok(row.map(traffic_probe_target_from_row))
    }

    async fn delete_traffic_probe_target(&self, id: i64) -> AppResult<bool> {
        let result = sqlx::query(DIALECT.delete_traffic_probe_target())
            .bind(Self::now())
            .bind(id)
            .execute(&self.pool)
            .await?;
        Ok(result.rows_affected() > 0)
    }

    async fn create_traffic_probe_result(
        &self,
        input: CreateTrafficProbeResultRecord,
    ) -> AppResult<TrafficProbeResultEntry> {
        let now = Self::now();
        let mut tx = self.pool.begin().await?;
        let id = sqlx::query_scalar::<_, i64>(DIALECT.create_traffic_probe_result())
            .bind(input.target_id)
            .bind(&input.status)
            .bind(&input.detail_json)
            .bind(&now)
            .fetch_one(&mut *tx)
            .await?;
        sqlx::query(DIALECT.update_traffic_probe_target_status())
            .bind(&input.status)
            .bind(input.target_id)
            .execute(&mut *tx)
            .await?;
        tx.commit().await?;
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
        let rows = if let Some(target_id) = target_id {
            sqlx::query_as::<_, TrafficProbeResultRow>(DIALECT.traffic_probe_results_for_target())
                .bind(target_id)
                .bind(limit)
                .fetch_all(&self.pool)
                .await?
        } else {
            sqlx::query_as::<_, TrafficProbeResultRow>(DIALECT.traffic_probe_results_all())
                .bind(limit)
                .fetch_all(&self.pool)
                .await?
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
        let id = sqlx::query_scalar::<_, i64>(DIALECT.create_traffic_probe_alert())
            .bind(input.target_id)
            .bind(input.result_id)
            .bind(&input.severity)
            .bind(&input.reason)
            .bind(&input.detail_json)
            .bind(&now)
            .fetch_one(&self.pool)
            .await?;
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
        let rows = match (target_id, status) {
            (Some(target_id), Some(status)) => {
                sqlx::query_as::<_, TrafficProbeAlertRow>(
                    DIALECT.traffic_probe_alerts_target_status(),
                )
                .bind(target_id)
                .bind(status)
                .bind(limit)
                .fetch_all(&self.pool)
                .await?
            }
            (Some(target_id), None) => {
                sqlx::query_as::<_, TrafficProbeAlertRow>(DIALECT.traffic_probe_alerts_target())
                    .bind(target_id)
                    .bind(limit)
                    .fetch_all(&self.pool)
                    .await?
            }
            (None, Some(status)) => {
                sqlx::query_as::<_, TrafficProbeAlertRow>(DIALECT.traffic_probe_alerts_status())
                    .bind(status)
                    .bind(limit)
                    .fetch_all(&self.pool)
                    .await?
            }
            (None, None) => {
                sqlx::query_as::<_, TrafficProbeAlertRow>(DIALECT.traffic_probe_alerts_all())
                    .bind(limit)
                    .fetch_all(&self.pool)
                    .await?
            }
        };
        rows.into_iter().map(traffic_probe_alert_from_row).collect()
    }

    async fn acknowledge_traffic_probe_alert(&self, id: i64) -> AppResult<bool> {
        let result = sqlx::query(DIALECT.acknowledge_traffic_probe_alert())
            .bind(Self::now())
            .bind(id)
            .execute(&self.pool)
            .await?;
        Ok(result.rows_affected() > 0)
    }

    async fn resolve_traffic_probe_alert(&self, id: i64) -> AppResult<bool> {
        let result = sqlx::query(DIALECT.resolve_traffic_probe_alert())
            .bind(Self::now())
            .bind(id)
            .execute(&self.pool)
            .await?;
        Ok(result.rows_affected() > 0)
    }

    async fn resolve_traffic_probe_alerts_for_target(&self, target_id: i64) -> AppResult<i64> {
        let result = sqlx::query(DIALECT.resolve_traffic_probe_alerts_for_target())
            .bind(Self::now())
            .bind(target_id)
            .execute(&self.pool)
            .await?;
        Ok(result.rows_affected().min(i64::MAX as u64) as i64)
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
        .map_err(|err| AppError::Internal(format!("系统配置 JSON 解析失败：{err}")))
}

fn split_csv(value: Option<String>) -> Vec<String> {
    value
        .unwrap_or_default()
        .split(',')
        .map(str::trim)
        .filter(|item| !item.is_empty())
        .map(ToOwned::to_owned)
        .collect()
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

#[derive(sqlx::FromRow)]
struct NotificationOutboxRow {
    id: i64,
    org_id: Option<i64>,
    user_id: Option<i64>,
    product_code: String,
    channel: String,
    template_code: String,
    recipient: String,
    related_kind: String,
    related_id: i64,
    payload_json: String,
    status: String,
    available_at: String,
    created_at: String,
    locked_at: Option<String>,
    delivered_at: Option<String>,
    failed_at: Option<String>,
    failure_reason: Option<String>,
    attempt_count: i64,
    delivery_secret_ciphertext: Option<String>,
}

#[derive(sqlx::FromRow)]
struct NotificationDeadLetterRow {
    id: i64,
    org_id: Option<i64>,
    user_id: Option<i64>,
    product_code: String,
    channel: String,
    template_code: String,
    recipient: String,
    related_kind: String,
    related_id: i64,
    status: String,
    created_at: String,
    failed_at: Option<String>,
    failure_reason: Option<String>,
    attempt_count: i64,
    delivery_secret_status: Option<String>,
    delivery_secret_ciphertext: Option<String>,
    delivery_secret_purged_at: Option<String>,
}

fn notification_outbox_from_row(row: NotificationOutboxRow) -> NotificationOutboxItem {
    NotificationOutboxItem {
        id: row.id,
        org_id: row.org_id,
        user_id: row.user_id,
        product_code: row.product_code,
        channel: row.channel,
        template_code: row.template_code,
        recipient: row.recipient,
        related_kind: row.related_kind,
        related_id: row.related_id,
        payload_json: row.payload_json,
        status: row.status,
        available_at: row.available_at,
        created_at: row.created_at,
        locked_at: row.locked_at,
        delivered_at: row.delivered_at,
        failed_at: row.failed_at,
        failure_reason: row.failure_reason,
        attempt_count: row.attempt_count,
        delivery_secret_ciphertext: row.delivery_secret_ciphertext,
    }
}

fn notification_dead_letter_from_row(
    row: NotificationDeadLetterRow,
) -> NotificationDeadLetterRecord {
    NotificationDeadLetterRecord {
        id: row.id,
        org_id: row.org_id,
        user_id: row.user_id,
        product_code: row.product_code,
        channel: row.channel,
        template_code: row.template_code,
        recipient: row.recipient,
        related_kind: row.related_kind,
        related_id: row.related_id,
        status: row.status,
        created_at: row.created_at,
        failed_at: row.failed_at,
        failure_reason: row.failure_reason,
        attempt_count: row.attempt_count,
        delivery_secret_status: row.delivery_secret_status,
        delivery_secret_ciphertext: row.delivery_secret_ciphertext,
        delivery_secret_purged_at: row.delivery_secret_purged_at,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::Settings;
    use crate::infrastructure::database::{self, DatabaseConnection};
    use crate::repository::{CreateInitialAdminRecord, IamRepository, NotificationRepository};
    use uuid::Uuid;

    #[tokio::test]
    async fn notification_failure_retries_before_final_failure() {
        let mut settings = Settings::default();
        settings.database.url = format!("sqlite://target/test-dbs/{}.sqlite", Uuid::new_v4());
        let connection = database::connect(&settings)
            .await
            .expect("connect sqlite test database");
        let pool = match connection {
            DatabaseConnection::Sqlite(pool) => pool,
            _ => panic!("sqlite repository test must use SQLite connection"),
        };
        let repo = SqliteRepository::new(pool.clone());
        let now = Utc::now().to_rfc3339();
        let outbox_id = sqlx::query_scalar::<_, i64>(DIALECT.create_notification_outbox())
            .bind(None::<i64>)
            .bind(None::<i64>)
            .bind("console")
            .bind("email")
            .bind("iam.password_reset.requested")
            .bind("user@example.com")
            .bind("iam_password_reset")
            .bind(1_i64)
            .bind("{}")
            .bind(&now)
            .bind(&now)
            .fetch_one(&pool)
            .await
            .expect("insert outbox");
        sqlx::query(DIALECT.create_notification_delivery_secret())
            .bind(outbox_id)
            .bind("encrypted-token-placeholder")
            .bind(&now)
            .execute(&pool)
            .await
            .expect("insert delivery secret");

        let claimed = repo
            .claim_due_notifications(10, 300)
            .await
            .expect("claim notification");
        assert_eq!(claimed.len(), 1);
        assert_eq!(claimed[0].attempt_count, 0);

        let retry = repo
            .mark_notification_failed(outbox_id, "smtp timeout", 60, 2)
            .await
            .expect("mark retry");
        assert!(retry.retried);
        assert!(!retry.failed);
        let retry_row = sqlx::query(
            "select o.status, o.attempt_count, s.secret_ciphertext
             from iam_notification_outbox o
             left join iam_notification_delivery_secrets s on s.outbox_id = o.id
             where o.id = ?",
        )
        .bind(outbox_id)
        .fetch_one(&pool)
        .await
        .expect("retry row");
        assert_eq!(retry_row.get::<String, _>("status"), "pending");
        assert_eq!(retry_row.get::<i64, _>("attempt_count"), 1);
        assert_eq!(
            retry_row.get::<Option<String>, _>("secret_ciphertext"),
            Some("encrypted-token-placeholder".into())
        );

        let final_failure = repo
            .mark_notification_failed(outbox_id, "smtp timeout again", 60, 2)
            .await
            .expect("mark final failure");
        assert!(!final_failure.retried);
        assert!(final_failure.failed);
        let failed_row = sqlx::query(
            "select o.status, o.attempt_count, s.secret_ciphertext, s.status as secret_status
             from iam_notification_outbox o
             left join iam_notification_delivery_secrets s on s.outbox_id = o.id
             where o.id = ?",
        )
        .bind(outbox_id)
        .fetch_one(&pool)
        .await
        .expect("failed row");
        assert_eq!(failed_row.get::<String, _>("status"), "failed");
        assert_eq!(failed_row.get::<i64, _>("attempt_count"), 2);
        assert_eq!(
            failed_row.get::<Option<String>, _>("secret_ciphertext"),
            None
        );
        assert_eq!(failed_row.get::<String, _>("secret_status"), "purged");

        let failed_notifications = repo
            .list_failed_notifications(10)
            .await
            .expect("list failed notifications");
        let failed_notification = failed_notifications
            .iter()
            .find(|item| item.id == outbox_id)
            .expect("failed notification is listed");
        assert_eq!(failed_notification.recipient, "user@example.com");
        assert_eq!(
            failed_notification.delivery_secret_status.as_deref(),
            Some("purged")
        );
        assert!(failed_notification.delivery_secret_ciphertext.is_none());

        let purged_requeue = repo
            .requeue_failed_notification(outbox_id)
            .await
            .expect("purged secret must not requeue");
        assert!(!purged_requeue);

        let replay_id = sqlx::query_scalar::<_, i64>(DIALECT.create_notification_outbox())
            .bind(None::<i64>)
            .bind(None::<i64>)
            .bind("console")
            .bind("email")
            .bind("iam.email_verification.requested")
            .bind("replay@example.com")
            .bind("iam_email_verification")
            .bind(2_i64)
            .bind("{}")
            .bind(&now)
            .bind(&now)
            .fetch_one(&pool)
            .await
            .expect("insert replay outbox");
        sqlx::query(DIALECT.create_notification_delivery_secret())
            .bind(replay_id)
            .bind("encrypted-replay-token-placeholder")
            .bind(&now)
            .execute(&pool)
            .await
            .expect("insert replay delivery secret");
        sqlx::query(
            "update iam_notification_outbox
             set status = 'failed',
                 attempt_count = 2,
                 failed_at = ?,
                 failure_reason = ?
             where id = ?",
        )
        .bind(&now)
        .bind("operator requested replay")
        .bind(replay_id)
        .execute(&pool)
        .await
        .expect("mark replay failed");

        let pending_secret_requeue = repo
            .requeue_failed_notification(replay_id)
            .await
            .expect("pending secret can requeue");
        assert!(pending_secret_requeue);
        let replay_row = sqlx::query(
            "select o.status, o.attempt_count, o.failed_at, o.failure_reason, s.secret_ciphertext
             from iam_notification_outbox o
             left join iam_notification_delivery_secrets s on s.outbox_id = o.id
             where o.id = ?",
        )
        .bind(replay_id)
        .fetch_one(&pool)
        .await
        .expect("replay row");
        assert_eq!(replay_row.get::<String, _>("status"), "pending");
        assert_eq!(replay_row.get::<i64, _>("attempt_count"), 0);
        assert_eq!(replay_row.get::<Option<String>, _>("failed_at"), None);
        assert_eq!(replay_row.get::<Option<String>, _>("failure_reason"), None);
        assert_eq!(
            replay_row.get::<Option<String>, _>("secret_ciphertext"),
            Some("encrypted-replay-token-placeholder".into())
        );
    }

    #[tokio::test]
    async fn pending_flows_roll_back_when_notification_outbox_fails() {
        let mut settings = Settings::default();
        settings.database.url = format!("sqlite://target/test-dbs/{}.sqlite", Uuid::new_v4());
        let connection = database::connect(&settings)
            .await
            .expect("connect sqlite test database");
        let pool = match connection {
            DatabaseConnection::Sqlite(pool) => pool,
            _ => panic!("sqlite repository test must use SQLite connection"),
        };
        let repo = SqliteRepository::new(pool.clone());
        let (owner, org) = repo
            .create_initial_admin(CreateInitialAdminRecord {
                email: "owner@example.com".into(),
                password_hash: "owner-password-hash".into(),
                display_name: "Owner".into(),
                organization_code: "owner-org".into(),
                organization_name: "Owner Org".into(),
                product_code: "console".into(),
            })
            .await
            .expect("create owner");

        // 用 trigger 模拟通知基础设施写入失败，确保 pending 主数据不会半提交。
        sqlx::query(
            "create trigger fail_pending_notification_outbox
             before insert on iam_notification_outbox
             begin
               select raise(abort, 'forced notification outbox failure');
             end",
        )
        .execute(&pool)
        .await
        .expect("create failing outbox trigger");

        let now = Utc::now().to_rfc3339();
        repo.create_registration_with_email_verification(
            CreateRegistrationRecord {
                email: "rollback-register@example.com".into(),
                password_hash: "pending-password-hash".into(),
                display_name: "Rollback Register".into(),
                organization_code: "rollback-register-org".into(),
                organization_name: "Rollback Register Org".into(),
                product_code: "console".into(),
                email_verification_token_hash: "registration-token-hash".into(),
                email_verification_expires_at: now.clone(),
            },
            notification(
                "rollback-register@example.com",
                "iam_email_verification",
                &now,
            ),
        )
        .await
        .expect_err("registration must fail when notification outbox fails");
        assert_eq!(
            count_rows(
                &pool,
                "select count(*) from iam_users where email = 'rollback-register@example.com'",
            )
            .await,
            0
        );
        assert_eq!(
            count_rows(
                &pool,
                "select count(*) from iam_organizations where code = 'rollback-register-org'",
            )
            .await,
            0
        );
        assert_eq!(
            count_rows(
                &pool,
                "select count(*) from iam_email_verifications where email = 'rollback-register@example.com'",
            )
            .await,
            0
        );

        repo.create_invitation_with_notification(
            CreateInvitationRecord {
                org_id: org.id,
                email: "rollback-invite@example.com".into(),
                role_code: "owner".into(),
                token_hash: "invitation-token-hash".into(),
                expires_at: now.clone(),
            },
            notification("rollback-invite@example.com", "iam_invitation", &now),
        )
        .await
        .expect_err("invitation must fail when notification outbox fails");
        assert_eq!(
            count_rows(
                &pool,
                "select count(*) from iam_invitations where email = 'rollback-invite@example.com'",
            )
            .await,
            0
        );

        repo.create_password_reset_with_notification(
            CreatePasswordResetRecord {
                user_id: owner.id,
                token_hash: "password-reset-token-hash".into(),
                expires_at: now.clone(),
            },
            notification("owner@example.com", "iam_password_reset", &now),
        )
        .await
        .expect_err("password reset must fail when notification outbox fails");
        assert_eq!(
            count_rows(
                &pool,
                "select count(*) from iam_password_resets where token_hash = 'password-reset-token-hash'",
            )
            .await,
            0
        );

        repo.create_email_verification_with_notification(
            CreateEmailVerificationRecord {
                user_id: owner.id,
                email: "owner@example.com".into(),
                token_hash: "email-verification-token-hash".into(),
                expires_at: now.clone(),
            },
            notification("owner@example.com", "iam_email_verification", &now),
        )
        .await
        .expect_err("email verification must fail when notification outbox fails");
        assert_eq!(
            count_rows(
                &pool,
                "select count(*) from iam_email_verifications where token_hash = 'email-verification-token-hash'",
            )
            .await,
            0
        );
        assert_eq!(
            count_rows(&pool, "select count(*) from iam_notification_outbox").await,
            0
        );
        assert_eq!(
            count_rows(
                &pool,
                "select count(*) from iam_notification_delivery_secrets"
            )
            .await,
            0
        );
    }

    fn notification(
        recipient: &str,
        related_kind: &str,
        available_at: &str,
    ) -> CreateNotificationOutboxRecord {
        CreateNotificationOutboxRecord {
            org_id: None,
            user_id: None,
            product_code: "console".into(),
            channel: "email".into(),
            template_code: "iam.test.pending".into(),
            recipient: recipient.into(),
            related_kind: related_kind.into(),
            payload_json: "{}".into(),
            available_at: available_at.into(),
            delivery_secret_ciphertext: Some("encrypted-test-token".into()),
        }
    }

    async fn count_rows(pool: &Pool<Sqlite>, query: &str) -> i64 {
        sqlx::query_scalar(query)
            .fetch_one(pool)
            .await
            .expect("count rows")
    }
}
