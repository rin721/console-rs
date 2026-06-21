use std::sync::Arc;

use axum::Json;
use axum::extract::{Path, State};
use axum::http::HeaderMap;
use axum::response::IntoResponse;

use crate::app::AppError;
use crate::app::AppState;
use crate::domain::iam::{
    AcceptInvitationRequest, ConfirmEmailVerificationRequest, CreateAPITokenRequest,
    CreateRoleRequest, ForgotPasswordRequest, InitialAdminRequest, InviteUserRequest,
    InviteUserResult, LoginRequest, RegisterRequest, RequestEmailVerification,
    ResetPasswordRequest, UpdateOrgUserRequest, UpdateRoleRequest, VerifyMfaRequest,
};
use crate::handler::http::error::HttpResult;
use crate::transport::http::request_context::{
    auth_credential_from_headers, clear_auth_cookies, refresh_cookie, request_context_from_headers,
    session_cookie, set_auth_cookies,
};
use crate::transport::http::route_registry;

pub async fn setup_status(
    State(state): State<Arc<AppState>>,
) -> HttpResult<Json<crate::domain::iam::SetupStatus>> {
    Ok(Json(state.iam.setup_status().await?))
}

pub async fn initial_admin(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Json(payload): Json<InitialAdminRequest>,
) -> HttpResult<impl IntoResponse> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    let (snapshot, tokens) = state.iam.initial_admin(payload, ctx).await?;
    Ok((set_auth_cookies(&state.settings, &tokens), Json(snapshot)))
}

pub async fn login(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Json(payload): Json<LoginRequest>,
) -> HttpResult<impl IntoResponse> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    let (snapshot, tokens) = state.iam.login(payload, ctx).await?;
    Ok((set_auth_cookies(&state.settings, &tokens), Json(snapshot)))
}

pub async fn register(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Json(payload): Json<RegisterRequest>,
) -> HttpResult<Json<crate::domain::iam::NotificationDelivery>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    Ok(Json(state.iam.register(payload, ctx).await?))
}

pub async fn refresh_session(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> HttpResult<impl IntoResponse> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    let (snapshot, tokens) = state
        .iam
        .refresh_session(refresh_cookie(&headers, &state.settings), ctx)
        .await?;
    Ok((set_auth_cookies(&state.settings, &tokens), Json(snapshot)))
}

pub async fn me_session(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> HttpResult<Json<crate::domain::iam::SessionSnapshot>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    Ok(Json(
        state
            .iam
            .current_session(session_cookie(&headers, &state.settings), ctx)
            .await?,
    ))
}

pub async fn logout(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> HttpResult<impl IntoResponse> {
    let result = state
        .iam
        .logout(
            session_cookie(&headers, &state.settings),
            refresh_cookie(&headers, &state.settings),
        )
        .await?;
    Ok((clear_auth_cookies(&state.settings), Json(result)))
}

pub async fn list_organizations(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> HttpResult<Json<Vec<crate::domain::iam::OrganizationSummary>>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    let permission = required_permission(&state, "iam.organizations.list")?;
    Ok(Json(
        state
            .iam
            .list_organizations(
                auth_credential_from_headers(&headers, &state.settings),
                ctx,
                &permission,
            )
            .await?,
    ))
}

pub async fn list_org_users(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(org_id): Path<i64>,
) -> HttpResult<Json<Vec<crate::domain::iam::OrganizationUserSummary>>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    let permission = required_permission(&state, "iam.org-users.list")?;
    Ok(Json(
        state
            .iam
            .list_org_users(
                auth_credential_from_headers(&headers, &state.settings),
                org_id,
                ctx,
                &permission,
            )
            .await?,
    ))
}

pub async fn update_org_user(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path((org_id, user_id)): Path<(i64, i64)>,
    Json(payload): Json<UpdateOrgUserRequest>,
) -> HttpResult<Json<crate::domain::iam::OrganizationUserSummary>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    let permission = required_permission(&state, "iam.org-users.update")?;
    Ok(Json(
        state
            .iam
            .update_org_user(
                auth_credential_from_headers(&headers, &state.settings),
                org_id,
                user_id,
                payload,
                ctx,
                &permission,
            )
            .await?,
    ))
}

pub async fn list_org_roles(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(org_id): Path<i64>,
) -> HttpResult<Json<Vec<crate::domain::iam::RoleSummary>>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    let permission = required_permission(&state, "iam.org-roles.list")?;
    Ok(Json(
        state
            .iam
            .list_org_roles(
                auth_credential_from_headers(&headers, &state.settings),
                org_id,
                ctx,
                &permission,
            )
            .await?,
    ))
}

pub async fn create_org_role(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(org_id): Path<i64>,
    Json(payload): Json<CreateRoleRequest>,
) -> HttpResult<Json<crate::domain::iam::RoleSummary>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    let permission = required_permission(&state, "iam.org-roles.create")?;
    Ok(Json(
        state
            .iam
            .create_org_role(
                auth_credential_from_headers(&headers, &state.settings),
                org_id,
                payload,
                ctx,
                &permission,
            )
            .await?,
    ))
}

pub async fn update_org_role(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path((org_id, role_id)): Path<(i64, i64)>,
    Json(payload): Json<UpdateRoleRequest>,
) -> HttpResult<Json<crate::domain::iam::RoleSummary>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    let permission = required_permission(&state, "iam.org-roles.update")?;
    Ok(Json(
        state
            .iam
            .update_org_role(
                auth_credential_from_headers(&headers, &state.settings),
                org_id,
                role_id,
                payload,
                ctx,
                &permission,
            )
            .await?,
    ))
}

pub async fn delete_org_role(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path((org_id, role_id)): Path<(i64, i64)>,
) -> HttpResult<Json<crate::domain::iam::BooleanResult>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    let permission = required_permission(&state, "iam.org-roles.delete")?;
    Ok(Json(
        state
            .iam
            .delete_org_role(
                auth_credential_from_headers(&headers, &state.settings),
                org_id,
                role_id,
                ctx,
                &permission,
            )
            .await?,
    ))
}

pub async fn list_permissions(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> HttpResult<Json<Vec<crate::domain::iam::PermissionSummary>>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    let permission = required_permission(&state, "iam.permissions.list")?;
    Ok(Json(
        state
            .iam
            .list_permissions(
                auth_credential_from_headers(&headers, &state.settings),
                ctx,
                &permission,
            )
            .await?,
    ))
}

pub async fn create_api_token(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(org_id): Path<i64>,
    Json(payload): Json<CreateAPITokenRequest>,
) -> HttpResult<Json<crate::domain::iam::CreateAPITokenResult>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    let permission = required_permission(&state, "iam.api-tokens.create")?;
    Ok(Json(
        state
            .iam
            .create_api_token(
                auth_credential_from_headers(&headers, &state.settings),
                org_id,
                payload,
                ctx,
                &permission,
            )
            .await?,
    ))
}

pub async fn list_api_tokens(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(org_id): Path<i64>,
) -> HttpResult<Json<Vec<crate::domain::iam::APITokenSummary>>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    let permission = required_permission(&state, "iam.api-tokens.list")?;
    Ok(Json(
        state
            .iam
            .list_api_tokens(
                auth_credential_from_headers(&headers, &state.settings),
                org_id,
                ctx,
                &permission,
            )
            .await?,
    ))
}

pub async fn revoke_api_token(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path((org_id, token_id)): Path<(i64, i64)>,
) -> HttpResult<Json<crate::domain::iam::BooleanResult>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    let permission = required_permission(&state, "iam.api-tokens.revoke")?;
    Ok(Json(
        state
            .iam
            .revoke_api_token(
                auth_credential_from_headers(&headers, &state.settings),
                org_id,
                token_id,
                ctx,
                &permission,
            )
            .await?,
    ))
}

pub async fn invite_user(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(org_id): Path<i64>,
    Json(payload): Json<InviteUserRequest>,
) -> HttpResult<Json<InviteUserResult>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    let permission = required_permission(&state, "iam.invitations.create")?;
    let (invitation, delivery) = state
        .iam
        .invite_user(
            auth_credential_from_headers(&headers, &state.settings),
            org_id,
            payload,
            ctx,
            &permission,
        )
        .await?;
    Ok(Json(InviteUserResult {
        item: invitation,
        delivery,
    }))
}

pub async fn list_invitations(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(org_id): Path<i64>,
) -> HttpResult<Json<Vec<crate::domain::iam::InvitationSummary>>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    let permission = required_permission(&state, "iam.invitations.list")?;
    Ok(Json(
        state
            .iam
            .list_invitations(
                auth_credential_from_headers(&headers, &state.settings),
                org_id,
                ctx,
                &permission,
            )
            .await?,
    ))
}

pub async fn revoke_invitation(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path((org_id, invitation_id)): Path<(i64, i64)>,
) -> HttpResult<Json<crate::domain::iam::BooleanResult>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    let permission = required_permission(&state, "iam.invitations.revoke")?;
    Ok(Json(
        state
            .iam
            .revoke_invitation(
                auth_credential_from_headers(&headers, &state.settings),
                org_id,
                invitation_id,
                ctx,
                &permission,
            )
            .await?,
    ))
}

pub async fn accept_invitation(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Json(payload): Json<AcceptInvitationRequest>,
) -> HttpResult<impl IntoResponse> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    let (snapshot, tokens) = state.iam.accept_invitation(payload, ctx).await?;
    Ok((set_auth_cookies(&state.settings, &tokens), Json(snapshot)))
}

pub async fn forgot_password(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<ForgotPasswordRequest>,
) -> HttpResult<Json<crate::domain::iam::NotificationDelivery>> {
    Ok(Json(state.iam.forgot_password(payload).await?))
}

pub async fn reset_password(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<ResetPasswordRequest>,
) -> HttpResult<Json<crate::domain::iam::BooleanResult>> {
    Ok(Json(state.iam.reset_password(payload).await?))
}

pub async fn request_email_verification(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<RequestEmailVerification>,
) -> HttpResult<Json<crate::domain::iam::NotificationDelivery>> {
    Ok(Json(state.iam.request_email_verification(payload).await?))
}

pub async fn confirm_email_verification(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<ConfirmEmailVerificationRequest>,
) -> HttpResult<Json<crate::domain::iam::BooleanResult>> {
    Ok(Json(
        state.iam.confirm_email_verification(payload.token).await?,
    ))
}

pub async fn setup_mfa(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> HttpResult<Json<crate::domain::iam::MfaSetupResult>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    Ok(Json(
        state
            .iam
            .setup_mfa(auth_credential_from_headers(&headers, &state.settings), ctx)
            .await?,
    ))
}

pub async fn list_mfa_factors(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> HttpResult<Json<Vec<crate::domain::iam::MfaFactorSummary>>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    Ok(Json(
        state
            .iam
            .list_mfa_factors(auth_credential_from_headers(&headers, &state.settings), ctx)
            .await?,
    ))
}

pub async fn verify_mfa(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Json(payload): Json<VerifyMfaRequest>,
) -> HttpResult<Json<crate::domain::iam::MfaVerifyResult>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    Ok(Json(
        state
            .iam
            .verify_mfa(
                auth_credential_from_headers(&headers, &state.settings),
                ctx,
                payload,
            )
            .await?,
    ))
}

pub async fn list_mfa_recovery_codes(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> HttpResult<Json<Vec<crate::domain::iam::MfaRecoveryCodeSummary>>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    Ok(Json(
        state
            .iam
            .list_mfa_recovery_codes(auth_credential_from_headers(&headers, &state.settings), ctx)
            .await?,
    ))
}

pub async fn rotate_mfa_recovery_codes(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> HttpResult<Json<crate::domain::iam::MfaRecoveryCodesResult>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    Ok(Json(
        state
            .iam
            .rotate_mfa_recovery_codes(auth_credential_from_headers(&headers, &state.settings), ctx)
            .await?,
    ))
}

pub async fn revoke_mfa(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(factor_id): Path<i64>,
) -> HttpResult<Json<crate::domain::iam::BooleanResult>> {
    let ctx = request_context_from_headers(&headers, &state.settings);
    Ok(Json(
        state
            .iam
            .revoke_mfa(
                auth_credential_from_headers(&headers, &state.settings),
                ctx,
                factor_id,
            )
            .await?,
    ))
}

fn required_permission(state: &AppState, route_id: &str) -> HttpResult<String> {
    Ok(
        route_registry::required_permission(&state.settings, route_id).ok_or_else(|| {
            AppError::Internal(format!("路由 {route_id} 缺少权限元数据，无法执行授权"))
        })?,
    )
}
