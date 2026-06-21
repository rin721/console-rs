import { API_ENDPOINTS } from "./endpoints";
import { configureApiClientAuthRuntime } from "./client";
import { apiClient } from "./runtime";
import type {
  HealthStatus,
  ReadyStatus,
  SystemAPIGroup,
  SystemAPISyncResult,
  SystemConfigSnapshot,
  SystemDictionary,
  SystemDictionaryCatalog,
  SystemDictionaryItem,
  SystemMediaAsset,
  SystemMediaAssetPage,
  SystemMediaCategory,
  SystemMediaCategoryCatalog,
  SystemMediaResumableAbortResult,
  SystemMediaResumableCheckInput,
  SystemMediaResumableCheckResult,
  SystemMediaResumableChunkMetadata,
  SystemMediaResumableChunkResult,
  SystemMediaResumableCompleteInput,
  SystemMediaResumableCompleteResult,
  SystemMediaURLImportInput,
  SystemMediaURLImportResult,
  SystemMenuGroup,
  SystemOperationRecordPage,
  SystemParameter,
  SystemParameterPage,
  SystemPermissionSyncResult,
  SystemPublicSettings,
  SystemServerInfo,
  SystemServerMetricsHistory,
  SystemTrafficHijackEvent,
  SystemTrafficHijackEventPage,
  SystemTrafficHijackOverview,
  SystemTrafficProbeResult,
  SystemTrafficProbeResultPage,
  SystemTrafficProbeTarget,
  SystemVersionPage,
  SystemVersionDetail,
  SystemVersionExportInput,
  SystemVersionImportResult,
  SystemVersionPackage,
  SystemVersionSourceCatalog,
} from "./types";

type RequestOptions = {
  signal?: AbortSignal;
};

export type SystemVersionListQuery = {
  endCreatedAt?: string;
  page?: number;
  pageSize?: number;
  startCreatedAt?: string;
  versionCode?: string;
  versionName?: string;
};

export type SystemMediaAssetListQuery = {
  categoryId?: number | string;
  keyword?: string;
  page?: number;
  pageSize?: number;
};

export type SystemMediaCategoryInput = {
  id?: number | string;
  name: string;
  parentId?: number | string;
  sort?: number;
};

export type SystemMediaAssetUpdateInput = {
  displayName: string;
};

export type SystemConfigUpdateItem = {
  key: string;
  value: unknown;
};

export type SystemConfigUpdateInput = {
  items: SystemConfigUpdateItem[];
  persist?: boolean;
};

export type SystemParameterListQuery = {
  endCreatedAt?: string;
  key?: string;
  name?: string;
  page?: number;
  pageSize?: number;
  startCreatedAt?: string;
};

export type SystemParameterInput = {
  description?: string;
  key: string;
  name: string;
  value: string;
};

export type SystemParameterUpdateInput = {
  description?: string;
  key?: string;
  name?: string;
  value?: string;
};

export type SystemDictionaryInput = {
  code: string;
  description?: string;
  name: string;
  status?: string;
};

export type SystemDictionaryUpdateInput = {
  description?: string;
  name?: string;
  status?: string;
};

export type SystemDictionaryItemInput = {
  extra?: string;
  label: string;
  sort?: number;
  status?: string;
  value: string;
};

export type SystemDictionaryItemUpdateInput = {
  extra?: string;
  label?: string;
  sort?: number;
  status?: string;
  value?: string;
};

export type SystemOperationRecordListQuery = {
  method?: string;
  page?: number;
  pageSize?: number;
  path?: string;
  status?: number | string;
  statusClass?: string;
};

export type SystemTrafficProbeTargetInput = {
  alertChannels?: string[];
  allowPrivateNetwork?: boolean;
  emailRecipients?: string[];
  enabled?: boolean;
  expectedContentKeyword?: string;
  expectedFinalHost?: string;
  expectedIpCidrs?: string[];
  expectedStatusCodes?: string;
  expectedTlsFingerprint?: string;
  intervalSeconds?: number;
  method?: "GET" | "HEAD";
  name: string;
  timeoutSeconds?: number;
  url: string;
};

export type SystemTrafficProbeTargetUpdateInput = Partial<SystemTrafficProbeTargetInput>;

export type SystemTrafficProbeResultListQuery = {
  cursor?: number | string;
  limit?: number;
  targetId?: number | string;
};

export type SystemTrafficHijackEventListQuery = {
  page?: number;
  pageSize?: number;
  severity?: string;
  state?: string;
  targetId?: number | string;
};

export const systemApi = {
  getHealth: (options: RequestOptions = {}) =>
    apiClient.request<HealthStatus>(API_ENDPOINTS.health, {
      auth: false,
      signal: options.signal,
    }),
  getReady: (options: RequestOptions = {}) =>
    apiClient.request<ReadyStatus>(API_ENDPOINTS.ready, {
      auth: false,
      signal: options.signal,
    }),
  getPublicSettings: async (options: RequestOptions = {}) => {
    const settings = await apiClient.request<SystemPublicSettings>(API_ENDPOINTS.system.publicSettings, {
      auth: false,
      signal: options.signal,
    });
    configureApiClientAuthRuntime(settings.auth);
    return settings;
  },
  getServerInfo: (options: RequestOptions = {}) =>
    apiClient.request<SystemServerInfo>(API_ENDPOINTS.system.serverInfo, {
      signal: options.signal,
    }),
  getServerMetricsHistory: (options: RequestOptions = {}) =>
    apiClient.request<SystemServerMetricsHistory>(API_ENDPOINTS.system.serverMetricsHistory, {
      signal: options.signal,
    }),
  getTrafficHijackOverview: (options: RequestOptions = {}) =>
    apiClient.request<SystemTrafficHijackOverview>(API_ENDPOINTS.system.trafficHijack.overview, {
      signal: options.signal,
    }),
  listTrafficProbeTargets: (options: RequestOptions = {}) =>
    apiClient.request<SystemTrafficProbeTarget[]>(API_ENDPOINTS.system.trafficHijack.targets, {
      signal: options.signal,
    }),
  createTrafficProbeTarget: (body: SystemTrafficProbeTargetInput, options: RequestOptions = {}) =>
    apiClient.request<SystemTrafficProbeTarget>(API_ENDPOINTS.system.trafficHijack.targets, {
      body,
      method: "POST",
      signal: options.signal,
    }),
  updateTrafficProbeTarget: (
    targetId: number | string,
    body: SystemTrafficProbeTargetUpdateInput,
    options: RequestOptions = {},
  ) =>
    apiClient.request<SystemTrafficProbeTarget>(
      API_ENDPOINTS.system.trafficHijack.target(targetId),
      {
        body,
        method: "PATCH",
        signal: options.signal,
      },
    ),
  deleteTrafficProbeTarget: (targetId: number | string, options: RequestOptions = {}) =>
    apiClient.request<{ deleted: boolean }>(API_ENDPOINTS.system.trafficHijack.target(targetId), {
      method: "DELETE",
      signal: options.signal,
    }),
  runTrafficProbe: (targetId: number | string, options: RequestOptions = {}) =>
    apiClient.request<SystemTrafficProbeResult>(
      API_ENDPOINTS.system.trafficHijack.probe(targetId),
      {
        method: "POST",
        signal: options.signal,
      },
    ),
  listTrafficProbeResults: (
    query: SystemTrafficProbeResultListQuery = {},
    options: RequestOptions = {},
  ) =>
    apiClient.request<SystemTrafficProbeResultPage>(API_ENDPOINTS.system.trafficHijack.results, {
      query,
      signal: options.signal,
    }),
  listTrafficHijackEvents: (
    query: SystemTrafficHijackEventListQuery = {},
    options: RequestOptions = {},
  ) =>
    apiClient.request<SystemTrafficHijackEventPage>(API_ENDPOINTS.system.trafficHijack.events, {
      query,
      signal: options.signal,
    }),
  resolveTrafficHijackEvent: (eventId: number | string, options: RequestOptions = {}) =>
    apiClient.request<SystemTrafficHijackEvent>(
      API_ENDPOINTS.system.trafficHijack.eventResolve(eventId),
      {
        method: "POST",
        signal: options.signal,
      },
    ),
  listAPIs: (options: RequestOptions = {}) =>
    apiClient.request<SystemAPIGroup[]>(API_ENDPOINTS.system.apis, {
      signal: options.signal,
    }),
  syncAPIs: (options: RequestOptions = {}) =>
    apiClient.request<SystemAPISyncResult>(API_ENDPOINTS.system.apiSync, {
      method: "POST",
      signal: options.signal,
    }),
  syncAPIPermissions: (options: RequestOptions = {}) =>
    apiClient.request<SystemPermissionSyncResult>(API_ENDPOINTS.system.apiPermissionsSync, {
      method: "POST",
      signal: options.signal,
    }),
  getConfig: (options: RequestOptions = {}) =>
    apiClient.request<SystemConfigSnapshot>(API_ENDPOINTS.system.config, {
      signal: options.signal,
    }),
  updateConfig: (body: SystemConfigUpdateInput, options: RequestOptions = {}) =>
    apiClient.request<SystemConfigSnapshot>(API_ENDPOINTS.system.config, {
      body,
      method: "PATCH",
      signal: options.signal,
    }),
  listMenus: (options: RequestOptions = {}) =>
    apiClient.request<SystemMenuGroup[]>(API_ENDPOINTS.system.menus, {
      signal: options.signal,
    }),
  listDictionaries: (options: RequestOptions = {}) =>
    apiClient.request<SystemDictionaryCatalog>(API_ENDPOINTS.system.dictionaries, {
      signal: options.signal,
    }),
  createDictionary: (body: SystemDictionaryInput, options: RequestOptions = {}) =>
    apiClient.request<SystemDictionary>(API_ENDPOINTS.system.dictionaries, {
      body,
      method: "POST",
      signal: options.signal,
    }),
  updateDictionary: (
    dictionaryId: number | string,
    body: SystemDictionaryUpdateInput,
    options: RequestOptions = {},
  ) =>
    apiClient.request<SystemDictionary>(API_ENDPOINTS.system.dictionary(dictionaryId), {
      body,
      method: "PATCH",
      signal: options.signal,
    }),
  deleteDictionary: (dictionaryId: number | string, options: RequestOptions = {}) =>
    apiClient.request<{ deleted: boolean }>(API_ENDPOINTS.system.dictionary(dictionaryId), {
      method: "DELETE",
      signal: options.signal,
    }),
  createDictionaryItem: (
    dictionaryId: number | string,
    body: SystemDictionaryItemInput,
    options: RequestOptions = {},
  ) =>
    apiClient.request<SystemDictionaryItem>(API_ENDPOINTS.system.dictionaryItems(dictionaryId), {
      body,
      method: "POST",
      signal: options.signal,
    }),
  updateDictionaryItem: (
    itemId: number | string,
    body: SystemDictionaryItemUpdateInput,
    options: RequestOptions = {},
  ) =>
    apiClient.request<SystemDictionaryItem>(API_ENDPOINTS.system.dictionaryItem(itemId), {
      body,
      method: "PATCH",
      signal: options.signal,
    }),
  deleteDictionaryItem: (itemId: number | string, options: RequestOptions = {}) =>
    apiClient.request<{ deleted: boolean }>(API_ENDPOINTS.system.dictionaryItem(itemId), {
      method: "DELETE",
      signal: options.signal,
    }),
  listParameters: (query: SystemParameterListQuery = {}, options: RequestOptions = {}) =>
    apiClient.request<SystemParameterPage>(API_ENDPOINTS.system.parameters, {
      query,
      signal: options.signal,
    }),
  createParameter: (body: SystemParameterInput, options: RequestOptions = {}) =>
    apiClient.request<SystemParameter>(API_ENDPOINTS.system.parameters, {
      body,
      method: "POST",
      signal: options.signal,
    }),
  getParameterByKey: (key: string, options: RequestOptions = {}) =>
    apiClient.request<SystemParameter>(API_ENDPOINTS.system.parameterValue, {
      query: { key },
      signal: options.signal,
    }),
  updateParameter: (
    parameterId: number | string,
    body: SystemParameterUpdateInput,
    options: RequestOptions = {},
  ) =>
    apiClient.request<SystemParameter>(API_ENDPOINTS.system.parameter(parameterId), {
      body,
      method: "PATCH",
      signal: options.signal,
    }),
  deleteParameter: (parameterId: number | string, options: RequestOptions = {}) =>
    apiClient.request<{ deleted: boolean }>(API_ENDPOINTS.system.parameter(parameterId), {
      method: "DELETE",
      signal: options.signal,
    }),
  deleteParameters: (ids: Array<number | string>, options: RequestOptions = {}) =>
    apiClient.request<{ deleted: boolean }>(API_ENDPOINTS.system.parameters, {
      body: { ids },
      method: "DELETE",
      signal: options.signal,
    }),
  listOperationRecords: (
    query: SystemOperationRecordListQuery = {},
    options: RequestOptions = {},
  ) =>
    apiClient.request<SystemOperationRecordPage>(API_ENDPOINTS.system.operationRecords, {
      query,
      signal: options.signal,
    }),
  deleteOperationRecords: (ids: Array<number | string>, options: RequestOptions = {}) =>
    apiClient.request<{ deleted: boolean }>(API_ENDPOINTS.system.operationRecords, {
      body: { ids },
      method: "DELETE",
      signal: options.signal,
    }),
  listMediaCategories: (options: RequestOptions = {}) =>
    apiClient.request<SystemMediaCategoryCatalog>(API_ENDPOINTS.system.media.categories, {
      signal: options.signal,
    }),
  upsertMediaCategory: (body: SystemMediaCategoryInput, options: RequestOptions = {}) =>
    apiClient.request<SystemMediaCategory>(API_ENDPOINTS.system.media.categories, {
      body,
      method: "POST",
      signal: options.signal,
    }),
  deleteMediaCategory: (categoryId: number | string, options: RequestOptions = {}) =>
    apiClient.request<{ deleted: boolean }>(API_ENDPOINTS.system.media.category(categoryId), {
      method: "DELETE",
      signal: options.signal,
    }),
  listMediaAssets: (query: SystemMediaAssetListQuery = {}, options: RequestOptions = {}) =>
    apiClient.request<SystemMediaAssetPage>(API_ENDPOINTS.system.media.assets, {
      query,
      signal: options.signal,
    }),
  updateMediaAsset: (
    assetId: number | string,
    body: SystemMediaAssetUpdateInput,
    options: RequestOptions = {},
  ) =>
    apiClient.request<SystemMediaAsset>(API_ENDPOINTS.system.media.asset(assetId), {
      body,
      method: "PATCH",
      signal: options.signal,
    }),
  downloadMediaAsset: (assetId: number | string, options: RequestOptions = {}) =>
    apiClient.download(API_ENDPOINTS.system.media.assetDownload(assetId), {
      signal: options.signal,
    }),
  deleteMediaAsset: (assetId: number | string, options: RequestOptions = {}) =>
    apiClient.request<{ deleted: boolean }>(API_ENDPOINTS.system.media.asset(assetId), {
      method: "DELETE",
      signal: options.signal,
    }),
  uploadMediaAsset: (file: Blob, categoryId?: number | string, options: RequestOptions = {}) => {
    const body = new FormData();
    body.append("file", file);
    if (categoryId !== undefined && categoryId !== null && categoryId !== "") {
      body.append("categoryId", String(categoryId));
    }

    return apiClient.request<SystemMediaAsset>(API_ENDPOINTS.system.media.assetUpload, {
      body,
      method: "POST",
      signal: options.signal,
    });
  },
  importMediaURLs: (body: SystemMediaURLImportInput, options: RequestOptions = {}) =>
    apiClient.request<SystemMediaURLImportResult>(API_ENDPOINTS.system.media.importURL, {
      body,
      method: "POST",
      signal: options.signal,
    }),
  checkMediaResumableUpload: (body: SystemMediaResumableCheckInput, options: RequestOptions = {}) =>
    apiClient.request<SystemMediaResumableCheckResult>(API_ENDPOINTS.system.media.resumableCheck, {
      body,
      method: "POST",
      signal: options.signal,
    }),
  uploadMediaChunk: (
    file: Blob,
    metadata: SystemMediaResumableChunkMetadata,
    options: RequestOptions = {},
  ) => {
    const body = new FormData();
    body.append("file", file);
    body.append("chunkHash", metadata.chunkHash);
    body.append("chunkIndex", String(metadata.chunkIndex));
    body.append("chunkTotal", String(metadata.chunkTotal));
    body.append("fileHash", metadata.fileHash);
    body.append("fileName", metadata.fileName);
    body.append("sessionId", String(metadata.sessionId));

    return apiClient.request<SystemMediaResumableChunkResult>(
      API_ENDPOINTS.system.media.resumableChunks,
      {
        body,
        method: "POST",
        signal: options.signal,
      },
    );
  },
  completeMediaResumableUpload: (
    body: SystemMediaResumableCompleteInput,
    options: RequestOptions = {},
  ) =>
    apiClient.request<SystemMediaResumableCompleteResult>(
      API_ENDPOINTS.system.media.resumableComplete,
      {
        body,
        method: "POST",
        signal: options.signal,
      },
    ),
  abortMediaResumableUpload: (
    body: SystemMediaResumableCompleteInput,
    options: RequestOptions = {},
  ) =>
    apiClient.request<SystemMediaResumableAbortResult>(API_ENDPOINTS.system.media.resumableAbort, {
      body,
      method: "POST",
      signal: options.signal,
    }),
  listVersions: (query: SystemVersionListQuery = {}, options: RequestOptions = {}) =>
    apiClient.request<SystemVersionPage>(API_ENDPOINTS.system.versions, {
      query,
      signal: options.signal,
    }),
  listVersionSources: (options: RequestOptions = {}) =>
    apiClient.request<SystemVersionSourceCatalog>(API_ENDPOINTS.system.versionSources, {
      signal: options.signal,
    }),
  exportVersion: (body: SystemVersionExportInput, options: RequestOptions = {}) =>
    apiClient.request<SystemVersionDetail>(API_ENDPOINTS.system.versionExport, {
      body,
      method: "POST",
      signal: options.signal,
    }),
  importVersion: (versionData: string, options: RequestOptions = {}) =>
    apiClient.request<SystemVersionImportResult>(API_ENDPOINTS.system.versionImport, {
      body: { versionData },
      method: "POST",
      signal: options.signal,
    }),
  getVersion: (versionId: number | string, options: RequestOptions = {}) =>
    apiClient.request<SystemVersionDetail>(API_ENDPOINTS.system.version(versionId), {
      signal: options.signal,
    }),
  downloadVersion: (versionId: number | string, options: RequestOptions = {}) =>
    apiClient.request<SystemVersionPackage>(API_ENDPOINTS.system.versionDownload(versionId), {
      signal: options.signal,
    }),
  deleteVersion: (versionId: number | string, options: RequestOptions = {}) =>
    apiClient.request<{ deleted: boolean }>(API_ENDPOINTS.system.version(versionId), {
      method: "DELETE",
      signal: options.signal,
    }),
  deleteVersions: (ids: Array<number | string>, options: RequestOptions = {}) =>
    apiClient.request<{ deleted: boolean }>(API_ENDPOINTS.system.versions, {
      body: { ids },
      method: "DELETE",
      signal: options.signal,
    }),
};
