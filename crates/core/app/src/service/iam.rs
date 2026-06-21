use std::collections::BTreeSet;
use std::sync::Arc;

use chrono::{DateTime, Duration, Utc};
use uuid::Uuid;

use crate::app::{AppError, AppResult};
use crate::config::Settings;
use crate::domain::iam::{
    API_TOKEN_SECRET_PREFIX, APITokenSummary, AcceptInvitationRequest, AuthCredential,
    BooleanResult, CreateAPITokenRequest, CreateAPITokenResult, CreateRoleRequest,
    EMAIL_VERIFICATION_TOKEN_SECRET_PREFIX, ForgotPasswordRequest, INVITATION_TOKEN_SECRET_PREFIX,
    InitialAdminRequest, InvitationSummary, InviteUserRequest, LoginRequest, LogoutResult,
    MFA_RECOVERY_CODE_SECRET_PREFIX, MfaFactorSummary, MfaRecoveryCodeSummary,
    MfaRecoveryCodesResult, MfaSetupResult, MfaVerifyResult, NotificationDelivery,
    OrganizationSummary, OrganizationUserSummary, PASSWORD_RESET_TOKEN_SECRET_PREFIX,
    PermissionSummary, REFRESH_TOKEN_SECRET_PREFIX, RegisterRequest, RequestContext,
    RequestEmailVerification, ResetPasswordRequest, RoleSummary, SESSION_TOKEN_SECRET_PREFIX,
    SessionSnapshot, SessionTokens, SetupStatus, UpdateOrgUserRequest, UpdateRoleRequest, User,
    VerifyMfaRequest,
};
use crate::repository::{
    AcceptInvitationRecord, CreateAPITokenRecord, CreateAuditLogRecord,
    CreateEmailVerificationRecord, CreateInitialAdminRecord, CreateInvitationRecord,
    CreateMfaRecoveryCodeRecord, CreateNotificationOutboxRecord, CreatePasswordResetRecord,
    CreateRegistrationRecord, CreateRoleRecord, CreateSessionRecord, IamRepository,
    UpdateOrgUserRecord, UpdateRoleRecord,
};
use crate::service::crypto_error::CryptoResultExt;
use crypto;

pub struct IamService {
    settings: Settings,
    repo: Arc<dyn IamRepository>,
}

struct CreatedSession {
    tokens: SessionTokens,
    expires_at: String,
    refresh_expires_at: String,
}

struct GeneratedMfaRecoveryCodes {
    raw_codes: Vec<String>,
    records: Vec<CreateMfaRecoveryCodeRecord>,
}

impl IamService {
    pub fn new(settings: Settings, repo: Arc<dyn IamRepository>) -> Self {
        Self { settings, repo }
    }

    fn encrypt_delivery_token(&self, raw_token: &str) -> AppResult<String> {
        crypto::encrypt_secret(raw_token, &self.settings.notification.delivery_secret_key)
            .into_app()
    }

    pub async fn setup_status(&self) -> AppResult<SetupStatus> {
        Ok(SetupStatus {
            initialized: self.repo.has_any_user().await?,
        })
    }

    pub async fn initial_admin(
        &self,
        request: InitialAdminRequest,
        ctx: RequestContext,
    ) -> AppResult<(SessionSnapshot, SessionTokens)> {
        self.validate_initial_admin(&request)?;
        let password_hash = crypto::hash_password(&request.password).into_app()?;

        let (user, org) = self
            .repo
            .create_initial_admin(CreateInitialAdminRecord {
                email: request.email.trim().to_lowercase(),
                password_hash,
                display_name: request.display_name.trim().into(),
                organization_code: request.organization_code.trim().to_lowercase(),
                organization_name: request.organization_name.trim().into(),
                product_code: ctx.product_code.clone(),
            })
            .await?;

        let created_session = self.create_session(user.id, org.id, &ctx).await?;
        Ok((
            self.snapshot(
                user,
                org,
                ctx,
                created_session.expires_at.clone(),
                Some(created_session.refresh_expires_at.clone()),
                true,
            )
            .await?,
            created_session.tokens,
        ))
    }

    pub async fn register(
        &self,
        request: RegisterRequest,
        ctx: RequestContext,
    ) -> AppResult<NotificationDelivery> {
        if !self.settings.auth.self_signup_enabled {
            return Err(AppError::Forbidden);
        }
        self.validate_registration(&request)?;
        let email = request.email.trim().to_lowercase();
        if self.repo.find_user_by_identifier(&email).await?.is_some() {
            return Err(AppError::Conflict("注册邮箱已存在".into()));
        }

        let raw_token = crypto::new_secret(EMAIL_VERIFICATION_TOKEN_SECRET_PREFIX);
        let token_hash = crypto::hash_secret(&raw_token, &self.settings.auth.session_secret);
        let expires_at = (Utc::now()
            + Duration::seconds(self.settings.auth.email_verification_ttl_seconds))
        .to_rfc3339();
        let password_hash = crypto::hash_password(&request.password).into_app()?;
        let organization_code = normalize_org_code(&request.organization_code)?;
        let organization_name = normalize_org_name(&request.organization_name)?;
        let display_name = normalize_user_display_name(&request.display_name)?;
        let product_code = ctx.product_code.clone();

        self.repo
            .create_registration_with_email_verification(
                CreateRegistrationRecord {
                    email: email.clone(),
                    password_hash,
                    display_name,
                    organization_code,
                    organization_name,
                    product_code: product_code.clone(),
                    email_verification_token_hash: token_hash,
                    email_verification_expires_at: expires_at.clone(),
                },
                CreateNotificationOutboxRecord {
                    org_id: None,
                    user_id: None,
                    product_code,
                    channel: "email".into(),
                    template_code: "iam.registration.email_verification".into(),
                    recipient: email,
                    related_kind: "iam_email_verification".into(),
                    payload_json: serde_json::json!({
                        "expires_at": expires_at,
                        "registration": true,
                        "delivery_boundary": "raw_token_encrypted_in_delivery_secret_vault"
                    })
                    .to_string(),
                    available_at: Utc::now().to_rfc3339(),
                    delivery_secret_ciphertext: Some(self.encrypt_delivery_token(&raw_token)?),
                },
            )
            .await?;

        Ok(accepted_delivery())
    }

    pub async fn login(
        &self,
        request: LoginRequest,
        ctx: RequestContext,
    ) -> AppResult<(SessionSnapshot, SessionTokens)> {
        let user = self
            .repo
            .find_user_by_identifier(request.identifier.trim())
            .await?
            .ok_or(AppError::Unauthorized)?;
        if user.status != "active"
            || !crypto::verify_password(&request.password, &user.password_hash).into_app()?
        {
            return Err(AppError::Unauthorized);
        }
        let org = self
            .repo
            .primary_organization_for_user(user.id)
            .await?
            .ok_or_else(|| AppError::Conflict("当前用户没有可用组织".into()))?;
        if let Some(factor) = self.repo.find_verified_mfa_factor(user.id).await? {
            let Some(code) = request.mfa_code.as_deref() else {
                return Err(AppError::MfaRequired);
            };
            if self.verify_login_mfa_code(user.id, &factor, code).await? {
                self.audit(
                    Some(org.id),
                    Some(user.id),
                    "iam.mfa.recovery-code.used",
                    "tenant",
                    &ctx.product_code,
                    "用户使用一次性 MFA 恢复码完成登录",
                )
                .await?;
            }
        }

        let created_session = self.create_session(user.id, org.id, &ctx).await?;
        let user = User {
            id: user.id,
            email: user.email,
            display_name: user.display_name,
            status: user.status,
        };
        Ok((
            self.snapshot(
                user,
                org,
                ctx,
                created_session.expires_at.clone(),
                Some(created_session.refresh_expires_at.clone()),
                true,
            )
            .await?,
            created_session.tokens,
        ))
    }

    pub async fn current_session(
        &self,
        raw_session: Option<String>,
        ctx: RequestContext,
    ) -> AppResult<SessionSnapshot> {
        let Some(raw_session) = raw_session else {
            return Ok(SessionSnapshot {
                authenticated: false,
                user: None,
                organization: None,
                product_code: ctx.product_code,
                client_type: ctx.client_type,
                permissions: vec![],
                mfa_enabled: false,
                expires_at: None,
                refresh_expires_at: None,
            });
        };
        let hash = crypto::hash_secret(&raw_session, &self.settings.auth.session_secret);
        let Some(record) = self.repo.find_session_by_hash(&hash).await? else {
            return Err(AppError::Unauthorized);
        };
        if record.revoked_at.is_some() {
            return Err(AppError::Unauthorized);
        }
        let expires_at = DateTime::parse_from_rfc3339(&record.expires_at)
            .map_err(|err| AppError::Internal(format!("会话过期时间格式无效：{err}")))?
            .with_timezone(&Utc);
        if expires_at <= Utc::now() {
            return Err(AppError::Unauthorized);
        }

        let permissions = self
            .repo
            .list_permissions_for_user(
                record.user.id,
                record.organization.id,
                &record.product_code,
                true,
            )
            .await?;
        let mfa_enabled = self.mfa_enabled(record.user.id).await?;

        Ok(SessionSnapshot {
            authenticated: true,
            user: Some(record.user),
            organization: Some(record.organization),
            product_code: record.product_code,
            client_type: record.client_type,
            permissions,
            mfa_enabled,
            expires_at: Some(record.expires_at),
            refresh_expires_at: record.refresh_expires_at,
        })
    }

    pub async fn refresh_session(
        &self,
        raw_refresh: Option<String>,
        ctx: RequestContext,
    ) -> AppResult<(SessionSnapshot, SessionTokens)> {
        let Some(raw_refresh) = raw_refresh else {
            return Err(AppError::Unauthorized);
        };
        let current_refresh_hash =
            crypto::hash_secret(&raw_refresh, &self.settings.auth.session_secret);
        let Some(record) = self
            .repo
            .find_session_by_refresh_hash(&current_refresh_hash)
            .await?
        else {
            return Err(AppError::Unauthorized);
        };
        if record.revoked_at.is_some() {
            return Err(AppError::Unauthorized);
        }
        if record.product_code != ctx.product_code || record.client_type != ctx.client_type {
            return Err(AppError::Unauthorized);
        }
        let refresh_expires_at = record
            .refresh_expires_at
            .clone()
            .ok_or(AppError::Unauthorized)?;
        let refresh_expires_at = DateTime::parse_from_rfc3339(&refresh_expires_at)
            .map_err(|err| AppError::Internal(format!("刷新令牌过期时间格式无效：{err}")))?
            .with_timezone(&Utc);
        if refresh_expires_at <= Utc::now() {
            return Err(AppError::Unauthorized);
        }

        let rotated = self.rotated_session_tokens();
        let rotated_ok = self
            .repo
            .rotate_session_tokens(
                &record.id,
                &current_refresh_hash,
                crypto::hash_secret(
                    &rotated.tokens.raw_session,
                    &self.settings.auth.session_secret,
                ),
                crypto::hash_secret(
                    &rotated.tokens.raw_refresh,
                    &self.settings.auth.session_secret,
                ),
                rotated.expires_at.clone(),
                rotated.refresh_expires_at.clone(),
            )
            .await?;
        if !rotated_ok {
            return Err(AppError::Unauthorized);
        }
        self.audit(
            Some(record.organization.id),
            Some(record.user.id),
            "iam.session.refreshed",
            "tenant",
            &ctx.product_code,
            "refresh token 已轮换，未记录原始令牌",
        )
        .await?;
        Ok((
            self.snapshot(
                record.user,
                record.organization,
                ctx,
                rotated.expires_at.clone(),
                Some(rotated.refresh_expires_at.clone()),
                true,
            )
            .await?,
            rotated.tokens,
        ))
    }

    pub async fn logout(
        &self,
        raw_session: Option<String>,
        raw_refresh: Option<String>,
    ) -> AppResult<LogoutResult> {
        if let Some(raw_session) = raw_session {
            let hash = crypto::hash_secret(&raw_session, &self.settings.auth.session_secret);
            self.repo.revoke_session_by_hash(&hash).await?;
        }
        if let Some(raw_refresh) = raw_refresh {
            let hash = crypto::hash_secret(&raw_refresh, &self.settings.auth.session_secret);
            self.repo.revoke_session_by_refresh_hash(&hash).await?;
        }
        Ok(LogoutResult { logged_out: true })
    }

    pub async fn list_organizations(
        &self,
        credential: AuthCredential,
        ctx: RequestContext,
        required_permission: &str,
    ) -> AppResult<Vec<OrganizationSummary>> {
        self.require_permission(credential, ctx, None, required_permission)
            .await?;
        self.repo.list_organizations().await
    }

    pub async fn list_org_users(
        &self,
        credential: AuthCredential,
        org_id: i64,
        ctx: RequestContext,
        required_permission: &str,
    ) -> AppResult<Vec<OrganizationUserSummary>> {
        self.require_permission(credential, ctx, Some(org_id), required_permission)
            .await?;
        self.repo.list_org_users(org_id).await
    }

    pub async fn update_org_user(
        &self,
        credential: AuthCredential,
        org_id: i64,
        user_id: i64,
        request: UpdateOrgUserRequest,
        ctx: RequestContext,
        required_permission: &str,
    ) -> AppResult<OrganizationUserSummary> {
        validate_positive_id(user_id, "用户 ID")?;
        let snapshot = self
            .require_permission(credential, ctx, Some(org_id), required_permission)
            .await?;
        let actor_user_id = snapshot.user.as_ref().map(|user| user.id);
        let user = self
            .repo
            .update_org_user(UpdateOrgUserRecord {
                org_id,
                user_id,
                display_name: normalize_user_display_name(&request.display_name)?,
                status: normalize_user_status(&request.status)?,
                role_codes: normalize_role_codes(request.role_codes)?,
            })
            .await?;
        self.audit(
            Some(org_id),
            actor_user_id,
            "iam.user.updated",
            "tenant",
            &snapshot.product_code,
            "组织用户资料、状态或角色已更新",
        )
        .await?;
        Ok(user)
    }

    pub async fn list_org_roles(
        &self,
        credential: AuthCredential,
        org_id: i64,
        ctx: RequestContext,
        required_permission: &str,
    ) -> AppResult<Vec<RoleSummary>> {
        self.require_permission(credential, ctx, Some(org_id), required_permission)
            .await?;
        self.repo.list_org_roles(org_id).await
    }

    pub async fn create_org_role(
        &self,
        credential: AuthCredential,
        org_id: i64,
        request: CreateRoleRequest,
        ctx: RequestContext,
        required_permission: &str,
    ) -> AppResult<RoleSummary> {
        let snapshot = self
            .require_permission(credential, ctx, Some(org_id), required_permission)
            .await?;
        let user_id = snapshot.user.as_ref().map(|user| user.id);
        let role = self
            .repo
            .create_org_role(CreateRoleRecord {
                org_id,
                code: normalize_role_code(&request.code)?,
                name: normalize_role_name(&request.name)?,
                product_code: snapshot.product_code.clone(),
                permission_codes: normalize_permission_codes(request.permission_codes)?,
            })
            .await?;
        self.audit(
            Some(org_id),
            user_id,
            "iam.role.created",
            "tenant",
            &snapshot.product_code,
            "组织角色已创建",
        )
        .await?;
        Ok(role)
    }

    pub async fn update_org_role(
        &self,
        credential: AuthCredential,
        org_id: i64,
        role_id: i64,
        request: UpdateRoleRequest,
        ctx: RequestContext,
        required_permission: &str,
    ) -> AppResult<RoleSummary> {
        validate_positive_id(role_id, "角色 ID")?;
        let snapshot = self
            .require_permission(credential, ctx, Some(org_id), required_permission)
            .await?;
        let user_id = snapshot.user.as_ref().map(|user| user.id);
        let role = self
            .repo
            .update_org_role(UpdateRoleRecord {
                org_id,
                role_id,
                name: normalize_role_name(&request.name)?,
                product_code: snapshot.product_code.clone(),
                permission_codes: normalize_permission_codes(request.permission_codes)?,
            })
            .await?;
        self.audit(
            Some(org_id),
            user_id,
            "iam.role.updated",
            "tenant",
            &snapshot.product_code,
            "组织角色已更新",
        )
        .await?;
        Ok(role)
    }

    pub async fn delete_org_role(
        &self,
        credential: AuthCredential,
        org_id: i64,
        role_id: i64,
        ctx: RequestContext,
        required_permission: &str,
    ) -> AppResult<BooleanResult> {
        validate_positive_id(role_id, "角色 ID")?;
        let snapshot = self
            .require_permission(credential, ctx, Some(org_id), required_permission)
            .await?;
        let user_id = snapshot.user.as_ref().map(|user| user.id);
        let deleted = self.repo.delete_org_role(org_id, role_id).await?;
        if deleted {
            self.audit(
                Some(org_id),
                user_id,
                "iam.role.deleted",
                "tenant",
                &snapshot.product_code,
                "组织角色已删除",
            )
            .await?;
        }
        Ok(BooleanResult {
            revoked: None,
            accepted: None,
            reset: None,
            verified: None,
            deleted: Some(deleted),
        })
    }

    pub async fn list_permissions(
        &self,
        credential: AuthCredential,
        ctx: RequestContext,
        required_permission: &str,
    ) -> AppResult<Vec<PermissionSummary>> {
        self.require_permission(credential, ctx.clone(), None, required_permission)
            .await?;
        self.repo.list_permissions(&ctx.product_code).await
    }

    pub async fn create_api_token(
        &self,
        credential: AuthCredential,
        org_id: i64,
        request: CreateAPITokenRequest,
        ctx: RequestContext,
        required_permission: &str,
    ) -> AppResult<CreateAPITokenResult> {
        let snapshot = self
            .require_permission(credential, ctx, Some(org_id), required_permission)
            .await?;
        let user = snapshot.user.ok_or(AppError::Unauthorized)?;

        let raw_token = crypto::new_secret(API_TOKEN_SECRET_PREFIX);
        let token_hash = crypto::hash_secret(&raw_token, &self.settings.auth.session_secret);
        let token_prefix = raw_token.chars().take(16).collect::<String>();
        let days = request
            .expires_in_days
            .unwrap_or(self.settings.auth.api_token_default_days)
            .clamp(1, 365);
        let expires_at = Some((Utc::now() + Duration::days(days)).to_rfc3339());
        let item = self
            .repo
            .create_api_token(CreateAPITokenRecord {
                org_id,
                user_id: user.id,
                token_hash,
                token_prefix,
                expires_at,
            })
            .await?;
        self.audit(
            Some(org_id),
            Some(user.id),
            "iam.api_token.created",
            "tenant",
            &snapshot.product_code,
            "组织 API Token 已签发",
        )
        .await?;
        Ok(CreateAPITokenResult {
            item,
            token: raw_token,
        })
    }

    pub async fn list_api_tokens(
        &self,
        credential: AuthCredential,
        org_id: i64,
        ctx: RequestContext,
        required_permission: &str,
    ) -> AppResult<Vec<APITokenSummary>> {
        self.require_permission(credential, ctx, Some(org_id), required_permission)
            .await?;
        self.repo.list_api_tokens(org_id).await
    }

    pub async fn revoke_api_token(
        &self,
        credential: AuthCredential,
        org_id: i64,
        token_id: i64,
        ctx: RequestContext,
        required_permission: &str,
    ) -> AppResult<BooleanResult> {
        let snapshot = self
            .require_permission(credential, ctx, Some(org_id), required_permission)
            .await?;
        let user_id = snapshot.user.as_ref().map(|user| user.id);
        let revoked = self.repo.revoke_api_token(org_id, token_id).await?;
        if revoked {
            self.audit(
                Some(org_id),
                user_id,
                "iam.api_token.revoked",
                "tenant",
                &snapshot.product_code,
                "组织 API Token 已撤销",
            )
            .await?;
        }
        Ok(BooleanResult {
            revoked: Some(revoked),
            accepted: None,
            reset: None,
            verified: None,
            deleted: None,
        })
    }

    pub async fn invite_user(
        &self,
        credential: AuthCredential,
        org_id: i64,
        request: InviteUserRequest,
        ctx: RequestContext,
        required_permission: &str,
    ) -> AppResult<(InvitationSummary, NotificationDelivery)> {
        let snapshot = self
            .require_permission(credential, ctx, Some(org_id), required_permission)
            .await?;
        let user_id = snapshot.user.as_ref().map(|user| user.id);
        if !request.email.contains('@') {
            return Err(AppError::Validation("邮箱格式无效".into()));
        }
        let role_code = request
            .role_code
            .as_deref()
            .map(str::trim)
            .filter(|value| !value.is_empty())
            .unwrap_or("owner")
            .to_ascii_lowercase();
        validate_role_code(&role_code)?;
        if !self.repo.org_role_exists(org_id, &role_code).await? {
            return Err(AppError::Validation("邀请角色不存在".into()));
        }

        let raw_token = crypto::new_secret(INVITATION_TOKEN_SECRET_PREFIX);
        let token_hash = crypto::hash_secret(&raw_token, &self.settings.auth.session_secret);
        let expires_at = (Utc::now()
            + Duration::seconds(self.settings.auth.invitation_ttl_seconds))
        .to_rfc3339();
        let email = request.email.trim().to_lowercase();
        let invitation = self
            .repo
            .create_invitation_with_notification(
                CreateInvitationRecord {
                    org_id,
                    email: email.clone(),
                    role_code: role_code.clone(),
                    token_hash,
                    expires_at: expires_at.clone(),
                },
                CreateNotificationOutboxRecord {
                    org_id: Some(org_id),
                    user_id,
                    product_code: snapshot.product_code.clone(),
                    channel: "email".into(),
                    template_code: "iam.invitation.created".into(),
                    recipient: email.clone(),
                    related_kind: "iam_invitation".into(),
                    payload_json: serde_json::json!({
                        "org_id": org_id,
                        "email": email,
                        "role_code": role_code,
                        "expires_at": expires_at,
                        "delivery_boundary": "raw_token_encrypted_in_delivery_secret_vault"
                    })
                    .to_string(),
                    available_at: Utc::now().to_rfc3339(),
                    delivery_secret_ciphertext: Some(self.encrypt_delivery_token(&raw_token)?),
                },
            )
            .await?;
        self.audit(
            Some(org_id),
            user_id,
            "iam.invitation.created",
            "tenant",
            &snapshot.product_code,
            "组织邀请已创建",
        )
        .await?;
        Ok((invitation, accepted_delivery()))
    }

    pub async fn accept_invitation(
        &self,
        request: AcceptInvitationRequest,
        ctx: RequestContext,
    ) -> AppResult<(SessionSnapshot, SessionTokens)> {
        let token = request.token.trim();
        if token.is_empty() {
            return Err(AppError::Validation("邀请令牌不能为空".into()));
        }
        if request.password.len() < self.settings.auth.password_policy.min_length {
            return Err(AppError::Validation(format!(
                "密码长度不能小于 {}",
                self.settings.auth.password_policy.min_length
            )));
        }
        let display_name = request.display_name.trim().to_string();
        if display_name.is_empty() {
            return Err(AppError::Validation("显示名不能为空".into()));
        }

        let token_hash = crypto::hash_secret(token, &self.settings.auth.session_secret);
        let Some(invitation) = self.repo.find_invitation_by_hash(&token_hash).await? else {
            return Err(AppError::Unauthorized);
        };
        ensure_pending_token(
            &invitation.status,
            &invitation.expires_at,
            invitation.accepted_at.as_deref(),
        )?;
        if invitation.revoked_at.is_some() {
            return Err(AppError::Unauthorized);
        }
        if !self
            .repo
            .org_role_exists(invitation.org_id, &invitation.role_code)
            .await?
        {
            return Err(AppError::Conflict("邀请绑定的角色不存在".into()));
        }
        if self
            .repo
            .find_user_by_identifier(&invitation.email)
            .await?
            .is_some()
        {
            return Err(AppError::Conflict("受邀邮箱已存在用户".into()));
        }

        let password_hash = crypto::hash_password(&request.password).into_app()?;
        let (user, org) = self
            .repo
            .accept_invitation_with_user(AcceptInvitationRecord {
                invitation_id: invitation.id,
                org_id: invitation.org_id,
                email: invitation.email,
                role_code: invitation.role_code,
                display_name,
                password_hash,
            })
            .await?;
        self.audit(
            Some(org.id),
            Some(user.id),
            "iam.invitation.accepted",
            "tenant",
            &ctx.product_code,
            "用户已通过邀请加入组织",
        )
        .await?;

        let created_session = self.create_session(user.id, org.id, &ctx).await?;
        Ok((
            self.snapshot(
                user,
                org,
                ctx,
                created_session.expires_at.clone(),
                Some(created_session.refresh_expires_at.clone()),
                false,
            )
            .await?,
            created_session.tokens,
        ))
    }

    pub async fn list_invitations(
        &self,
        credential: AuthCredential,
        org_id: i64,
        ctx: RequestContext,
        required_permission: &str,
    ) -> AppResult<Vec<InvitationSummary>> {
        self.require_permission(credential, ctx, Some(org_id), required_permission)
            .await?;
        self.repo.list_invitations(org_id).await
    }

    pub async fn revoke_invitation(
        &self,
        credential: AuthCredential,
        org_id: i64,
        invitation_id: i64,
        ctx: RequestContext,
        required_permission: &str,
    ) -> AppResult<BooleanResult> {
        let snapshot = self
            .require_permission(credential, ctx, Some(org_id), required_permission)
            .await?;
        let user_id = snapshot.user.as_ref().map(|user| user.id);
        let revoked = self.repo.revoke_invitation(org_id, invitation_id).await?;
        if revoked {
            self.audit(
                Some(org_id),
                user_id,
                "iam.invitation.revoked",
                "tenant",
                &snapshot.product_code,
                "组织邀请已撤销",
            )
            .await?;
        }
        Ok(BooleanResult {
            revoked: Some(revoked),
            accepted: None,
            reset: None,
            verified: None,
            deleted: None,
        })
    }

    pub async fn forgot_password(
        &self,
        request: ForgotPasswordRequest,
    ) -> AppResult<NotificationDelivery> {
        if let Some(user) = self
            .repo
            .find_user_by_identifier(request.email.trim())
            .await?
        {
            let user_id = user.id;
            let raw_token = crypto::new_secret(PASSWORD_RESET_TOKEN_SECRET_PREFIX);
            let token_hash = crypto::hash_secret(&raw_token, &self.settings.auth.session_secret);
            let expires_at = (Utc::now()
                + Duration::seconds(self.settings.auth.password_reset_ttl_seconds))
            .to_rfc3339();
            self.repo
                .create_password_reset_with_notification(
                    CreatePasswordResetRecord {
                        user_id,
                        token_hash,
                        expires_at: expires_at.clone(),
                    },
                    CreateNotificationOutboxRecord {
                        org_id: None,
                        user_id: Some(user_id),
                        product_code: self.settings.app.product_code.clone(),
                        channel: "email".into(),
                        template_code: "iam.password_reset.requested".into(),
                        recipient: user.email,
                        related_kind: "iam_password_reset".into(),
                        payload_json: serde_json::json!({
                            "user_id": user_id,
                            "expires_at": expires_at,
                            "delivery_boundary": "raw_token_encrypted_in_delivery_secret_vault"
                        })
                        .to_string(),
                        available_at: Utc::now().to_rfc3339(),
                        delivery_secret_ciphertext: Some(self.encrypt_delivery_token(&raw_token)?),
                    },
                )
                .await?;
            self.audit(
                None,
                Some(user_id),
                "iam.password_reset.requested",
                "platform",
                &self.settings.app.product_code,
                "密码重置通知已创建",
            )
            .await?;
        }
        Ok(accepted_delivery())
    }

    pub async fn reset_password(&self, request: ResetPasswordRequest) -> AppResult<BooleanResult> {
        if request.password.len() < self.settings.auth.password_policy.min_length {
            return Err(AppError::Validation(format!(
                "密码长度不能小于 {}",
                self.settings.auth.password_policy.min_length
            )));
        }
        let token_hash = crypto::hash_secret(&request.token, &self.settings.auth.session_secret);
        let Some(reset) = self.repo.find_password_reset_by_hash(&token_hash).await? else {
            return Err(AppError::Unauthorized);
        };
        ensure_pending_token(&reset.status, &reset.expires_at, reset.used_at.as_deref())?;
        let password_hash = crypto::hash_password(&request.password).into_app()?;
        let reset_id = reset.id;
        let reset_user_id = reset.user_id;
        let reset_done = self
            .repo
            .reset_password_with_token(reset_id, reset_user_id, password_hash)
            .await?;
        if reset_done {
            self.audit(
                None,
                Some(reset_user_id),
                "iam.password_reset.completed",
                "platform",
                &self.settings.app.product_code,
                "密码已通过重置令牌更新",
            )
            .await?;
        }
        Ok(BooleanResult {
            revoked: None,
            accepted: None,
            reset: Some(reset_done),
            verified: None,
            deleted: None,
        })
    }

    pub async fn request_email_verification(
        &self,
        request: RequestEmailVerification,
    ) -> AppResult<NotificationDelivery> {
        if let Some(user) = self
            .repo
            .find_user_by_identifier(request.email.trim())
            .await?
        {
            let user_id = user.id;
            let email = user.email;
            let raw_token = crypto::new_secret(EMAIL_VERIFICATION_TOKEN_SECRET_PREFIX);
            let token_hash = crypto::hash_secret(&raw_token, &self.settings.auth.session_secret);
            let expires_at = (Utc::now()
                + Duration::seconds(self.settings.auth.email_verification_ttl_seconds))
            .to_rfc3339();
            self.repo
                .create_email_verification_with_notification(
                    CreateEmailVerificationRecord {
                        user_id,
                        email: email.clone(),
                        token_hash,
                        expires_at: expires_at.clone(),
                    },
                    CreateNotificationOutboxRecord {
                        org_id: None,
                        user_id: Some(user_id),
                        product_code: self.settings.app.product_code.clone(),
                        channel: "email".into(),
                        template_code: "iam.email_verification.requested".into(),
                        recipient: email,
                        related_kind: "iam_email_verification".into(),
                        payload_json: serde_json::json!({
                            "user_id": user_id,
                            "expires_at": expires_at,
                            "delivery_boundary": "raw_token_encrypted_in_delivery_secret_vault"
                        })
                        .to_string(),
                        available_at: Utc::now().to_rfc3339(),
                        delivery_secret_ciphertext: Some(self.encrypt_delivery_token(&raw_token)?),
                    },
                )
                .await?;
            self.audit(
                None,
                Some(user_id),
                "iam.email_verification.requested",
                "platform",
                &self.settings.app.product_code,
                "邮箱验证通知已创建",
            )
            .await?;
        }
        Ok(accepted_delivery())
    }

    pub async fn confirm_email_verification(&self, token: String) -> AppResult<BooleanResult> {
        let token_hash = crypto::hash_secret(&token, &self.settings.auth.session_secret);
        let Some(verification) = self
            .repo
            .find_email_verification_by_hash(&token_hash)
            .await?
        else {
            return Err(AppError::Unauthorized);
        };
        ensure_pending_token(
            &verification.status,
            &verification.expires_at,
            verification.verified_at.as_deref(),
        )?;
        let verified = self
            .repo
            .confirm_email_verification(verification.id, verification.user_id)
            .await?;
        if verified {
            self.audit(
                None,
                Some(verification.user_id),
                "iam.email_verification.completed",
                "platform",
                &self.settings.app.product_code,
                "邮箱验证已完成",
            )
            .await?;
        }
        Ok(BooleanResult {
            revoked: None,
            accepted: None,
            reset: None,
            verified: Some(verified),
            deleted: None,
        })
    }

    pub async fn setup_mfa(
        &self,
        credential: AuthCredential,
        ctx: RequestContext,
    ) -> AppResult<MfaSetupResult> {
        let snapshot = self.require_authenticated_session(credential, ctx).await?;
        let user = snapshot.user.ok_or(AppError::Unauthorized)?;
        let org_id = snapshot.organization.as_ref().map(|org| org.id);
        let secret = crypto::new_totp_secret();
        let secret_ciphertext =
            crypto::encrypt_secret(&secret, &self.settings.auth.mfa_secret_key).into_app()?;
        let factor = self
            .repo
            .create_pending_mfa_factor(crate::repository::CreateMfaFactorRecord {
                user_id: user.id,
                kind: "totp".into(),
                secret_ciphertext,
            })
            .await?;
        self.audit(
            org_id,
            Some(user.id),
            "iam.mfa.setup",
            "tenant",
            &snapshot.product_code,
            "TOTP MFA 因子已创建，等待验证码确认",
        )
        .await?;
        Ok(MfaSetupResult {
            factor,
            otpauth_url: crypto::totp_otpauth_url(
                &self.settings.auth.mfa_issuer,
                &user.email,
                &secret,
            ),
            secret,
        })
    }

    pub async fn list_mfa_factors(
        &self,
        credential: AuthCredential,
        ctx: RequestContext,
    ) -> AppResult<Vec<MfaFactorSummary>> {
        let snapshot = self.require_authenticated_session(credential, ctx).await?;
        let user = snapshot.user.ok_or(AppError::Unauthorized)?;
        self.repo.list_mfa_factors(user.id).await
    }

    pub async fn verify_mfa(
        &self,
        credential: AuthCredential,
        ctx: RequestContext,
        request: VerifyMfaRequest,
    ) -> AppResult<MfaVerifyResult> {
        let snapshot = self.require_authenticated_session(credential, ctx).await?;
        let user = snapshot.user.ok_or(AppError::Unauthorized)?;
        let org_id = snapshot.organization.as_ref().map(|org| org.id);
        let factor = self
            .repo
            .find_pending_mfa_factor(user.id)
            .await?
            .ok_or(AppError::Unauthorized)?;
        self.verify_mfa_factor_code(&factor, &request.code).await?;
        let recovery_codes = self.generate_mfa_recovery_codes();
        let verified = self
            .repo
            .activate_mfa_factor_with_recovery_codes(user.id, factor.id, recovery_codes.records)
            .await?;
        if verified {
            self.audit(
                org_id,
                Some(user.id),
                "iam.mfa.verified",
                "tenant",
                &snapshot.product_code,
                "TOTP MFA 因子已验证并启用",
            )
            .await?;
        }
        Ok(MfaVerifyResult {
            verified,
            recovery_codes: if verified {
                recovery_codes.raw_codes
            } else {
                Vec::new()
            },
        })
    }

    pub async fn list_mfa_recovery_codes(
        &self,
        credential: AuthCredential,
        ctx: RequestContext,
    ) -> AppResult<Vec<MfaRecoveryCodeSummary>> {
        let snapshot = self.require_authenticated_session(credential, ctx).await?;
        let user = snapshot.user.ok_or(AppError::Unauthorized)?;
        self.repo.list_mfa_recovery_codes(user.id).await
    }

    pub async fn rotate_mfa_recovery_codes(
        &self,
        credential: AuthCredential,
        ctx: RequestContext,
    ) -> AppResult<MfaRecoveryCodesResult> {
        let snapshot = self.require_authenticated_session(credential, ctx).await?;
        let user = snapshot.user.ok_or(AppError::Unauthorized)?;
        if self.repo.find_verified_mfa_factor(user.id).await?.is_none() {
            return Err(AppError::Validation(
                "启用 TOTP MFA 后才能轮换恢复码".into(),
            ));
        }
        let org_id = snapshot.organization.as_ref().map(|org| org.id);
        let recovery_codes = self.generate_mfa_recovery_codes();
        let items = self
            .repo
            .replace_mfa_recovery_codes(user.id, recovery_codes.records)
            .await?;
        self.audit(
            org_id,
            Some(user.id),
            "iam.mfa.recovery-codes.rotated",
            "tenant",
            &snapshot.product_code,
            "MFA 恢复码已轮换",
        )
        .await?;
        Ok(MfaRecoveryCodesResult {
            items,
            recovery_codes: recovery_codes.raw_codes,
        })
    }

    pub async fn revoke_mfa(
        &self,
        credential: AuthCredential,
        ctx: RequestContext,
        factor_id: i64,
    ) -> AppResult<BooleanResult> {
        if factor_id <= 0 {
            return Err(AppError::Validation("MFA 因子 ID 必须大于 0".into()));
        }
        let snapshot = self.require_authenticated_session(credential, ctx).await?;
        let user = snapshot.user.ok_or(AppError::Unauthorized)?;
        let org_id = snapshot.organization.as_ref().map(|org| org.id);
        let revoked = self.repo.revoke_mfa_factor(user.id, factor_id).await?;
        if revoked {
            self.audit(
                org_id,
                Some(user.id),
                "iam.mfa.revoked",
                "tenant",
                &snapshot.product_code,
                "TOTP MFA 因子已撤销",
            )
            .await?;
        }
        Ok(BooleanResult {
            revoked: Some(revoked),
            accepted: None,
            reset: None,
            verified: None,
            deleted: None,
        })
    }

    pub async fn require_permission(
        &self,
        credential: AuthCredential,
        ctx: RequestContext,
        org_id: Option<i64>,
        required_permission: &str,
    ) -> AppResult<SessionSnapshot> {
        let snapshot = self.authenticate(credential, ctx).await?;
        if !snapshot.authenticated {
            return Err(AppError::Unauthorized);
        }
        if let Some(expected_org_id) = org_id {
            let Some(org) = &snapshot.organization else {
                return Err(AppError::Forbidden);
            };
            if org.id != expected_org_id {
                return Err(AppError::Forbidden);
            }
        }
        if !snapshot
            .permissions
            .iter()
            .any(|permission| permission == required_permission)
        {
            return Err(AppError::Forbidden);
        }
        Ok(snapshot)
    }

    pub async fn actor_user_id(
        &self,
        credential: AuthCredential,
        ctx: RequestContext,
    ) -> AppResult<Option<i64>> {
        let snapshot = self.authenticate(credential, ctx).await?;
        Ok(snapshot.user.map(|user| user.id))
    }

    fn validate_initial_admin(&self, request: &InitialAdminRequest) -> AppResult<()> {
        if !request.email.contains('@') {
            return Err(AppError::Validation("邮箱格式无效".into()));
        }
        if request.password.len() < self.settings.auth.password_policy.min_length {
            return Err(AppError::Validation(format!(
                "密码长度不能小于 {}",
                self.settings.auth.password_policy.min_length
            )));
        }
        if request.organization_code.trim().is_empty()
            || request.organization_name.trim().is_empty()
        {
            return Err(AppError::Validation("组织编码和名称不能为空".into()));
        }
        Ok(())
    }

    fn validate_registration(&self, request: &RegisterRequest) -> AppResult<()> {
        if !request.email.contains('@') {
            return Err(AppError::Validation("邮箱格式无效".into()));
        }
        if request.password.len() < self.settings.auth.password_policy.min_length {
            return Err(AppError::Validation(format!(
                "密码长度不能小于 {}",
                self.settings.auth.password_policy.min_length
            )));
        }
        normalize_user_display_name(&request.display_name)?;
        normalize_org_code(&request.organization_code)?;
        normalize_org_name(&request.organization_name)?;
        Ok(())
    }

    async fn authenticate(
        &self,
        credential: AuthCredential,
        ctx: RequestContext,
    ) -> AppResult<SessionSnapshot> {
        if credential.raw_session.is_some() {
            return self.current_session(credential.raw_session, ctx).await;
        }
        if let Some(raw_api_token) = credential.raw_api_token {
            return self.api_token_session(raw_api_token, ctx).await;
        }
        Ok(SessionSnapshot {
            authenticated: false,
            user: None,
            organization: None,
            product_code: ctx.product_code,
            client_type: ctx.client_type,
            permissions: vec![],
            mfa_enabled: false,
            expires_at: None,
            refresh_expires_at: None,
        })
    }

    async fn api_token_session(
        &self,
        raw_api_token: String,
        ctx: RequestContext,
    ) -> AppResult<SessionSnapshot> {
        let hash = crypto::hash_secret(&raw_api_token, &self.settings.auth.session_secret);
        let Some(record) = self.repo.find_api_token_by_hash(&hash).await? else {
            return Err(AppError::Unauthorized);
        };
        if record.status != "active"
            || record.revoked_at.is_some()
            || record.user.status != "active"
        {
            return Err(AppError::Unauthorized);
        }
        if let Some(expires_at) = &record.expires_at {
            let expires_at = DateTime::parse_from_rfc3339(expires_at)
                .map_err(|err| AppError::Internal(format!("API Token 过期时间格式无效：{err}")))?
                .with_timezone(&Utc);
            if expires_at <= Utc::now() {
                return Err(AppError::Unauthorized);
            }
        }
        let permissions = self
            .repo
            .list_permissions_for_user(
                record.user.id,
                record.organization.id,
                &ctx.product_code,
                false,
            )
            .await?;
        let mfa_enabled = self.mfa_enabled(record.user.id).await?;
        Ok(SessionSnapshot {
            authenticated: true,
            user: Some(record.user),
            organization: Some(record.organization),
            product_code: ctx.product_code,
            client_type: ctx.client_type,
            permissions,
            mfa_enabled,
            expires_at: record.expires_at,
            refresh_expires_at: None,
        })
    }

    async fn create_session(
        &self,
        user_id: i64,
        org_id: i64,
        ctx: &RequestContext,
    ) -> AppResult<CreatedSession> {
        let created = self.rotated_session_tokens();
        self.repo
            .create_session(CreateSessionRecord {
                id: Uuid::new_v4().to_string(),
                token_hash: crypto::hash_secret(
                    &created.tokens.raw_session,
                    &self.settings.auth.session_secret,
                ),
                refresh_token_hash: crypto::hash_secret(
                    &created.tokens.raw_refresh,
                    &self.settings.auth.session_secret,
                ),
                user_id,
                org_id,
                product_code: ctx.product_code.clone(),
                client_type: ctx.client_type.clone(),
                expires_at: created.expires_at.clone(),
                refresh_expires_at: created.refresh_expires_at.clone(),
            })
            .await?;
        Ok(created)
    }

    fn rotated_session_tokens(&self) -> CreatedSession {
        CreatedSession {
            tokens: SessionTokens {
                raw_session: crypto::new_secret(SESSION_TOKEN_SECRET_PREFIX),
                raw_refresh: crypto::new_secret(REFRESH_TOKEN_SECRET_PREFIX),
            },
            expires_at: self.expires_at(),
            refresh_expires_at: self.refresh_expires_at(),
        }
    }

    fn expires_at(&self) -> String {
        (Utc::now() + Duration::seconds(self.settings.auth.session_ttl_seconds)).to_rfc3339()
    }

    fn refresh_expires_at(&self) -> String {
        (Utc::now() + Duration::seconds(self.settings.auth.refresh_ttl_seconds)).to_rfc3339()
    }

    async fn snapshot(
        &self,
        user: User,
        org: crate::domain::iam::Organization,
        ctx: RequestContext,
        expires_at: String,
        refresh_expires_at: Option<String>,
        include_platform: bool,
    ) -> AppResult<SessionSnapshot> {
        let permissions = self
            .repo
            .list_permissions_for_user(user.id, org.id, &ctx.product_code, include_platform)
            .await?;
        let mfa_enabled = self.mfa_enabled(user.id).await?;
        Ok(SessionSnapshot {
            authenticated: true,
            user: Some(user),
            organization: Some(org),
            product_code: ctx.product_code,
            client_type: ctx.client_type,
            permissions,
            mfa_enabled,
            expires_at: Some(expires_at),
            refresh_expires_at,
        })
    }

    async fn require_authenticated_session(
        &self,
        credential: AuthCredential,
        ctx: RequestContext,
    ) -> AppResult<SessionSnapshot> {
        if credential.raw_api_token.is_some() {
            return Err(AppError::Forbidden);
        }
        let snapshot = self.current_session(credential.raw_session, ctx).await?;
        if !snapshot.authenticated {
            return Err(AppError::Unauthorized);
        }
        Ok(snapshot)
    }

    async fn mfa_enabled(&self, user_id: i64) -> AppResult<bool> {
        Ok(self.repo.find_verified_mfa_factor(user_id).await?.is_some())
    }

    fn generate_mfa_recovery_codes(&self) -> GeneratedMfaRecoveryCodes {
        let mut raw_codes = Vec::with_capacity(self.settings.auth.mfa_recovery_code_count);
        let mut records = Vec::with_capacity(self.settings.auth.mfa_recovery_code_count);
        for _ in 0..self.settings.auth.mfa_recovery_code_count {
            let raw_code = crypto::new_secret(MFA_RECOVERY_CODE_SECRET_PREFIX);
            records.push(CreateMfaRecoveryCodeRecord {
                code_hash: crypto::hash_secret(&raw_code, &self.settings.auth.session_secret),
                code_prefix: secret_prefix(&raw_code),
            });
            raw_codes.push(raw_code);
        }
        GeneratedMfaRecoveryCodes { raw_codes, records }
    }

    async fn verify_login_mfa_code(
        &self,
        user_id: i64,
        factor: &crate::domain::iam::StoredMfaFactor,
        code: &str,
    ) -> AppResult<bool> {
        if self.mfa_factor_code_matches(factor, code).await? {
            return Ok(false);
        }
        let recovery_hash = crypto::hash_secret(code.trim(), &self.settings.auth.session_secret);
        if self
            .repo
            .consume_mfa_recovery_code(user_id, &recovery_hash)
            .await?
        {
            return Ok(true);
        }
        Err(AppError::Unauthorized)
    }

    async fn verify_mfa_factor_code(
        &self,
        factor: &crate::domain::iam::StoredMfaFactor,
        code: &str,
    ) -> AppResult<()> {
        if self.mfa_factor_code_matches(factor, code).await? {
            Ok(())
        } else {
            Err(AppError::Unauthorized)
        }
    }

    async fn mfa_factor_code_matches(
        &self,
        factor: &crate::domain::iam::StoredMfaFactor,
        code: &str,
    ) -> AppResult<bool> {
        let secret = crypto::decrypt_secret(
            &factor.secret_ciphertext,
            &self.settings.auth.mfa_secret_key,
        )
        .into_app()?;
        crypto::verify_totp_code(&secret, code).into_app()
    }

    async fn audit(
        &self,
        org_id: Option<i64>,
        user_id: Option<i64>,
        action: &str,
        scope: &str,
        product_code: &str,
        detail: &str,
    ) -> AppResult<()> {
        self.repo
            .record_audit(CreateAuditLogRecord {
                org_id,
                user_id,
                action: action.into(),
                scope: scope.into(),
                product_code: product_code.into(),
                detail: detail.into(),
            })
            .await
    }
}

fn accepted_delivery() -> NotificationDelivery {
    NotificationDelivery {
        accepted: true,
        channel: "notification-outbox".into(),
    }
}

fn secret_prefix(value: &str) -> String {
    if let Some(random_part) = value.strip_prefix(&format!("{MFA_RECOVERY_CODE_SECRET_PREFIX}_")) {
        let random_prefix: String = random_part.chars().take(8).collect();
        return format!("{MFA_RECOVERY_CODE_SECRET_PREFIX}_{random_prefix}");
    }
    value.chars().take(12).collect()
}

fn normalize_role_code(value: &str) -> AppResult<String> {
    let code = value.trim().to_ascii_lowercase();
    validate_role_code(&code)?;
    Ok(code)
}

fn normalize_role_name(value: &str) -> AppResult<String> {
    let name = value.trim().to_string();
    if name.is_empty() || name.chars().count() > 80 {
        return Err(AppError::Validation(
            "角色名称不能为空且不能超过 80 个字符".into(),
        ));
    }
    Ok(name)
}

fn normalize_user_display_name(value: &str) -> AppResult<String> {
    let name = value.trim().to_string();
    if name.is_empty() || name.chars().count() > 80 {
        return Err(AppError::Validation(
            "用户显示名不能为空且不能超过 80 个字符".into(),
        ));
    }
    Ok(name)
}

fn normalize_org_code(value: &str) -> AppResult<String> {
    let code = value.trim().to_ascii_lowercase();
    if code.is_empty() || code.len() > 64 {
        return Err(AppError::Validation(
            "组织编码不能为空且不能超过 64 个字符".into(),
        ));
    }
    if !code
        .chars()
        .all(|ch| ch.is_ascii_alphanumeric() || matches!(ch, '_' | '-'))
    {
        return Err(AppError::Validation(
            "组织编码只能包含字母、数字、下划线和中划线".into(),
        ));
    }
    Ok(code)
}

fn normalize_org_name(value: &str) -> AppResult<String> {
    let name = value.trim().to_string();
    if name.is_empty() || name.chars().count() > 80 {
        return Err(AppError::Validation(
            "组织名称不能为空且不能超过 80 个字符".into(),
        ));
    }
    Ok(name)
}

fn normalize_user_status(value: &str) -> AppResult<String> {
    let status = value.trim().to_ascii_lowercase();
    if matches!(status.as_str(), "active" | "disabled") {
        Ok(status)
    } else {
        Err(AppError::Validation(
            "用户状态只能是 active 或 disabled".into(),
        ))
    }
}

fn normalize_role_codes(values: Vec<String>) -> AppResult<Vec<String>> {
    if values.is_empty() {
        return Err(AppError::Validation("组织用户至少需要一个租户角色".into()));
    }
    let mut seen = BTreeSet::new();
    for value in values {
        seen.insert(normalize_role_code(&value)?);
    }
    Ok(seen.into_iter().collect())
}

fn normalize_permission_codes(values: Vec<String>) -> AppResult<Vec<String>> {
    if values.is_empty() {
        return Err(AppError::Validation("角色至少需要绑定一个权限".into()));
    }
    let mut seen = BTreeSet::new();
    for value in values {
        let code = value.trim().to_ascii_lowercase();
        if code.is_empty() || code.len() > 120 {
            return Err(AppError::Validation(
                "权限编码不能为空且不能超过 120 个字符".into(),
            ));
        }
        if !code
            .chars()
            .all(|ch| ch.is_ascii_alphanumeric() || matches!(ch, '_' | '-' | ':' | '.'))
        {
            return Err(AppError::Validation(
                "权限编码只能包含字母、数字、下划线、中划线、冒号和点".into(),
            ));
        }
        seen.insert(code);
    }
    Ok(seen.into_iter().collect())
}

fn validate_positive_id(id: i64, label: &str) -> AppResult<()> {
    if id <= 0 {
        return Err(AppError::Validation(format!("{label} 必须大于 0")));
    }
    Ok(())
}

fn ensure_pending_token(
    status: &str,
    expires_at: &str,
    consumed_at: Option<&str>,
) -> AppResult<()> {
    if status != "pending" || consumed_at.is_some() {
        return Err(AppError::Unauthorized);
    }
    let expires_at = DateTime::parse_from_rfc3339(expires_at)
        .map_err(|err| AppError::Internal(format!("令牌过期时间格式无效：{err}")))?
        .with_timezone(&Utc);
    if expires_at <= Utc::now() {
        return Err(AppError::Unauthorized);
    }
    Ok(())
}

fn validate_role_code(value: &str) -> AppResult<()> {
    if value.is_empty() || value.len() > 80 {
        return Err(AppError::Validation(
            "角色编码不能为空且不能超过 80 个字符".into(),
        ));
    }
    if !value
        .chars()
        .all(|ch| ch.is_ascii_alphanumeric() || matches!(ch, '_' | '-' | '.'))
    {
        return Err(AppError::Validation(
            "角色编码只能包含字母、数字、下划线、中划线和点".into(),
        ));
    }
    Ok(())
}
