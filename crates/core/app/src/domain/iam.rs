use serde::{Deserialize, Serialize};

pub const API_TOKEN_SECRET_PREFIX: &str = "api_token";
pub const INVITATION_TOKEN_SECRET_PREFIX: &str = "invitation_token";
pub const PASSWORD_RESET_TOKEN_SECRET_PREFIX: &str = "password_reset_token";
pub const EMAIL_VERIFICATION_TOKEN_SECRET_PREFIX: &str = "email_verify_token";
pub const SESSION_TOKEN_SECRET_PREFIX: &str = "session_token";
pub const REFRESH_TOKEN_SECRET_PREFIX: &str = "refresh_token";
pub const MFA_RECOVERY_CODE_SECRET_PREFIX: &str = "mfa_recovery_code";

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct User {
    pub id: i64,
    pub email: String,
    pub display_name: String,
    pub status: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct Organization {
    pub id: i64,
    pub code: String,
    pub name: String,
    pub scope: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct OrganizationSummary {
    pub id: i64,
    pub code: String,
    pub name: String,
    pub scope: String,
    pub status: String,
    pub created_at: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct OrganizationUserSummary {
    pub id: i64,
    pub email: String,
    pub display_name: String,
    pub status: String,
    pub email_verified_at: Option<String>,
    pub role_codes: Vec<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct RoleSummary {
    pub id: i64,
    pub org_id: Option<i64>,
    pub code: String,
    pub name: String,
    pub scope: String,
    pub system_builtin: bool,
    pub permissions: Vec<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct CreateRoleRequest {
    pub code: String,
    pub name: String,
    pub permission_codes: Vec<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct UpdateRoleRequest {
    pub name: String,
    pub permission_codes: Vec<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct UpdateOrgUserRequest {
    pub display_name: String,
    pub status: String,
    pub role_codes: Vec<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct PermissionSummary {
    pub id: i64,
    pub product_code: String,
    pub scope: String,
    pub code: String,
    pub name: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct InitialAdminRequest {
    pub email: String,
    pub password: String,
    pub display_name: String,
    pub organization_code: String,
    pub organization_name: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct RegisterRequest {
    pub email: String,
    pub password: String,
    pub display_name: String,
    pub organization_code: String,
    pub organization_name: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct LoginRequest {
    pub identifier: String,
    pub password: String,
    #[serde(default, alias = "mfaCode")]
    pub mfa_code: Option<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct SetupStatus {
    pub initialized: bool,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct SessionSnapshot {
    pub authenticated: bool,
    pub user: Option<User>,
    pub organization: Option<Organization>,
    pub product_code: String,
    pub client_type: String,
    pub permissions: Vec<String>,
    pub mfa_enabled: bool,
    pub expires_at: Option<String>,
    pub refresh_expires_at: Option<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct LogoutResult {
    pub logged_out: bool,
}

#[derive(Clone, Debug)]
pub struct SessionTokens {
    pub raw_session: String,
    pub raw_refresh: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct CreateAPITokenRequest {
    pub expires_in_days: Option<i64>,
    pub remark: Option<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct APITokenSummary {
    pub id: i64,
    pub org_id: i64,
    pub user_id: i64,
    pub token_prefix: String,
    pub status: String,
    pub expires_at: Option<String>,
    pub created_at: String,
    pub revoked_at: Option<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct CreateAPITokenResult {
    pub item: APITokenSummary,
    pub token: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct BooleanResult {
    pub revoked: Option<bool>,
    pub accepted: Option<bool>,
    pub reset: Option<bool>,
    pub verified: Option<bool>,
    pub deleted: Option<bool>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct MfaFactorSummary {
    pub id: i64,
    pub kind: String,
    pub status: String,
    pub created_at: String,
    pub verified_at: Option<String>,
    pub revoked_at: Option<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct MfaSetupResult {
    pub factor: MfaFactorSummary,
    pub secret: String,
    pub otpauth_url: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct VerifyMfaRequest {
    pub code: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct MfaVerifyResult {
    pub verified: bool,
    pub recovery_codes: Vec<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct MfaRecoveryCodeSummary {
    pub id: i64,
    pub code_prefix: String,
    pub status: String,
    pub created_at: String,
    pub used_at: Option<String>,
    pub revoked_at: Option<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct MfaRecoveryCodesResult {
    pub items: Vec<MfaRecoveryCodeSummary>,
    pub recovery_codes: Vec<String>,
}

#[derive(Clone, Debug)]
pub struct StoredMfaFactor {
    pub id: i64,
    pub user_id: i64,
    pub kind: String,
    pub secret_ciphertext: String,
    pub status: String,
    pub created_at: String,
    pub verified_at: Option<String>,
    pub revoked_at: Option<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct InviteUserRequest {
    pub email: String,
    pub role_code: Option<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct InvitationSummary {
    pub id: i64,
    pub org_id: i64,
    pub email: String,
    pub role_code: String,
    pub status: String,
    pub expires_at: String,
    pub created_at: String,
    pub accepted_at: Option<String>,
    pub revoked_at: Option<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct AcceptInvitationRequest {
    pub token: String,
    pub password: String,
    pub display_name: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct InviteUserResult {
    pub item: InvitationSummary,
    pub delivery: NotificationDelivery,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct NotificationDelivery {
    pub accepted: bool,
    pub channel: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct ForgotPasswordRequest {
    pub email: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct ResetPasswordRequest {
    pub token: String,
    pub password: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct RequestEmailVerification {
    pub email: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct ConfirmEmailVerificationRequest {
    pub token: String,
}

#[derive(Clone, Debug)]
pub struct RequestContext {
    pub product_code: String,
    pub client_type: String,
}

#[derive(Clone, Debug, Default)]
pub struct AuthCredential {
    pub raw_session: Option<String>,
    pub raw_api_token: Option<String>,
}

#[derive(Clone, Debug)]
pub struct StoredUser {
    pub id: i64,
    pub email: String,
    pub display_name: String,
    pub password_hash: String,
    pub status: String,
}

#[derive(Clone, Debug)]
pub struct SessionRecord {
    pub id: String,
    pub token_hash: String,
    pub refresh_token_hash: Option<String>,
    pub user: User,
    pub organization: Organization,
    pub product_code: String,
    pub client_type: String,
    pub expires_at: String,
    pub refresh_expires_at: Option<String>,
    pub revoked_at: Option<String>,
}

#[derive(Clone, Debug)]
pub struct APITokenAuthRecord {
    pub id: i64,
    pub user: User,
    pub organization: Organization,
    pub status: String,
    pub expires_at: Option<String>,
    pub revoked_at: Option<String>,
}

#[derive(Clone, Debug)]
pub struct InvitationAcceptRecord {
    pub id: i64,
    pub org_id: i64,
    pub email: String,
    pub role_code: String,
    pub status: String,
    pub expires_at: String,
    pub accepted_at: Option<String>,
    pub revoked_at: Option<String>,
}

#[derive(Clone, Debug)]
pub struct PasswordResetRecord {
    pub id: i64,
    pub user_id: i64,
    pub status: String,
    pub expires_at: String,
    pub used_at: Option<String>,
}

#[derive(Clone, Debug)]
pub struct EmailVerificationRecord {
    pub id: i64,
    pub user_id: i64,
    pub status: String,
    pub expires_at: String,
    pub verified_at: Option<String>,
}
