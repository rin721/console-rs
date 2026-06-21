import { describe, expect, it } from "vitest";

import type { SetupFieldSchema, SetupSchema, SetupStepSchema } from "~/lib/api/setup";

import {
  buildRunRequest,
  hydrateValues,
  normalizeStep,
  routeSlugForKey,
  syncDefaultLocaleValue,
  validateStepFields,
  visibleConfigValues,
  visibleStepValues,
} from "./wizard-helpers";

const textField = (key: string, required = true, configPath = ""): SetupFieldSchema => {
  const field: SetupFieldSchema = {
    key,
    label: key,
    required,
    sensitive: false,
    type: "text",
  };
  return configPath ? { ...field, configPath } : field;
};

const step = (
  key: string,
  fields: SetupFieldSchema[],
  overrides: Partial<SetupStepSchema> = {},
): SetupStepSchema => ({
  description: key,
  fields,
  key,
  order: 1,
  phase: "test",
  required: true,
  skippable: false,
  testable: false,
  title: key,
  ...overrides,
});

describe("setup wizard helpers", () => {
  it("keeps frontend setup routes aligned with backend step slugs", () => {
    expect(routeSlugForKey("config.source")).toBe("config-source");
    expect(routeSlugForKey("dependencies.check")).toBe("preflight");
    expect(routeSlugForKey("catalog.sync")).toBe("permissions");
    expect(routeSlugForKey("site.configure")).toBe("site");
    expect(routeSlugForKey("optional.finalize")).toBe("finalize");
  });

  it("does not hydrate sensitive backend values into browser form state", () => {
    const schema: SetupSchema = {
      steps: [
        step("database.configure", [
          {
            default: "postgres-secret",
            key: "database.password",
            label: "database.password",
            required: false,
            sensitive: true,
            type: "password",
            value: "existing-secret",
          },
          {
            default: "sqlite",
            key: "database.driver",
            label: "database.driver",
            required: true,
            sensitive: false,
            type: "select",
          },
        ]),
      ],
    };

    expect(hydrateValues({}, schema, "en")).toEqual({
      "database.configure": {
        "database.driver": "sqlite",
        "database.password": "",
      },
    });
    expect(
      hydrateValues({ "database.configure": { "database.password": "entered" } }, schema, "en"),
    ).toMatchObject({
      "database.configure": {
        "database.password": "entered",
      },
    });
  });

  it("normalizes grouped fields when the backend leaves top-level fields empty", () => {
    const groupedStep = step("database.configure", [], {
      groups: [
        {
          fields: [
            {
              default: "sqlite",
              key: "database.driver",
              label: "database.driver",
              required: true,
              sensitive: false,
              type: "select",
            },
          ],
          key: "sqlite",
          title: "SQLite",
        },
      ],
    });

    expect(normalizeStep(groupedStep).fields.map((field) => field.key)).toEqual([
      "database.driver",
    ]);
    expect(hydrateValues({}, { steps: [groupedStep] }, "zh-CN")).toEqual({
      "database.configure": {
        "database.driver": "sqlite",
      },
    });
  });

  it("maps setup default locale fields to backend locale codes", () => {
    const schema: SetupSchema = {
      steps: [
        step("system.configure", [
          {
            key: "i18n.defaultLocale",
            label: "i18n.defaultLocale",
            options: [
              { label: "zh-CN", value: "zh-CN" },
              { label: "en-US", value: "en-US" },
            ],
            required: true,
            sensitive: false,
            type: "select",
          },
        ]),
      ],
    };

    expect(hydrateValues({}, schema, "en")).toMatchObject({
      "system.configure": {
        "i18n.defaultLocale": "en-US",
      },
    });
    expect(
      hydrateValues(
        { "system.configure": { "i18n.defaultLocale": "zh-CN" } },
        schema,
        "en",
        "zh-CN",
      ),
    ).toMatchObject({
      "system.configure": {
        "i18n.defaultLocale": "en-US",
      },
    });
    expect(
      hydrateValues(
        { "system.configure": { "i18n.defaultLocale": "en-US" } },
        schema,
        "zh-CN",
        "en",
      ),
    ).toMatchObject({
      "system.configure": {
        "i18n.defaultLocale": "zh-CN",
      },
    });
    expect(
      hydrateValues(
        { "system.configure": { "i18n.defaultLocale": "en-US" } },
        schema,
        "zh-CN",
        "zh-CN",
      ),
    ).toMatchObject({
      "system.configure": {
        "i18n.defaultLocale": "en-US",
      },
    });
  });

  it("syncs explicit setup language changes into backend default locale fields", () => {
    const schema: SetupSchema = {
      steps: [
        step("system.configure", [], {
          groups: [
            {
              fields: [
                {
                  default: "zh-CN",
                  key: "i18n.defaultLocale",
                  label: "i18n.defaultLocale",
                  options: [
                    { label: "zh-CN", value: "zh-CN" },
                    { label: "en-US", value: "en-US" },
                  ],
                  required: true,
                  sensitive: false,
                  type: "select",
                },
              ],
              key: "locale",
              title: "locale",
            },
          ],
        }),
      ],
    };

    expect(
      syncDefaultLocaleValue(
        { "system.configure": { "i18n.defaultLocale": "zh-CN" } },
        schema,
        "en",
      ),
    ).toMatchObject({
      "system.configure": {
        "i18n.defaultLocale": "en-US",
      },
    });
  });

  it("validates the backend password policy for the owner step", () => {
    const ownerStep = step("iam.owner", [
      {
        key: "password",
        label: "password",
        required: true,
        sensitive: true,
        type: "password",
      },
    ]);

    expect(
      validateStepFields(
        ownerStep,
        { password: "SHORT" },
        {
          minLength: 12,
          requireLower: true,
          requireNumber: true,
          requireSymbol: true,
          requireUpper: true,
        },
      ),
    ).toEqual({
      fieldKey: "password",
      failures: ["minLength", "lower", "number", "symbol"],
      minLength: 12,
      type: "passwordPolicy",
    });
  });

  it("requires matching owner password confirmation without backend schema changes", () => {
    const ownerStep = step("iam.owner", [
      {
        key: "password",
        label: "password",
        required: true,
        sensitive: true,
        type: "password",
      },
    ]);

    expect(
      validateStepFields(ownerStep, {
        password: "Aoi-password-123!",
      }),
    ).toEqual({
      fieldKey: "passwordConfirm",
      type: "passwordConfirmRequired",
    });
    expect(
      validateStepFields(ownerStep, {
        password: "Aoi-password-123!",
        passwordConfirm: "different-password",
      }),
    ).toEqual({
      fieldKey: "passwordConfirm",
      type: "passwordMismatch",
    });
    expect(
      validateStepFields(ownerStep, {
        password: "Aoi-password-123!",
        passwordConfirm: "Aoi-password-123!",
      }),
    ).toBeNull();
  });

  it("builds the initialization request from visible setup fields", () => {
    const ownerStep = step("iam.owner", [
      textField("orgCode"),
      textField("orgName"),
      textField("username"),
      textField("email"),
      { ...textField("password"), sensitive: true, type: "password" },
    ]);
    const finalizeStep = step("optional.finalize", [
      { ...textField("createServiceToken", false), type: "boolean" },
      { ...textField("serviceTokenDays", false), type: "number" },
      textField("serviceTokenRemark", false),
    ]);
    const hiddenStep = step("cache.configure", [], {
      groups: [
        {
          fields: [
            textField("cache.driver", true, "cache.driver"),
            {
              ...textField("redis.address", true, "redis.address"),
              visibleWhen: { equals: "redis", field: "cache.driver" },
            },
          ],
          key: "cache",
          title: "cache",
        },
      ],
    });
    const values = {
      "cache.configure": {
        "cache.driver": "local",
        "redis.address": "127.0.0.1:6379",
      },
      "iam.owner": {
        email: "owner@example.com",
        orgCode: "default",
        orgName: "Default",
        password: "Aoi-password-123!",
        passwordConfirm: "Aoi-password-123!",
        username: "owner",
      },
      "optional.finalize": {
        createServiceToken: true,
        serviceTokenDays: 90,
        serviceTokenRemark: "bootstrap",
      },
    };

    expect(visibleStepValues(hiddenStep, values["cache.configure"])).toEqual({
      "cache.driver": "local",
    });
    expect(visibleConfigValues(hiddenStep, values["cache.configure"])).toEqual({
      "cache.driver": "local",
    });
    const request = buildRunRequest(values, [ownerStep, finalizeStep, hiddenStep], "token");

    expect(request).toMatchObject({
      createServiceToken: true,
      email: "owner@example.com",
      orgCode: "default",
      orgName: "Default",
      password: "Aoi-password-123!",
      serviceTokenDays: 90,
      serviceTokenRemark: "bootstrap",
      setupToken: "token",
      username: "owner",
    });
    expect(request).not.toHaveProperty("values");
    expect(request).not.toHaveProperty("passwordConfirm");
  });
});
