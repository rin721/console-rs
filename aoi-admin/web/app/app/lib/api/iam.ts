import { API_ENDPOINTS } from "./endpoints";
import { apiClient } from "./runtime";
import type {
  IAMAPITokenPage,
  IAMAPITokenRevokeResult,
  IAMAuditLog,
  IAMCreateAPITokenInput,
  IAMCreateAPITokenResult,
  IAMCreateOrganizationInput,
  IAMCreateRoleInput,
  IAMInvitation,
  IAMInvitationRevokeResult,
  IAMInviteUserInput,
  IAMOrganization,
  IAMOrganizationUser,
  IAMOrganizationPage,
  IAMOrganizationUserPage,
  IAMPermission,
  IAMRole,
  IAMSessionPage,
  IAMSessionRevokeResult,
  IAMUpdateOrganizationInput,
  IAMUpdateRoleInput,
  IAMUpdateUserInput,
  NotificationDelivery,
} from "./types";

type RequestOptions = {
  signal?: AbortSignal;
};

export type IAMOrganizationListQuery = {
  code?: string;
  desc?: boolean;
  keyword?: string;
  name?: string;
  orderKey?: string;
  page?: number;
  pageSize?: number;
  status?: string;
};

export type IAMUserListQuery = {
  desc?: boolean;
  displayName?: string;
  email?: string;
  keyword?: string;
  orderKey?: string;
  page?: number;
  pageSize?: number;
  roleCode?: string;
  status?: string;
  username?: string;
};

export type IAMSessionListQuery = {
  clientType?: string;
  desc?: boolean;
  ipAddress?: string;
  keyword?: string;
  orderKey?: string;
  page?: number;
  pageSize?: number;
  productCode?: string;
  scope?: string;
  status?: string;
  userId?: number | string;
};

export type IAMAPITokenListQuery = {
  page?: number;
  pageSize?: number;
  status?: string;
  userId?: number | string;
};

export type IAMAuditLogListQuery = {
  action?: string;
  cursor?: number | string;
  from?: string;
  limit?: number;
  to?: string;
  userId?: number | string;
};

export const iamApi = {
  createAPIToken: (
    orgId: number | string,
    input: IAMCreateAPITokenInput,
    options: RequestOptions = {},
  ) =>
    apiClient.request<IAMCreateAPITokenResult>(API_ENDPOINTS.orgs.apiTokens(orgId), {
      body: input,
      method: "POST",
      signal: options.signal,
    }),
  createOrganization: (input: IAMCreateOrganizationInput, options: RequestOptions = {}) =>
    apiClient.request<IAMOrganization>(API_ENDPOINTS.orgs.collection, {
      body: input,
      method: "POST",
      signal: options.signal,
    }),
  createRole: (orgId: number | string, input: IAMCreateRoleInput, options: RequestOptions = {}) =>
    apiClient.request<IAMRole>(API_ENDPOINTS.orgs.roles(orgId), {
      body: input,
      method: "POST",
      signal: options.signal,
    }),
  inviteUser: (orgId: number | string, input: IAMInviteUserInput, options: RequestOptions = {}) =>
    apiClient.request<NotificationDelivery>(API_ENDPOINTS.orgs.userInvitations(orgId), {
      body: input,
      method: "POST",
      signal: options.signal,
    }),
  listAuditLogs: (
    orgId: number | string,
    query: IAMAuditLogListQuery = {},
    options: RequestOptions = {},
  ) =>
    apiClient.request<IAMAuditLog[]>(API_ENDPOINTS.orgs.auditLogs(orgId), {
      query,
      signal: options.signal,
    }),
  listAPITokens: (
    orgId: number | string,
    query: IAMAPITokenListQuery = {},
    options: RequestOptions = {},
  ) =>
    apiClient.request<IAMAPITokenPage>(API_ENDPOINTS.orgs.apiTokens(orgId), {
      query,
      signal: options.signal,
    }),
  listInvitations: (orgId: number | string, options: RequestOptions = {}) =>
    apiClient.request<IAMInvitation[]>(API_ENDPOINTS.orgs.invitations(orgId), {
      signal: options.signal,
    }),
  listOrganizations: (query: IAMOrganizationListQuery = {}, options: RequestOptions = {}) =>
    apiClient.request<IAMOrganizationPage>(API_ENDPOINTS.orgs.collection, {
      query,
      signal: options.signal,
    }),
  listPermissions: (orgId: number | string, options: RequestOptions = {}) =>
    apiClient.request<IAMPermission[]>(API_ENDPOINTS.orgs.permissions(orgId), {
      signal: options.signal,
    }),
  listRoles: (orgId: number | string, options: RequestOptions = {}) =>
    apiClient.request<IAMRole[]>(API_ENDPOINTS.orgs.roles(orgId), {
      signal: options.signal,
    }),
  listSessions: (
    orgId: number | string,
    query: IAMSessionListQuery = {},
    options: RequestOptions = {},
  ) =>
    apiClient.request<IAMSessionPage>(API_ENDPOINTS.orgs.sessions(orgId), {
      query,
      signal: options.signal,
    }),
  revokeSession: (
    orgId: number | string,
    sessionId: number | string,
    options: RequestOptions = {},
  ) =>
    apiClient.request<IAMSessionRevokeResult>(API_ENDPOINTS.orgs.session(orgId, sessionId), {
      method: "DELETE",
      signal: options.signal,
    }),
  revokeAPIToken: (
    orgId: number | string,
    tokenId: number | string,
    options: RequestOptions = {},
  ) =>
    apiClient.request<IAMAPITokenRevokeResult>(API_ENDPOINTS.orgs.apiToken(orgId, tokenId), {
      method: "DELETE",
      signal: options.signal,
    }),
  revokeInvitation: (
    orgId: number | string,
    invitationId: number | string,
    options: RequestOptions = {},
  ) =>
    apiClient.request<IAMInvitationRevokeResult>(
      API_ENDPOINTS.orgs.invitation(orgId, invitationId),
      {
        method: "DELETE",
        signal: options.signal,
      },
    ),
  updateUser: (
    orgId: number | string,
    userId: number | string,
    input: IAMUpdateUserInput,
    options: RequestOptions = {},
  ) =>
    apiClient.request<IAMOrganizationUser>(API_ENDPOINTS.orgs.user(orgId, userId), {
      body: input,
      method: "PATCH",
      signal: options.signal,
    }),
  updateRole: (
    orgId: number | string,
    roleId: number | string,
    input: IAMUpdateRoleInput,
    options: RequestOptions = {},
  ) =>
    apiClient.request<IAMRole>(API_ENDPOINTS.orgs.role(orgId, roleId), {
      body: input,
      method: "PATCH",
      signal: options.signal,
    }),
  updateOrganization: (
    orgId: number | string,
    input: IAMUpdateOrganizationInput,
    options: RequestOptions = {},
  ) =>
    apiClient.request<IAMOrganization>(API_ENDPOINTS.orgs.item(orgId), {
      body: input,
      method: "PATCH",
      signal: options.signal,
    }),
  listUsers: (orgId: number | string, query: IAMUserListQuery = {}, options: RequestOptions = {}) =>
    apiClient.request<IAMOrganizationUserPage>(API_ENDPOINTS.orgs.users(orgId), {
      query,
      signal: options.signal,
    }),
};
