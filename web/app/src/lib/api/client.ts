import { endpoints } from "./endpoints";
import type { Locale, PublicSettings, SessionSnapshot } from "./types";

export type ApiRuntime = {
  locale: Locale;
  productHeader?: string;
  clientTypeHeader?: string;
  productCode?: string;
  clientType?: string;
  csrfEnabled: boolean;
  csrfCookieName?: string;
  csrfHeaderName?: string;
};

export class ApiError extends Error {
  status: number;
  code: string;

  constructor(message: string, status: number, code: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

const defaultRuntime: ApiRuntime = {
  locale: "zh-CN",
  csrfEnabled: false
};

let runtime: ApiRuntime = { ...defaultRuntime };
let onRefresh: ((session: SessionSnapshot) => void) | undefined;
let onUnauthorized: (() => void) | undefined;
let refreshPromise: Promise<SessionSnapshot | null> | null = null;

export function configureApi(settings: Partial<ApiRuntime>) {
  runtime = { ...runtime, ...settings };
}

export function configureApiFromPublicSettings(settings: PublicSettings, locale: Locale) {
  configureApi({
    locale,
    productCode: settings.product_code,
    productHeader: settings.auth.product_header,
    clientTypeHeader: settings.auth.client_type_header,
    clientType: settings.auth.default_client_type,
    csrfEnabled: settings.auth.csrf_enabled,
    csrfCookieName: settings.auth.csrf_cookie_name,
    csrfHeaderName: settings.auth.csrf_header_name
  });
}

export function configureAuthCallbacks(callbacks: {
  onRefresh?: (session: SessionSnapshot) => void;
  onUnauthorized?: () => void;
}) {
  onRefresh = callbacks.onRefresh;
  onUnauthorized = callbacks.onUnauthorized;
}

export async function apiGet<T>(endpoint: string, query?: Record<string, unknown>) {
  return apiRequest<T>(endpoint, { method: "GET", query });
}

export async function apiPost<T>(endpoint: string, body?: Record<string, unknown> | null) {
  return apiRequest<T>(endpoint, { body, method: "POST" });
}

export async function apiPostForm<T>(endpoint: string, body: FormData) {
  return apiRequest<T>(endpoint, { body, method: "POST" });
}

export async function apiPut<T>(endpoint: string, body?: Record<string, unknown> | null) {
  return apiRequest<T>(endpoint, { body, method: "PUT" });
}

export async function apiDelete<T>(endpoint: string, body?: Record<string, unknown> | null) {
  return apiRequest<T>(endpoint, { body, method: "DELETE" });
}

async function apiRequest<T>(
  endpoint: string,
  options: {
    body?: FormData | Record<string, unknown> | null;
    method: "DELETE" | "GET" | "POST" | "PUT";
    query?: Record<string, unknown>;
    retryAuth?: boolean;
  },
): Promise<T> {
  try {
    return await send<T>(endpoint, options);
  } catch (error) {
    if (
      error instanceof ApiError &&
      error.status === 401 &&
      options.retryAuth !== false &&
      endpoint !== endpoints.auth.refresh
    ) {
      const refreshed = await refreshSession();
      if (refreshed) {
        return apiRequest<T>(endpoint, { ...options, retryAuth: false });
      }
    }
    if (error instanceof ApiError && error.status === 401) {
      onUnauthorized?.();
    }
    throw error;
  }
}

async function send<T>(
  endpoint: string,
  options: {
    body?: FormData | Record<string, unknown> | null;
    method: "DELETE" | "GET" | "POST" | "PUT";
    query?: Record<string, unknown>;
  },
): Promise<T> {
  const headers = new Headers({
    Accept: "application/json",
    "X-Locale": runtime.locale
  });
  if (runtime.productHeader && runtime.productCode) {
    headers.set(runtime.productHeader, runtime.productCode);
  }
  if (runtime.clientTypeHeader && runtime.clientType) {
    headers.set(runtime.clientTypeHeader, runtime.clientType);
  }
  if (runtime.csrfEnabled && runtime.csrfCookieName && runtime.csrfHeaderName && options.method !== "GET") {
    const token = cookieValue(runtime.csrfCookieName);
    if (token) headers.set(runtime.csrfHeaderName, token);
  }

  const response = await fetch(urlWithQuery(endpoint, options.query), {
    body: requestBody(options.body),
    credentials: "include",
    headers: options.body == null || options.body instanceof FormData ? headers : withJson(headers),
    method: options.method
  });
  const payload = await readPayload(response);
  if (!response.ok) {
    const apiBody = payload && typeof payload === "object" ? (payload as { code?: string; message?: string }) : {};
    throw new ApiError(apiBody.message || response.statusText, response.status, apiBody.code || String(response.status));
  }
  return payload as T;
}

async function refreshSession() {
  if (!refreshPromise) {
    refreshPromise = send<SessionSnapshot>(endpoints.auth.refresh, {
      method: "POST"
    })
      .then((session) => {
        onRefresh?.(session);
        return session;
      })
      .catch(() => {
        onUnauthorized?.();
        return null;
      })
      .finally(() => {
        refreshPromise = null;
      });
  }
  return refreshPromise;
}

function requestBody(body: FormData | Record<string, unknown> | null | undefined) {
  if (body == null) return undefined;
  if (body instanceof FormData) return body;
  return JSON.stringify(body);
}

function withJson(headers: Headers) {
  headers.set("Content-Type", "application/json");
  return headers;
}

async function readPayload(response: Response) {
  if (response.status === 204) return null;
  const text = await response.text();
  if (!text) return null;
  try {
    return JSON.parse(text) as unknown;
  } catch {
    return text;
  }
}

function urlWithQuery(endpoint: string, query?: Record<string, unknown>) {
  const url = new URL(endpoint, window.location.origin);
  for (const [key, value] of Object.entries(query ?? {})) {
    if (value === undefined || value === null || value === "") continue;
    url.searchParams.set(key, String(value));
  }
  return `${url.pathname}${url.search}`;
}

function cookieValue(name: string) {
  const prefix = `${encodeURIComponent(name)}=`;
  return (
    document.cookie
      .split(";")
      .map((part) => part.trim())
      .find((part) => part.startsWith(prefix))
      ?.slice(prefix.length) ?? ""
  );
}
