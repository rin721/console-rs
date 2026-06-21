use chrono::Utc;
use sqlx::any::{AnyPoolOptions, install_default_drivers};
use sqlx::{Any, Pool, Row, Transaction};

use crate::app::{AppError, AppResult};
use crate::domain::iam::{
    APITokenAuthRecord, APITokenSummary, EmailVerificationRecord, InvitationAcceptRecord,
    InvitationSummary, MfaFactorSummary, MfaRecoveryCodeSummary, Organization, OrganizationSummary,
    OrganizationUserSummary, PasswordResetRecord, PermissionSummary, RoleSummary, SessionRecord,
    StoredMfaFactor, StoredUser, User,
};
use crate::infrastructure::sql_templates::SqlDialect;
use crate::repository::{
    AcceptInvitationRecord, CreateAPITokenRecord, CreateAuditLogRecord,
    CreateEmailVerificationRecord, CreateInitialAdminRecord, CreateInvitationRecord,
    CreateMfaFactorRecord, CreateMfaRecoveryCodeRecord, CreateNotificationOutboxRecord,
    CreatePasswordResetRecord, CreateRegistrationRecord, CreateRoleRecord, CreateSessionRecord,
    IamRepository, UpdateOrgUserRecord, UpdateRoleRecord,
};

#[derive(Clone)]
pub struct ExternalIamRepository {
    pool: Pool<Any>,
    dialect: SqlDialect,
}

impl ExternalIamRepository {
    pub fn new(dialect: SqlDialect, pool: Pool<Any>) -> Self {
        Self { pool, dialect }
    }

    pub async fn connect(
        dialect: SqlDialect,
        url: &str,
        max_connections: u32,
    ) -> anyhow::Result<Self> {
        install_default_drivers();
        let pool = AnyPoolOptions::new()
            .max_connections(max_connections)
            .connect(url)
            .await?;
        Ok(Self::new(dialect, pool))
    }

    fn now() -> String {
        Utc::now().to_rfc3339()
    }

    async fn mysql_last_insert_id(tx: &mut Transaction<'_, Any>) -> AppResult<i64> {
        let id = sqlx::query_scalar::<_, i64>("select cast(last_insert_id() as signed)")
            .fetch_one(&mut **tx)
            .await?;
        Ok(id)
    }

    async fn create_tenant_organization(
        &self,
        tx: &mut Transaction<'_, Any>,
        code: &str,
        name: &str,
        now: &str,
    ) -> AppResult<i64> {
        if self.dialect == SqlDialect::Mysql {
            sqlx::query(self.dialect.create_tenant_organization())
                .bind(code)
                .bind(name)
                .bind(now)
                .bind(now)
                .execute(&mut **tx)
                .await?;
            Self::mysql_last_insert_id(tx).await
        } else {
            Ok(
                sqlx::query_scalar::<_, i64>(self.dialect.create_tenant_organization())
                    .bind(code)
                    .bind(name)
                    .bind(now)
                    .bind(now)
                    .fetch_one(&mut **tx)
                    .await?,
            )
        }
    }

    async fn create_active_user(
        &self,
        tx: &mut Transaction<'_, Any>,
        email: &str,
        display_name: &str,
        password_hash: &str,
        now: &str,
    ) -> AppResult<i64> {
        if self.dialect == SqlDialect::Mysql {
            sqlx::query(self.dialect.create_active_user())
                .bind(email)
                .bind(display_name)
                .bind(password_hash)
                .bind(now)
                .bind(now)
                .bind(now)
                .execute(&mut **tx)
                .await?;
            Self::mysql_last_insert_id(tx).await
        } else {
            Ok(
                sqlx::query_scalar::<_, i64>(self.dialect.create_active_user())
                    .bind(email)
                    .bind(display_name)
                    .bind(password_hash)
                    .bind(now)
                    .bind(now)
                    .bind(now)
                    .fetch_one(&mut **tx)
                    .await?,
            )
        }
    }

    async fn create_pending_verification_user(
        &self,
        tx: &mut Transaction<'_, Any>,
        email: &str,
        display_name: &str,
        password_hash: &str,
        now: &str,
    ) -> AppResult<i64> {
        if self.dialect == SqlDialect::Mysql {
            sqlx::query(self.dialect.create_pending_verification_user())
                .bind(email)
                .bind(display_name)
                .bind(password_hash)
                .bind(now)
                .bind(now)
                .execute(&mut **tx)
                .await?;
            Self::mysql_last_insert_id(tx).await
        } else {
            Ok(
                sqlx::query_scalar::<_, i64>(self.dialect.create_pending_verification_user())
                    .bind(email)
                    .bind(display_name)
                    .bind(password_hash)
                    .bind(now)
                    .bind(now)
                    .fetch_one(&mut **tx)
                    .await?,
            )
        }
    }

    async fn create_tenant_owner_role(
        &self,
        tx: &mut Transaction<'_, Any>,
        org_id: i64,
        now: &str,
    ) -> AppResult<i64> {
        if self.dialect == SqlDialect::Mysql {
            sqlx::query(self.dialect.create_tenant_owner_role())
                .bind(org_id)
                .bind(now)
                .bind(now)
                .execute(&mut **tx)
                .await?;
            Self::mysql_last_insert_id(tx).await
        } else {
            Ok(
                sqlx::query_scalar::<_, i64>(self.dialect.create_tenant_owner_role())
                    .bind(org_id)
                    .bind(now)
                    .bind(now)
                    .fetch_one(&mut **tx)
                    .await?,
            )
        }
    }

    async fn create_platform_owner_role(
        &self,
        tx: &mut Transaction<'_, Any>,
        now: &str,
    ) -> AppResult<i64> {
        if self.dialect == SqlDialect::Mysql {
            sqlx::query(self.dialect.create_platform_owner_role())
                .bind(now)
                .bind(now)
                .execute(&mut **tx)
                .await?;
            Self::mysql_last_insert_id(tx).await
        } else {
            Ok(
                sqlx::query_scalar::<_, i64>(self.dialect.create_platform_owner_role())
                    .bind(now)
                    .bind(now)
                    .fetch_one(&mut **tx)
                    .await?,
            )
        }
    }

    async fn create_org_role_id(
        &self,
        tx: &mut Transaction<'_, Any>,
        org_id: i64,
        code: &str,
        name: &str,
        now: &str,
    ) -> AppResult<i64> {
        if self.dialect == SqlDialect::Mysql {
            sqlx::query(self.dialect.create_org_role())
                .bind(org_id)
                .bind(code)
                .bind(name)
                .bind(now)
                .bind(now)
                .execute(&mut **tx)
                .await?;
            Self::mysql_last_insert_id(tx).await
        } else {
            Ok(sqlx::query_scalar::<_, i64>(self.dialect.create_org_role())
                .bind(org_id)
                .bind(code)
                .bind(name)
                .bind(now)
                .bind(now)
                .fetch_one(&mut **tx)
                .await?)
        }
    }

    async fn insert_notification_outbox(
        &self,
        tx: &mut Transaction<'_, Any>,
        input: &CreateNotificationOutboxRecord,
        related_id: i64,
        created_at: &str,
    ) -> AppResult<()> {
        let outbox_id = if self.dialect == SqlDialect::Mysql {
            sqlx::query(self.dialect.create_notification_outbox())
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
                .execute(&mut **tx)
                .await?;
            Self::mysql_last_insert_id(tx).await?
        } else {
            sqlx::query_scalar::<_, i64>(self.dialect.create_notification_outbox())
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
                .fetch_one(&mut **tx)
                .await?
        };

        if let Some(secret_ciphertext) = &input.delivery_secret_ciphertext {
            sqlx::query(self.dialect.create_notification_delivery_secret())
                .bind(outbox_id)
                .bind(secret_ciphertext)
                .bind(created_at)
                .execute(&mut **tx)
                .await?;
        }
        Ok(())
    }

    async fn insert_mfa_recovery_codes(
        &self,
        tx: &mut Transaction<'_, Any>,
        user_id: i64,
        records: &[CreateMfaRecoveryCodeRecord],
        created_at: &str,
    ) -> AppResult<Vec<MfaRecoveryCodeSummary>> {
        let mut summaries = Vec::with_capacity(records.len());
        for record in records {
            let id = if self.dialect == SqlDialect::Mysql {
                sqlx::query(self.dialect.create_mfa_recovery_code())
                    .bind(user_id)
                    .bind(&record.code_hash)
                    .bind(&record.code_prefix)
                    .bind(created_at)
                    .execute(&mut **tx)
                    .await?;
                Self::mysql_last_insert_id(tx).await?
            } else {
                sqlx::query_scalar::<_, i64>(self.dialect.create_mfa_recovery_code())
                    .bind(user_id)
                    .bind(&record.code_hash)
                    .bind(&record.code_prefix)
                    .bind(created_at)
                    .fetch_one(&mut **tx)
                    .await?
            };
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
        let rows = sqlx::query(self.dialect.role_permissions_for_role())
            .bind(role_id)
            .fetch_all(&self.pool)
            .await?;
        Ok(rows.into_iter().map(|row| row.get("code")).collect())
    }

    async fn bind_tenant_role_permissions(
        &self,
        tx: &mut Transaction<'_, Any>,
        role_id: i64,
        product_code: &str,
        permission_codes: &[String],
    ) -> AppResult<()> {
        let mut permission_ids = Vec::with_capacity(permission_codes.len());
        for code in permission_codes {
            let permission_id =
                sqlx::query_scalar::<_, i64>(self.dialect.tenant_permission_id_by_code())
                    .bind(product_code)
                    .bind(code)
                    .fetch_optional(&mut **tx)
                    .await?;
            let Some(permission_id) = permission_id else {
                return Err(AppError::Validation(format!(
                    "租户角色不能绑定不存在或平台级权限：{code}"
                )));
            };
            permission_ids.push(permission_id);
        }

        sqlx::query(self.dialect.delete_role_permissions())
            .bind(role_id)
            .execute(&mut **tx)
            .await?;
        for permission_id in permission_ids {
            sqlx::query(self.dialect.role_permission_values())
                .bind(role_id)
                .bind(permission_id)
                .execute(&mut **tx)
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
impl IamRepository for ExternalIamRepository {
    async fn has_any_user(&self) -> AppResult<bool> {
        let count = sqlx::query_scalar::<_, i64>(self.dialect.users_count())
            .fetch_one(&self.pool)
            .await?;
        Ok(count > 0)
    }

    async fn create_initial_admin(
        &self,
        input: CreateInitialAdminRecord,
    ) -> AppResult<(User, Organization)> {
        let mut tx = self.pool.begin().await?;
        let count = sqlx::query_scalar::<_, i64>(self.dialect.users_count())
            .fetch_one(&mut *tx)
            .await?;
        if count > 0 {
            return Err(AppError::Conflict("平台已存在初始化管理员".into()));
        }

        let now = Self::now();
        let org_id = self
            .create_tenant_organization(
                &mut tx,
                &input.organization_code,
                &input.organization_name,
                &now,
            )
            .await?;
        let user_id = self
            .create_active_user(
                &mut tx,
                &input.email,
                &input.display_name,
                &input.password_hash,
                &now,
            )
            .await?;
        let tenant_role_id = self.create_tenant_owner_role(&mut tx, org_id, &now).await?;
        let platform_role_id = self.create_platform_owner_role(&mut tx, &now).await?;

        sqlx::query(self.dialect.create_active_membership())
            .bind(org_id)
            .bind(user_id)
            .bind("owner")
            .bind(&now)
            .bind(&now)
            .execute(&mut *tx)
            .await?;
        sqlx::query(self.dialect.create_active_membership())
            .bind(None::<i64>)
            .bind(user_id)
            .bind("platform_owner")
            .bind(&now)
            .bind(&now)
            .execute(&mut *tx)
            .await?;
        sqlx::query(self.dialect.role_permissions_for_tenant_scopes())
            .bind(tenant_role_id)
            .bind(&input.product_code)
            .execute(&mut *tx)
            .await?;
        sqlx::query(self.dialect.role_permissions_for_platform_scope())
            .bind(platform_role_id)
            .bind(&input.product_code)
            .execute(&mut *tx)
            .await?;
        sqlx::query(self.dialect.create_audit_log())
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
        let existing_user = sqlx::query_scalar::<_, i64>(self.dialect.user_email_count())
            .bind(&input.email)
            .fetch_one(&mut *tx)
            .await?;
        if existing_user > 0 {
            return Err(AppError::Conflict("注册邮箱已存在".into()));
        }
        let existing_org = sqlx::query_scalar::<_, i64>(self.dialect.organization_code_count())
            .bind(&input.organization_code)
            .fetch_one(&mut *tx)
            .await?;
        if existing_org > 0 {
            return Err(AppError::Conflict("组织编码已存在".into()));
        }

        let now = Self::now();
        let org_id = self
            .create_tenant_organization(
                &mut tx,
                &input.organization_code,
                &input.organization_name,
                &now,
            )
            .await?;
        let user_id = self
            .create_pending_verification_user(
                &mut tx,
                &input.email,
                &input.display_name,
                &input.password_hash,
                &now,
            )
            .await?;
        let owner_role_id = self.create_tenant_owner_role(&mut tx, org_id, &now).await?;
        sqlx::query(self.dialect.create_active_membership())
            .bind(org_id)
            .bind(user_id)
            .bind("owner")
            .bind(&now)
            .bind(&now)
            .execute(&mut *tx)
            .await?;
        sqlx::query(self.dialect.role_permissions_for_tenant_scopes())
            .bind(owner_role_id)
            .bind(&input.product_code)
            .execute(&mut *tx)
            .await?;

        let verification_id = if self.dialect == SqlDialect::Mysql {
            sqlx::query(self.dialect.create_email_verification())
                .bind(user_id)
                .bind(&input.email)
                .bind(&input.email_verification_token_hash)
                .bind(&input.email_verification_expires_at)
                .bind(&now)
                .execute(&mut *tx)
                .await?;
            Self::mysql_last_insert_id(&mut tx).await?
        } else {
            sqlx::query_scalar::<_, i64>(self.dialect.create_email_verification())
                .bind(user_id)
                .bind(&input.email)
                .bind(&input.email_verification_token_hash)
                .bind(&input.email_verification_expires_at)
                .bind(&now)
                .fetch_one(&mut *tx)
                .await?
        };
        let notification = CreateNotificationOutboxRecord {
            org_id: Some(org_id),
            user_id: Some(user_id),
            ..notification
        };
        self.insert_notification_outbox(&mut tx, &notification, verification_id, &now)
            .await?;
        sqlx::query(self.dialect.create_audit_log())
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
        let row = sqlx::query(self.dialect.user_by_identifier())
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
        let row = sqlx::query(self.dialect.primary_organization_for_user())
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
        let rows = sqlx::query(self.dialect.organizations_list())
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
        let rows = sqlx::query(self.dialect.org_users_list())
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
        let user_row = sqlx::query(self.dialect.org_user_membership_context())
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
            let role_exists =
                sqlx::query_scalar::<_, i64>(self.dialect.tenant_org_role_code_count())
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
                sqlx::query_scalar::<_, i64>(self.dialect.org_active_owner_count_except_user())
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

        sqlx::query(self.dialect.update_org_user_profile_status())
            .bind(&input.display_name)
            .bind(&input.status)
            .bind(&now)
            .bind(input.user_id)
            .execute(&mut *tx)
            .await?;
        sqlx::query(self.dialect.delete_org_user_memberships())
            .bind(input.org_id)
            .bind(input.user_id)
            .execute(&mut *tx)
            .await?;
        for role_code in &input.role_codes {
            sqlx::query(self.dialect.create_active_membership())
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
        let rows = sqlx::query(self.dialect.org_roles_list())
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
        let existing_count = sqlx::query_scalar::<_, i64>(self.dialect.org_role_code_count())
            .bind(input.org_id)
            .bind(&input.code)
            .fetch_one(&mut *tx)
            .await?;
        if existing_count > 0 {
            return Err(AppError::Conflict("组织角色编码已存在".into()));
        }

        let role_id = self
            .create_org_role_id(&mut tx, input.org_id, &input.code, &input.name, &now)
            .await?;
        self.bind_tenant_role_permissions(
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
        let row = sqlx::query(self.dialect.tenant_org_role_by_id())
            .bind(input.role_id)
            .bind(input.org_id)
            .fetch_optional(&mut *tx)
            .await?
            .ok_or_else(|| AppError::NotFound("组织角色不存在".into()))?;
        if row.get::<i64, _>("system_builtin") == 1 {
            return Err(AppError::Conflict("内置角色不能修改".into()));
        }
        sqlx::query(self.dialect.update_org_role_name())
            .bind(&input.name)
            .bind(&now)
            .bind(input.role_id)
            .execute(&mut *tx)
            .await?;
        self.bind_tenant_role_permissions(
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
        let row = sqlx::query(self.dialect.tenant_org_role_by_id())
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
        let member_count =
            sqlx::query_scalar::<_, i64>(self.dialect.org_role_active_member_count())
                .bind(org_id)
                .bind(&role_code)
                .fetch_one(&mut *tx)
                .await?;
        if member_count > 0 {
            return Err(AppError::Conflict("角色仍被组织成员使用，不能删除".into()));
        }
        let pending_invitation_count =
            sqlx::query_scalar::<_, i64>(self.dialect.org_role_pending_invitation_count())
                .bind(org_id)
                .bind(&role_code)
                .fetch_one(&mut *tx)
                .await?;
        if pending_invitation_count > 0 {
            return Err(AppError::Conflict(
                "角色仍被待处理邀请使用，不能删除".into(),
            ));
        }
        sqlx::query(self.dialect.delete_org_role())
            .bind(role_id)
            .bind(org_id)
            .execute(&mut *tx)
            .await?;
        tx.commit().await?;
        Ok(true)
    }

    async fn list_permissions(&self, product_code: &str) -> AppResult<Vec<PermissionSummary>> {
        let rows = sqlx::query(self.dialect.permissions_list())
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
        sqlx::query(self.dialect.create_session())
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
        let row = sqlx::query_as::<_, SessionRecordRow>(self.dialect.session_by_token_hash())
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
        let row = sqlx::query_as::<_, SessionRecordRow>(self.dialect.session_by_refresh_hash())
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
        let result = sqlx::query(self.dialect.rotate_session_tokens())
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
        sqlx::query(self.dialect.revoke_session_by_token_hash())
            .bind(&now)
            .bind(&now)
            .bind(token_hash)
            .execute(&self.pool)
            .await?;
        Ok(())
    }

    async fn revoke_session_by_refresh_hash(&self, refresh_hash: &str) -> AppResult<()> {
        let now = Self::now();
        sqlx::query(self.dialect.revoke_session_by_refresh_hash())
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
        let rows = sqlx::query(self.dialect.permissions_for_user())
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
        let mut tx = self.pool.begin().await?;
        let id = if self.dialect == SqlDialect::Mysql {
            sqlx::query(self.dialect.create_api_token())
                .bind(input.org_id)
                .bind(input.user_id)
                .bind(input.token_hash)
                .bind(&input.token_prefix)
                .bind(&input.expires_at)
                .bind(&now)
                .execute(&mut *tx)
                .await?;
            Self::mysql_last_insert_id(&mut tx).await?
        } else {
            sqlx::query_scalar::<_, i64>(self.dialect.create_api_token())
                .bind(input.org_id)
                .bind(input.user_id)
                .bind(input.token_hash)
                .bind(&input.token_prefix)
                .bind(&input.expires_at)
                .bind(&now)
                .fetch_one(&mut *tx)
                .await?
        };
        tx.commit().await?;
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
        let row = sqlx::query_as::<_, APITokenAuthRow>(self.dialect.api_token_by_hash())
            .bind(token_hash)
            .bind(1_i64)
            .fetch_optional(&self.pool)
            .await?;
        Ok(row.map(api_token_auth_from_row))
    }

    async fn list_api_tokens(&self, org_id: i64) -> AppResult<Vec<APITokenSummary>> {
        let rows = sqlx::query_as::<_, APITokenSummaryRow>(self.dialect.api_tokens_for_org())
            .bind(org_id)
            .fetch_all(&self.pool)
            .await?;
        Ok(rows.into_iter().map(api_token_from_row).collect())
    }

    async fn revoke_api_token(&self, org_id: i64, token_id: i64) -> AppResult<bool> {
        let now = Self::now();
        let result = sqlx::query(self.dialect.revoke_api_token())
            .bind(now)
            .bind(org_id)
            .bind(token_id)
            .execute(&self.pool)
            .await?;
        Ok(result.rows_affected() > 0)
    }

    async fn org_role_exists(&self, org_id: i64, role_code: &str) -> AppResult<bool> {
        let count = sqlx::query_scalar::<_, i64>(self.dialect.org_role_code_count())
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
        let invitation_id = if self.dialect == SqlDialect::Mysql {
            sqlx::query(self.dialect.create_invitation())
                .bind(input.org_id)
                .bind(&input.email)
                .bind(&input.role_code)
                .bind(input.token_hash)
                .bind(&input.expires_at)
                .bind(&now)
                .execute(&mut *tx)
                .await?;
            Self::mysql_last_insert_id(&mut tx).await?
        } else {
            sqlx::query_scalar::<_, i64>(self.dialect.create_invitation())
                .bind(input.org_id)
                .bind(&input.email)
                .bind(&input.role_code)
                .bind(input.token_hash)
                .bind(&input.expires_at)
                .bind(&now)
                .fetch_one(&mut *tx)
                .await?
        };
        self.insert_notification_outbox(&mut tx, &notification, invitation_id, &now)
            .await?;
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
        let rows = sqlx::query_as::<_, InvitationSummaryRow>(self.dialect.invitations_for_org())
            .bind(org_id)
            .fetch_all(&self.pool)
            .await?;
        Ok(rows.into_iter().map(invitation_from_row).collect())
    }

    async fn revoke_invitation(&self, org_id: i64, invitation_id: i64) -> AppResult<bool> {
        let now = Self::now();
        let result = sqlx::query(self.dialect.revoke_invitation())
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
        let row = sqlx::query(self.dialect.invitation_by_hash())
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
        let org_row = sqlx::query(self.dialect.organization_by_id())
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

        let user_id = self
            .create_active_user(
                &mut tx,
                &input.email,
                &input.display_name,
                &input.password_hash,
                &now,
            )
            .await?;
        sqlx::query(self.dialect.create_active_membership())
            .bind(input.org_id)
            .bind(user_id)
            .bind(&input.role_code)
            .bind(&now)
            .bind(&now)
            .execute(&mut *tx)
            .await?;
        let accept_result = sqlx::query(self.dialect.accept_invitation())
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
        let reset_id = if self.dialect == SqlDialect::Mysql {
            sqlx::query(self.dialect.create_password_reset())
                .bind(input.user_id)
                .bind(input.token_hash)
                .bind(input.expires_at)
                .bind(&now)
                .execute(&mut *tx)
                .await?;
            Self::mysql_last_insert_id(&mut tx).await?
        } else {
            sqlx::query_scalar::<_, i64>(self.dialect.create_password_reset())
                .bind(input.user_id)
                .bind(input.token_hash)
                .bind(input.expires_at)
                .bind(&now)
                .fetch_one(&mut *tx)
                .await?
        };
        self.insert_notification_outbox(&mut tx, &notification, reset_id, &now)
            .await?;
        tx.commit().await?;
        Ok(())
    }

    async fn find_password_reset_by_hash(
        &self,
        token_hash: &str,
    ) -> AppResult<Option<PasswordResetRecord>> {
        let row = sqlx::query(self.dialect.password_reset_by_hash())
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
        let reset_result = sqlx::query(self.dialect.mark_password_reset_used())
            .bind(&now)
            .bind(reset_id)
            .bind(user_id)
            .execute(&mut *tx)
            .await?;
        if reset_result.rows_affected() == 0 {
            tx.rollback().await?;
            return Ok(false);
        }
        sqlx::query(self.dialect.update_user_password_hash())
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
        let verification_id = if self.dialect == SqlDialect::Mysql {
            sqlx::query(self.dialect.create_email_verification())
                .bind(input.user_id)
                .bind(input.email)
                .bind(input.token_hash)
                .bind(input.expires_at)
                .bind(&now)
                .execute(&mut *tx)
                .await?;
            Self::mysql_last_insert_id(&mut tx).await?
        } else {
            sqlx::query_scalar::<_, i64>(self.dialect.create_email_verification())
                .bind(input.user_id)
                .bind(input.email)
                .bind(input.token_hash)
                .bind(input.expires_at)
                .bind(&now)
                .fetch_one(&mut *tx)
                .await?
        };
        self.insert_notification_outbox(&mut tx, &notification, verification_id, &now)
            .await?;
        tx.commit().await?;
        Ok(())
    }

    async fn find_email_verification_by_hash(
        &self,
        token_hash: &str,
    ) -> AppResult<Option<EmailVerificationRecord>> {
        let row = sqlx::query(self.dialect.email_verification_by_hash())
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
        let verify_result = sqlx::query(self.dialect.mark_email_verification_verified())
            .bind(&now)
            .bind(verification_id)
            .bind(user_id)
            .execute(&mut *tx)
            .await?;
        if verify_result.rows_affected() == 0 {
            tx.rollback().await?;
            return Ok(false);
        }
        sqlx::query(self.dialect.mark_user_email_verified())
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
        sqlx::query(self.dialect.revoke_pending_mfa_factors())
            .bind(&now)
            .bind(input.user_id)
            .bind(&input.kind)
            .execute(&mut *tx)
            .await?;
        let id = if self.dialect == SqlDialect::Mysql {
            sqlx::query(self.dialect.create_mfa_factor())
                .bind(input.user_id)
                .bind(&input.kind)
                .bind(&input.secret_ciphertext)
                .bind(&now)
                .execute(&mut *tx)
                .await?;
            Self::mysql_last_insert_id(&mut tx).await?
        } else {
            sqlx::query_scalar::<_, i64>(self.dialect.create_mfa_factor())
                .bind(input.user_id)
                .bind(&input.kind)
                .bind(&input.secret_ciphertext)
                .bind(&now)
                .fetch_one(&mut *tx)
                .await?
        };
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
        let rows = sqlx::query_as::<_, MfaFactorSummaryRow>(self.dialect.mfa_factors_for_user())
            .bind(user_id)
            .fetch_all(&self.pool)
            .await?;
        Ok(rows.into_iter().map(mfa_factor_summary_from_row).collect())
    }

    async fn find_pending_mfa_factor(&self, user_id: i64) -> AppResult<Option<StoredMfaFactor>> {
        let row =
            sqlx::query_as::<_, StoredMfaFactorRow>(self.dialect.pending_mfa_factor_for_user())
                .bind(user_id)
                .fetch_optional(&self.pool)
                .await?;
        Ok(row.map(stored_mfa_factor_from_row))
    }

    async fn find_verified_mfa_factor(&self, user_id: i64) -> AppResult<Option<StoredMfaFactor>> {
        let row =
            sqlx::query_as::<_, StoredMfaFactorRow>(self.dialect.verified_mfa_factor_for_user())
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
        let activate = sqlx::query(self.dialect.activate_mfa_factor())
            .bind(&now)
            .bind(factor_id)
            .bind(user_id)
            .execute(&mut *tx)
            .await?;
        if activate.rows_affected() == 0 {
            tx.rollback().await?;
            return Ok(false);
        }
        sqlx::query(self.dialect.revoke_active_mfa_recovery_codes())
            .bind(&now)
            .bind(user_id)
            .execute(&mut *tx)
            .await?;
        self.insert_mfa_recovery_codes(&mut tx, user_id, &recovery_codes, &now)
            .await?;
        sqlx::query(self.dialect.revoke_other_mfa_factors())
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
        let result = sqlx::query(self.dialect.revoke_mfa_factor())
            .bind(&now)
            .bind(factor_id)
            .bind(user_id)
            .execute(&mut *tx)
            .await?;
        let revoked = result.rows_affected() > 0;
        if revoked {
            sqlx::query(self.dialect.revoke_active_mfa_recovery_codes())
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
        let rows = sqlx::query_as::<_, MfaRecoveryCodeSummaryRow>(
            self.dialect.mfa_recovery_codes_for_user(),
        )
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
        sqlx::query(self.dialect.revoke_active_mfa_recovery_codes())
            .bind(&now)
            .bind(user_id)
            .execute(&mut *tx)
            .await?;
        let summaries = self
            .insert_mfa_recovery_codes(&mut tx, user_id, &recovery_codes, &now)
            .await?;
        tx.commit().await?;
        Ok(summaries)
    }

    async fn consume_mfa_recovery_code(&self, user_id: i64, code_hash: &str) -> AppResult<bool> {
        let now = Self::now();
        let result = sqlx::query(self.dialect.consume_mfa_recovery_code())
            .bind(&now)
            .bind(user_id)
            .bind(code_hash)
            .execute(&self.pool)
            .await?;
        Ok(result.rows_affected() > 0)
    }

    async fn record_audit(&self, input: CreateAuditLogRecord) -> AppResult<()> {
        sqlx::query(self.dialect.create_audit_log())
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

fn split_csv(value: Option<String>) -> Vec<String> {
    value
        .unwrap_or_default()
        .split(',')
        .map(str::trim)
        .filter(|item| !item.is_empty())
        .map(ToOwned::to_owned)
        .collect()
}
