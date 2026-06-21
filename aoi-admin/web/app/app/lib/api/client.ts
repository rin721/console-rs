import { i18n } from "~/i18n/i18n";
import { toBackendLocale } from "~/i18n/locales";
import { API_ENDPOINTS } from "./endpoints";
import type { ApiResult, SessionSnapshot } from "./types";

type RequestMethod = "DELETE" | "GET" | "PATCH" | "POST" | "PUT";

export type ApiRequestOptions = {
  auth?: boolean;
  body?: BodyInit | Record<string, unknown> | null;
  headers?: HeadersInit;
  method?: RequestMethod;
  query?: Record<string, unknown>;
  retryAuth?: boolean;
  signal?: AbortSignal;
};

export type ApiDownload = {
  blob: Blob;
  contentType: string;
  filename: string;
};

export type ApiClientAuthRuntime = {
  clientTypeHeader?: string;
  csrfCookieName?: string;
  csrfEnabled?: boolean;
  csrfHeaderName?: string;
  defaultClientType?: string;
  defaultProductCode?: string;
  productHeader?: string;
};

type ApiClientConfig = {
  onRefresh?: (session: SessionSnapshot) => void;
  onUnauthorized?: () => void;
};

const defaultAuthRuntime: Required<ApiClientAuthRuntime> = {
  clientTypeHeader: "X-Aoi-Client-Type",
  csrfCookieName: "aoi_csrf",
  csrfEnabled: true,
  csrfHeaderName: "X-CSRF-Token",
  defaultClientType: "pc_web",
  defaultProductCode: "aoi-admin",
  productHeader: "X-Aoi-Product-Code",
};

let authRuntime = { ...defaultAuthRuntime };

export function configureApiClientAuthRuntime(settings: ApiClientAuthRuntime) {
  authRuntime = {
    ...authRuntime,
    ...Object.fromEntries(
      Object.entries(settings).filter(([, value]) => value !== undefined && value !== ""),
    ),
  };
}

export class ApiError extends Error {
  code: number | string;
  endpoint: string;
  payload?: unknown;
  status: number;
  traceId?: string;

  constructor(
    message: string,
    status: number,
    endpoint: string,
    code: number | string,
    payload?: unknown,
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.endpoint = endpoint;
    this.code = code;

    if (payload && typeof payload === "object" && "traceId" in payload) {
      this.traceId = String(payload.traceId);
    }
    this.payload = payload;
  }
}

export function resolveApiBaseUrl() {
  const envBase = import.meta.env.VITE_PUBLIC_API_BASE_URL?.trim();
  if (envBase) {
    return envBase.replace(/\/+$/, "");
  }
  if (typeof window !== "undefined") {
    return window.location.origin;
  }
  return "";
}

export function createApiClient(config: ApiClientConfig = {}) {
  let refreshPromise: Promise<SessionSnapshot | null> | null = null;

  async function request<T>(endpoint: string, options: ApiRequestOptions = {}): Promise<T> {
    return requestWithAuthRetry<T>(endpoint, options, options.retryAuth !== false);
  }

  async function download(endpoint: string, options: ApiRequestOptions = {}): Promise<ApiDownload> {
    return downloadWithAuthRetry(endpoint, options, options.retryAuth !== false);
  }

  async function requestWithAuthRetry<T>(
    endpoint: string,
    options: ApiRequestOptions,
    allowRefresh: boolean,
  ): Promise<T> {
    try {
      return await send<T>(endpoint, options);
    } catch (error) {
      if (
        error instanceof ApiError &&
        error.status === 401 &&
        allowRefresh &&
        options.auth !== false &&
        endpoint !== API_ENDPOINTS.auth.refresh
      ) {
        const refreshed = await refreshSession();
        if (refreshed) {
          return requestWithAuthRetry<T>(endpoint, options, false);
        }
      }

      if (error instanceof ApiError && error.status === 401 && options.auth !== false) {
        config.onUnauthorized?.();
      }
      throw error;
    }
  }

  async function downloadWithAuthRetry(
    endpoint: string,
    options: ApiRequestOptions,
    allowRefresh: boolean,
  ): Promise<ApiDownload> {
    try {
      return await sendDownload(endpoint, options);
    } catch (error) {
      if (
        error instanceof ApiError &&
        error.status === 401 &&
        allowRefresh &&
        options.auth !== false &&
        endpoint !== API_ENDPOINTS.auth.refresh
      ) {
        const refreshed = await refreshSession();
        if (refreshed) {
          return downloadWithAuthRetry(endpoint, options, false);
        }
      }

      if (error instanceof ApiError && error.status === 401 && options.auth !== false) {
        config.onUnauthorized?.();
      }
      throw error;
    }
  }

  async function send<T>(endpoint: string, options: ApiRequestOptions): Promise<T> {
    const headers = new Headers(options.headers);
    headers.set("Accept", "application/json");
    headers.set("X-Locale", toBackendLocale(i18n.language));
    applyAuthRuntimeHeaders(headers, options.method ?? "GET");

    const body = normalizeBody(options.body, headers);
    const response = await fetch(resolveEndpointUrl(endpoint, options.query), {
      body,
      credentials: "include",
      headers,
      method: options.method ?? "GET",
      signal: options.signal,
    });

    const payload = await readPayload<T>(response);
    if (!response.ok) {
      throw toApiError(response, endpoint, payload);
    }
    if (typeof payload === "string" && !isJsonResponse(response)) {
      throw new ApiError(
        i18n.t("errors.api.requestFailed"),
        response.status,
        endpoint,
        "NON_JSON_RESPONSE",
        payload,
      );
    }

    if (isApiResult(payload)) {
      if (payload.code !== 0) {
        throw new ApiError(
          payload.message || i18n.t("errors.api.requestFailed"),
          response.status,
          endpoint,
          payload.code,
          payload,
        );
      }
      return payload.data as T;
    }

    return payload as T;
  }

  async function sendDownload(endpoint: string, options: ApiRequestOptions): Promise<ApiDownload> {
    const headers = new Headers(options.headers);
    headers.set("Accept", "application/octet-stream");
    headers.set("X-Locale", toBackendLocale(i18n.language));
    applyAuthRuntimeHeaders(headers, options.method ?? "GET");

    const response = await fetch(resolveEndpointUrl(endpoint, options.query), {
      credentials: "include",
      headers,
      method: options.method ?? "GET",
      signal: options.signal,
    });

    if (!response.ok) {
      const payload = await readPayload<unknown>(response);
      throw toApiError(response, endpoint, payload);
    }

    return {
      blob: await response.blob(),
      contentType: response.headers.get("content-type") || "application/octet-stream",
      filename: filenameFromContentDisposition(response.headers.get("content-disposition")),
    };
  }

  async function refreshSession() {
    if (!refreshPromise) {
      refreshPromise = performRefreshSession().finally(() => {
        refreshPromise = null;
      });
    }
    return refreshPromise;
  }

  async function performRefreshSession() {
    try {
      const session = await send<SessionSnapshot>(API_ENDPOINTS.auth.refresh, {
        auth: false,
        method: "POST",
        retryAuth: false,
      });
      config.onRefresh?.(session);
      return session;
    } catch {
      config.onUnauthorized?.();
      return null;
    }
  }

  return { download, request };
}

function applyAuthRuntimeHeaders(headers: Headers, method: RequestMethod) {
  if (authRuntime.defaultProductCode) {
    headers.set(authRuntime.productHeader, authRuntime.defaultProductCode);
  }
  if (authRuntime.defaultClientType) {
    headers.set(authRuntime.clientTypeHeader, authRuntime.defaultClientType);
  }
  if (!isSafeMethod(method) && authRuntime.csrfEnabled) {
    const csrfToken = cookieValue(authRuntime.csrfCookieName);
    if (csrfToken) {
      headers.set(authRuntime.csrfHeaderName, csrfToken);
    }
  }
}

function isSafeMethod(method: RequestMethod) {
  return method === "GET";
}

function cookieValue(name: string) {
  if (typeof document === "undefined" || !name) {
    return "";
  }
  const prefix = `${encodeURIComponent(name)}=`;
  return (
    document.cookie
      .split(";")
      .map((part) => part.trim())
      .find((part) => part.startsWith(prefix))
      ?.slice(prefix.length) ?? ""
  );
}

export function resolveEndpointUrl(endpoint: string, query?: Record<string, unknown>) {
  const base = resolveApiBaseUrl();
  const url = endpoint.startsWith("http")
    ? new URL(endpoint)
    : new URL(endpoint, base || "http://localhost");

  for (const [key, value] of Object.entries(query ?? {})) {
    if (value === undefined || value === null || value === "") {
      continue;
    }
    if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
      url.searchParams.set(key, String(value));
    }
  }

  if (!base && !endpoint.startsWith("http")) {
    return `${url.pathname}${url.search}`;
  }
  return url.toString();
}

function normalizeBody(body: ApiRequestOptions["body"], headers: Headers) {
  if (body === undefined || body === null) {
    return undefined;
  }
  if (body instanceof FormData || body instanceof Blob || typeof body === "string") {
    return body;
  }

  headers.set("Content-Type", "application/json");
  return JSON.stringify(body);
}

async function readPayload<T>(response: Response): Promise<ApiResult<T> | T | null> {
  if (response.status === 204) {
    return null;
  }

  const text = await response.text();
  if (!text) {
    return null;
  }

  try {
    return JSON.parse(text) as ApiResult<T> | T;
  } catch {
    return text as T;
  }
}

function toApiError(response: Response, endpoint: string, payload: unknown) {
  if (isApiResult(payload)) {
    return new ApiError(
      payload.message || response.statusText || i18n.t("errors.api.requestFailed"),
      response.status,
      endpoint,
      payload.code,
      payload,
    );
  }

  return new ApiError(
    response.statusText || i18n.t("errors.api.requestFailed"),
    response.status,
    endpoint,
    response.status,
    payload,
  );
}

function isJsonResponse(response: Response) {
  return response.headers.get("content-type")?.toLowerCase().includes("application/json") ?? false;
}

function isApiResult(value: unknown): value is ApiResult<unknown> {
  return Boolean(value && typeof value === "object" && "code" in value);
}

function filenameFromContentDisposition(value: string | null) {
  if (!value) {
    return "";
  }

  const utf8Match = value.match(/filename\*=UTF-8''([^;]+)/i);
  if (utf8Match?.[1]) {
    try {
      return decodeURIComponent(utf8Match[1]);
    } catch {
      return utf8Match[1];
    }
  }

  const quoted = value.match(/filename="([^"]+)"/i);
  if (quoted?.[1]) {
    return quoted[1];
  }

  const plain = value.match(/filename=([^;]+)/i);
  return plain?.[1]?.trim() || "";
}
