import { afterEach, describe, expect, it, vi } from "vitest";

import { ApiError, createApiClient, resolveEndpointUrl } from "./client";
import { API_ENDPOINTS } from "./endpoints";

describe("api client", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("resolves relative API URLs with query parameters", () => {
    expect(resolveEndpointUrl(API_ENDPOINTS.orgs.users(1), { page: 2, empty: "" })).toBe(
      "http://localhost:3000/api/v1/orgs/1/users?page=2",
    );
  });

  it("unwraps backend Result payloads", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ code: 0, data: { ok: true } }), { status: 200 }),
    );

    const client = createApiClient();
    await expect(client.request<{ ok: boolean }>(API_ENDPOINTS.health)).resolves.toEqual({ ok: true });
  });

  it("passes abort signals to fetch", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ code: 0, data: { ok: true } }), { status: 200 }),
    );
    const controller = new AbortController();

    const client = createApiClient();
    await client.request(API_ENDPOINTS.auth.captcha, { auth: false, signal: controller.signal });

    expect(fetchMock).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({ signal: controller.signal }),
    );
  });

  it("normalizes failed backend Result payloads", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ code: "BAD_REQUEST", message: "Invalid request" }), {
        status: 400,
      }),
    );

    const client = createApiClient();
    await expect(client.request(API_ENDPOINTS.health)).rejects.toBeInstanceOf(ApiError);
  });

  it("rejects successful non-json API fallback payloads", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response("<!doctype html><html></html>", {
        headers: { "content-type": "text/html" },
        status: 200,
      }),
    );

    const client = createApiClient();
    await expect(client.request(API_ENDPOINTS.setup.status)).rejects.toMatchObject({
      code: "NON_JSON_RESPONSE",
      status: 200,
    });
  });
});
