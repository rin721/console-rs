type EndpointID = number | string;

const apiV1 = "/api/v1";

function pathValue(value: EndpointID) {
  return encodeURIComponent(String(value));
}

const orgPath = (orgId: EndpointID) => `${apiV1}/orgs/${pathValue(orgId)}`;
const systemPath = `${apiV1}/system`;
const systemMediaPath = `${systemPath}/media`;
const systemMediaAssetsPath = `${systemMediaPath}/assets`;

export const API_ENDPOINTS = {
  auth: {
    captcha: `${apiV1}/auth/captcha`,
    emailVerificationConfirm: (token: string) =>
      `${apiV1}/auth/email-verifications/${pathValue(token)}/confirm`,
    forgotPassword: `${apiV1}/auth/password/forgot`,
    initialAdminSetup: `${apiV1}/auth/setup/initial-admin`,
    login: `${apiV1}/auth/login`,
    logout: `${apiV1}/auth/logout`,
    mfaSetup: `${apiV1}/auth/mfa/setup`,
    mfaVerify: `${apiV1}/auth/mfa/verify`,
    passwordReset: `${apiV1}/auth/password/reset`,
    refresh: `${apiV1}/auth/refresh`,
    setupStatus: `${apiV1}/auth/setup/status`,
    signup: `${apiV1}/auth/signup`,
    switchOrg: `${apiV1}/auth/switch-org`,
  },
  health: "/health",
  invitations: {
    accept: (token: string) => `${apiV1}/invitations/${pathValue(token)}/accept`,
  },
  me: {
    organizations: `${apiV1}/me/orgs`,
    profile: `${apiV1}/me`,
    session: `${apiV1}/me/session`,
  },
  orgs: {
    apiToken: (orgId: EndpointID, tokenId: EndpointID) =>
      `${orgPath(orgId)}/api-tokens/${pathValue(tokenId)}`,
    apiTokens: (orgId: EndpointID) => `${orgPath(orgId)}/api-tokens`,
    auditLogs: (orgId: EndpointID) => `${orgPath(orgId)}/audit-logs`,
    collection: `${apiV1}/orgs`,
    invitation: (orgId: EndpointID, invitationId: EndpointID) =>
      `${orgPath(orgId)}/invitations/${pathValue(invitationId)}`,
    invitations: (orgId: EndpointID) => `${orgPath(orgId)}/invitations`,
    item: (orgId: EndpointID) => orgPath(orgId),
    permissions: (orgId: EndpointID) => `${orgPath(orgId)}/permissions`,
    role: (orgId: EndpointID, roleId: EndpointID) => `${orgPath(orgId)}/roles/${pathValue(roleId)}`,
    roles: (orgId: EndpointID) => `${orgPath(orgId)}/roles`,
    session: (orgId: EndpointID, sessionId: EndpointID) =>
      `${orgPath(orgId)}/sessions/${pathValue(sessionId)}`,
    sessions: (orgId: EndpointID) => `${orgPath(orgId)}/sessions`,
    user: (orgId: EndpointID, userId: EndpointID) => `${orgPath(orgId)}/users/${pathValue(userId)}`,
    userInvitations: (orgId: EndpointID) => `${orgPath(orgId)}/users/invitations`,
    users: (orgId: EndpointID) => `${orgPath(orgId)}/users`,
  },
  plugins: {
    capabilities: (pluginId: string) => `${apiV1}/plugins/${pathValue(pluginId)}/capabilities`,
    collection: `${apiV1}/plugins`,
    health: (pluginId: string) => `${apiV1}/plugins/${pathValue(pluginId)}/health`,
    item: (pluginId: string) => `${apiV1}/plugins/${pathValue(pluginId)}`,
  },
  ready: "/ready",
  setup: {
    complete: `${apiV1}/setup/complete`,
    config: (stepKey: EndpointID) => `${apiV1}/setup/configs/${pathValue(stepKey)}`,
    configTest: (stepKey: EndpointID) => `${apiV1}/setup/configs/${pathValue(stepKey)}/test`,
    runLogs: (runId: EndpointID) => `${apiV1}/setup/runs/${pathValue(runId)}/logs`,
    runRetry: (runId: EndpointID) => `${apiV1}/setup/runs/${pathValue(runId)}/retry`,
    runStepSkip: (runId: EndpointID, stepKey: EndpointID) =>
      `${apiV1}/setup/runs/${pathValue(runId)}/steps/${pathValue(stepKey)}/skip`,
    runs: `${apiV1}/setup/runs`,
    schema: `${apiV1}/setup/schema`,
    status: `${apiV1}/setup/status`,
  },
  system: {
    apis: `${systemPath}/apis`,
    apiPermissionsSync: `${systemPath}/apis/permissions/sync`,
    apiSync: `${systemPath}/apis/sync`,
    config: `${systemPath}/config`,
    dictionary: (dictionaryId: EndpointID) =>
      `${systemPath}/dictionaries/${pathValue(dictionaryId)}`,
    dictionaryItem: (itemId: EndpointID) => `${systemPath}/dictionary-items/${pathValue(itemId)}`,
    dictionaryItems: (dictionaryId: EndpointID) =>
      `${systemPath}/dictionaries/${pathValue(dictionaryId)}/items`,
    dictionaries: `${systemPath}/dictionaries`,
    media: {
      asset: (assetId: EndpointID) => `${systemMediaAssetsPath}/${pathValue(assetId)}`,
      assetDownload: (assetId: EndpointID) =>
        `${systemMediaAssetsPath}/${pathValue(assetId)}/download`,
      assetUpload: `${systemMediaAssetsPath}/upload`,
      assets: systemMediaAssetsPath,
      categories: `${systemMediaPath}/categories`,
      category: (categoryId: EndpointID) =>
        `${systemMediaPath}/categories/${pathValue(categoryId)}`,
      importURL: `${systemMediaAssetsPath}/import-url`,
      resumableAbort: `${systemMediaAssetsPath}/resumable/abort`,
      resumableCheck: `${systemMediaAssetsPath}/resumable/check`,
      resumableChunks: `${systemMediaAssetsPath}/resumable/chunks`,
      resumableComplete: `${systemMediaAssetsPath}/resumable/complete`,
    },
    menus: `${systemPath}/menus`,
    operationRecords: `${systemPath}/operation-records`,
    parameter: (parameterId: EndpointID) => `${systemPath}/parameters/${pathValue(parameterId)}`,
    parameterValue: `${systemPath}/parameters/value`,
    parameters: `${systemPath}/parameters`,
    publicSettings: `${systemPath}/public-settings`,
    serverInfo: `${systemPath}/server-info`,
    serverMetricsHistory: `${systemPath}/server-metrics/history`,
    trafficHijack: {
      eventResolve: (eventId: EndpointID) =>
        `${systemPath}/traffic-hijack/events/${pathValue(eventId)}/resolve`,
      events: `${systemPath}/traffic-hijack/events`,
      overview: `${systemPath}/traffic-hijack/overview`,
      probe: (targetId: EndpointID) =>
        `${systemPath}/traffic-hijack/targets/${pathValue(targetId)}/probe`,
      results: `${systemPath}/traffic-hijack/results`,
      stream: `${systemPath}/traffic-hijack/stream`,
      target: (targetId: EndpointID) =>
        `${systemPath}/traffic-hijack/targets/${pathValue(targetId)}`,
      targets: `${systemPath}/traffic-hijack/targets`,
    },
    version: (versionId: EndpointID) => `${systemPath}/versions/${pathValue(versionId)}`,
    versionDownload: (versionId: EndpointID) =>
      `${systemPath}/versions/${pathValue(versionId)}/download`,
    versionExport: `${systemPath}/versions/export`,
    versionImport: `${systemPath}/versions/import`,
    versionSources: `${systemPath}/versions/sources`,
    versions: `${systemPath}/versions`,
  },
} as const;
