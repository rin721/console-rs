export type ApiResult<T> = {
  code: number | string;
  data?: T;
  message?: string;
  messageArgs?: Record<string, unknown>;
  messageKey?: string;
  serverTime?: string;
  traceId?: string;
};

type BackendString<T extends string> = T | (string & Record<never, never>);

export type SessionSnapshot = {
  accessExpiresAt?: string;
  clientType: string;
  orgId: number | string;
  productCode: string;
  refreshExpiresAt?: string;
  sessionId: number | string;
  userId: number | string;
};

export type CaptchaChallenge = {
  captchaId?: string;
  enabled: boolean;
  expiresAt?: string;
  image?: string;
};

export type LoginRequest = {
  captchaCode?: string;
  captchaId?: string;
  identifier: string;
  mfaCode?: string;
  orgCode?: string;
  password: string;
};

export type SignupRequest = {
  displayName?: string;
  email: string;
  orgCode: string;
  orgName: string;
  password: string;
  username: string;
};

export type RegistrationMode =
  | "disabled"
  | "direct"
  | "email_verification"
  | "invite_only";

export type SignupStatus = "authenticated" | "verification_pending";

export type SignupResult = {
  delivery?: NotificationDelivery;
  session?: SessionSnapshot;
  status: BackendString<SignupStatus>;
};

export type ForgotPasswordRequest = {
  email: string;
};

export type NotificationDelivery = {
  debug?: boolean;
  token?: string;
  url?: string;
};

export type ResetPasswordRequest = {
  newPassword: string;
  token: string;
};

export type AcceptInvitationRequest = {
  displayName?: string;
  password: string;
  username: string;
};

export type AcceptInvitationResult = {
  email?: string;
  orgId?: number | string;
  sessionId?: number | string;
  userId?: number | string;
};

export type CurrentUser = {
  displayName?: string;
  email?: string;
  id: number | string;
  mfaEnabled?: boolean;
  username?: string;
};

export type Organization = {
  code?: string;
  id: number | string;
  name?: string;
};

export type MFASetupPayload = {
  otpauthUrl: string;
  secret: string;
};

export type MFAVerifyResult = {
  verified: boolean;
};

export type PluginCapability = {
  description?: string;
  input_schema?: unknown;
  name: string;
  output_schema?: unknown;
  permissions?: string[];
  scope?: string;
  secret_policy?: string;
  version?: string;
};

export type PluginSnapshot = {
  capabilities?: PluginCapability[];
  created_at: string;
  endpoint?: string;
  hooks?: string[];
  instance_id: string;
  last_heartbeat_at: string;
  lease_expires_at?: string;
  lease_ttl_seconds?: number;
  metadata?: Record<string, string>;
  name: string;
  owner_host?: string;
  permissions?: string[];
  plugin_id: string;
  protocol: string;
  registered_at: string;
  runtime_status?: string;
  schema_version?: string;
  status: string;
  transport?: string;
  updated_at: string;
  version: string;
};

export type PluginHealthStatus = {
  error?: string;
  instance_id?: string;
  last_heartbeat_at: string;
  lease_expires_at?: string;
  plugin_id: string;
  runtime_status?: string;
  status: string;
};

export type PluginCapabilitiesResponse = {
  capabilities: PluginCapability[];
};

export type IAMStatus = BackendString<
  "active" | "disabled" | "expired" | "pending" | "revoked" | "used"
>;

export type IAMOrganization = {
  code: string;
  createdAt: string;
  id: number | string;
  name: string;
  status: IAMStatus;
  updatedAt: string;
};

export type IAMOrganizationPage = {
  items: IAMOrganization[];
  page: number;
  pageSize: number;
  storageStatus: string;
  total: number;
};

export type IAMCreateOrganizationInput = {
  code: string;
  name: string;
};

export type IAMUpdateOrganizationInput = {
  name: string;
};

export type IAMUser = {
  createdAt: string;
  displayName: string;
  email: string;
  id: number | string;
  lastLoginAt?: string | null;
  lockedUntil?: string | null;
  mfaEnabled: boolean;
  status: IAMStatus;
  updatedAt: string;
  username: string;
};

export type IAMOrganizationUser = {
  membershipStatus: IAMStatus;
  roles: string[];
  user: IAMUser;
};

export type IAMOrganizationUserPage = {
  items: IAMOrganizationUser[];
  page: number;
  pageSize: number;
  storageStatus: string;
  total: number;
};

export type IAMInvitation = {
  acceptedBy?: number | string | null;
  createdAt: string;
  email: string;
  expiresAt: string;
  id: number | string;
  invitedBy: number | string;
  orgId: number | string;
  roleCode: string;
  status: IAMStatus;
  updatedAt: string;
};

export type IAMInviteUserInput = {
  email: string;
  roleCode: string;
};

export type IAMUpdateUserInput = {
  roles?: string[];
  status?: string;
};

export type IAMCreateRoleInput = {
  code: string;
  description?: string;
  name: string;
  permissions: string[];
};

export type IAMUpdateRoleInput = {
  description?: string;
  name?: string;
  permissions?: string[];
};

export type IAMInvitationRevokeResult = {
  revoked: boolean;
};

export type IAMRole = {
  code: string;
  createdAt: string;
  description: string;
  id: number | string;
  name: string;
  orgId: number | string;
  permissions?: string[];
  system: boolean;
  updatedAt: string;
};

export type IAMPermission = {
  code: string;
  createdAt: string;
  description: string;
  id: number | string;
  name: string;
  updatedAt: string;
};

export type IAMSession = {
  clientType: string;
  createdAt: string;
  expiresAt: string;
  id: number | string;
  ipAddress: string;
  lastUsedAt?: string | null;
  orgId: number | string;
  productCode: string;
  revokedAt?: string | null;
  updatedAt: string;
  userAgent: string;
  userId: number | string;
};

export type IAMSessionPage = {
  items: IAMSession[];
  page: number;
  pageSize: number;
  storageStatus: string;
  total: number;
};

export type IAMSessionRevokeResult = {
  revoked: boolean;
};

export type IAMAPIToken = {
  createdAt: string;
  createdBy: number | string;
  expiresAt?: string | null;
  id: number | string;
  lastUsedAt?: string | null;
  lastUsedIpAddress: string;
  orgId: number | string;
  remark: string;
  revokedAt?: string | null;
  revokedBy?: number | string | null;
  roleCode: string;
  status: IAMStatus;
  tokenPrefix: string;
  updatedAt: string;
  userDisplayName: string;
  userId: number | string;
  username: string;
};

export type IAMAPITokenPage = {
  items: IAMAPIToken[];
  page: number;
  pageSize: number;
  storageStatus: string;
  total: number;
};

export type IAMCreateAPITokenInput = {
  days?: number;
  remark?: string;
  roleCode: string;
  userId: number | string;
};

export type IAMCreateAPITokenResult = {
  item: IAMAPIToken;
  token: string;
};

export type IAMAPITokenRevokeResult = {
  revoked: boolean;
};

export type IAMAuditLog = {
  action: string;
  clientType: string;
  createdAt: string;
  id: number | string;
  ipAddress: string;
  metadata: string;
  orgId?: number | string | null;
  productCode: string;
  resource: string;
  resourceId: string;
  userAgent: string;
  userId?: number | string | null;
};

export type HealthStatus = {
  status: string;
};

export type ReadyStatus = {
  checks?: Record<string, string>;
  status: string;
};

export type SystemPublicBrandSettings = {
  productCode: string;
  productName: string;
  versionName: string;
};

export type SystemPublicAuthSettings = {
  clientTypeHeader: string;
  csrfCookieName: string;
  csrfEnabled: boolean;
  csrfHeaderName: string;
  defaultClientType: string;
  defaultProductCode: string;
  productHeader: string;
  registrationMode: BackendString<RegistrationMode>;
};

export type SystemPublicSettings = {
  auth: SystemPublicAuthSettings;
  brand: SystemPublicBrandSettings;
  defaultLocale: string;
  fallbackLocale: string;
  supportedLocales: string[];
};

export type SystemServerDiskInfo = {
  fsType: string;
  mountPoint: string;
  totalGb: number;
  totalMb: number;
  usedGb: number;
  usedMb: number;
  usedPercent: number;
};

export type SystemServerInfo = {
  build: {
    goVersion: string;
    module: string;
    path: string;
    settings: Array<{ key: string; value: string }>;
    version: string;
  };
  cpu: {
    cores: number;
    percent: number[];
  };
  disk: SystemServerDiskInfo[];
  gc: {
    lastGcAt?: string;
    nextGcMb: number;
    numGc: number;
    pauseTotalNs: number;
  };
  memory: {
    allocMb: number;
    heapAllocMb: number;
    heapIdleMb: number;
    heapInuseMb: number;
    heapObjects: number;
    heapReleasedMb: number;
    heapSysMb: number;
    stackInuseMb: number;
    stackSysMb: number;
    sysMb: number;
    totalAllocMb: number;
  };
  os: {
    compiler: string;
    goarch: string;
    goos: string;
    goVersion: string;
    numCpu: number;
    numGoroutine: number;
  };
  ram: {
    totalMb: number;
    usedMb: number;
    usedPercent: number;
  };
  refreshedAt: string;
  runtime: {
    startTime: string;
    uptime: string;
    uptimeSeconds: number;
  };
};

export type SystemServerMetricsSample = {
  cpuUsedPercent: number;
  diskIo: SystemServerDiskIOSample[];
  diskIoLatencyMs: number;
  diskMaxUsedPercent: number;
  diskReadMbPerSecond: number;
  diskReadOpsPerSecond: number;
  diskWriteMbPerSecond: number;
  diskWriteOpsPerSecond: number;
  goroutines: number;
  heapAllocMb: number;
  networkReceiveKbPerSecond: number;
  networkTransmitKbPerSecond: number;
  ramUsedPercent: number;
  sampledAt: string;
};

export type SystemServerDiskIOSample = {
  ioLatencyMs: number;
  name: string;
  readMbPerSecond: number;
  readOpsPerSecond: number;
  writeMbPerSecond: number;
  writeOpsPerSecond: number;
};

export type SystemServerMetricsHistory = {
  intervalSeconds: number;
  samples: SystemServerMetricsSample[];
  windowSeconds: number;
};

export type SystemTrafficProbeStatus = BackendString<
  "critical" | "healthy" | "pending" | "warning"
>;

export type SystemTrafficProbeSeverity = BackendString<
  "critical" | "high" | "low" | "medium" | "ok"
>;

export type SystemTrafficProbeTarget = {
  alertChannels: string;
  allowPrivateNetwork: boolean;
  createdAt: string;
  emailRecipients: string;
  enabled: boolean;
  expectedContentKeyword: string;
  expectedFinalHost: string;
  expectedIpCidrs: string;
  expectedStatusCodes: string;
  expectedTlsFingerprint: string;
  id: number | string;
  intervalSeconds: number;
  lastCheckedAt?: string | null;
  lastError: string;
  lastReason: string;
  lastSeverity: SystemTrafficProbeSeverity;
  lastStatus: SystemTrafficProbeStatus;
  method: BackendString<"GET" | "HEAD">;
  name: string;
  nextProbeAt?: string | null;
  timeoutSeconds: number;
  updatedAt: string;
  url: string;
};

export type SystemTrafficProbeResult = {
  connectDurationMs: number;
  createdAt: string;
  dnsDurationMs: number;
  dnsIps: string;
  errorMessage: string;
  evidenceJson: string;
  finalUrl: string;
  id: number | string;
  method: string;
  reason: string;
  severity: SystemTrafficProbeSeverity;
  stage: string;
  status: SystemTrafficProbeStatus;
  statusCode: number;
  targetId: number | string;
  targetName: string;
  tlsDurationMs: number;
  tlsFingerprintSha256: string;
  tlsIssuer: string;
  tlsNotAfter?: string | null;
  tlsSubject: string;
  totalDurationMs: number;
  ttfbMs: number;
  url: string;
};

export type SystemTrafficHijackEventState = BackendString<"open" | "resolved">;

export type SystemTrafficHijackEvent = {
  createdAt: string;
  evidenceHash: string;
  evidenceJson: string;
  firstSeenAt: string;
  id: number | string;
  lastSeenAt: string;
  notificationStatus: string;
  occurrences: number;
  reason: string;
  resolvedAt?: string | null;
  severity: SystemTrafficProbeSeverity;
  state: SystemTrafficHijackEventState;
  targetId: number | string;
  targetName: string;
  updatedAt: string;
};

export type SystemTrafficHijackOverview = {
  criticalTargets: number;
  enabledTargets: number;
  healthyTargets: number;
  recentEvents: SystemTrafficHijackEvent[];
  recentResults: SystemTrafficProbeResult[];
  severityCounts: Record<string, number>;
  storageStatus: string;
  targets: SystemTrafficProbeTarget[];
  totalTargets: number;
  warningTargets: number;
};

export type SystemTrafficProbeResultPage = {
  items: SystemTrafficProbeResult[];
  limit: number;
  nextCursor?: number | string;
  storageStatus: string;
};

export type SystemTrafficHijackEventPage = {
  items: SystemTrafficHijackEvent[];
  page: number;
  pageSize: number;
  storageStatus: string;
  total: number;
};

export type SystemTrafficHijackStreamEvent = {
  data: unknown;
  event: string;
  sentAt: string;
};

export type SystemAPIEntry = {
  access: BackendString<"authenticated" | "permission" | "public">;
  code: string;
  description: string;
  group: string;
  method: string;
  order: number;
  path: string;
  permission?: string;
  permissionRegistered: boolean;
  productCode?: string;
  scope?: BackendString<"platform" | "product" | "tenant">;
  synced: boolean;
  syncedAt?: string;
};

export type SystemAPIGroup = {
  code: string;
  count: number;
  items: SystemAPIEntry[];
  label: string;
};

export type SystemAPISyncResult = {
  created: number;
  groups: SystemAPIGroup[];
  persisted: boolean;
  stale: number;
  storageStatus: string;
  syncedAt: string;
  total: number;
  updated: number;
};

export type SystemPermissionSyncItem = {
  code: string;
  created: boolean;
  description: string;
  exists: boolean;
  name: string;
  productCode?: string;
  scope?: BackendString<"platform" | "product" | "tenant">;
};

export type SystemPermissionSyncResult = {
  created: number;
  items: SystemPermissionSyncItem[];
  persisted: boolean;
  skipped: number;
  storageStatus: string;
  syncedAt: string;
  total: number;
};

export type SystemMenuItem = {
  code: string;
  description?: string;
  descriptionKey?: string;
  icon: string;
  label: string;
  labelKey?: string;
  mobile: boolean;
  order: number;
  path: string;
  permission?: string;
  productCode?: string;
  scope?: BackendString<"platform" | "product" | "tenant">;
};

export type SystemMenuGroup = {
  code: string;
  description?: string;
  descriptionKey?: string;
  items: SystemMenuItem[];
  label: string;
  labelKey?: string;
  order: number;
};

export type SystemDictionaryStatus = BackendString<"active" | "disabled">;

export type SystemDictionaryItem = {
  createdAt: string;
  dictionaryId: number | string;
  extra: string;
  id: number | string;
  label: string;
  sort: number;
  status: SystemDictionaryStatus;
  updatedAt: string;
  value: string;
};

export type SystemDictionary = {
  code: string;
  createdAt: string;
  description: string;
  id: number | string;
  items: SystemDictionaryItem[];
  name: string;
  status: SystemDictionaryStatus;
  updatedAt: string;
};

export type SystemDictionaryCatalog = {
  items: SystemDictionary[];
  storageStatus: string;
  total: number;
};

export type SystemConfigValueType = BackendString<
  "array" | "boolean" | "number" | "object" | "string" | "unknown"
>;

export type SystemConfigVisibilityCondition = {
  field: string;
  in: string[];
};

export type SystemConfigOption = {
  description?: string;
  descriptionKey?: string;
  label: string;
  labelKey?: string;
  value: string;
};

export type SystemConfigItem = {
  description: string;
  descriptionKey?: string;
  editable: boolean;
  editor?: string;
  groupKey?: string;
  key: string;
  label: string;
  labelKey?: string;
  options?: SystemConfigOption[];
  risk?: string;
  secret: boolean;
  source: string;
  testable: boolean;
  value: unknown;
  valueType: SystemConfigValueType;
  visibleWhen?: SystemConfigVisibilityCondition;
};

export type SystemConfigGroup = {
  description?: string;
  descriptionKey?: string;
  items: SystemConfigItem[];
  key: string;
  label: string;
  labelKey?: string;
  risk?: string;
  testable: boolean;
  visibleWhen?: SystemConfigVisibilityCondition;
};

export type SystemConfigSection = {
  code: string;
  description: string;
  descriptionKey?: string;
  groups: SystemConfigGroup[];
  icon: string;
  items: SystemConfigItem[];
  label: string;
  labelKey?: string;
  order: number;
};

export type SystemConfigSnapshot = {
  sections: SystemConfigSection[];
};

export type SystemParameter = {
  createdAt: string;
  description: string;
  id: number | string;
  key: string;
  name: string;
  updatedAt: string;
  value: string;
};

export type SystemParameterPage = {
  items: SystemParameter[];
  page: number;
  pageSize: number;
  storageStatus: string;
  total: number;
};

export type SystemOperationRecord = {
  body: string;
  createdAt: string;
  errorMessage: string;
  id: number | string;
  ipAddress: string;
  latencyMs: number;
  method: string;
  path: string;
  response: string;
  status: number;
  traceId: string;
  userAgent: string;
  userId: number | string;
  username: string;
};

export type SystemOperationRecordPage = {
  items: SystemOperationRecord[];
  page: number;
  pageSize: number;
  storageStatus: string;
  total: number;
};

export type SystemMediaCategory = {
  children?: SystemMediaCategory[];
  createdAt: string;
  id: number | string;
  name: string;
  parentId?: number | string;
  sort: number;
  updatedAt: string;
};

export type SystemMediaCategoryCatalog = {
  items: SystemMediaCategory[];
  storageStatus: string;
  total: number;
};

export type SystemMediaAsset = {
  categoryId?: number | string;
  createdAt: string;
  displayName: string;
  extension: string;
  external: boolean;
  id: number | string;
  mimeType: string;
  originalName: string;
  sizeBytes: number;
  source: BackendString<"resumable" | "upload" | "url">;
  storageKey: string;
  updatedAt: string;
  uploadedBy: number | string;
  uploadedByUsername: string;
  url: string;
};

export type SystemMediaAssetPage = {
  items: SystemMediaAsset[];
  objectStorage: string;
  page: number;
  pageSize: number;
  storageStatus: string;
  total: number;
  uploadMaxBytes: number;
  uploadMaxMb: number;
  uploadUnavailable: boolean;
};

export type SystemMediaURLImportItem = {
  name?: string;
  url: string;
};

export type SystemMediaURLImportInput = {
  categoryId?: number | string;
  items?: SystemMediaURLImportItem[];
  text?: string;
};

export type SystemMediaURLImportResult = {
  imported: number;
  items: SystemMediaAsset[];
  storageStatus: string;
};

export type SystemMediaUploadStatus = BackendString<"aborted" | "active" | "completed" | "expired">;

export type SystemMediaUploadSession = {
  chunkSize: number;
  chunkTotal: number;
  completedAt?: string | null;
  createdAt: string;
  expiresAt: string;
  fileHash: string;
  fileName: string;
  finalAssetId?: number | string | null;
  id: number | string;
  sizeBytes: number;
  status: SystemMediaUploadStatus;
  updatedAt: string;
  uploadedBy: number | string;
  uploadedByUsername: string;
};

export type SystemMediaResumableCheckInput = {
  categoryId?: number | string;
  chunkSize?: number;
  chunkTotal?: number;
  fileHash: string;
  fileName: string;
  sizeBytes: number;
};

export type SystemMediaResumableCheckResult = {
  asset?: SystemMediaAsset;
  chunkSize: number;
  missingChunks: number[];
  progress: number;
  session: SystemMediaUploadSession;
  storageStatus: string;
  uploadMaxBytes: number;
  uploadMaxMb: number;
  uploadUnavailable: boolean;
  uploadedChunks: number[];
};

export type SystemMediaResumableChunkMetadata = {
  chunkHash: string;
  chunkIndex: number;
  chunkTotal: number;
  fileHash: string;
  fileName: string;
  sessionId: number | string;
};

export type SystemMediaResumableChunkResult = {
  chunkIndex: number;
  missingChunks: number[];
  progress: number;
  storageStatus: string;
  uploadedChunks: number[];
};

export type SystemMediaResumableCompleteInput = {
  fileHash: string;
  sessionId: number | string;
};

export type SystemMediaResumableCompleteResult = {
  asset: SystemMediaAsset;
  sessionId: number | string;
  storageStatus: string;
};

export type SystemMediaResumableAbortResult = {
  aborted: boolean;
  sessionId: number | string;
  storageStatus: string;
};

export type SystemVersionRecord = {
  apiCount: number;
  createdAt: string;
  createdBy: number | string;
  createdByUsername: string;
  description: string;
  dictionaryCount: number;
  id: number | string;
  menuCount: number;
  source: BackendString<"export" | "import">;
  updatedAt: string;
  versionCode: string;
  versionName: string;
};

export type SystemVersionPage = {
  items: SystemVersionRecord[];
  page: number;
  pageSize: number;
  storageStatus: string;
  total: number;
};

export type SystemVersionPackageInfo = {
  code: string;
  createdAt?: string;
  description?: string;
  name: string;
};

export type SystemVersionPackage = {
  apis: SystemAPIGroup[];
  dictionaries: SystemDictionary[];
  exportedAt: string;
  menus: SystemMenuGroup[];
  version: SystemVersionPackageInfo;
};

export type SystemVersionDetail = {
  item: SystemVersionRecord;
  package: SystemVersionPackage;
  storageStatus: string;
};

export type SystemVersionSourceCatalog = {
  apiCount: number;
  apis: SystemAPIGroup[];
  dictionaries: SystemDictionary[];
  dictionaryCount: number;
  menuCount: number;
  menus: SystemMenuGroup[];
  storageStatus: string;
};

export type SystemVersionExportInput = {
  apiCodes: string[];
  description?: string;
  dictionaryCodes: string[];
  menuCodes: string[];
  versionCode: string;
  versionName: string;
};

export type SystemVersionImportResult = {
  apisSkipped: number;
  dictionariesCreated: number;
  dictionariesSkipped: number;
  dictionaryItemsCreated: number;
  importedAt: string;
  item: SystemVersionRecord;
  menusSkipped: number;
  storageStatus: string;
};
