import { API_ENDPOINTS } from "./endpoints";
import { apiClient } from "./runtime";
import type { PluginCapabilitiesResponse, PluginHealthStatus, PluginSnapshot } from "./types";

type RequestOptions = {
  signal?: AbortSignal;
};

export const pluginsApi = {
  getPlugin: (pluginId: string, options: RequestOptions = {}) =>
    apiClient.request<PluginSnapshot>(API_ENDPOINTS.plugins.item(pluginId), {
      signal: options.signal,
    }),
  getPluginHealth: (pluginId: string, options: RequestOptions = {}) =>
    apiClient.request<PluginHealthStatus>(API_ENDPOINTS.plugins.health(pluginId), {
      signal: options.signal,
    }),
  listPluginCapabilities: (pluginId: string, options: RequestOptions = {}) =>
    apiClient.request<PluginCapabilitiesResponse>(API_ENDPOINTS.plugins.capabilities(pluginId), {
      signal: options.signal,
    }),
  listPlugins: (options: RequestOptions = {}) =>
    apiClient.request<PluginSnapshot[]>(API_ENDPOINTS.plugins.collection, {
      signal: options.signal,
    }),
};
