import { Fragment, useEffect, useId, useMemo, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";
import { useTranslation } from "react-i18next";

import { Badge } from "~/components/aoi/primitives/Badge";
import { Button } from "~/components/aoi/primitives/Button";
import { FormField } from "~/components/aoi/patterns/FormField";
import { FormSkeleton } from "~/components/aoi/patterns/LoadingSkeletons";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import {
  StepWizard,
  type StepWizardItem,
  type StepWizardStatus,
} from "~/components/aoi/patterns/StepWizard";
import {
  useCompleteSetupMutation,
  useCreateSetupRunMutation,
  useRetrySetupRunMutation,
  useSaveSetupConfigMutation,
  useSetupRunLogsQuery,
  useSetupSchemaQuery,
  useSetupStatusQuery,
  useSkipSetupStepMutation,
  useTestSetupConfigMutation,
} from "~/features/setup/queries";
import {
  buildRunRequest,
  hydrateValues,
  inputTypeForField,
  isVisible,
  languageStepKey,
  normalizeStep,
  routeSlugForStep,
  scalarString,
  syncDefaultLocaleValue,
  validateStepFields,
  visibleConfigValues,
  type PasswordPolicyFailure,
  type StepValidationResult,
  type StepValues,
  type WizardStep,
} from "~/features/setup/wizard-helpers";
import { confirmSetupLanguage } from "~/features/setup/setup-progress";
import { supportedLocales } from "~/i18n/locales";
import type { AppLocale } from "~/i18n/resources";
import type {
  InitializationReport,
  InitializationStep,
  SetupConfigSaveResult,
  SetupFieldSchema,
  SetupStepSchema,
  SetupTestResult,
} from "~/lib/api/setup";
import { useAuthStore } from "~/stores/auth-store";

export default function SetupWizardRoute() {
  const { i18n, t } = useTranslation();
  const navigate = useNavigate();
  const params = useParams();
  const [searchParams] = useSearchParams();
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, Record<string, string>>>({});
  const [notice, setNotice] = useState("");
  const [activeRunId, setActiveRunId] = useState<string | null>(null);
  const [runConfirmation, setRunConfirmation] = useState({ confirmed: false, stepKey: "" });
  const [values, setValues] = useState<StepValues>({});
  const [testResults, setTestResults] = useState<Record<string, SetupTestResult>>({});
  const [testedPayloads, setTestedPayloads] = useState<Record<string, string>>({});
  const previousLocaleRef = useRef(i18n.language as AppLocale);
  const setSession = useAuthStore((state) => state.setSession);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const setupToken = searchParams.get("setupToken") ?? undefined;
  const statusQuery = useSetupStatusQuery();
  const schemaQuery = useSetupSchemaQuery(setupToken);
  const status = statusQuery.data ?? null;
  const schema = schemaQuery.data ?? null;
  const runId = activeRunId ?? status?.lastRun?.id ?? null;
  const runLogsQuery = useSetupRunLogsQuery(runId);
  const saveConfigMutation = useSaveSetupConfigMutation();
  const testConfigMutation = useTestSetupConfigMutation();
  const createRunMutation = useCreateSetupRunMutation();
  const retryRunMutation = useRetrySetupRunMutation();
  const skipStepMutation = useSkipSetupStepMutation();
  const completeSetupMutation = useCompleteSetupMutation();

  const schemaSteps = useMemo(() => (schema?.steps ?? []).map(normalizeStep), [schema]);
  const steps = useMemo<WizardStep[]>(
    () => [createLanguageStep(t), ...schemaSteps],
    [schemaSteps, t],
  );
  const statusStepsByKey = useMemo(
    () => new Map((status?.steps ?? []).map((step) => [step.key, step])),
    [status?.steps],
  );
  const activeSlug = params.step ?? "language";
  const activeStep = steps.find((step) => routeSlugForStep(step) === activeSlug) ?? steps[0];
  const activeStepKey = activeStep.key;
  const activeIndex = Math.max(
    0,
    steps.findIndex((step) => step.key === activeStepKey),
  );
  const activeReport = activeStep.local ? undefined : statusStepsByKey.get(activeStepKey);
  const activeTest = activeStep.local
    ? undefined
    : testResults[activeStepKey] || testResultFromReport(activeStep, activeReport);
  const activeTestPayload = activeStep.local
    ? ""
    : payloadSnapshotForStep(activeStep, values[activeStepKey] ?? {});
  const activeTestCurrent =
    !activeStep.local && testedPayloads[activeStepKey]
      ? testedPayloads[activeStepKey] === activeTestPayload
      : testFingerprintIsCurrent(activeStep, activeReport);
  const progress = Math.round(((activeIndex + 1) / Math.max(1, steps.length)) * 100);
  const hasBackendSteps = schemaSteps.length > 0;
  const isFinalStep = activeStep && activeIndex === steps.length - 1 && hasBackendSteps;
  const runConfirmed = Boolean(
    isFinalStep && runConfirmation.stepKey === activeStepKey && runConfirmation.confirmed,
  );
  const canComplete = Boolean(runId && status?.lastRun?.status === "succeeded");
  const canRetry = Boolean(runId && status?.lastRun?.status === "failed");
  const canSkip = Boolean(runId && activeStep && !activeStep.local && activeStep.skippable);
  const loading = statusQuery.isLoading || schemaQuery.isLoading;
  const queryError = statusQuery.error ?? schemaQuery.error;
  const displayError =
    error || (queryError ? messageFromError(queryError, t("setup.status.loadFailed")) : "");
  const displayErrorTitle = error ? t("setup.status.actionFailed") : t("setup.status.loadFailed");
  const wizardItems = useMemo(
    () =>
      steps.map<StepWizardItem>((step, index) => {
        const itemStatus = stepStatusForWizard(
          step,
          index,
          activeIndex,
          activeStepKey,
          statusStepsByKey,
        );
        return {
          description: step.description,
          disabled: itemStatus === "blocked",
          key: step.key,
          status: itemStatus,
          statusLabel: setupStatusLabel(t, itemStatus),
          title: step.title,
        };
      }),
    [activeIndex, activeStepKey, statusStepsByKey, steps, t],
  );

  useEffect(() => {
    if (!schema) {
      return;
    }
    const currentLocale = i18n.language as AppLocale;
    setValues((current) =>
      hydrateValues(current, schema, currentLocale, previousLocaleRef.current),
    );
    previousLocaleRef.current = currentLocale;
  }, [i18n.language, schema]);

  useEffect(() => {
    if (status && !status.required) {
      void navigate(isAuthenticated ? "/admin" : "/login", { replace: true });
    }
  }, [isAuthenticated, navigate, status]);

  async function refreshSetup() {
    setError("");
    setNotice("");
    await Promise.all([
      statusQuery.refetch(),
      schemaQuery.refetch(),
      runId ? runLogsQuery.refetch() : Promise.resolve(),
    ]);
  }

  function updateValue(stepKey: string, fieldKey: string, value: unknown) {
    if (isFinalStep) {
      setRunConfirmation((current) =>
        current.stepKey === activeStepKey ? { ...current, confirmed: false } : current,
      );
    }
    setFieldErrors((current) => {
      const next = clearFieldError(current, stepKey, fieldKey);
      return fieldKey === "password" ? clearFieldError(next, stepKey, "passwordConfirm") : next;
    });
    setValues((current) => ({
      ...current,
      [stepKey]: {
        ...(current[stepKey] ?? {}),
        [fieldKey]: value,
      },
    }));
  }

  async function goToStep(stepKey: string) {
    const target = steps.find((step) => step.key === stepKey);
    if (target) {
      if (target.key !== activeStepKey && stepIsBlockedByDependencies(target, statusStepsByKey)) {
        setError(t("setup.errors.stepBlocked"));
        return;
      }
      setRunConfirmation({ confirmed: false, stepKey: "" });
      await navigate(`/setup/${routeSlugForStep(target)}`);
    }
  }

  async function goNext() {
    const next = steps[activeIndex + 1];
    if (next) {
      await goToStep(next.key);
    }
  }

  async function goBack() {
    const previous = steps[activeIndex - 1];
    if (previous) {
      await goToStep(previous.key);
    }
  }

  function changeSetupLanguage(locale: AppLocale) {
    setValues((current) => syncDefaultLocaleValue(current, schema, locale));
    previousLocaleRef.current = locale;
    void i18n.changeLanguage(locale);
  }

  async function saveStep() {
    if (!activeStep || activeStep.local) {
      if (activeStep?.key === languageStepKey) {
        confirmSetupLanguage();
      }
      await goNext();
      return;
    }
    if (!validateActiveStep(activeStep)) {
      return;
    }
    const configValues = visibleConfigValues(activeStep, values[activeStepKey] ?? {});
    if (!Object.keys(configValues).length) {
      await goNext();
      return;
    }
    setBusy("save");
    setError("");
    setNotice("");
    try {
      const result = await saveConfigMutation.mutateAsync({
        persist: true,
        setupToken,
        stepKey: activeStepKey,
        test: activeStep.testable,
        values: configValues,
      });
      if (result.test) {
        setTestResults((current) => ({ ...current, [activeStepKey]: result.test! }));
        setTestedPayloads((current) => ({
          ...current,
          [activeStepKey]: activeTestPayload,
        }));
      }
      setNotice(setupSaveNotice(result, t));
      if (!result.restartRequired) {
        await goNext();
      }
    } catch (err) {
      setError(messageFromError(err, t("errors.api.requestFailed")));
    } finally {
      setBusy("");
    }
  }

  async function testStep() {
    if (!activeStep || activeStep.local) {
      return;
    }
    if (!validateActiveStep(activeStep)) {
      return;
    }
    setBusy("test");
    setError("");
    setNotice("");
    try {
      const configValues = visibleConfigValues(activeStep, values[activeStepKey] ?? {});
      const payloadSnapshot = payloadSnapshotForStep(activeStep, values[activeStepKey] ?? {});
      const result = await testConfigMutation.mutateAsync({
        setupToken,
        stepKey: activeStepKey,
        values: configValues,
      });
      setTestResults((current) => ({ ...current, [activeStepKey]: result }));
      setTestedPayloads((current) => ({ ...current, [activeStepKey]: payloadSnapshot }));
      if (result.status === "failed") {
        setError(result.error || t("setup.errors.testFailed"));
      } else {
        setNotice(result.summary || t("setup.status.testPassed"));
      }
    } catch (err) {
      setError(messageFromError(err, t("errors.api.requestFailed")));
    } finally {
      setBusy("");
    }
  }

  async function runSetup() {
    if (!runConfirmed) {
      setError(t("setup.errors.confirmRequired"));
      return;
    }
    const payload = buildRunRequest(values, schemaSteps, setupToken);
    if (!payload) {
      setError(t("setup.errors.ownerRequired"));
      return;
    }
    setBusy("run");
    setError("");
    setNotice(t("setup.status.running"));
    try {
      const result = await createRunMutation.mutateAsync(payload);
      setActiveRunId(result.run.id);
      setRunConfirmation({ confirmed: false, stepKey: activeStepKey });
      if (result.loginTokensIssued && result.loginTokens) {
        setSession(result.loginTokens);
        setNotice(t("setup.status.completed"));
        await navigate("/admin");
        return;
      }
      setNotice(result.report?.summary || result.run.status);
      await refreshSetup();
    } catch (err) {
      setError(messageFromError(err, t("errors.api.requestFailed")));
      await refreshSetup();
    } finally {
      setBusy("");
    }
  }

  async function retryRun() {
    if (!runId) {
      return;
    }
    const payload = buildRunRequest(values, schemaSteps, setupToken);
    if (!payload) {
      setError(t("setup.errors.ownerRequired"));
      return;
    }
    setBusy("retry");
    setError("");
    setNotice(t("setup.status.running"));
    try {
      const result = await retryRunMutation.mutateAsync({ body: payload, runId, setupToken });
      setActiveRunId(result.run.id);
      setNotice(result.report?.summary || t("setup.status.retryExecuted"));
      await refreshSetup();
    } catch (err) {
      setError(messageFromError(err, t("errors.api.requestFailed")));
      await refreshSetup();
    } finally {
      setBusy("");
    }
  }

  async function skipStep() {
    if (!runId || !activeStep || activeStep.local) {
      setError(t("setup.errors.skipNeedsRun"));
      return;
    }
    setBusy("skip");
    setError("");
    setNotice("");
    try {
      const result = await skipStepMutation.mutateAsync({
        reason: t("setup.skip.reason"),
        runId,
        setupToken,
        stepKey: activeStepKey,
      });
      setActiveRunId(result.run.id);
      setNotice(t("setup.status.skipped"));
      await refreshSetup();
      await goNext();
    } catch (err) {
      setError(messageFromError(err, t("errors.api.requestFailed")));
    } finally {
      setBusy("");
    }
  }

  async function completeSetup() {
    setBusy("complete");
    setError("");
    setNotice("");
    try {
      const result = await completeSetupMutation.mutateAsync(setupToken);
      setNotice(result.report.summary || t("setup.status.completed"));
      if (result.completed) {
        await navigate(isAuthenticated ? "/admin" : "/login");
        return;
      }
      await refreshSetup();
    } catch (err) {
      setError(messageFromError(err, t("errors.api.requestFailed")));
    } finally {
      setBusy("");
    }
  }

  async function loadRunLogs() {
    if (!runId) {
      return;
    }
    setBusy("logs");
    setError("");
    try {
      await runLogsQuery.refetch();
    } catch (err) {
      setError(messageFromError(err, t("errors.api.requestFailed")));
    } finally {
      setBusy("");
    }
  }

  function validateActiveStep(step: SetupStepSchema) {
    const validation = validateStepFields(step, values[step.key] ?? {}, status?.passwordPolicy);
    if (validation) {
      const message = validationMessage(validation, t);
      setError(message);
      setFieldErrors((current) => ({
        ...current,
        [step.key]: {
          ...(current[step.key] ?? {}),
          [validation.fieldKey]: message,
        },
      }));
      return false;
    }
    setFieldErrors((current) => clearStepErrors(current, step.key));
    return true;
  }

  return (
    <main className="aoi-setup-layout">
      <aside className="aoi-setup-steps">
        <StepWizard
          ariaLabel={t("setup.progress.label")}
          currentKey={activeStepKey}
          items={wizardItems}
          progressLabel={t("setup.progress.text", {
            current: activeIndex + 1,
            total: steps.length,
          })}
          progressValue={progress}
          onSelect={(stepKey) => void goToStep(stepKey)}
        />
      </aside>

      <section className="aoi-setup-panel" aria-labelledby="setup-step-title">
        <header className="aoi-setup-panel__header">
          <Badge>
            {activeReport
              ? setupStatusLabel(t, activeReport.status)
              : setupStatusLabel(t, activeStep.local ? "current" : "pending")}
          </Badge>
          <h1 id="setup-step-title">{activeStep.title}</h1>
          <p>{activeStep.description}</p>
        </header>

        {loading ? <FormSkeleton fields={4} /> : null}
        {displayError ? (
          <StateBlock intent="danger" title={displayErrorTitle} description={displayError} />
        ) : null}
        {notice ? <StateBlock title={t("common.labels.status")} description={notice} /> : null}
        {!loading && !hasBackendSteps ? (
          <StateBlock title={t("setup.empty.title")} description={t("setup.empty.description")} />
        ) : null}

        <StateBlock
          title={t("setup.security.title")}
          description={t("setup.security.description")}
        />

        {activeReport ? <SetupStepReport step={activeReport} /> : null}
        {runId && status?.report ? <SetupReportSummary report={status.report} /> : null}

        {activeStep.local ? (
          <SetupLanguageStep value={i18n.language as AppLocale} onChange={changeSetupLanguage} />
        ) : (
          <SetupStepForm
            fieldErrors={fieldErrors[activeStepKey] ?? {}}
            step={activeStep}
            values={values[activeStepKey] ?? {}}
            onChange={updateValue}
          />
        )}

        {activeTest ? <SetupTestFeedback current={activeTestCurrent} result={activeTest} /> : null}

        {isFinalStep ? (
          <SetupRunConfirmation
            checked={runConfirmed}
            onChange={(confirmed) => setRunConfirmation({ confirmed, stepKey: activeStepKey })}
          />
        ) : null}

        <div className="aoi-setup-actions">
          <Button
            appearance="secondary"
            disabled={activeIndex === 0 || Boolean(busy)}
            onClick={() => void goBack()}
          >
            {t("setup.actions.back")}
          </Button>
          {activeStep.testable ? (
            <Button
              appearance="secondary"
              loading={busy === "test"}
              onClick={() => void testStep()}
            >
              {t("setup.actions.test")}
            </Button>
          ) : null}
          {canSkip ? (
            <Button appearance="ghost" loading={busy === "skip"} onClick={() => void skipStep()}>
              {t("setup.actions.skip")}
            </Button>
          ) : null}
          {isFinalStep ? (
            canComplete ? (
              <Button loading={busy === "complete"} onClick={() => void completeSetup()}>
                {t("setup.actions.complete")}
              </Button>
            ) : (
              <Button
                disabled={!runConfirmed || Boolean(busy)}
                loading={busy === "run"}
                onClick={() => void runSetup()}
              >
                {t("setup.actions.run")}
              </Button>
            )
          ) : (
            <Button loading={busy === "save"} onClick={() => void saveStep()}>
              {activeStep.local ? t("setup.actions.continue") : t("setup.actions.save")}
            </Button>
          )}
          {canRetry ? (
            <Button
              appearance="secondary"
              loading={busy === "retry"}
              onClick={() => void retryRun()}
            >
              {t("setup.actions.retry")}
            </Button>
          ) : null}
          {runId ? (
            <Button appearance="ghost" loading={busy === "logs"} onClick={() => void loadRunLogs()}>
              {t("setup.actions.logs")}
            </Button>
          ) : null}
          <Button
            appearance="ghost"
            loading={statusQuery.isFetching || schemaQuery.isFetching}
            onClick={() => void refreshSetup()}
          >
            {t("setup.actions.refresh")}
          </Button>
        </div>

        {status?.restartRequired ? (
          <StateBlock
            intent="danger"
            title={t("setup.status.restartRequired")}
            description={status.restartReason || t("setup.status.loadFailed")}
          />
        ) : null}
        {runLogsQuery.data?.length ? <SetupRunLogs steps={runLogsQuery.data} /> : null}
      </section>
    </main>
  );
}

function SetupRunLogs({ steps }: { steps: InitializationStep[] }) {
  const { t } = useTranslation();

  return (
    <section className="aoi-setup-log-panel" aria-labelledby="setup-logs-title">
      <h2 id="setup-logs-title">{t("setup.logs.title")}</h2>
      <ol className="aoi-setup-log-list">
        {steps.map((step) => (
          <li key={step.key} data-status={step.status}>
            <span>{setupStatusLabel(t, step.status)}</span>
            <strong>{step.title}</strong>
            {step.outputSummary ? <p>{step.outputSummary}</p> : null}
            {step.errorMessage ? <p>{step.errorMessage}</p> : null}
            {step.repairHint || step.recovery ? <p>{step.repairHint || step.recovery}</p> : null}
          </li>
        ))}
      </ol>
    </section>
  );
}

function SetupReportSummary({ report }: { report: InitializationReport }) {
  const { t } = useTranslation();

  return (
    <section className="aoi-setup-report" aria-labelledby="setup-report-title">
      <h2 id="setup-report-title">{t("setup.report.title")}</h2>
      <p>{report.summary}</p>
      <dl className="aoi-setup-report__stats">
        <div>
          <dt>{t("setup.report.successful")}</dt>
          <dd>{report.successful}</dd>
        </div>
        <div>
          <dt>{t("setup.report.failed")}</dt>
          <dd>{report.failed}</dd>
        </div>
        <div>
          <dt>{t("setup.report.skipped")}</dt>
          <dd>{report.skipped}</dd>
        </div>
        <div>
          <dt>{t("setup.report.risk")}</dt>
          <dd>{report.risk}</dd>
        </div>
      </dl>
    </section>
  );
}

function SetupStepReport({ step }: { step: InitializationStep }) {
  const { t } = useTranslation();
  const rows = [
    { label: t("setup.report.goal"), value: step.goal },
    { label: t("setup.report.outputSummary"), value: step.outputSummary },
    { label: t("setup.report.testSummary"), value: step.testSummary },
    { label: t("setup.report.completionMark"), value: step.completionMark },
    { label: t("setup.report.recovery"), value: step.repairHint || step.recovery },
    {
      label: t("setup.report.automaticActions"),
      value: step.automaticActions?.length ? step.automaticActions.join(", ") : "",
    },
  ].filter((row) => row.value);

  if (!rows.length && !step.errorMessage && !step.restartRequired) {
    return null;
  }

  return (
    <section className="aoi-setup-report" aria-labelledby="setup-step-report-title">
      <h2 id="setup-step-report-title">{t("setup.report.stepTitle")}</h2>
      <dl>
        <div>
          <dt>{t("common.labels.status")}</dt>
          <dd>{setupStatusLabel(t, step.status)}</dd>
        </div>
        {rows.map((row) => (
          <div key={row.label}>
            <dt>{row.label}</dt>
            <dd>{row.value}</dd>
          </div>
        ))}
        {step.errorMessage ? (
          <div>
            <dt>{t("setup.report.error")}</dt>
            <dd>{step.errorMessage}</dd>
          </div>
        ) : null}
        {step.restartRequired ? (
          <div>
            <dt>{t("setup.status.restartRequired")}</dt>
            <dd>{t("setup.report.restartRequired")}</dd>
          </div>
        ) : null}
      </dl>
    </section>
  );
}

function SetupTestFeedback({ current, result }: { current?: boolean; result: SetupTestResult }) {
  const { t } = useTranslation();
  const failed = result.status === "failed";

  return (
    <section
      className="aoi-setup-test-feedback"
      data-status={result.status}
      aria-labelledby="setup-test-feedback-title"
    >
      <div>
        <h2 id="setup-test-feedback-title">
          {failed ? t("setup.test.failedTitle") : t("setup.test.passedTitle")}
        </h2>
        <p>
          {t("common.labels.status")}: {setupStatusLabel(t, result.status)}
        </p>
      </div>
      <p>{result.error || result.summary}</p>
      {result.repairHint ? (
        <p className="aoi-setup-test-feedback__hint">
          {t("setup.test.repairHint")}: {result.repairHint}
        </p>
      ) : null}
      {current === false ? (
        <p className="aoi-setup-test-feedback__hint">{t("setup.test.stale")}</p>
      ) : null}
      {result.restartRequired ? (
        <p className="aoi-setup-test-feedback__hint">{t("setup.test.restartRequired")}</p>
      ) : null}
    </section>
  );
}

function createLanguageStep(t: ReturnType<typeof useTranslation>["t"]): WizardStep {
  return {
    description: t("setup.language.description"),
    fields: [],
    key: languageStepKey,
    local: true,
    order: 0,
    phase: "language",
    required: true,
    routeSlug: "language",
    skippable: false,
    testable: false,
    title: t("setup.language.title"),
  };
}

function SetupLanguageStep({
  onChange,
  value,
}: {
  onChange: (locale: AppLocale) => void;
  value: AppLocale;
}) {
  const { t } = useTranslation();

  return (
    <div className="aoi-form-field">
      <label htmlFor="setup-language">{t("setup.language.field")}</label>
      <select
        id="setup-language"
        className="aoi-language-select"
        value={value}
        onChange={(event) => onChange(event.target.value as AppLocale)}
      >
        {supportedLocales.map((locale) => (
          <option key={locale} value={locale}>
            {locale}
          </option>
        ))}
      </select>
      <span className="aoi-form-field__help">{t("setup.language.description")}</span>
    </div>
  );
}

function SetupStepForm({
  fieldErrors,
  onChange,
  step,
  values,
}: {
  fieldErrors: Record<string, string>;
  onChange: (stepKey: string, fieldKey: string, value: unknown) => void;
  step?: SetupStepSchema;
  values: Record<string, unknown>;
}) {
  const { t } = useTranslation();

  if (!step) {
    return null;
  }

  const groups = step.groups?.length
    ? step.groups
    : [{ fields: step.fields, key: step.key, title: step.title }];

  return (
    <div className="aoi-setup-form">
      {groups.map((group) =>
        isVisible(group.visibleWhen, values) ? (
          <fieldset className="aoi-setup-fieldset" key={group.key}>
            <legend>{group.title}</legend>
            {group.description ? <p>{group.description}</p> : null}
            {group.fields
              .filter((field) => isVisible(field.visibleWhen, values))
              .map((field) => (
                <Fragment key={field.key}>
                  <SetupField
                    error={fieldErrors[field.key]}
                    field={field}
                    stepKey={step.key}
                    value={values[field.key]}
                    onChange={onChange}
                  />
                  {isOwnerPasswordField(step, field) ? (
                    <FormField
                      autoComplete="new-password"
                      error={fieldErrors.passwordConfirm}
                      help={t("setup.owner.passwordConfirm.help")}
                      id={`${step.key}-passwordConfirm`}
                      label={t("setup.owner.passwordConfirm.label")}
                      type="password"
                      value={scalarString(values.passwordConfirm)}
                      onChange={(event) =>
                        onChange(step.key, "passwordConfirm", event.target.value)
                      }
                    />
                  ) : null}
                </Fragment>
              ))}
          </fieldset>
        ) : null,
      )}
    </div>
  );
}

function SetupField({
  error,
  field,
  onChange,
  stepKey,
  value,
}: {
  error?: string;
  field: SetupFieldSchema;
  onChange: (stepKey: string, fieldKey: string, value: unknown) => void;
  stepKey: string;
  value: unknown;
}) {
  const currentValue = value ?? field.default ?? "";
  const fieldId = `${stepKey}-${field.key}`;
  const helpId = field.help ? `${fieldId}-help` : undefined;
  const errorId = error ? `${fieldId}-error` : undefined;
  const describedBy = [helpId, errorId].filter(Boolean).join(" ") || undefined;

  if (field.type === "boolean") {
    return (
      <label className="aoi-setup-check">
        <input
          id={fieldId}
          aria-describedby={describedBy}
          aria-invalid={Boolean(error)}
          checked={Boolean(currentValue)}
          type="checkbox"
          onChange={(event) => onChange(stepKey, field.key, event.target.checked)}
        />
        <span>
          <strong>{field.label}</strong>
          {field.help ? (
            <span id={helpId} className="aoi-form-field__help">
              {field.help}
            </span>
          ) : null}
          {error ? (
            <span id={errorId} className="aoi-form-field__error">
              {error}
            </span>
          ) : null}
        </span>
      </label>
    );
  }

  if (field.type === "select" && field.options?.length) {
    return (
      <div className="aoi-form-field">
        <label htmlFor={fieldId}>{field.label}</label>
        <select
          id={fieldId}
          aria-describedby={describedBy}
          aria-invalid={Boolean(error)}
          className="aoi-language-select"
          value={scalarString(currentValue)}
          onChange={(event) => onChange(stepKey, field.key, event.target.value)}
        >
          {field.options.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
        {field.help ? (
          <span id={helpId} className="aoi-form-field__help">
            {field.help}
          </span>
        ) : null}
        {error ? (
          <span id={errorId} className="aoi-form-field__error">
            {error}
          </span>
        ) : null}
      </div>
    );
  }

  return (
    <FormField
      id={fieldId}
      autoComplete={field.key === "password" ? "new-password" : undefined}
      error={error}
      help={field.help}
      label={field.label}
      placeholder={field.placeholder}
      type={inputTypeForField(field)}
      value={scalarString(currentValue)}
      onChange={(event) =>
        onChange(
          stepKey,
          field.key,
          field.type === "number"
            ? event.target.value === ""
              ? ""
              : Number(event.target.value)
            : event.target.value,
        )
      }
    />
  );
}

function SetupRunConfirmation({
  checked,
  onChange,
}: {
  checked: boolean;
  onChange: (checked: boolean) => void;
}) {
  const { t } = useTranslation();
  const inputId = useId();

  return (
    <div className="aoi-setup-check aoi-setup-confirm">
      <input
        id={inputId}
        checked={checked}
        type="checkbox"
        onChange={(event) => onChange(event.target.checked)}
      />
      <span>
        <label htmlFor={inputId}>
          <strong>{t("setup.confirm.title")}</strong>
        </label>
        <span className="aoi-form-field__help">{t("setup.confirm.description")}</span>
      </span>
    </div>
  );
}

function isOwnerPasswordField(step: SetupStepSchema, field: SetupFieldSchema) {
  return step.key === "iam.owner" && field.key === "password";
}

function clearFieldError(
  current: Record<string, Record<string, string>>,
  stepKey: string,
  fieldKey: string,
) {
  const currentStepErrors = current[stepKey];
  if (!currentStepErrors?.[fieldKey]) {
    return current;
  }
  const remainingStepErrors = { ...currentStepErrors };
  delete remainingStepErrors[fieldKey];
  if (!Object.keys(remainingStepErrors).length) {
    const remainingErrors = { ...current };
    delete remainingErrors[stepKey];
    return remainingErrors;
  }
  return {
    ...current,
    [stepKey]: remainingStepErrors,
  };
}

function clearStepErrors(current: Record<string, Record<string, string>>, stepKey: string) {
  if (!current[stepKey]) {
    return current;
  }
  const remainingErrors = { ...current };
  delete remainingErrors[stepKey];
  return remainingErrors;
}

function stepStatusForWizard(
  step: WizardStep,
  index: number,
  activeIndex: number,
  activeKey: string,
  statusStepsByKey: Map<string, InitializationStep>,
): StepWizardStatus {
  if (stepIsBlockedByDependencies(step, statusStepsByKey) && step.key !== activeKey) {
    return "blocked";
  }
  const status = statusStepsByKey.get(step.key)?.status;
  if (isKnownStepStatus(status)) {
    if (status === "pending" && step.key === activeKey) {
      return "current";
    }
    return status;
  }
  if (step.key === activeKey) {
    return "current";
  }
  if (step.local && index < activeIndex) {
    return "succeeded";
  }
  return "pending";
}

function isKnownStepStatus(status: string | undefined): status is StepWizardStatus {
  return Boolean(
    status &&
    ["blocked", "current", "failed", "pending", "running", "skipped", "succeeded"].includes(status),
  );
}

function stepIsBlockedByDependencies(
  step: WizardStep,
  statusStepsByKey: Map<string, InitializationStep>,
) {
  if (step.local || !step.dependencies?.length) {
    return false;
  }
  return step.dependencies.some((dependencyKey) => {
    const status = statusStepsByKey.get(dependencyKey)?.status;
    return status === "failed" || status === "running";
  });
}

function setupStatusLabel(t: ReturnType<typeof useTranslation>["t"], status: string) {
  const labels: Record<string, string> = {
    blocked: t("setup.stepStatus.blocked"),
    current: t("setup.stepStatus.current"),
    failed: t("setup.stepStatus.failed"),
    pending: t("setup.stepStatus.pending"),
    running: t("setup.stepStatus.running"),
    skipped: t("setup.stepStatus.skipped"),
    succeeded: t("setup.stepStatus.succeeded"),
  };
  return labels[status] ?? status;
}

function setupSaveNotice(result: SetupConfigSaveResult, t: ReturnType<typeof useTranslation>["t"]) {
  if (result.restartRequired || result.nextAction === "restart") {
    return result.restartReason || t("setup.status.restartRequired");
  }
  if (result.envManagedPathsOverwritten?.length) {
    return t("setup.messages.savedEnvManagedOverwritten", {
      paths: result.envManagedPathsOverwritten.join(t("setup.messages.pathSeparator")),
    });
  }
  return result.inputSummary || t("setup.status.saved");
}

function testResultFromReport(
  step: WizardStep,
  report: InitializationStep | undefined,
): SetupTestResult | undefined {
  if (!report?.testStatus || step.local) {
    return undefined;
  }
  return {
    error: report.testError,
    inputFingerprint: report.testInputFingerprint,
    repairHint: report.repairHint,
    restartRequired: Boolean(report.restartRequired),
    status: report.testStatus as SetupTestResult["status"],
    stepKey: step.key,
    summary: report.testSummary || "",
    testedAt: "",
  };
}

function testFingerprintIsCurrent(step: WizardStep, report: InitializationStep | undefined) {
  if (step.local) {
    return undefined;
  }
  if (!report?.testInputFingerprint || !step.inputFingerprint) {
    return undefined;
  }
  return report.testInputFingerprint === step.inputFingerprint;
}

function payloadSnapshotForStep(step: SetupStepSchema, values: Record<string, unknown>) {
  return JSON.stringify(
    Object.entries(visibleConfigValues(step, values)).sort(([left], [right]) =>
      left.localeCompare(right),
    ),
  );
}

function validationMessage(
  validation: StepValidationResult,
  t: ReturnType<typeof useTranslation>["t"],
) {
  if (validation.type === "required") {
    return t("setup.errors.fieldRequired", { field: validation.fieldLabel });
  }
  if (validation.type === "passwordConfirmRequired") {
    return t("setup.errors.passwordConfirmRequired");
  }
  if (validation.type === "passwordMismatch") {
    return t("setup.errors.passwordMismatch");
  }
  const rules = validation.failures
    .map((failure) => passwordPolicyFailureLabel(failure, validation.minLength, t))
    .join(t("setup.passwordPolicy.separator"));
  return t("setup.errors.passwordWeakWithRules", { rules });
}

function passwordPolicyFailureLabel(
  failure: PasswordPolicyFailure,
  minLength: number,
  t: ReturnType<typeof useTranslation>["t"],
) {
  if (failure === "minLength") {
    return t("setup.passwordPolicy.minLength", { count: minLength });
  }
  return t(`setup.passwordPolicy.${failure}`);
}

function messageFromError(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback;
}
