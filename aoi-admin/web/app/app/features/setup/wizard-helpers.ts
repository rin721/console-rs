import { toBackendLocale } from "~/i18n/locales";
import type { AppLocale } from "~/i18n/resources";
import type {
  InitializationRunRequest,
  PasswordPolicy,
  SetupFieldSchema,
  SetupSchema,
  SetupStepSchema,
  SetupVisibilityCondition,
} from "~/lib/api/setup";

export const languageStepKey = "__language";

export type WizardStep = SetupStepSchema & {
  local?: boolean;
};

export type StepValues = Record<string, Record<string, unknown>>;

export type PasswordPolicyFailure = "lower" | "minLength" | "number" | "symbol" | "upper";

export type StepValidationResult =
  | {
      fieldKey: string;
      fieldLabel: string;
      type: "required";
    }
  | {
      fieldKey: string;
      failures: PasswordPolicyFailure[];
      minLength: number;
      type: "passwordPolicy";
    }
  | {
      fieldKey: "passwordConfirm";
      type: "passwordConfirmRequired";
    }
  | {
      fieldKey: "passwordConfirm";
      type: "passwordMismatch";
    };

export function normalizeStep(step: SetupStepSchema): SetupStepSchema {
  const groups = Array.isArray(step.groups)
    ? step.groups.map((group) => ({
        ...group,
        fields: Array.isArray(group.fields) ? group.fields : [],
      }))
    : [];
  const topLevelFields = Array.isArray(step.fields) ? step.fields : [];
  const fields = topLevelFields.length ? topLevelFields : groups.flatMap((group) => group.fields);
  return {
    ...step,
    fields,
    groups,
    routeSlug: step.routeSlug || routeSlugForKey(step.key),
  };
}

export function routeSlugForStep(step: SetupStepSchema) {
  return step.routeSlug || routeSlugForKey(step.key);
}

export function routeSlugForKey(key: string) {
  const mapping: Record<string, string> = {
    [languageStepKey]: "language",
    "cache.configure": "cache",
    "catalog.sync": "permissions",
    "config.diagnostics": "diagnostics",
    "config.source": "config-source",
    "database.configure": "database",
    "database.migrate": "migrate",
    "dependencies.check": "preflight",
    "iam.owner": "owner",
    "optional.finalize": "finalize",
    "site.configure": "site",
    "storage.configure": "storage",
    "system.configure": "system",
    "system.seed": "seed",
    "verify.finish": "verify",
    welcome: "welcome",
  };
  return mapping[key] || key;
}

export function hydrateValues(
  current: StepValues,
  schema: SetupSchema,
  locale: AppLocale,
  previousLocale?: AppLocale,
): StepValues {
  const next = { ...current };
  const backendLocale = toBackendLocale(locale);
  const previousBackendLocale = previousLocale ? toBackendLocale(previousLocale) : undefined;
  for (const step of schema.steps.map(normalizeStep)) {
    const bucket = { ...(next[step.key] ?? {}) };
    for (const field of step.fields) {
      if (field.key === "i18n.defaultLocale") {
        if (
          bucket[field.key] === undefined ||
          scalarString(bucket[field.key]) === previousLocale ||
          scalarString(bucket[field.key]) === previousBackendLocale
        ) {
          bucket[field.key] = backendLocale;
        }
        continue;
      }
      if (bucket[field.key] !== undefined) {
        continue;
      }
      if (field.sensitive) {
        bucket[field.key] = defaultFieldValue(field);
      } else if (field.value !== undefined) {
        bucket[field.key] = field.value;
      } else if (field.default !== undefined) {
        bucket[field.key] = field.default;
      } else {
        bucket[field.key] = defaultFieldValue(field);
      }
    }
    next[step.key] = bucket;
  }
  return next;
}

export function syncDefaultLocaleValue(
  current: StepValues,
  schema: SetupSchema | null | undefined,
  locale: AppLocale,
): StepValues {
  if (!schema) {
    return current;
  }
  const backendLocale = toBackendLocale(locale);
  let changed = false;
  const next = { ...current };

  for (const step of schema.steps.map(normalizeStep)) {
    if (!step.fields.some((field) => field.key === "i18n.defaultLocale")) {
      continue;
    }
    const bucket = { ...(next[step.key] ?? {}) };
    if (bucket["i18n.defaultLocale"] !== backendLocale) {
      bucket["i18n.defaultLocale"] = backendLocale;
      changed = true;
    }
    next[step.key] = bucket;
  }

  return changed ? next : current;
}

export function defaultFieldValue(field: SetupFieldSchema) {
  if (field.type === "boolean") {
    return false;
  }
  if (field.type === "number") {
    return 0;
  }
  return "";
}

export function isVisible(
  condition: SetupVisibilityCondition | undefined,
  values: Record<string, unknown>,
) {
  if (!condition) {
    return true;
  }
  const value = values[condition.field];
  if (condition.in) {
    return condition.in.includes(scalarString(value));
  }
  if ("equals" in condition) {
    return value === condition.equals;
  }
  return true;
}

export function visibleStepValues(step: SetupStepSchema, values: Record<string, unknown>) {
  return visibleFieldsForStep(step, values).reduce<Record<string, unknown>>((acc, field) => {
    if (values[field.key] !== undefined) {
      acc[field.key] = values[field.key];
    }
    return acc;
  }, {});
}

export function visibleConfigValues(step: SetupStepSchema, values: Record<string, unknown>) {
  return visibleFieldsForStep(step, values).reduce<Record<string, unknown>>((acc, field) => {
    if (field.configPath && values[field.key] !== undefined) {
      acc[field.key] = values[field.key];
    }
    return acc;
  }, {});
}

export function validateStepFields(
  step: SetupStepSchema,
  values: Record<string, unknown>,
  passwordPolicy?: PasswordPolicy,
): StepValidationResult | null {
  const visibleFields = visibleFieldsForStep(step, values);
  for (const field of visibleFields) {
    if (!field.required) {
      continue;
    }
    const value = values[field.key];
    if (value === undefined || value === null || scalarString(value).trim() === "") {
      return {
        fieldKey: field.key,
        fieldLabel: field.label,
        type: "required",
      };
    }
    if (field.key === "password") {
      const failures = validatePasswordPolicy(scalarString(value), passwordPolicy);
      if (failures.length) {
        return {
          fieldKey: field.key,
          failures,
          minLength: passwordPolicy?.minLength ?? 0,
          type: "passwordPolicy",
        };
      }
    }
  }
  if (
    step.key === "iam.owner" &&
    visibleFields.some((field) => field.key === "password" && field.required)
  ) {
    const password = scalarString(values.password);
    const passwordConfirm = scalarString(values.passwordConfirm);
    if (password && !passwordConfirm) {
      return {
        fieldKey: "passwordConfirm",
        type: "passwordConfirmRequired",
      };
    }
    if (password && passwordConfirm !== password) {
      return {
        fieldKey: "passwordConfirm",
        type: "passwordMismatch",
      };
    }
  }
  return null;
}

export function validatePasswordPolicy(password: string, policy?: PasswordPolicy) {
  const failures: PasswordPolicyFailure[] = [];
  if (!policy) {
    return failures;
  }
  if (policy.minLength > 0 && password.length < policy.minLength) {
    failures.push("minLength");
  }
  if (policy.requireLower && !/[a-z]/.test(password)) {
    failures.push("lower");
  }
  if (policy.requireUpper && !/[A-Z]/.test(password)) {
    failures.push("upper");
  }
  if (policy.requireNumber && !/[0-9]/.test(password)) {
    failures.push("number");
  }
  if (policy.requireSymbol && !/[^A-Za-z0-9]/.test(password)) {
    failures.push("symbol");
  }
  return failures;
}

export function inputTypeForField(field: SetupFieldSchema) {
  if (field.sensitive || field.type === "password") {
    return "password";
  }
  if (field.type === "number") {
    return "number";
  }
  if (field.type === "email") {
    return "email";
  }
  return "text";
}

export function buildRunRequest(
  values: StepValues,
  _steps: SetupStepSchema[],
  setupToken?: string,
): InitializationRunRequest | null {
  const flatValues = Object.fromEntries(
    Object.values(values).flatMap((bucket) => Object.entries(bucket)),
  );
  const password = scalarString(flatValues.password);
  const request: InitializationRunRequest = {
    displayName: textValue(flatValues.displayName),
    email: textValue(flatValues.email),
    mode: "first_run",
    orgCode: textValue(flatValues.orgCode),
    orgName: textValue(flatValues.orgName),
    password,
    setupToken,
    username: textValue(flatValues.username),
  };
  const createServiceToken = booleanValue(flatValues.createServiceToken);
  const serviceTokenDays = numberValue(flatValues.serviceTokenDays);
  const serviceTokenRemark = textValue(flatValues.serviceTokenRemark);

  if (createServiceToken !== undefined) {
    request.createServiceToken = createServiceToken;
  }
  if (serviceTokenDays !== undefined) {
    request.serviceTokenDays = serviceTokenDays;
  }
  if (serviceTokenRemark) {
    request.serviceTokenRemark = serviceTokenRemark;
  }

  if (
    !request.email ||
    !request.orgCode ||
    !request.orgName ||
    !request.password.trim() ||
    !request.username
  ) {
    return null;
  }
  return request;
}

export function scalarString(value: unknown) {
  if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  return "";
}

function visibleFieldsForStep(step: SetupStepSchema, values: Record<string, unknown>) {
  if (step.groups?.length) {
    return step.groups.flatMap((group) => {
      if (!isVisible(group.visibleWhen, values)) {
        return [];
      }
      return group.fields.filter((field) => isVisible(field.visibleWhen, values));
    });
  }
  return step.fields.filter((field) => isVisible(field.visibleWhen, values));
}

function textValue(value: unknown) {
  return scalarString(value).trim();
}

function booleanValue(value: unknown) {
  if (typeof value === "boolean") {
    return value;
  }
  if (typeof value === "string") {
    if (value === "true") {
      return true;
    }
    if (value === "false") {
      return false;
    }
  }
  return undefined;
}

function numberValue(value: unknown) {
  if (value === undefined || value === null || value === "") {
    return undefined;
  }
  const parsed = typeof value === "number" ? value : Number(value);
  return Number.isFinite(parsed) ? parsed : undefined;
}
