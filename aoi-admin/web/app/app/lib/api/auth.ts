import { API_ENDPOINTS } from "./endpoints";
import type { ApiRequestOptions } from "./client";
import { apiClient } from "./runtime";
import type {
  AcceptInvitationRequest,
  AcceptInvitationResult,
  CaptchaChallenge,
  CurrentUser,
  ForgotPasswordRequest,
  LoginRequest,
  MFASetupPayload,
  MFAVerifyResult,
  NotificationDelivery,
  Organization,
  ResetPasswordRequest,
  SessionSnapshot,
  SignupRequest,
  SignupResult,
} from "./types";

type RequestOptions = Pick<ApiRequestOptions, "signal">;

export const authApi = {
  acceptInvitation: (token: string, body: AcceptInvitationRequest) =>
    apiClient.request<AcceptInvitationResult>(API_ENDPOINTS.invitations.accept(token), {
      auth: false,
      body,
      method: "POST",
    }),
  captcha: (options?: RequestOptions) =>
    apiClient.request<CaptchaChallenge>(API_ENDPOINTS.auth.captcha, {
      auth: false,
      signal: options?.signal,
    }),
  forgotPassword: (body: ForgotPasswordRequest) =>
    apiClient.request<NotificationDelivery>(API_ENDPOINTS.auth.forgotPassword, {
      auth: false,
      body,
      method: "POST",
    }),
  login: (body: LoginRequest) =>
    apiClient.request<SessionSnapshot>(API_ENDPOINTS.auth.login, {
      auth: false,
      body,
      method: "POST",
    }),
  signup: (body: SignupRequest) =>
    apiClient.request<SignupResult>(API_ENDPOINTS.auth.signup, {
      auth: false,
      body,
      method: "POST",
    }),
  confirmEmailVerification: (token: string) =>
    apiClient.request<SessionSnapshot>(API_ENDPOINTS.auth.emailVerificationConfirm(token), {
      auth: false,
      method: "POST",
    }),
  session: (options?: RequestOptions) =>
    apiClient.request<SessionSnapshot>(API_ENDPOINTS.me.session, options),
  getMe: (options?: RequestOptions) =>
    apiClient.request<CurrentUser>(API_ENDPOINTS.me.profile, options),
  listMyOrganizations: (options?: RequestOptions) =>
    apiClient.request<Organization[]>(API_ENDPOINTS.me.organizations, options),
  logout: () =>
    apiClient.request<{ loggedOut: boolean }>(API_ENDPOINTS.auth.logout, {
      method: "POST",
      retryAuth: false,
    }),
  setupMFA: () =>
    apiClient.request<MFASetupPayload>(API_ENDPOINTS.auth.mfaSetup, {
      method: "POST",
    }),
  resetPassword: (body: ResetPasswordRequest) =>
    apiClient.request<{ reset: boolean }>(API_ENDPOINTS.auth.passwordReset, {
      auth: false,
      body,
      method: "POST",
    }),
  switchOrg: (orgId: number | string) =>
    apiClient.request<SessionSnapshot>(API_ENDPOINTS.auth.switchOrg, {
      body: { orgId: numericID(orgId) },
      method: "POST",
    }),
  verifyMFA: (code: string) =>
    apiClient.request<MFAVerifyResult>(API_ENDPOINTS.auth.mfaVerify, {
      body: { code },
      method: "POST",
    }),
};

function numericID(value: number | string) {
  if (typeof value === "number") {
    return value;
  }
  const parsed = Number(value);
  return Number.isSafeInteger(parsed) ? parsed : value;
}
