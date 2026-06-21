import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";

import { queryKeys } from "~/lib/api/query-keys";
import { setupApi } from "~/lib/api/setup";
import type { InitializationRunRequest } from "~/lib/api/setup";

type SetupTokenInput = {
  setupToken?: string;
};

type SaveSetupConfigInput = SetupTokenInput & {
  persist?: boolean;
  stepKey: string;
  test?: boolean;
  values: Record<string, unknown>;
};

type TestSetupConfigInput = SetupTokenInput & {
  stepKey: string;
  values: Record<string, unknown>;
};

type RetrySetupRunInput = SetupTokenInput & {
  body: InitializationRunRequest;
  runId: string;
};

type SkipSetupStepInput = SetupTokenInput & {
  reason?: string;
  runId: string;
  stepKey: string;
};

export function useSetupStatusQuery() {
  const { i18n } = useTranslation();

  return useQuery({
    queryKey: queryKeys.setup.status(i18n.language),
    queryFn: ({ signal }) => setupApi.getStatus({ signal }),
  });
}

export function useSetupSchemaQuery(setupToken?: string) {
  const { i18n } = useTranslation();

  return useQuery({
    queryKey: queryKeys.setup.schema(setupToken, i18n.language),
    queryFn: ({ signal }) => setupApi.getSchema(setupToken, { signal }),
  });
}

export function useSetupRunLogsQuery(runId: string | null | undefined) {
  const { i18n } = useTranslation();

  return useQuery({
    enabled: Boolean(runId),
    queryKey: queryKeys.setup.logs(runId ?? "", i18n.language),
    queryFn: ({ signal }) => setupApi.getRunLogs(runId ?? "", { signal }),
  });
}

export function useSaveSetupConfigMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ persist, setupToken, stepKey, test, values }: SaveSetupConfigInput) =>
      setupApi.saveConfig(stepKey, { persist, setupToken, test, values }),
    onSuccess: () => invalidateSetupQueries(queryClient),
  });
}

export function useTestSetupConfigMutation() {
  return useMutation({
    mutationFn: ({ setupToken, stepKey, values }: TestSetupConfigInput) =>
      setupApi.testConfig(stepKey, { setupToken, values }),
  });
}

export function useCreateSetupRunMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (body: InitializationRunRequest) => setupApi.createRun(body),
    onSettled: () => invalidateSetupQueries(queryClient),
  });
}

export function useRetrySetupRunMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ body, runId, setupToken }: RetrySetupRunInput) =>
      setupApi.retryRun(runId, { ...body, setupToken }),
    onSettled: () => invalidateSetupQueries(queryClient),
  });
}

export function useSkipSetupStepMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ reason, runId, setupToken, stepKey }: SkipSetupStepInput) =>
      setupApi.skipStep(runId, stepKey, { reason, setupToken }),
    onSettled: () => invalidateSetupQueries(queryClient),
  });
}

export function useCompleteSetupMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (setupToken?: string) => setupApi.complete(setupToken),
    onSettled: () => invalidateSetupQueries(queryClient),
  });
}

export function invalidateSetupQueries(queryClient: QueryClient) {
  return queryClient.invalidateQueries({ queryKey: queryKeys.setup.root });
}
