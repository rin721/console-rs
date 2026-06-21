export type Locale = "zh-CN" | "en";

export type PublicSettings = {
  product_name: string;
  product_code: string;
  default_locale: Locale;
  supported_locales: Locale[];
  auth: {
    self_signup_enabled: boolean;
    session_cookie_name: string;
    refresh_cookie_name: string;
    product_header: string;
    client_type_header: string;
    default_client_type: string;
    csrf_enabled: boolean;
    csrf_cookie_name: string;
    csrf_header_name: string;
  };
};

export type SetupStatus = {
  completed: boolean;
  has_initial_admin: boolean;
  required_steps: Array<{
    key: string;
    title: string;
    status: string;
  }>;
};
export type IamSetupStatus = { initialized: boolean };

export type SetupConfigCheckSummary = {
  ready: boolean;
  checks: SetupConfigCheck[];
};

export type SetupConfigCheck = {
  key: string;
  title: string;
  status: "ok" | "warning" | "error";
  severity: "info" | "warning" | "error";
  message: string;
};

export type User = {
  id: number;
  email: string;
  display_name: string;
  status: string;
};

export type Organization = {
  id: number;
  code: string;
  name: string;
  scope: string;
};

export type SessionSnapshot = {
  authenticated: boolean;
  user: User | null;
  organization: Organization | null;
  product_code: string;
  client_type: string;
  permissions: string[];
  mfa_enabled: boolean;
  expires_at: string | null;
  refresh_expires_at: string | null;
};

export type SetupSchema = {
  locale: string;
  steps: Array<{
    key: string;
    title: string;
    fields: Array<{
      key: string;
      label: string;
      kind: string;
      required: boolean;
      sensitive: boolean;
    }>;
  }>;
};

export type RegisterRequest = {
  email: string;
  password: string;
  display_name: string;
  organization_code: string;
  organization_name: string;
};

export type SetupRun = {
  id: string;
  status: string;
  reason: string | null;
  created_at: string;
  updated_at: string;
};

export type CompleteSetupRequest = {
  confirm: boolean;
  run_id?: string;
};

export type SetupStepLog = {
  step_key: string;
  status: string;
  message: string;
  created_at: string;
};

export type ServerStatus = {
  source: string;
  collected_at: string;
  started_at: string;
  uptime_seconds: number;
  process_id: number;
  os: string;
  arch: string;
  available_parallelism: number;
  product_code: string;
  version: string;
  database_driver: string;
  metrics: ServerResourceMetrics;
};

export type ServerResourceMetrics = {
  source: string;
  cpu_usage_percent: number;
  process_cpu_usage_percent: number;
  total_memory_bytes: number;
  used_memory_bytes: number;
  available_memory_bytes: number;
  process_memory_bytes: number;
  process_virtual_memory_bytes: number;
  total_swap_bytes: number;
  used_swap_bytes: number;
  total_disk_bytes: number;
  used_disk_bytes: number;
  available_disk_bytes: number;
  disk_count: number;
  network_interface_count: number;
  network_received_bytes: number;
  network_transmitted_bytes: number;
  system_uptime_seconds: number;
  system_boot_time_seconds: number;
  load_average_one: number;
  load_average_five: number;
  load_average_fifteen: number;
};

export type ApiCatalogGroup = {
  tag: string;
  items: Array<{
    id: string;
    method: string;
    path: string;
    tag: string;
    summary: string;
    access: string;
    permission: string | null;
    scope: string;
    product_code: string;
  }>;
};

export type SystemMenu = {
  code: string;
  title: string;
  path: string;
  permission: string | null;
  scope: string;
  sort_order: number;
};

export type OrganizationSummary = Organization & {
  status: string;
  created_at: string;
};

export type OrganizationUserSummary = User & {
  email_verified_at: string | null;
  role_codes: string[];
};

export type UpdateOrgUserRequest = {
  display_name: string;
  status: "active" | "disabled";
  role_codes: string[];
};

export type RoleSummary = {
  id: number;
  org_id: number | null;
  code: string;
  name: string;
  scope: string;
  system_builtin: boolean;
  permissions: string[];
};

export type CreateRoleRequest = {
  code: string;
  name: string;
  permission_codes: string[];
};

export type UpdateRoleRequest = {
  name: string;
  permission_codes: string[];
};

export type PermissionSummary = {
  id: number;
  product_code: string;
  scope: string;
  code: string;
  name: string;
};

export type APITokenSummary = {
  id: number;
  org_id: number;
  user_id: number;
  token_prefix: string;
  status: string;
  expires_at: string | null;
  created_at: string;
  revoked_at: string | null;
};

export type CreateAPITokenRequest = {
  expires_in_days?: number | null;
  remark?: string | null;
};

export type CreateAPITokenResult = {
  item: APITokenSummary;
  token: string;
};

export type InvitationSummary = {
  id: number;
  org_id: number;
  email: string;
  role_code: string;
  status: string;
  expires_at: string;
  created_at: string;
};

export type InviteUserRequest = {
  email: string;
  role_code?: string | null;
};

export type InviteUserResult = {
  item: InvitationSummary;
};

export type AcceptInvitationRequest = {
  token: string;
  password: string;
  display_name: string;
};

export type ForgotPasswordRequest = {
  email: string;
};

export type ResetPasswordRequest = {
  token: string;
  password: string;
};

export type RequestEmailVerification = {
  email: string;
};

export type ConfirmEmailVerificationRequest = {
  token: string;
};

export type NotificationDelivery = {
  accepted: boolean;
  channel: string;
};

export type BooleanResult = {
  revoked?: boolean | null;
  accepted?: boolean | null;
  reset?: boolean | null;
  verified?: boolean | null;
  deleted?: boolean | null;
};

export type MfaFactorSummary = {
  id: number;
  kind: string;
  status: string;
  created_at: string;
  verified_at: string | null;
  revoked_at: string | null;
};

export type MfaSetupResult = {
  factor: MfaFactorSummary;
  secret: string;
  otpauth_url: string;
};

export type VerifyMfaRequest = {
  code: string;
};

export type MfaVerifyResult = {
  verified: boolean;
  recovery_codes: string[];
};

export type MfaRecoveryCodeSummary = {
  id: number;
  code_prefix: string;
  status: string;
  created_at: string;
  used_at: string | null;
  revoked_at: string | null;
};

export type MfaRecoveryCodesResult = {
  items: MfaRecoveryCodeSummary[];
  recovery_codes: string[];
};

export type OperationRecord = {
  id: number;
  actor_user_id: number | null;
  method: string;
  path: string;
  status: number;
  created_at: string;
};

export type OperationRecordCountBucket = {
  key: string;
  count: number;
};

export type OperationRecordPathBucket = {
  path: string;
  count: number;
  error_count: number;
  last_seen_at: string | null;
};

export type OperationRecordSummary = {
  generated_at: string;
  total_count: number;
  success_count: number;
  redirect_count: number;
  client_error_count: number;
  server_error_count: number;
  other_count: number;
  top_limit: number;
  by_method: OperationRecordCountBucket[];
  by_status_class: OperationRecordCountBucket[];
  top_paths: OperationRecordPathBucket[];
};

export type SystemConfigEntry = {
  key: string;
  value: unknown;
  updated_at: string;
};

export type SystemDictionaryEntry = {
  id: number;
  code: string;
  name: string;
  created_at: string;
};

export type SystemParameterEntry = {
  id: number;
  key: string;
  name: string;
  value: string;
  created_at: string;
  updated_at: string;
};

export type VersionPackageEntry = {
  id: number;
  version_name: string;
  version_code: string;
  manifest: unknown;
  status: string;
  published_at: string | null;
  retired_at: string | null;
  created_at: string;
};

export type VersionPackageActionResult = {
  event_id: number;
  previous_active_id: number | null;
  package: VersionPackageEntry;
};

export type VersionReleaseEventEntry = {
  id: number;
  package_id: number;
  previous_active_id: number | null;
  action: string;
  status: string;
  reason: string | null;
  created_at: string;
};

export type MediaAssetEntry = {
  id: number;
  category: string | null;
  display_name: string;
  storage_key: string;
  mime_type: string;
  size_bytes: number;
  created_at: string;
};

export type StorageObjectEntry = {
  storage_key: string;
  size_bytes: number;
  updated_at: string | null;
  e_tag: string | null;
};

export type TrafficProbeTarget = {
  id: number;
  name: string;
  url: string;
  expected_status: number;
  status: string;
  created_at: string;
};

export type TrafficProbeResult = {
  id: number;
  target_id: number;
  status: string;
  detail: unknown;
  probed_at: string;
};

export type TrafficProbeAlert = {
  id: number;
  target_id: number;
  result_id: number;
  severity: string;
  status: string;
  reason: string;
  detail: unknown;
  opened_at: string;
  acknowledged_at: string | null;
  resolved_at: string | null;
};

export type TrafficProbeAlertActionResult = {
  updated: boolean;
};
