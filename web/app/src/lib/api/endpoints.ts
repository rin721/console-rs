type EndpointID = number | string;

const apiV1 = "/api/v1";
const systemPath = `${apiV1}/system`;

const id = (value: EndpointID) => encodeURIComponent(String(value));

export const endpoints = {
  health: "/health",
  ready: "/ready",
  auth: {
    setupStatus: `${apiV1}/auth/setup/status`,
    acceptInvitation: `${apiV1}/auth/invitations/accept`,
    confirmEmailVerification: `${apiV1}/auth/email-verifications/confirm`,
    forgotPassword: `${apiV1}/auth/password/forgot`,
    initialAdmin: `${apiV1}/auth/setup/initial-admin`,
    login: `${apiV1}/auth/login`,
    logout: `${apiV1}/auth/logout`,
    register: `${apiV1}/auth/register`,
    mfaFactor: (factorId: EndpointID) => `${apiV1}/auth/mfa/factors/${id(factorId)}`,
    mfaFactors: `${apiV1}/auth/mfa/factors`,
    mfaRecoveryCodes: `${apiV1}/auth/mfa/recovery-codes`,
    mfaSetup: `${apiV1}/auth/mfa/setup`,
    mfaVerify: `${apiV1}/auth/mfa/verify`,
    refresh: `${apiV1}/auth/refresh`,
    requestEmailVerification: `${apiV1}/auth/email-verifications`,
    resetPassword: `${apiV1}/auth/password/reset`,
    session: `${apiV1}/me/session`
  },
  setup: {
    status: `${apiV1}/setup/status`,
    configChecks: `${apiV1}/setup/config-checks`,
    schema: `${apiV1}/setup/schema`,
    runs: `${apiV1}/setup/runs`,
    logs: (runId: EndpointID) => `${apiV1}/setup/runs/${id(runId)}/logs`,
    complete: `${apiV1}/setup/complete`
  },
  iam: {
    orgs: `${apiV1}/iam/orgs`,
    orgUsers: (orgId: EndpointID) => `${apiV1}/iam/orgs/${id(orgId)}/users`,
    orgUser: (orgId: EndpointID, userId: EndpointID) => `${apiV1}/iam/orgs/${id(orgId)}/users/${id(userId)}`,
    orgRoles: (orgId: EndpointID) => `${apiV1}/iam/orgs/${id(orgId)}/roles`,
    orgRole: (orgId: EndpointID, roleId: EndpointID) => `${apiV1}/iam/orgs/${id(orgId)}/roles/${id(roleId)}`,
    permissions: `${apiV1}/iam/permissions`
  },
  orgs: {
    apiTokens: (orgId: EndpointID) => `${apiV1}/orgs/${id(orgId)}/api-tokens`,
    apiToken: (orgId: EndpointID, tokenId: EndpointID) => `${apiV1}/orgs/${id(orgId)}/api-tokens/${id(tokenId)}`,
    invitations: (orgId: EndpointID) => `${apiV1}/orgs/${id(orgId)}/invitations`,
    invitation: (orgId: EndpointID, invitationId: EndpointID) => `${apiV1}/orgs/${id(orgId)}/invitations/${id(invitationId)}`,
    userInvitations: (orgId: EndpointID) => `${apiV1}/orgs/${id(orgId)}/users/invitations`
  },
  system: {
    publicSettings: `${systemPath}/public-settings`,
    menus: `${systemPath}/menus`,
    apis: `${systemPath}/apis`,
    operationRecords: `${systemPath}/operation-records`,
    operationRecordSummary: `${systemPath}/operation-records/summary`,
    serverStatus: `${systemPath}/server-status`,
    configs: `${systemPath}/configs`,
    config: (key: EndpointID) => `${systemPath}/configs/${id(key)}`,
    dictionaries: `${systemPath}/dictionaries`,
    dictionary: (code: EndpointID) => `${systemPath}/dictionaries/${id(code)}`,
    parameters: `${systemPath}/parameters`,
    parameter: (key: EndpointID) => `${systemPath}/parameters/${id(key)}`,
    versionPackages: `${systemPath}/version-packages`,
    versionPackage: (idValue: EndpointID) => `${systemPath}/version-packages/${id(idValue)}`,
    versionPackageReleases: `${systemPath}/version-packages/releases`,
    versionPackagePublish: (idValue: EndpointID) => `${systemPath}/version-packages/${id(idValue)}/publish`,
    versionPackageRollback: (idValue: EndpointID) => `${systemPath}/version-packages/${id(idValue)}/rollback`,
    mediaAssets: `${systemPath}/media-assets`,
    mediaAssetUpload: `${systemPath}/media-assets/upload`,
    mediaAsset: (idValue: EndpointID) => `${systemPath}/media-assets/${id(idValue)}`,
    storageObjects: `${systemPath}/storage-objects`,
    trafficProbeTargets: `${systemPath}/traffic-probes/targets`,
    trafficProbeTarget: (idValue: EndpointID) => `${systemPath}/traffic-probes/targets/${id(idValue)}`,
    trafficProbeRun: (idValue: EndpointID) => `${systemPath}/traffic-probes/targets/${id(idValue)}/run`,
    trafficProbeResults: `${systemPath}/traffic-probes/results`,
    trafficProbeAlerts: `${systemPath}/traffic-probes/alerts`,
    trafficProbeAlertAck: (idValue: EndpointID) => `${systemPath}/traffic-probes/alerts/${id(idValue)}/ack`,
    trafficProbeAlertResolve: (idValue: EndpointID) => `${systemPath}/traffic-probes/alerts/${id(idValue)}/resolve`
  }
} as const;
