export const queryKeys = {
  auth: {
    identity: ["auth", "identity"] as const,
    me: ["auth", "me"] as const,
    organizations: ["auth", "organizations"] as const,
    root: ["auth"] as const,
  },
  iam: {
    apiTokens: (
      locale: string,
      orgId: number | string,
      page: number,
      pageSize: number,
      filters: {
        status?: string;
        userId?: number | string;
      } = {},
    ) => ["iam", "api-tokens", locale, orgId, page, pageSize, filters] as const,
    auditLogs: (
      locale: string,
      orgId: number | string,
      filters: {
        action?: string;
        cursor?: number | string;
        from?: string;
        limit?: number;
        to?: string;
        userId?: number | string;
      } = {},
    ) => ["iam", "audit-logs", locale, orgId, filters] as const,
    organizations: (
      locale: string,
      page: number,
      pageSize: number,
      filters: {
        code?: string;
        keyword?: string;
        name?: string;
        status?: string;
      } = {},
    ) => ["iam", "organizations", locale, page, pageSize, filters] as const,
    permissions: (locale: string, orgId: number | string) =>
      ["iam", "permissions", locale, orgId] as const,
    invitations: (locale: string, orgId: number | string) =>
      ["iam", "invitations", locale, orgId] as const,
    roles: (locale: string, orgId: number | string) => ["iam", "roles", locale, orgId] as const,
    root: ["iam"] as const,
    sessions: (
      locale: string,
      orgId: number | string,
      page: number,
      pageSize: number,
      filters: {
        desc?: boolean;
        ipAddress?: string;
        keyword?: string;
        orderKey?: string;
        scope?: string;
        status?: string;
        userId?: number | string;
      } = {},
    ) => ["iam", "sessions", locale, orgId, page, pageSize, filters] as const,
    users: (
      locale: string,
      orgId: number | string,
      page: number,
      pageSize: number,
      filters: {
        desc?: boolean;
        displayName?: string;
        email?: string;
        keyword?: string;
        orderKey?: string;
        roleCode?: string;
        status?: string;
        username?: string;
      } = {},
    ) => ["iam", "users", locale, orgId, page, pageSize, filters] as const,
  },
  plugins: {
    capabilities: (locale: string, pluginId: string) =>
      ["plugins", "capabilities", locale, pluginId] as const,
    detail: (locale: string, pluginId: string) => ["plugins", "detail", locale, pluginId] as const,
    health: (locale: string, pluginId: string) => ["plugins", "health", locale, pluginId] as const,
    list: (locale: string) => ["plugins", "list", locale] as const,
    root: ["plugins"] as const,
  },
  setup: {
    logs: (runId: string, locale: string) => ["setup", "logs", runId, locale] as const,
    root: ["setup"] as const,
    schema: (setupToken: string | undefined, locale: string) =>
      ["setup", "schema", setupToken ?? "", locale] as const,
    status: (locale: string) => ["setup", "status", locale] as const,
  },
  system: {
    apiCatalog: (locale: string) => ["system", "apis", locale] as const,
    config: (locale: string) => ["system", "config", locale] as const,
    dictionaries: (locale: string) => ["system", "dictionaries", locale] as const,
    health: ["system", "health"] as const,
    mediaAssets: (
      locale: string,
      page: number,
      pageSize: number,
      filters: {
        categoryId?: number | string;
        keyword?: string;
      } = {},
    ) => ["system", "media", "assets", locale, page, pageSize, filters] as const,
    mediaCategories: (locale: string) => ["system", "media", "categories", locale] as const,
    menus: (locale: string) => ["system", "menus", locale] as const,
    parameters: (
      locale: string,
      page: number,
      pageSize: number,
      filters: {
        endCreatedAt?: string;
        key?: string;
        name?: string;
        startCreatedAt?: string;
      } = {},
    ) => ["system", "parameters", locale, page, pageSize, filters] as const,
    operationRecords: (
      locale: string,
      page: number,
      pageSize: number,
      filters: {
        method?: string;
        path?: string;
        status?: number | string;
        statusClass?: string;
      } = {},
    ) => ["system", "operation-records", locale, page, pageSize, filters] as const,
    publicSettings: (locale: string) => ["system", "public-settings", locale] as const,
    ready: ["system", "ready"] as const,
    root: ["system"] as const,
    serverInfo: ["system", "server-info"] as const,
    serverMetricsHistory: ["system", "server-metrics", "history"] as const,
    trafficHijackEvents: (
      locale: string,
      filters: {
        page?: number;
        pageSize?: number;
        severity?: string;
        state?: string;
        targetId?: number | string;
      } = {},
    ) => ["system", "traffic-hijack", "events", locale, filters] as const,
    trafficHijackOverview: (locale: string) =>
      ["system", "traffic-hijack", "overview", locale] as const,
    trafficProbeResults: (
      locale: string,
      filters: {
        cursor?: number | string;
        limit?: number;
        targetId?: number | string;
      } = {},
    ) => ["system", "traffic-hijack", "results", locale, filters] as const,
    trafficProbeTargets: (locale: string) =>
      ["system", "traffic-hijack", "targets", locale] as const,
    version: (locale: string, versionId: number | string) =>
      ["system", "versions", "detail", locale, versionId] as const,
    versionSources: (locale: string) => ["system", "versions", "sources", locale] as const,
    versions: (
      locale: string,
      page: number,
      pageSize: number,
      filters: {
        endCreatedAt?: string;
        startCreatedAt?: string;
        versionCode?: string;
        versionName?: string;
      } = {},
    ) => ["system", "versions", locale, page, pageSize, filters] as const,
  },
};
