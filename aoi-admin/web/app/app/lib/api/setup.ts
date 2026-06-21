import { API_ENDPOINTS } from "./endpoints";
import { apiClient } from "./runtime";
import type { SessionSnapshot } from "./types";

type BackendString<T extends string> = T | (string & Record<never, never>);

type RequestOptions = {
  signal?: AbortSignal;
};

export type SetupStepStatus = BackendString<
  "failed" | "pending" | "running" | "skipped" | "succeeded"
>;

export type PasswordPolicy = {
  minLength: number;
  requireLower: boolean;
  requireNumber: boolean;
  requireSymbol: boolean;
  requireUpper: boolean;
};

export type InitializationStep = {
  automaticActions?: string[];
  completionMark?: string;
  dependencies?: string[];
  errorCode?: string;
  errorMessage?: string;
  finishedAt?: string;
  goal?: string;
  key: string;
  outputSummary?: string;
  phase: string;
  recovery?: string;
  repairHint?: string;
  restartRequired?: boolean;
  schema?: SetupStepSchema;
  startedAt?: string;
  status: SetupStepStatus;
  testError?: string;
  testInputFingerprint?: string;
  testStatus?: string;
  testSummary?: string;
  title: string;
  userInputs?: string[];
};

export type SetupStatus = {
  allowedActions?: string[];
  completed?: boolean;
  currentStep?: string;
  lastRun?: {
    currentStep: string;
    id: string;
    status: string;
  };
  passwordPolicy?: PasswordPolicy;
  report?: InitializationReport;
  required: boolean;
  restartReason?: string;
  restartRequired?: boolean;
  steps?: InitializationStep[];
};

export type SetupVisibilityCondition = {
  equals?: unknown;
  field: string;
  in?: string[];
};

export type SetupFieldSchema = {
  configPath?: string;
  default?: unknown;
  help?: string;
  key: string;
  label: string;
  options?: Array<{ label: string; value: string }>;
  placeholder?: string;
  required: boolean;
  sensitive: boolean;
  type: BackendString<"boolean" | "email" | "number" | "password" | "select" | "text">;
  value?: unknown;
  visibleWhen?: SetupVisibilityCondition;
};

export type SetupFieldGroup = {
  description?: string;
  fields: SetupFieldSchema[];
  key: string;
  title: string;
  visibleWhen?: SetupVisibilityCondition;
};

export type SetupStepSchema = {
  dependencies?: string[];
  description: string;
  fields: SetupFieldSchema[];
  groups?: SetupFieldGroup[];
  inputFingerprint?: string;
  key: string;
  order: number;
  phase: string;
  required: boolean;
  routeSlug?: string;
  skippable: boolean;
  testable: boolean;
  title: string;
};

export type SetupSchema = {
  steps: SetupStepSchema[];
};

export type SetupTestResult = {
  error?: string;
  inputFingerprint?: string;
  repairHint?: string;
  restartRequired: boolean;
  status: BackendString<"failed" | "skipped" | "succeeded">;
  stepKey: string;
  summary: string;
  testedAt: string;
};

export type SetupConfigSaveResult = {
  envManagedPathsOverwritten?: string[];
  envManagedPersistence?: string;
  inputFingerprint?: string;
  inputSummary: string;
  nextAction?: string;
  restartReason?: string;
  restartRequired: boolean;
  steps?: InitializationStep[];
  stepKey: string;
  test?: SetupTestResult;
};

export type InitializationRunRequest = {
  createServiceToken?: boolean;
  displayName?: string;
  email: string;
  mode?: "first_run" | "repair" | "verify";
  orgCode: string;
  orgName: string;
  password: string;
  serviceTokenDays?: number;
  serviceTokenRemark?: string;
  setupToken?: string;
  username: string;
  values?: Record<string, unknown>;
};

export type InitializationRunResult = {
  loginTokens?: SessionSnapshot;
  loginTokensIssued: boolean;
  report?: InitializationReport;
  restartReason?: string;
  restartRequired?: boolean;
  run: {
    currentStep: string;
    id: string;
    status: string;
  };
  steps: InitializationStep[];
};

export type InitializationReport = {
  failed: number;
  generatedAt: string;
  restartReason?: string;
  restartRequired: boolean;
  risk: number;
  skipped: number;
  successful: number;
  summary: string;
};

export type SetupCompleteResult = {
  completed: boolean;
  report: InitializationReport;
  steps: InitializationStep[];
};

export const setupApi = {
  complete: (setupToken?: string) =>
    apiClient.request<SetupCompleteResult>(API_ENDPOINTS.setup.complete, {
      auth: false,
      body: { setupToken },
      method: "POST",
    }),
  createRun: (body: InitializationRunRequest) =>
    apiClient.request<InitializationRunResult>(API_ENDPOINTS.setup.runs, {
      auth: false,
      body,
      method: "POST",
    }),
  getRunLogs: (runId: string, options: RequestOptions = {}) =>
    apiClient.request<InitializationStep[]>(API_ENDPOINTS.setup.runLogs(runId), {
      auth: false,
      signal: options.signal,
    }),
  getSchema: (setupToken?: string, options: RequestOptions = {}) =>
    apiClient.request<SetupSchema>(API_ENDPOINTS.setup.schema, {
      auth: false,
      query: setupToken ? { setupToken } : undefined,
      signal: options.signal,
    }),
  getStatus: (options: RequestOptions = {}) =>
    apiClient.request<SetupStatus>(API_ENDPOINTS.setup.status, {
      auth: false,
      signal: options.signal,
    }),
  retryRun: (runId: string, body: InitializationRunRequest) =>
    apiClient.request<InitializationRunResult>(API_ENDPOINTS.setup.runRetry(runId), {
      auth: false,
      body,
      method: "POST",
    }),
  saveConfig: (
    stepKey: string,
    body: {
      persist?: boolean;
      setupToken?: string;
      test?: boolean;
      values: Record<string, unknown>;
    },
  ) =>
    apiClient.request<SetupConfigSaveResult>(API_ENDPOINTS.setup.config(stepKey), {
      auth: false,
      body,
      method: "PATCH",
    }),
  skipStep: (runId: string, stepKey: string, body: { reason?: string; setupToken?: string }) =>
    apiClient.request<InitializationRunResult>(API_ENDPOINTS.setup.runStepSkip(runId, stepKey), {
      auth: false,
      body,
      method: "POST",
    }),
  testConfig: (stepKey: string, body: { setupToken?: string; values: Record<string, unknown> }) =>
    apiClient.request<SetupTestResult>(API_ENDPOINTS.setup.configTest(stepKey), {
      auth: false,
      body,
      method: "POST",
    }),
};
