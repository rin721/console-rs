import {
  Activity,
  AlertTriangle,
  Boxes,
  CheckCircle2,
  Database,
  Home,
  Image,
  KeyRound,
  Languages,
  LogIn,
  LogOut,
  Menu,
  Package,
  Pencil,
  Play,
  Plus,
  Radar,
  RefreshCw,
  Server,
  Settings,
  ShieldCheck,
  Trash2,
  UserPlus
} from "lucide-react";
import { FormEvent, ReactNode, useCallback, useEffect, useMemo, useState } from "react";
import { createRoot } from "react-dom/client";
import {
  ApiError,
  apiDelete,
  apiGet,
  apiPost,
  apiPostForm,
  apiPut,
  configureApi,
  configureApiFromPublicSettings,
  configureAuthCallbacks
} from "./lib/api/client";
import { endpoints } from "./lib/api/endpoints";
import type {
  AcceptInvitationRequest,
  APITokenSummary,
  ApiCatalogGroup,
  BooleanResult,
  CompleteSetupRequest,
  ConfirmEmailVerificationRequest,
  CreateAPITokenRequest,
  CreateAPITokenResult,
  CreateRoleRequest,
  ForgotPasswordRequest,
  IamSetupStatus,
  InvitationSummary,
  InviteUserRequest,
  Locale,
  MediaAssetEntry,
  MfaFactorSummary,
  MfaRecoveryCodeSummary,
  MfaRecoveryCodesResult,
  MfaSetupResult,
  MfaVerifyResult,
  OperationRecord,
  OperationRecordSummary,
  OrganizationSummary,
  OrganizationUserSummary,
  PermissionSummary,
  PublicSettings,
  RegisterRequest,
  RequestEmailVerification,
  ResetPasswordRequest,
  RoleSummary,
  ServerStatus,
  SessionSnapshot,
  StorageObjectEntry,
  SetupConfigCheckSummary,
  SetupRun,
  SetupSchema,
  SetupStatus,
  SetupStepLog,
  SystemConfigEntry,
  SystemDictionaryEntry,
  SystemMenu,
  SystemParameterEntry,
  TrafficProbeAlert,
  TrafficProbeResult,
  TrafficProbeTarget,
  UpdateOrgUserRequest,
  UpdateRoleRequest,
  VerifyMfaRequest,
  VersionPackageEntry,
  VersionReleaseEventEntry
} from "./lib/api/types";
import { defaultLocale, detectInitialLocale, supportedLocales, translate } from "./i18n";
import "./styles.css";

type View = "account" | "admin" | "login" | "public" | "setup";
type AdminTab = "assets" | "iam" | "overview" | "system";

type Bootstrap = {
  apiOnline: boolean;
  authSetup: IamSetupStatus | null;
  publicSettings: PublicSettings | null;
  ready: boolean;
  setupConfigChecks: SetupConfigCheckSummary | null;
  setupStatus: SetupStatus | null;
};

type AdminData = {
  apiGroups: ApiCatalogGroup[];
  apiTokens: APITokenSummary[];
  configs: SystemConfigEntry[];
  dictionaries: SystemDictionaryEntry[];
  invitations: InvitationSummary[];
  media: MediaAssetEntry[];
  storageObjects: StorageObjectEntry[];
  mfaFactors: MfaFactorSummary[];
  mfaRecoveryCodes: MfaRecoveryCodeSummary[];
  menus: SystemMenu[];
  operations: OperationRecord[];
  operationSummary: OperationRecordSummary | null;
  orgs: OrganizationSummary[];
  parameters: SystemParameterEntry[];
  permissions: PermissionSummary[];
  probeAlerts: TrafficProbeAlert[];
  probeResults: TrafficProbeResult[];
  probes: TrafficProbeTarget[];
  roles: RoleSummary[];
  selectedOrgId: number | null;
  server: ServerStatus | null;
  users: OrganizationUserSummary[];
  versionReleases: VersionReleaseEventEntry[];
  versions: VersionPackageEntry[];
};

const emptyAdminData: AdminData = {
  apiGroups: [],
  apiTokens: [],
  configs: [],
  dictionaries: [],
  invitations: [],
  media: [],
  storageObjects: [],
  mfaFactors: [],
  mfaRecoveryCodes: [],
  menus: [],
  operations: [],
  operationSummary: null,
  orgs: [],
  parameters: [],
  permissions: [],
  probeAlerts: [],
  probeResults: [],
  probes: [],
  roles: [],
  selectedOrgId: null,
  server: null,
  users: [],
  versionReleases: [],
  versions: []
};

function App() {
  const [locale, setLocale] = useState<Locale>(() => detectInitialLocale());
  const [view, setView] = useState<View>(() => viewFromPath(window.location.pathname));
  const [tab, setTab] = useState<AdminTab>("overview");
  const [bootstrap, setBootstrap] = useState<Bootstrap>({
    apiOnline: false,
    authSetup: null,
    publicSettings: null,
    ready: false,
    setupConfigChecks: null,
    setupStatus: null
  });
  const [session, setSession] = useState<SessionSnapshot | null>(null);
  const [adminData, setAdminData] = useState<AdminData>(emptyAdminData);
  const [setupSchema, setSetupSchema] = useState<SetupSchema | null>(null);
  const [setupRun, setSetupRun] = useState<SetupRun | null>(null);
  const [setupRuns, setSetupRuns] = useState<SetupRun[]>([]);
  const [setupLogs, setSetupLogs] = useState<SetupStepLog[]>([]);
  const [notice, setNotice] = useState<string>("");
  const [busy, setBusy] = useState(false);
  const t = useMemo(() => (key: string, params?: Record<string, string | number>) => translate(locale, key, params), [locale]);

  useEffect(() => {
    document.documentElement.lang = locale;
    localStorage.setItem("console-locale", locale);
    configureApi({ locale });
  }, [locale]);

  useEffect(() => {
    const onPop = () => setView(viewFromPath(window.location.pathname));
    window.addEventListener("popstate", onPop);
    return () => window.removeEventListener("popstate", onPop);
  }, []);

  const navigate = useCallback((next: View) => {
    window.history.pushState(null, "", pathForView(next));
    setView(next);
  }, []);

  const accountAuthenticated = useCallback((snapshot: SessionSnapshot) => {
    setSession(snapshot);
    navigate("admin");
  }, [navigate]);

  const loadBootstrap = useCallback(async () => {
    const next: Bootstrap = {
      apiOnline: false,
      authSetup: null,
      publicSettings: null,
      ready: false,
      setupConfigChecks: null,
      setupStatus: null
    };
    try {
      await apiGet(endpoints.health);
      next.apiOnline = true;
    } catch {
      next.apiOnline = false;
    }
    try {
      await apiGet(endpoints.ready);
      next.ready = true;
    } catch {
      next.ready = false;
    }
    try {
      const settings = await apiGet<PublicSettings>(endpoints.system.publicSettings);
      next.publicSettings = settings;
      configureApiFromPublicSettings(settings, locale);
    } catch {
      next.publicSettings = null;
    }
    try {
      next.authSetup = await apiGet<IamSetupStatus>(endpoints.auth.setupStatus);
      next.setupStatus = await apiGet<SetupStatus>(endpoints.setup.status);
      next.setupConfigChecks = await apiGet<SetupConfigCheckSummary>(endpoints.setup.configChecks);
      setSetupSchema(await apiGet<SetupSchema>(endpoints.setup.schema));
      setSetupRuns(await apiGet<SetupRun[]>(endpoints.setup.runs));
    } catch {
      next.authSetup = null;
    }
    try {
      const current = await apiGet<SessionSnapshot>(endpoints.auth.session);
      setSession(current);
    } catch {
      setSession(null);
    }
    setBootstrap(next);
    if (next.authSetup && !next.authSetup.initialized) {
      navigate("setup");
    }
  }, [locale, navigate]);

  useEffect(() => {
    configureAuthCallbacks({
      onRefresh: setSession,
      onUnauthorized: () => setSession(null)
    });
    void loadBootstrap();
  }, [loadBootstrap]);

  const loadAdmin = useCallback(
    async (orgId?: number | null) => {
      if (!session?.authenticated) return;
      const [
        server,
        menus,
        apiGroups,
        operations,
        operationSummary,
        orgs,
        permissions,
        configs,
        dictionaries,
        parameters,
        versions,
        versionReleases,
        media,
        storageObjects,
        probes,
        probeResults,
        probeAlerts
      ] = await Promise.all([
        optional(apiGet<ServerStatus>(endpoints.system.serverStatus)),
        optional(apiGet<SystemMenu[]>(endpoints.system.menus)),
        optional(apiGet<ApiCatalogGroup[]>(endpoints.system.apis)),
        optional(apiGet<OperationRecord[]>(endpoints.system.operationRecords)),
        optional(apiGet<OperationRecordSummary>(endpoints.system.operationRecordSummary, { top_limit: 5 })),
        optional(apiGet<OrganizationSummary[]>(endpoints.iam.orgs)),
        optional(apiGet<PermissionSummary[]>(endpoints.iam.permissions)),
        optional(apiGet<SystemConfigEntry[]>(endpoints.system.configs)),
        optional(apiGet<SystemDictionaryEntry[]>(endpoints.system.dictionaries)),
        optional(apiGet<SystemParameterEntry[]>(endpoints.system.parameters)),
        optional(apiGet<VersionPackageEntry[]>(endpoints.system.versionPackages)),
        optional(apiGet<VersionReleaseEventEntry[]>(endpoints.system.versionPackageReleases)),
        optional(apiGet<MediaAssetEntry[]>(endpoints.system.mediaAssets)),
        optional(apiGet<StorageObjectEntry[]>(endpoints.system.storageObjects, { limit: 50 })),
        optional(apiGet<TrafficProbeTarget[]>(endpoints.system.trafficProbeTargets)),
        optional(apiGet<TrafficProbeResult[]>(endpoints.system.trafficProbeResults, { limit: 20 })),
        optional(apiGet<TrafficProbeAlert[]>(endpoints.system.trafficProbeAlerts, { limit: 20 }))
      ]);
      const selectedOrgId = orgId ?? session.organization?.id ?? orgs?.[0]?.id ?? null;
      const users = selectedOrgId ? await optional(apiGet<OrganizationUserSummary[]>(endpoints.iam.orgUsers(selectedOrgId))) : [];
      const roles = selectedOrgId ? await optional(apiGet<RoleSummary[]>(endpoints.iam.orgRoles(selectedOrgId))) : [];
      const apiTokens = selectedOrgId ? await optional(apiGet<APITokenSummary[]>(endpoints.orgs.apiTokens(selectedOrgId))) : [];
      const invitations = selectedOrgId ? await optional(apiGet<InvitationSummary[]>(endpoints.orgs.invitations(selectedOrgId))) : [];
      const mfaFactors = await optional(apiGet<MfaFactorSummary[]>(endpoints.auth.mfaFactors));
      const mfaRecoveryCodes = await optional(apiGet<MfaRecoveryCodeSummary[]>(endpoints.auth.mfaRecoveryCodes));
      setAdminData({
        apiGroups: apiGroups ?? [],
        apiTokens: apiTokens ?? [],
        configs: configs ?? [],
        dictionaries: dictionaries ?? [],
        invitations: invitations ?? [],
        media: media ?? [],
        storageObjects: storageObjects ?? [],
        mfaFactors: mfaFactors ?? [],
        mfaRecoveryCodes: mfaRecoveryCodes ?? [],
        menus: menus ?? [],
        operations: operations ?? [],
        operationSummary: operationSummary ?? null,
        orgs: orgs ?? [],
        parameters: parameters ?? [],
        permissions: permissions ?? [],
        probeAlerts: probeAlerts ?? [],
        probeResults: probeResults ?? [],
        probes: probes ?? [],
        roles: roles ?? [],
        selectedOrgId,
        server: server ?? null,
        users: users ?? [],
        versionReleases: versionReleases ?? [],
        versions: versions ?? []
      });
    },
    [session],
  );

  useEffect(() => {
    if (view === "admin" && session?.authenticated) {
      void loadAdmin(adminData.selectedOrgId);
    }
  }, [view, session?.authenticated]);

  async function submitInitialAdmin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setBusy(true);
    setNotice("");
    try {
      const nextSession = await apiPost<SessionSnapshot>(endpoints.auth.initialAdmin, formPayload(form, [
        "email",
        "password",
        "display_name",
        "organization_code",
        "organization_name"
      ]));
      setSession(nextSession);
      setNotice(t("setup.created"));
      await loadBootstrap();
      navigate("admin");
    } catch (error) {
      setNotice(errorMessage(error, t));
    } finally {
      setBusy(false);
    }
  }

  async function login(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setBusy(true);
    setNotice("");
    try {
      const payload = formPayload(form, ["identifier", "password", "mfaCode"]);
      if (!payload.mfaCode) delete payload.mfaCode;
      const nextSession = await apiPost<SessionSnapshot>(endpoints.auth.login, payload);
      setSession(nextSession);
      navigate("admin");
    } catch (error) {
      setNotice(errorMessage(error, t));
    } finally {
      setBusy(false);
    }
  }

  async function logout() {
    await apiPost(endpoints.auth.logout);
    setSession(null);
    navigate("login");
  }

  async function createSetupRun() {
    const run = await apiPost<SetupRun>(endpoints.setup.runs, { reason: "webui" });
    setSetupRun(run);
    setSetupRuns((current) => [run, ...current.filter((item) => item.id !== run.id)]);
    setSetupLogs(await apiGet<SetupStepLog[]>(endpoints.setup.logs(run.id)));
  }

  async function selectSetupRun(run: SetupRun) {
    setSetupRun(run);
    setSetupLogs(await apiGet<SetupStepLog[]>(endpoints.setup.logs(run.id)));
  }

  async function completeSetup() {
    const payload: CompleteSetupRequest = setupRun?.id
      ? { confirm: true, run_id: setupRun.id }
      : { confirm: true };
    await apiPost(endpoints.setup.complete, payload);
    if (setupRun?.id) {
      setSetupLogs(await apiGet<SetupStepLog[]>(endpoints.setup.logs(setupRun.id)));
    }
    setSetupRuns(await apiGet<SetupRun[]>(endpoints.setup.runs));
    await loadBootstrap();
  }

  async function mutate(action: () => Promise<unknown>) {
    setBusy(true);
    setNotice("");
    try {
      await action();
      await loadAdmin(adminData.selectedOrgId);
    } catch (error) {
      setNotice(errorMessage(error, t));
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="app-shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">{t("app.subtitle")}</p>
          <h1>{t("app.name")}</h1>
        </div>
        <div className="topbar-actions">
          <StatusPill ok={bootstrap.apiOnline} text={bootstrap.apiOnline ? t("app.apiOnline") : t("app.apiOffline")} />
          <button className="icon-button" onClick={() => void loadBootstrap()} title={t("app.refresh")} type="button">
            <RefreshCw size={18} />
          </button>
          <label className="locale-switch">
            <Languages size={16} />
            <select aria-label={t("app.language")} onChange={(event) => setLocale(event.currentTarget.value as Locale)} value={locale}>
              {supportedLocales.map((item) => (
                <option key={item} value={item}>{item === "zh-CN" ? t("app.zhCN") : t("app.en")}</option>
              ))}
            </select>
          </label>
          {session?.authenticated ? (
            <button className="text-button" onClick={() => void logout()} type="button">
              <LogOut size={16} />
              {t("app.logout")}
            </button>
          ) : null}
        </div>
      </header>

      <nav className="view-tabs">
        <button className={view === "public" ? "active" : ""} onClick={() => navigate("public")} type="button">
          <Home size={16} />
          {t("nav.public")}
        </button>
        <button className={view === "setup" ? "active" : ""} onClick={() => navigate("setup")} type="button">
          <UserPlus size={16} />
          {t("nav.setup")}
        </button>
        <button className={view === "login" ? "active" : ""} onClick={() => navigate("login")} type="button">
          <LogIn size={16} />
          {t("nav.login")}
        </button>
        <button className={view === "account" ? "active" : ""} onClick={() => navigate("account")} type="button">
          <KeyRound size={16} />
          {t("nav.account")}
        </button>
        <button className={view === "admin" ? "active" : ""} onClick={() => navigate("admin")} type="button">
          <ShieldCheck size={16} />
          {t("nav.admin")}
        </button>
      </nav>

      {notice ? <div className="notice">{notice}</div> : null}

      {view === "public" ? <PublicView bootstrap={bootstrap} navigate={navigate} t={t} /> : null}
      {view === "setup" ? (
        <SetupView
          busy={busy}
          completeSetup={completeSetup}
          createSetupRun={createSetupRun}
          schema={setupSchema}
          setupLogs={setupLogs}
          setupRun={setupRun}
          setupRuns={setupRuns}
          setupConfigChecks={bootstrap.setupConfigChecks}
          setupStatus={bootstrap.setupStatus}
          selectSetupRun={selectSetupRun}
          submitInitialAdmin={submitInitialAdmin}
          t={t}
        />
      ) : null}
      {view === "login" ? <LoginView busy={busy} login={login} navigate={navigate} session={session} t={t} /> : null}
      {view === "account" ? <AccountFlowView onAuthenticated={accountAuthenticated} publicSettings={bootstrap.publicSettings} t={t} /> : null}
      {view === "admin" ? (
        <AdminView
          data={adminData}
          loadAdmin={loadAdmin}
          mutate={mutate}
          refreshBootstrap={loadBootstrap}
          selectedTab={tab}
          session={session}
          setSelectedOrg={(orgId) => void loadAdmin(orgId)}
          setTab={setTab}
          t={t}
        />
      ) : null}
    </main>
  );
}

function PublicView({ bootstrap, navigate, t }: { bootstrap: Bootstrap; navigate: (view: View) => void; t: TFn }) {
  const productName = bootstrap.publicSettings?.product_name ?? t("app.name");
  const productCode = bootstrap.publicSettings?.product_code ?? "console";
  return (
    <section className="content-grid">
      <div className="panel span-2">
        <div className="section-heading">
          <div>
            <p className="eyebrow">{productCode}</p>
            <h2>{t("public.title", { product: productName })}</h2>
          </div>
          <StatusPill ok={bootstrap.ready} text={bootstrap.ready ? t("status.ready") : t("app.notReady")} />
        </div>
        <p className="muted">{t("public.description")}</p>
        <div className="public-actions">
          <button className="primary-button" onClick={() => navigate("admin")} type="button">
            <ShieldCheck size={16} />
            {t("nav.admin")}
          </button>
          <button className="text-button" onClick={() => navigate("setup")} type="button">
            <UserPlus size={16} />
            {t("nav.setup")}
          </button>
        </div>
      </div>
      <div className="panel">
        <h3>{t("public.runtime")}</h3>
        <dl className="kv-grid">
          <div>
            <dt>{t("public.backend")}</dt>
            <dd><StatusPill ok={bootstrap.apiOnline} text={bootstrap.apiOnline ? t("app.apiOnline") : t("app.apiOffline")} /></dd>
          </div>
          <div>
            <dt>{t("public.setupState")}</dt>
            <dd>{bootstrap.setupStatus?.completed ? t("status.initialized") : t("status.pending")}</dd>
          </div>
          <div>
            <dt>{t("public.locale")}</dt>
            <dd>{bootstrap.publicSettings?.default_locale ?? defaultLocale}</dd>
          </div>
          <div>
            <dt>{t("public.csrf")}</dt>
            <dd>{bootstrap.publicSettings?.auth.csrf_enabled ? t("status.enabled") : t("status.disabled")}</dd>
          </div>
        </dl>
      </div>
      <div className="panel span-3">
        <h3>{t("public.capabilities")}</h3>
        <MiniList
          empty={t("app.empty")}
          items={[
            t("public.capabilitySetup"),
            t("public.capabilityIam"),
            t("public.capabilitySystem"),
            t("public.capabilityContracts")
          ]}
        />
      </div>
    </section>
  );
}

function SetupView(props: {
  busy: boolean;
  completeSetup: () => Promise<void>;
  createSetupRun: () => Promise<void>;
  selectSetupRun: (run: SetupRun) => Promise<void>;
  schema: SetupSchema | null;
  setupLogs: SetupStepLog[];
  setupRun: SetupRun | null;
  setupRuns: SetupRun[];
  setupConfigChecks: SetupConfigCheckSummary | null;
  setupStatus: SetupStatus | null;
  submitInitialAdmin: (event: FormEvent<HTMLFormElement>) => void;
  t: TFn;
}) {
  const { busy, completeSetup, createSetupRun, schema, selectSetupRun, setupConfigChecks, setupLogs, setupRun, setupRuns, setupStatus, submitInitialAdmin, t } = props;
  return (
    <section className="content-grid">
      <div className="panel span-2">
        <div className="section-heading">
          <div>
            <p className="eyebrow">{setupStatus?.completed ? t("status.initialized") : t("status.pending")}</p>
            <h2>{t("setup.title")}</h2>
          </div>
          <button className="text-button" onClick={() => void createSetupRun()} type="button">
            <Play size={16} />
            {t("setup.startRun")}
          </button>
        </div>
        <p className="muted">{setupStatus?.completed ? t("setup.alreadyDone") : t("setup.description")}</p>
        <div className="step-list">
          {(schema?.steps ?? []).map((step) => (
            <div className="step-row" key={step.key}>
              <strong>{step.title}</strong>
              <span>{step.fields.map((field) => field.label).join(" / ") || step.key}</span>
            </div>
          ))}
        </div>
      </div>
      <div className="panel span-2">
        <div className="section-heading">
          <div>
            <p className="eyebrow">{setupConfigChecks?.ready ? t("status.ready") : t("status.blocked")}</p>
            <h2>{t("setup.configChecks")}</h2>
          </div>
        </div>
        <SimpleTable
          columns={[t("table.name"), t("table.status"), t("table.summary")]}
          rows={(setupConfigChecks?.checks ?? []).map((check) => [
            check.title,
            <StatusPill key={check.key} ok={check.status !== "error"} text={t(`status.${check.status}`)} />,
            check.message
          ])}
        />
      </div>
      <form className="panel form-panel" onSubmit={submitInitialAdmin}>
        <h2>{t("setup.adminTitle")}</h2>
        <Input label={t("setup.email")} name="email" required type="email" />
        <Input label={t("setup.password")} name="password" required type="password" />
        <Input label={t("setup.displayName")} name="display_name" required />
        <Input label={t("setup.organizationCode")} name="organization_code" required />
        <Input label={t("setup.organizationName")} name="organization_name" required />
        <button className="primary-button" disabled={busy} type="submit">
          <UserPlus size={16} />
          {t("setup.submit")}
        </button>
      </form>
      <div className="panel">
        <h2>{t("setup.runLogs")}</h2>
        {setupRun ? <p className="muted">{setupRun.id}</p> : null}
        <MiniList empty={t("app.empty")} items={setupLogs.map((log) => `${log.step_key} · ${log.status} · ${log.message}`)} />
        <h3>{t("setup.recentRuns")}</h3>
        <MiniList
          empty={t("app.empty")}
          items={setupRuns.map((run) => `${run.status} · ${run.reason ?? run.id} · ${run.updated_at}`)}
          renderAction={(index) => (
            <button className="icon-button" onClick={() => void selectSetupRun(setupRuns[index])} title={t("setup.viewRun")} type="button">
              <RefreshCw size={14} />
            </button>
          )}
        />
        <button className="text-button" onClick={() => void completeSetup()} type="button">
          <Settings size={16} />
          {t("setup.complete")}
        </button>
      </div>
    </section>
  );
}

function LoginView(props: { busy: boolean; login: (event: FormEvent<HTMLFormElement>) => void; navigate: (view: View) => void; session: SessionSnapshot | null; t: TFn }) {
  const { busy, login, navigate, session, t } = props;
  return (
    <section className="auth-layout">
      <form className="panel form-panel" onSubmit={login}>
        <h2>{t("auth.title")}</h2>
        <Input label={t("auth.identifier")} name="identifier" required type="email" />
        <Input label={t("auth.password")} name="password" required type="password" />
        <Input help={t("auth.mfaHelp")} label={t("auth.mfaCode")} name="mfaCode" />
        <button className="primary-button" disabled={busy} type="submit">
          <LogIn size={16} />
          {t("auth.submit")}
        </button>
        <button className="text-button" onClick={() => navigate("account")} type="button">
          <KeyRound size={16} />
          {t("auth.accountFlows")}
        </button>
      </form>
      <div className="panel">
        <h2>{t("admin.session")}</h2>
        <p className="muted">
          {session?.authenticated && session.user ? t("auth.loggedIn", { email: session.user.email }) : t("admin.needLogin")}
        </p>
      </div>
    </section>
  );
}

function AccountFlowView({ onAuthenticated, publicSettings, t }: { onAuthenticated: (snapshot: SessionSnapshot) => void; publicSettings: PublicSettings | null; t: TFn }) {
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);
  const selfSignupEnabled = publicSettings?.auth.self_signup_enabled === true;
  const [initialTokens] = useState(() => {
    const params = new URLSearchParams(window.location.search);
    return {
      invite: "",
      reset: "",
      verify: "",
      email: params.get("email") || ""
    };
  });

  useEffect(() => {
    if (window.location.pathname.startsWith("/account") && window.location.search) {
      window.history.replaceState(null, "", "/account");
    }
  }, []);

  async function run(action: () => Promise<string | void>) {
    setBusy(true);
    setNotice("");
    try {
      const message = await action();
      if (message) setNotice(message);
    } catch (error) {
      setNotice(errorMessage(error, t));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="content-grid">
      {notice ? <div className="notice span-3">{notice}</div> : null}
      {selfSignupEnabled ? (
        <form
          className="panel form-panel"
          onSubmit={(event) => {
            event.preventDefault();
            const form = event.currentTarget;
            const formData = new FormData(form);
            const payload: RegisterRequest = {
              display_name: String(formData.get("display_name") ?? "").trim(),
              email: String(formData.get("email") ?? "").trim(),
              organization_code: String(formData.get("organization_code") ?? "").trim(),
              organization_name: String(formData.get("organization_name") ?? "").trim(),
              password: String(formData.get("password") ?? "")
            };
            void run(async () => {
              await apiPost(endpoints.auth.register, payload);
              form.reset();
              return t("account.registrationAccepted");
            });
          }}
        >
          <h2>{t("account.register")}</h2>
          <Input label={t("setup.email")} name="email" required type="email" />
          <Input label={t("setup.password")} name="password" required type="password" />
          <Input label={t("setup.displayName")} name="display_name" required />
          <Input label={t("setup.organizationCode")} name="organization_code" required />
          <Input label={t("setup.organizationName")} name="organization_name" required />
          <button className="primary-button" disabled={busy} type="submit"><UserPlus size={16} />{t("account.register")}</button>
        </form>
      ) : null}
      <form
        className="panel form-panel"
        onSubmit={(event) => {
          event.preventDefault();
          const form = event.currentTarget;
          const formData = new FormData(form);
          const payload: AcceptInvitationRequest = {
            display_name: String(formData.get("display_name") ?? "").trim(),
            password: String(formData.get("password") ?? ""),
            token: String(formData.get("token") ?? "").trim()
          };
          void run(async () => {
            const snapshot = await apiPost<SessionSnapshot>(endpoints.auth.acceptInvitation, payload);
            form.reset();
            onAuthenticated(snapshot);
          });
        }}
      >
        <h2>{t("account.acceptInvitation")}</h2>
        <Input defaultValue={initialTokens.invite} label={t("account.invitationToken")} name="token" required />
        <Input label={t("setup.displayName")} name="display_name" required />
        <Input label={t("setup.password")} name="password" required type="password" />
        <button className="primary-button" disabled={busy} type="submit"><UserPlus size={16} />{t("account.acceptInvitation")}</button>
      </form>
      <form
        className="panel form-panel"
        onSubmit={(event) => {
          event.preventDefault();
          const form = event.currentTarget;
          const formData = new FormData(form);
          const payload: ForgotPasswordRequest = { email: String(formData.get("email") ?? "").trim() };
          void run(async () => {
            await apiPost(endpoints.auth.forgotPassword, payload);
            form.reset();
            return t("account.deliveryAccepted");
          });
        }}
      >
        <h2>{t("account.forgotPassword")}</h2>
        <Input defaultValue={initialTokens.email} label={t("setup.email")} name="email" required type="email" />
        <button className="text-button" disabled={busy} type="submit"><KeyRound size={16} />{t("account.requestPasswordReset")}</button>
      </form>
      <form
        className="panel form-panel"
        onSubmit={(event) => {
          event.preventDefault();
          const form = event.currentTarget;
          const formData = new FormData(form);
          const payload: ResetPasswordRequest = {
            password: String(formData.get("password") ?? ""),
            token: String(formData.get("token") ?? "").trim()
          };
          void run(async () => {
            const result = await apiPost<BooleanResult>(endpoints.auth.resetPassword, payload);
            form.reset();
            return result.reset ? t("account.passwordResetDone") : t("app.error");
          });
        }}
      >
        <h2>{t("account.resetPassword")}</h2>
        <Input defaultValue={initialTokens.reset} label={t("account.resetToken")} name="token" required />
        <Input label={t("setup.password")} name="password" required type="password" />
        <button className="text-button" disabled={busy} type="submit"><KeyRound size={16} />{t("account.resetPassword")}</button>
      </form>
      <form
        className="panel form-panel"
        onSubmit={(event) => {
          event.preventDefault();
          const form = event.currentTarget;
          const formData = new FormData(form);
          const payload: RequestEmailVerification = { email: String(formData.get("email") ?? "").trim() };
          void run(async () => {
            await apiPost(endpoints.auth.requestEmailVerification, payload);
            form.reset();
            return t("account.deliveryAccepted");
          });
        }}
      >
        <h2>{t("account.requestEmailVerification")}</h2>
        <Input defaultValue={initialTokens.email} label={t("setup.email")} name="email" required type="email" />
        <button className="text-button" disabled={busy} type="submit"><ShieldCheck size={16} />{t("account.requestEmailVerification")}</button>
      </form>
      <form
        className="panel form-panel"
        onSubmit={(event) => {
          event.preventDefault();
          const form = event.currentTarget;
          const payload: ConfirmEmailVerificationRequest = {
            token: String(new FormData(form).get("token") ?? "").trim()
          };
          void run(async () => {
            const result = await apiPost<BooleanResult>(endpoints.auth.confirmEmailVerification, payload);
            form.reset();
            return result.verified ? t("account.emailVerified") : t("app.error");
          });
        }}
      >
        <h2>{t("account.confirmEmailVerification")}</h2>
        <Input defaultValue={initialTokens.verify} label={t("account.emailVerificationToken")} name="token" required />
        <button className="text-button" disabled={busy} type="submit"><ShieldCheck size={16} />{t("account.confirmEmailVerification")}</button>
      </form>
    </section>
  );
}

function AdminView(props: {
  data: AdminData;
  loadAdmin: (orgId?: number | null) => Promise<void>;
  mutate: (action: () => Promise<unknown>) => Promise<void>;
  refreshBootstrap: () => Promise<void>;
  selectedTab: AdminTab;
  session: SessionSnapshot | null;
  setSelectedOrg: (orgId: number) => void;
  setTab: (tab: AdminTab) => void;
  t: TFn;
}) {
  const { data, loadAdmin, mutate, refreshBootstrap, selectedTab, session, setSelectedOrg, setTab, t } = props;
  if (!session?.authenticated) {
    return <div className="panel"><h2>{t("admin.title")}</h2><p className="muted">{t("admin.needLogin")}</p></div>;
  }
  return (
    <section className="admin-layout">
      <aside className="side-panel">
        <h2>{t("admin.title")}</h2>
        <p className="muted">{t("admin.description")}</p>
        <nav className="side-nav">
          <TabButton active={selectedTab === "overview"} icon={<Activity size={16} />} label={t("nav.overview")} onClick={() => setTab("overview")} />
          <TabButton active={selectedTab === "iam"} icon={<KeyRound size={16} />} label={t("nav.iam")} onClick={() => setTab("iam")} />
          <TabButton active={selectedTab === "system"} icon={<Database size={16} />} label={t("nav.system")} onClick={() => setTab("system")} />
          <TabButton active={selectedTab === "assets"} icon={<Boxes size={16} />} label={t("nav.assets")} onClick={() => setTab("assets")} />
        </nav>
      </aside>
      <div className="admin-content">
        <div className="section-heading">
          <div>
            <p className="eyebrow">{session.organization?.name ?? t("app.unknown")}</p>
            <h2>{tabTitle(selectedTab, t)}</h2>
          </div>
          <button className="icon-button" onClick={() => void loadAdmin(data.selectedOrgId)} title={t("app.refresh")} type="button">
            <RefreshCw size={18} />
          </button>
        </div>
        {selectedTab === "overview" ? <OverviewTab data={data} session={session} t={t} /> : null}
        {selectedTab === "iam" ? <IamTab data={data} mutate={mutate} refreshBootstrap={refreshBootstrap} session={session} setSelectedOrg={setSelectedOrg} t={t} /> : null}
        {selectedTab === "system" ? <SystemTab data={data} mutate={mutate} t={t} /> : null}
        {selectedTab === "assets" ? <AssetsTab data={data} mutate={mutate} t={t} /> : null}
      </div>
    </section>
  );
}

function OverviewTab({ data, session, t }: { data: AdminData; session: SessionSnapshot; t: TFn }) {
  const metrics = data.server?.metrics;
  const operationSummary = data.operationSummary;
  return (
    <div className="content-grid">
      <Metric icon={<ShieldCheck />} label={t("admin.permissions")} value={session.permissions.length} />
      <Metric icon={<Server />} label={t("admin.server")} value={data.server?.version ?? t("app.unknown")} />
      <Metric icon={<Activity />} label={t("admin.cpuUsage")} value={metrics ? formatPercent(metrics.cpu_usage_percent) : t("app.unknown")} />
      <Metric icon={<Database />} label={t("admin.memoryUsage")} value={metrics ? `${formatBytes(metrics.used_memory_bytes)} / ${formatBytes(metrics.total_memory_bytes)}` : t("app.unknown")} />
      <Metric icon={<Menu />} label={t("admin.menus")} value={data.menus.length} />
      <Metric icon={<Activity />} label={t("admin.operations")} value={operationSummary?.total_count ?? data.operations.length} />
      <Metric icon={<AlertTriangle />} label={t("admin.clientErrors")} value={operationSummary?.client_error_count ?? 0} />
      <Metric icon={<AlertTriangle />} label={t("admin.serverErrors")} value={operationSummary?.server_error_count ?? 0} />
      <div className="panel span-2">
        <h3>{t("admin.server")}</h3>
        <p className="muted">{t("admin.realMetricsOnly")}</p>
        <KeyValueGrid values={data.server ?? {}} />
      </div>
      <div className="panel">
        <h3>{t("admin.resourceMetrics")}</h3>
        <KeyValueGrid values={metrics ? resourceMetricValues(metrics, t) : {}} />
      </div>
      <div className="panel">
        <h3>{t("admin.menus")}</h3>
        <MiniList empty={t("app.empty")} items={data.menus.map((item) => `${item.title} · ${item.permission ?? "-"}`)} />
      </div>
      <div className="panel span-2">
        <h3>{t("admin.auditSummary")}</h3>
        <SimpleTable
          columns={[t("table.path"), t("table.count"), t("table.errors"), t("table.createdAt")]}
          rows={(operationSummary?.top_paths ?? []).map((item) => [item.path, item.count, item.error_count, item.last_seen_at ?? "-"])}
        />
      </div>
      <div className="panel">
        <h3>{t("admin.statusClasses")}</h3>
        <MiniList empty={t("app.empty")} items={(operationSummary?.by_status_class ?? []).map((item) => `${item.key} · ${item.count}`)} />
      </div>
      <div className="panel span-3">
        <h3>{t("admin.operations")}</h3>
        <SimpleTable
          columns={[t("table.method"), t("table.path"), t("table.status"), t("table.createdAt")]}
          rows={data.operations.map((item) => [item.method, item.path, item.status, item.created_at])}
        />
      </div>
    </div>
  );
}

function IamTab({
  data,
  mutate,
  refreshBootstrap,
  session,
  setSelectedOrg,
  t
}: {
  data: AdminData;
  mutate: (action: () => Promise<unknown>) => Promise<void>;
  refreshBootstrap: () => Promise<void>;
  session: SessionSnapshot;
  setSelectedOrg: (orgId: number) => void;
  t: TFn;
}) {
  return (
    <div className="content-grid">
      <div className="panel">
        <h3>{t("admin.organizations")}</h3>
        <select className="full-control" onChange={(event) => setSelectedOrg(Number(event.currentTarget.value))} value={data.selectedOrgId ?? ""}>
          {data.orgs.map((org) => <option key={org.id} value={org.id}>{org.name}</option>)}
        </select>
        <SimpleTable
          columns={[t("table.code"), t("table.name"), t("table.scope"), t("table.status")]}
          rows={data.orgs.map((item) => [item.code, item.name, item.scope, item.status])}
        />
      </div>
      <UserManagementPanel data={data} mutate={mutate} t={t} />
      <RoleManagementPanel data={data} mutate={mutate} t={t} />
      <APITokenPanel data={data} mutate={mutate} t={t} />
      <InvitationPanel data={data} mutate={mutate} t={t} />
      <MfaPanel data={data} mutate={mutate} refreshBootstrap={refreshBootstrap} session={session} t={t} />
      <div className="panel span-3">
        <h3>{t("admin.permissionCatalog")}</h3>
        <SimpleTable
          columns={[t("table.code"), t("table.scope"), t("table.summary")]}
          rows={data.permissions.map((item) => [item.code, item.scope, item.name])}
        />
      </div>
    </div>
  );
}

function UserManagementPanel({
  data,
  mutate,
  t
}: {
  data: AdminData;
  mutate: (action: () => Promise<unknown>) => Promise<void>;
  t: TFn;
}) {
  const selectedOrgId = data.selectedOrgId;
  const [editUserId, setEditUserId] = useState<number | null>(null);
  const roleCodes = useMemo(() => data.roles.map((role) => role.code), [data.roles]);
  useEffect(() => {
    if (!data.users.length) {
      setEditUserId(null);
      return;
    }
    if (!editUserId || !data.users.some((user) => user.id === editUserId)) {
      setEditUserId(data.users[0].id);
    }
  }, [data.users, editUserId]);
  const editingUser = data.users.find((user) => user.id === editUserId) ?? null;
  return (
    <div className="panel span-2">
      <div className="section-heading">
        <div>
          <p className="eyebrow">{t("admin.tenantScope")}</p>
          <h3>{t("admin.userManagement")}</h3>
        </div>
      </div>
      <SimpleTable
        columns={[t("table.id"), t("auth.identifier"), t("setup.displayName"), t("table.status"), t("admin.roles")]}
        rows={data.users.map((item) => [item.id, item.email, item.display_name, item.status, item.role_codes.join(", ")])}
      />
      {editingUser && selectedOrgId ? (
        <form
          aria-label={t("admin.editUser")}
          className="form-panel"
          key={editingUser.id}
          onSubmit={(event) => {
            event.preventDefault();
            const form = event.currentTarget;
            const formData = new FormData(form);
            const payload: UpdateOrgUserRequest = {
              display_name: String(formData.get("display_name") ?? "").trim(),
              status: String(formData.get("status") ?? "active") as UpdateOrgUserRequest["status"],
              role_codes: formData.getAll("role_codes").map(String)
            };
            void mutate(() => apiPut(endpoints.iam.orgUser(selectedOrgId, editingUser.id), payload));
          }}
        >
          <h3><Pencil size={16} /> {t("admin.editUser")}</h3>
          <label className="field">
            <span>{t("admin.users")}</span>
            <select aria-label={t("admin.users")} onChange={(event) => setEditUserId(Number(event.currentTarget.value))} value={editingUser.id}>
              {data.users.map((user) => <option key={user.id} value={user.id}>{user.email}</option>)}
            </select>
          </label>
          <Input defaultValue={editingUser.display_name} label={t("setup.displayName")} name="display_name" required />
          <label className="field">
            <span>{t("table.status")}</span>
            <select aria-label={t("table.status")} defaultValue={editingUser.status} name="status">
              <option value="active">{t("status.enabled")}</option>
              <option value="disabled">{t("status.disabled")}</option>
            </select>
          </label>
          <label className="field">
            <span>{t("admin.roles")}</span>
            <select aria-label={t("admin.roles")} defaultValue={editingUser.role_codes} multiple name="role_codes" size={Math.max(2, Math.min(5, roleCodes.length || 2))}>
              {roleCodes.map((roleCode) => <option key={roleCode} value={roleCode}>{roleCode}</option>)}
            </select>
          </label>
          <button className="text-button" type="submit"><Pencil size={16} />{t("admin.updateUser")}</button>
        </form>
      ) : <p className="muted">{t("admin.noUsers")}</p>}
    </div>
  );
}

function RoleManagementPanel({
  data,
  mutate,
  t
}: {
  data: AdminData;
  mutate: (action: () => Promise<unknown>) => Promise<void>;
  t: TFn;
}) {
  const editableRoles = useMemo(() => data.roles.filter((role) => !role.system_builtin), [data.roles]);
  const bindablePermissions = useMemo(
    () => data.permissions.filter((permission) => permission.scope === "tenant" || permission.scope === "product"),
    [data.permissions],
  );
  const [editRoleId, setEditRoleId] = useState<number | null>(null);
  useEffect(() => {
    if (!editableRoles.length) {
      setEditRoleId(null);
      return;
    }
    if (!editRoleId || !editableRoles.some((role) => role.id === editRoleId)) {
      setEditRoleId(editableRoles[0].id);
    }
  }, [editRoleId, editableRoles]);
  const selectedRole = editableRoles.find((role) => role.id === editRoleId) ?? null;
  const selectedOrgId = data.selectedOrgId;

  return (
    <div className="panel span-2">
      <div className="section-heading">
        <div>
          <p className="eyebrow">{t("admin.tenantScope")}</p>
          <h3>{t("admin.roleManagement")}</h3>
        </div>
      </div>
      <SimpleTable
        columns={[t("table.code"), t("table.name"), t("table.scope"), t("table.permission"), t("table.actions")]}
        rows={data.roles.map((role) => [
          role.code,
          role.system_builtin ? `${role.name} · ${t("admin.builtinRole")}` : role.name,
          role.scope,
          role.permissions.length,
          selectedOrgId ? (
            <button
              className="icon-button"
              disabled={role.system_builtin}
              onClick={() => {
                if (window.confirm(t("admin.confirmDeleteRole", { code: role.code }))) {
                  void mutate(() => apiDelete(endpoints.iam.orgRole(selectedOrgId, role.id)));
                }
              }}
              title={role.system_builtin ? t("admin.builtinRole") : t("app.delete")}
              type="button"
            >
              <Trash2 size={16} />
            </button>
          ) : "-",
        ])}
      />
      {selectedOrgId ? (
        <div className="role-forms">
          <RoleCreateForm bindablePermissions={bindablePermissions} mutate={mutate} orgId={selectedOrgId} t={t} />
          <RoleEditForm
            bindablePermissions={bindablePermissions}
            editableRoles={editableRoles}
            key={selectedRole?.id ?? "empty"}
            mutate={mutate}
            orgId={selectedOrgId}
            role={selectedRole}
            selectedRoleId={editRoleId}
            setSelectedRoleId={setEditRoleId}
            t={t}
          />
        </div>
      ) : (
        <p className="muted">{t("admin.selectOrg")}</p>
      )}
    </div>
  );
}

function RoleCreateForm({
  bindablePermissions,
  mutate,
  orgId,
  t
}: {
  bindablePermissions: PermissionSummary[];
  mutate: (action: () => Promise<unknown>) => Promise<void>;
  orgId: number;
  t: TFn;
}) {
  return (
    <form
      aria-label={t("admin.createRole")}
      className="inline-form"
      onSubmit={(event) => {
        event.preventDefault();
        const form = event.currentTarget;
        const formData = new FormData(form);
        const payload: CreateRoleRequest = {
          code: String(formData.get("code") ?? "").trim(),
          name: String(formData.get("name") ?? "").trim(),
          permission_codes: selectedFormValues(formData, "permission_codes")
        };
        void mutate(async () => {
          await apiPost(endpoints.iam.orgRoles(orgId), payload);
          form.reset();
        });
      }}
    >
      <h3><Plus size={16} /> {t("admin.createRole")}</h3>
      <Input label={t("admin.fields.roleCode")} name="code" required />
      <Input label={t("admin.fields.roleName")} name="name" required />
      <PermissionSelect defaultValues={[]} name="permission_codes" permissions={bindablePermissions} t={t} />
      <button className="text-button" type="submit"><Plus size={16} />{t("admin.createRole")}</button>
    </form>
  );
}

function RoleEditForm({
  bindablePermissions,
  editableRoles,
  mutate,
  orgId,
  role,
  selectedRoleId,
  setSelectedRoleId,
  t
}: {
  bindablePermissions: PermissionSummary[];
  editableRoles: RoleSummary[];
  mutate: (action: () => Promise<unknown>) => Promise<void>;
  orgId: number;
  role: RoleSummary | null;
  selectedRoleId: number | null;
  setSelectedRoleId: (roleId: number | null) => void;
  t: TFn;
}) {
  if (!editableRoles.length || !role) {
    return (
      <div className="inline-form">
        <h3><Pencil size={16} /> {t("admin.editRole")}</h3>
        <p className="muted">{t("admin.noEditableRoles")}</p>
      </div>
    );
  }
  return (
    <form
      aria-label={t("admin.editRole")}
      className="inline-form"
      onSubmit={(event) => {
        event.preventDefault();
        const formData = new FormData(event.currentTarget);
        const payload: UpdateRoleRequest = {
          name: String(formData.get("name") ?? "").trim(),
          permission_codes: selectedFormValues(formData, "permission_codes")
        };
        void mutate(() => apiPut(endpoints.iam.orgRole(orgId, role.id), payload));
      }}
    >
      <h3><Pencil size={16} /> {t("admin.editRole")}</h3>
      <label className="field">
        <span>{t("admin.fields.role")}</span>
        <select
          aria-label={t("admin.fields.role")}
          name="role_id"
          onChange={(event) => setSelectedRoleId(Number(event.currentTarget.value))}
          value={selectedRoleId ?? ""}
        >
          {editableRoles.map((item) => <option key={item.id} value={item.id}>{item.code}</option>)}
        </select>
      </label>
      <Input defaultValue={role.name} label={t("admin.fields.roleName")} name="name" required />
      <PermissionSelect defaultValues={role.permissions} name="permission_codes" permissions={bindablePermissions} t={t} />
      <button className="text-button" type="submit"><Pencil size={16} />{t("admin.updateRole")}</button>
    </form>
  );
}

function PermissionSelect({
  defaultValues,
  name,
  permissions,
  t
}: {
  defaultValues: string[];
  name: string;
  permissions: PermissionSummary[];
  t: TFn;
}) {
  if (!permissions.length) return <p className="muted">{t("admin.noBindablePermissions")}</p>;
  return (
    <label className="field">
      <span>{t("admin.fields.permissions")}</span>
      <select aria-label={t("admin.fields.permissions")} className="multi-select" defaultValue={defaultValues} multiple name={name}>
      {permissions.map((permission) => (
        <option key={permission.code} value={permission.code}>{permission.code} · {permission.name}</option>
      ))}
      </select>
    </label>
  );
}

function APITokenPanel({ data, mutate, t }: { data: AdminData; mutate: (action: () => Promise<unknown>) => Promise<void>; t: TFn }) {
  const [tokenReveal, setTokenReveal] = useState<CreateAPITokenResult | null>(null);
  const selectedOrgId = data.selectedOrgId;
  return (
    <div className="panel span-2">
      <div className="section-heading">
        <div>
          <p className="eyebrow">{t("admin.secretSafety")}</p>
          <h3>{t("admin.apiTokens")}</h3>
        </div>
      </div>
      {tokenReveal ? (
        <div className="secret-reveal">
          <strong>{t("admin.tokenCreated")}</strong>
          <code>{tokenReveal.token}</code>
          <small>{t("admin.tokenRevealWarning")}</small>
        </div>
      ) : null}
      {selectedOrgId ? (
        <form
          aria-label={t("admin.createApiToken")}
          className="inline-form compact-form"
          onSubmit={(event) => {
            event.preventDefault();
            const form = event.currentTarget;
            const formData = new FormData(form);
            const expires = Number(formData.get("expires_in_days") || 0);
            const payload: CreateAPITokenRequest = {
              expires_in_days: expires > 0 ? expires : null,
              remark: String(formData.get("remark") ?? "").trim() || null
            };
            void mutate(async () => {
              const created = await apiPost<CreateAPITokenResult>(endpoints.orgs.apiTokens(selectedOrgId), payload);
              setTokenReveal(created);
              form.reset();
            });
          }}
        >
          <h3><KeyRound size={16} /> {t("admin.createApiToken")}</h3>
          <Input label={t("admin.fields.expiresInDays")} name="expires_in_days" type="number" />
          <Input label={t("admin.fields.remark")} name="remark" />
          <button className="text-button" type="submit"><Plus size={16} />{t("admin.createApiToken")}</button>
        </form>
      ) : null}
      <SimpleTable
        columns={[t("table.id"), t("table.prefix"), t("table.status"), t("table.expiresAt"), t("table.actions")]}
        rows={data.apiTokens.map((token) => [
          token.id,
          token.token_prefix,
          token.status,
          token.expires_at ?? "-",
          selectedOrgId && !token.revoked_at ? (
            <button
              className="icon-button"
              onClick={() => {
                if (window.confirm(t("admin.confirmRevokeToken", { prefix: token.token_prefix }))) {
                  void mutate(() => apiDelete(endpoints.orgs.apiToken(selectedOrgId, token.id)));
                }
              }}
              title={t("admin.revokeToken")}
              type="button"
            >
              <Trash2 size={16} />
            </button>
          ) : "-",
        ])}
      />
    </div>
  );
}

function InvitationPanel({ data, mutate, t }: { data: AdminData; mutate: (action: () => Promise<unknown>) => Promise<void>; t: TFn }) {
  const selectedOrgId = data.selectedOrgId;
  const roleCodes = data.roles.map((role) => role.code);
  return (
    <div className="panel span-2">
      <div className="section-heading">
        <div>
          <p className="eyebrow">{t("admin.pendingFlows")}</p>
          <h3>{t("admin.invitations")}</h3>
        </div>
      </div>
      {selectedOrgId ? (
        <form
          aria-label={t("admin.createInvitation")}
          className="inline-form compact-form"
          onSubmit={(event) => {
            event.preventDefault();
            const form = event.currentTarget;
            const formData = new FormData(form);
            const payload: InviteUserRequest = {
              email: String(formData.get("email") ?? "").trim(),
              role_code: String(formData.get("role_code") ?? "").trim() || null
            };
            void mutate(async () => {
              await apiPost(endpoints.orgs.userInvitations(selectedOrgId), payload);
              form.reset();
            });
          }}
        >
          <h3><UserPlus size={16} /> {t("admin.createInvitation")}</h3>
          <Input label={t("setup.email")} name="email" required type="email" />
          <label className="field">
            <span>{t("admin.fields.role")}</span>
            <select aria-label={t("admin.fields.role")} name="role_code">
              {roleCodes.map((roleCode) => <option key={roleCode} value={roleCode}>{roleCode}</option>)}
            </select>
          </label>
          <button className="text-button" type="submit"><UserPlus size={16} />{t("admin.createInvitation")}</button>
        </form>
      ) : null}
      <SimpleTable
        columns={[t("auth.identifier"), t("admin.fields.role"), t("table.status"), t("table.expiresAt"), t("table.actions")]}
        rows={data.invitations.map((invitation) => [
          invitation.email,
          invitation.role_code,
          invitation.status,
          invitation.expires_at,
          selectedOrgId && invitation.status === "pending" ? (
            <button
              className="icon-button"
              onClick={() => {
                if (window.confirm(t("admin.confirmRevokeInvitation", { email: invitation.email }))) {
                  void mutate(() => apiDelete(endpoints.orgs.invitation(selectedOrgId, invitation.id)));
                }
              }}
              title={t("admin.revokeInvitation")}
              type="button"
            >
              <Trash2 size={16} />
            </button>
          ) : "-",
        ])}
      />
    </div>
  );
}

function MfaPanel({
  data,
  mutate,
  refreshBootstrap,
  session,
  t
}: {
  data: AdminData;
  mutate: (action: () => Promise<unknown>) => Promise<void>;
  refreshBootstrap: () => Promise<void>;
  session: SessionSnapshot;
  t: TFn;
}) {
  const [setupResult, setSetupResult] = useState<MfaSetupResult | null>(null);
  const [recoveryReveal, setRecoveryReveal] = useState<string[]>([]);
  const revokableFactors = data.mfaFactors.filter((factor) => !factor.revoked_at && factor.status !== "revoked");
  return (
    <div className="panel span-2">
      <div className="section-heading">
        <div>
          <p className="eyebrow">{session.mfa_enabled ? t("status.enabled") : t("status.disabled")}</p>
          <h3>{t("admin.mfa")}</h3>
        </div>
      </div>
      <p className="muted">{t("admin.mfaDescription")}</p>
      {setupResult ? (
        <div className="secret-reveal">
          <strong>{t("admin.mfaSecret")}</strong>
          <code>{setupResult.secret}</code>
          <small>{setupResult.otpauth_url}</small>
        </div>
      ) : null}
      {recoveryReveal.length ? (
        <div className="secret-reveal" aria-label={t("admin.mfaRecoveryReveal")}>
          <strong>{t("admin.mfaRecoveryReveal")}</strong>
          <div className="secret-list">
            {recoveryReveal.map((code) => <code key={code}>{code}</code>)}
          </div>
          <small>{t("admin.mfaRecoveryRevealWarning")}</small>
        </div>
      ) : null}
      <div className="form-panel">
        <button
          className="text-button"
          onClick={() => {
            void mutate(async () => {
              setSetupResult(await apiPost<MfaSetupResult>(endpoints.auth.mfaSetup));
            });
          }}
          type="button"
        >
          <KeyRound size={16} />{t("admin.startMfaSetup")}
        </button>
        {setupResult ? (
          <form
            className="inline-form"
            onSubmit={(event) => {
              event.preventDefault();
              const form = event.currentTarget;
              const formData = new FormData(form);
              const payload: VerifyMfaRequest = { code: String(formData.get("code") ?? "").trim() };
              void mutate(async () => {
                const verified = await apiPost<MfaVerifyResult>(endpoints.auth.mfaVerify, payload);
                setRecoveryReveal(verified.recovery_codes);
                setSetupResult(null);
                await refreshBootstrap();
              });
            }}
          >
            <Input label={t("auth.mfaCode")} name="code" required />
            <button className="text-button" type="submit"><ShieldCheck size={16} />{t("admin.verifyMfa")}</button>
          </form>
        ) : null}
      </div>
      <SimpleTable
        columns={[t("table.id"), t("table.kind"), t("table.status"), t("table.createdAt"), t("table.actions")]}
        rows={data.mfaFactors.map((factor) => [
          factor.id,
          factor.kind,
          factor.status,
          factor.created_at,
          !factor.revoked_at && factor.status !== "revoked" ? (
            <button
              className="icon-button"
              onClick={() => {
                if (window.confirm(t("admin.confirmRevokeMfa", { id: factor.id }))) {
                  void mutate(async () => {
                    await apiDelete(endpoints.auth.mfaFactor(factor.id));
                    if (setupResult?.factor.id === factor.id) setSetupResult(null);
                    await refreshBootstrap();
                  });
                }
              }}
              title={t("admin.revokeCurrentMfa")}
              type="button"
            >
              <Trash2 size={16} />
            </button>
          ) : "-",
        ])}
      />
      {session.mfa_enabled && revokableFactors.length === 0 ? <p className="muted">{t("admin.mfaNoRevokableFactors")}</p> : null}
      <div className="section-heading compact-heading">
        <div>
          <p className="eyebrow">{t("admin.secretSafety")}</p>
          <h3>{t("admin.mfaRecoveryCodes")}</h3>
        </div>
        <button
          className="text-button"
          disabled={!session.mfa_enabled}
          onClick={() => {
            if (window.confirm(t("admin.confirmRotateMfaRecoveryCodes"))) {
              void mutate(async () => {
                const rotated = await apiPost<MfaRecoveryCodesResult>(endpoints.auth.mfaRecoveryCodes);
                setRecoveryReveal(rotated.recovery_codes);
                await refreshBootstrap();
              });
            }
          }}
          type="button"
        >
          <RefreshCw size={16} />{t("admin.rotateMfaRecoveryCodes")}
        </button>
      </div>
      <p className="muted">{t("admin.mfaRecoveryDescription")}</p>
      <SimpleTable
        columns={[t("table.id"), t("table.prefix"), t("table.status"), t("table.createdAt"), t("table.usedAt"), t("table.revokedAt")]}
        rows={data.mfaRecoveryCodes.map((code) => [
          code.id,
          code.code_prefix,
          code.status,
          code.created_at,
          code.used_at ?? "-",
          code.revoked_at ?? "-"
        ])}
      />
    </div>
  );
}

function SystemTab({ data, mutate, t }: { data: AdminData; mutate: (action: () => Promise<unknown>) => Promise<void>; t: TFn }) {
  return (
    <div className="content-grid">
      <JsonUpsertPanel
        title={t("admin.configs")}
        fields={[["key", t("admin.fields.key")], ["value", t("admin.fields.value")]]}
        onSubmit={(form) => mutate(() => apiPut(endpoints.system.config(String(form.get("key"))), { value: parseJsonValue(String(form.get("value") || "{}")) }))}
        t={t}
      />
      <NamedUpsertPanel
        title={t("admin.dictionaries")}
        codeLabel={t("admin.fields.code")}
        nameLabel={t("admin.fields.name")}
        onSubmit={(form) => mutate(() => apiPut(endpoints.system.dictionary(String(form.get("code"))), { name: String(form.get("name")) }))}
        t={t}
      />
      <NamedUpsertPanel
        title={t("admin.parameters")}
        codeLabel={t("admin.fields.key")}
        nameLabel={t("admin.fields.name")}
        valueLabel={t("admin.fields.value")}
        onSubmit={(form) => mutate(() => apiPut(endpoints.system.parameter(String(form.get("code"))), { name: String(form.get("name")), value: String(form.get("value")) }))}
        t={t}
      />
      <div className="panel span-3">
        <h3>{t("admin.apiCatalog")}</h3>
        <SimpleTable
          columns={[t("table.method"), t("table.path"), t("table.permission"), t("table.summary")]}
          rows={data.apiGroups.flatMap((group) => group.items.map((item) => [item.method, item.path, item.permission ?? "-", item.summary]))}
        />
      </div>
      <div className="panel">
        <h3>{t("admin.configs")}</h3>
        <MiniList
          empty={t("app.empty")}
          items={data.configs.map((item) => `${item.key} · ${JSON.stringify(item.value)}`)}
          renderAction={(index) => (
            <DeleteIconButton
              label={t("app.delete")}
              onClick={() => confirmDelete(t, data.configs[index].key, () => mutate(() => apiDelete(endpoints.system.config(data.configs[index].key))))}
            />
          )}
        />
      </div>
      <div className="panel">
        <h3>{t("admin.dictionaries")}</h3>
        <MiniList
          empty={t("app.empty")}
          items={data.dictionaries.map((item) => `${item.code} · ${item.name}`)}
          renderAction={(index) => (
            <DeleteIconButton
              label={t("app.delete")}
              onClick={() => confirmDelete(t, data.dictionaries[index].code, () => mutate(() => apiDelete(endpoints.system.dictionary(data.dictionaries[index].code))))}
            />
          )}
        />
      </div>
      <div className="panel">
        <h3>{t("admin.parameters")}</h3>
        <MiniList
          empty={t("app.empty")}
          items={data.parameters.map((item) => `${item.key} · ${item.value}`)}
          renderAction={(index) => (
            <DeleteIconButton
              label={t("app.delete")}
              onClick={() => confirmDelete(t, data.parameters[index].key, () => mutate(() => apiDelete(endpoints.system.parameter(data.parameters[index].key))))}
            />
          )}
        />
      </div>
    </div>
  );
}

function AssetsTab({ data, mutate, t }: { data: AdminData; mutate: (action: () => Promise<unknown>) => Promise<void>; t: TFn }) {
  return (
    <div className="content-grid">
      <VersionForm mutate={mutate} t={t} />
      <MediaUploadForm mutate={mutate} t={t} />
      <MediaForm mutate={mutate} t={t} />
      <ProbeForm mutate={mutate} t={t} />
      <div className="panel">
        <h3><Package size={16} /> {t("admin.versions")}</h3>
        <SimpleTable
          columns={[
            t("table.id"),
            t("table.name"),
            t("table.code"),
            t("table.status"),
            t("table.publishedAt"),
            t("table.retiredAt"),
            t("table.actions")
          ]}
          rows={data.versions.map((item) => [
            item.id,
            item.version_name,
            item.version_code,
            item.status,
            item.published_at ?? "-",
            item.retired_at ?? "-",
            <div className="inline-actions">
              {item.status === "retired" ? (
                <button
                  className="icon-button"
                  onClick={() => confirmAction(t("admin.confirmRollbackVersion", { name: item.version_name }), () => mutate(() => apiPost(endpoints.system.versionPackageRollback(item.id), { reason: null })))}
                  title={t("app.rollback")}
                  type="button"
                >
                  <RefreshCw size={16} />
                </button>
              ) : null}
              {item.status !== "active" && item.status !== "retired" ? (
                <button
                  className="icon-button"
                  onClick={() => confirmAction(t("admin.confirmPublishVersion", { name: item.version_name }), () => mutate(() => apiPost(endpoints.system.versionPackagePublish(item.id), { reason: null })))}
                  title={t("app.publish")}
                  type="button"
                >
                  <Play size={16} />
                </button>
              ) : null}
              {item.status !== "active" ? (
                <DeleteIconButton
                  label={t("app.delete")}
                  onClick={() => confirmDelete(t, item.version_name, () => mutate(() => apiDelete(endpoints.system.versionPackage(item.id))))}
                />
              ) : <span className="muted">-</span>}
            </div>
          ])}
        />
      </div>
      <div className="panel span-3">
        <h3><RefreshCw size={16} /> {t("admin.versionReleases")}</h3>
        <SimpleTable
          columns={[
            t("table.id"),
            t("table.action"),
            t("table.status"),
            t("table.package"),
            t("table.previous"),
            t("table.reason"),
            t("table.createdAt")
          ]}
          rows={data.versionReleases.map((item) => [
            item.id,
            item.action,
            item.status,
            item.package_id,
            item.previous_active_id ?? "-",
            item.reason ?? "-",
            item.created_at
          ])}
        />
      </div>
      <div className="panel">
        <h3><Image size={16} /> {t("admin.media")}</h3>
        <MiniList
          empty={t("app.empty")}
          items={data.media.map((item) => `${item.display_name} · ${item.storage_key}`)}
          renderAction={(index) => (
            <DeleteIconButton
              label={t("app.delete")}
              onClick={() => confirmDelete(t, data.media[index].display_name, () => mutate(() => apiDelete(endpoints.system.mediaAsset(data.media[index].id))))}
            />
          )}
        />
      </div>
      <div className="panel span-2">
        <h3><Boxes size={16} /> {t("admin.storageObjects")}</h3>
        <SimpleTable
          columns={[
            t("admin.fields.storageKey"),
            t("admin.fields.sizeBytes"),
            t("table.updatedAt"),
            t("table.actions")
          ]}
          rows={data.storageObjects.map((item) => [
            item.storage_key,
            formatBytes(item.size_bytes),
            item.updated_at ?? "-",
            <DeleteIconButton
              label={t("app.delete")}
              onClick={() => confirmDelete(t, item.storage_key, () => mutate(() => apiDelete(endpoints.system.storageObjects, { storage_key: item.storage_key })))}
            />
          ])}
        />
      </div>
      <div className="panel">
        <h3><Radar size={16} /> {t("admin.probes")}</h3>
        <MiniList
          empty={t("app.empty")}
          items={data.probes.map((item) => `${item.name} · ${item.status} · ${item.url}`)}
          renderAction={(index) => (
            <>
              <button className="icon-button" onClick={() => void mutate(() => apiPost(endpoints.system.trafficProbeRun(data.probes[index].id)))} title={t("app.run")} type="button">
                <Play size={16} />
              </button>
              <DeleteIconButton
                label={t("app.delete")}
                onClick={() => confirmDelete(t, data.probes[index].name, () => mutate(() => apiDelete(endpoints.system.trafficProbeTarget(data.probes[index].id))))}
              />
            </>
          )}
        />
      </div>
      <div className="panel span-3">
        <h3>{t("admin.probeResults")}</h3>
        <SimpleTable columns={[t("table.id"), t("table.status"), t("table.createdAt")]} rows={data.probeResults.map((item) => [item.id, item.status, item.probed_at])} />
      </div>
      <div className="panel span-3">
        <h3><AlertTriangle size={16} /> {t("admin.probeAlerts")}</h3>
        <SimpleTable
          columns={[
            t("table.id"),
            t("table.severity"),
            t("table.status"),
            t("table.reason"),
            t("table.target"),
            t("table.result"),
            t("table.openedAt"),
            t("table.actions")
          ]}
          rows={data.probeAlerts.map((item) => [
            item.id,
            item.severity,
            item.status,
            item.reason,
            item.target_id,
            item.result_id,
            item.opened_at,
            <div className="inline-actions">
              {item.status === "open" ? (
                <button className="icon-button" onClick={() => void mutate(() => apiPost(endpoints.system.trafficProbeAlertAck(item.id)))} title={t("app.acknowledge")} type="button">
                  <CheckCircle2 size={16} />
                </button>
              ) : null}
              {item.status !== "resolved" ? (
                <button className="icon-button" onClick={() => void mutate(() => apiPost(endpoints.system.trafficProbeAlertResolve(item.id)))} title={t("app.resolve")} type="button">
                  <CheckCircle2 size={16} />
                </button>
              ) : <span className="muted">-</span>}
            </div>
          ])}
        />
      </div>
    </div>
  );
}

function VersionForm({ mutate, t }: { mutate: (action: () => Promise<unknown>) => Promise<void>; t: TFn }) {
  return (
    <form className="panel form-panel" onSubmit={(event) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      void mutate(() => apiPost(endpoints.system.versionPackages, {
        manifest: parseJsonValue(String(form.get("manifest") || "{}")),
        version_code: String(form.get("version_code")),
        version_name: String(form.get("version_name"))
      }));
      event.currentTarget.reset();
    }}>
      <h3>{t("admin.versions")}</h3>
      <Input label={t("admin.fields.versionName")} name="version_name" required />
      <Input label={t("admin.fields.versionCode")} name="version_code" required />
      <Input label={t("admin.fields.manifest")} name="manifest" />
      <button className="text-button" type="submit"><Package size={16} />{t("app.create")}</button>
    </form>
  );
}

function MediaUploadForm({ mutate, t }: { mutate: (action: () => Promise<unknown>) => Promise<void>; t: TFn }) {
  return (
    <form className="panel form-panel" onSubmit={(event) => {
      event.preventDefault();
      const target = event.currentTarget;
      const form = new FormData(target);
      void mutate(() => apiPostForm<MediaAssetEntry>(endpoints.system.mediaAssetUpload, form));
      target.reset();
    }}>
      <h3>{t("admin.mediaUpload")}</h3>
      <Input label={t("admin.fields.displayName")} name="display_name" required />
      <label className="field">
        <span>{t("admin.fields.file")}</span>
        <input name="file" required type="file" />
      </label>
      <Input label={t("admin.fields.category")} name="category" />
      <button className="text-button" type="submit"><Image size={16} />{t("app.upload")}</button>
    </form>
  );
}

function MediaForm({ mutate, t }: { mutate: (action: () => Promise<unknown>) => Promise<void>; t: TFn }) {
  return (
    <form className="panel form-panel" onSubmit={(event) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      void mutate(() => apiPost(endpoints.system.mediaAssets, {
        category: String(form.get("category") || ""),
        display_name: String(form.get("display_name")),
        mime_type: String(form.get("mime_type")),
        size_bytes: Number(form.get("size_bytes") || 0),
        storage_key: String(form.get("storage_key"))
      }));
      event.currentTarget.reset();
    }}>
      <h3>{t("admin.mediaRegister")}</h3>
      <Input label={t("admin.fields.displayName")} name="display_name" required />
      <Input label={t("admin.fields.storageKey")} name="storage_key" required />
      <Input label={t("admin.fields.mimeType")} name="mime_type" required />
      <Input label={t("admin.fields.sizeBytes")} name="size_bytes" required type="number" />
      <Input label={t("admin.fields.category")} name="category" />
      <button className="text-button" type="submit"><Image size={16} />{t("app.create")}</button>
    </form>
  );
}

function ProbeForm({ mutate, t }: { mutate: (action: () => Promise<unknown>) => Promise<void>; t: TFn }) {
  return (
    <form className="panel form-panel" onSubmit={(event) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      void mutate(() => apiPost(endpoints.system.trafficProbeTargets, {
        expected_status: Number(form.get("expected_status") || 200),
        name: String(form.get("name")),
        url: String(form.get("url"))
      }));
      event.currentTarget.reset();
    }}>
      <h3>{t("admin.probes")}</h3>
      <Input label={t("admin.fields.name")} name="name" required />
      <Input label={t("admin.fields.url")} name="url" required type="url" />
      <Input label={t("admin.fields.expectedStatus")} name="expected_status" required type="number" />
      <button className="text-button" type="submit"><Radar size={16} />{t("app.create")}</button>
    </form>
  );
}

function JsonUpsertPanel(props: {
  fields: Array<[string, string]>;
  onSubmit: (form: FormData) => void;
  t: TFn;
  title: string;
}) {
  return (
    <form className="panel form-panel" onSubmit={(event) => {
      event.preventDefault();
      props.onSubmit(new FormData(event.currentTarget));
      event.currentTarget.reset();
    }}>
      <h3>{props.title}</h3>
      {props.fields.map(([name, label]) => <Input key={name} label={label} name={name} required={name === "key"} />)}
      <button className="text-button" type="submit"><Settings size={16} />{props.t("app.save")}</button>
    </form>
  );
}

function NamedUpsertPanel(props: {
  codeLabel: string;
  nameLabel: string;
  onSubmit: (form: FormData) => void;
  t: TFn;
  title: string;
  valueLabel?: string;
}) {
  return (
    <form className="panel form-panel" onSubmit={(event) => {
      event.preventDefault();
      props.onSubmit(new FormData(event.currentTarget));
      event.currentTarget.reset();
    }}>
      <h3>{props.title}</h3>
      <Input label={props.codeLabel} name="code" required />
      <Input label={props.nameLabel} name="name" required />
      {props.valueLabel ? <Input label={props.valueLabel} name="value" required /> : null}
      <button className="text-button" type="submit"><Settings size={16} />{props.t("app.save")}</button>
    </form>
  );
}

function Input({
  defaultValue,
  help,
  label,
  name,
  required,
  type = "text"
}: {
  defaultValue?: string | number;
  help?: string;
  label: string;
  name: string;
  required?: boolean;
  type?: string;
}) {
  const helpId = `${name}-help`;
  return (
    <label className="field">
      <span>{label}</span>
      <input aria-describedby={help ? helpId : undefined} defaultValue={defaultValue} name={name} required={required} type={type} />
      {help ? <small id={helpId}>{help}</small> : null}
    </label>
  );
}

function TabButton({ active, icon, label, onClick }: { active: boolean; icon: ReactNode; label: string; onClick: () => void }) {
  return <button className={active ? "active" : ""} onClick={onClick} type="button">{icon}{label}</button>;
}

function StatusPill({ ok, text }: { ok: boolean; text: string }) {
  return <span className={ok ? "status-pill ok" : "status-pill"}>{text}</span>;
}

function DeleteIconButton({ label, onClick }: { label: string; onClick: () => void }) {
  return <button className="icon-button" onClick={onClick} title={label} type="button"><Trash2 size={16} /></button>;
}

function confirmDelete(t: TFn, name: string, action: () => void) {
  if (window.confirm(t("admin.confirmDeleteItem", { name }))) {
    action();
  }
}

function confirmAction(message: string, action: () => void) {
  if (window.confirm(message)) {
    action();
  }
}

function Metric({ icon, label, value }: { icon: ReactNode; label: string; value: ReactNode }) {
  return <div className="metric">{icon}<span>{label}</span><strong>{value}</strong></div>;
}

function MiniList({ empty, items, renderAction }: { empty: string; items: string[]; renderAction?: (index: number) => ReactNode }) {
  if (!items.length) return <p className="muted">{empty}</p>;
  return <ul className="mini-list">{items.map((item, index) => <li key={`${item}-${index}`}><span>{item}</span>{renderAction?.(index)}</li>)}</ul>;
}

function SimpleTable({ columns, rows }: { columns: ReactNode[]; rows: Array<ReactNode[]> }) {
  if (!rows.length) return <p className="muted">-</p>;
  return (
    <div className="table-wrap">
      <table>
        <thead><tr>{columns.map((column, index) => <th key={index}>{column}</th>)}</tr></thead>
        <tbody>{rows.map((row, rowIndex) => <tr key={rowIndex}>{row.map((cell, cellIndex) => <td key={cellIndex}>{cell}</td>)}</tr>)}</tbody>
      </table>
    </div>
  );
}

function KeyValueGrid({ values }: { values: Record<string, unknown> }) {
  return <dl className="kv-grid">{Object.entries(values).map(([key, value]) => <div key={key}><dt>{key}</dt><dd>{formatDisplayValue(value)}</dd></div>)}</dl>;
}

function resourceMetricValues(metrics: NonNullable<ServerStatus["metrics"]>, t: TFn) {
  return {
    [t("admin.cpuUsage")]: formatPercent(metrics.cpu_usage_percent),
    [t("admin.processCpuUsage")]: formatPercent(metrics.process_cpu_usage_percent),
    [t("admin.memoryUsage")]: `${formatBytes(metrics.used_memory_bytes)} / ${formatBytes(metrics.total_memory_bytes)}`,
    [t("admin.memoryAvailable")]: formatBytes(metrics.available_memory_bytes),
    [t("admin.processMemory")]: formatBytes(metrics.process_memory_bytes),
    [t("admin.processVirtualMemory")]: formatBytes(metrics.process_virtual_memory_bytes),
    [t("admin.swapUsage")]: `${formatBytes(metrics.used_swap_bytes)} / ${formatBytes(metrics.total_swap_bytes)}`,
    [t("admin.diskUsage")]: `${formatBytes(metrics.used_disk_bytes)} / ${formatBytes(metrics.total_disk_bytes)}`,
    [t("admin.diskAvailable")]: formatBytes(metrics.available_disk_bytes),
    [t("admin.diskCount")]: metrics.disk_count,
    [t("admin.networkInterfaces")]: metrics.network_interface_count,
    [t("admin.networkReceived")]: formatBytes(metrics.network_received_bytes),
    [t("admin.networkTransmitted")]: formatBytes(metrics.network_transmitted_bytes),
    [t("admin.systemUptime")]: `${metrics.system_uptime_seconds}s`,
    [t("admin.systemBootTime")]: formatUnixSeconds(metrics.system_boot_time_seconds),
    [t("admin.loadAverage")]: `${metrics.load_average_one.toFixed(2)} / ${metrics.load_average_five.toFixed(2)} / ${metrics.load_average_fifteen.toFixed(2)}`,
    [t("admin.metricsSource")]: metrics.source
  };
}

function formatPercent(value: number) {
  return `${value.toFixed(1)}%`;
}

function formatBytes(value: number) {
  if (value <= 0) return "0 B";
  const units = ["B", "KiB", "MiB", "GiB", "TiB"];
  let next = value;
  let unitIndex = 0;
  while (next >= 1024 && unitIndex < units.length - 1) {
    next /= 1024;
    unitIndex += 1;
  }
  return `${next.toFixed(unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
}

function formatUnixSeconds(value: number) {
  if (value <= 0) return "";
  return new Date(value * 1000).toISOString();
}

function formatDisplayValue(value: unknown) {
  if (value === null || value === undefined) return "";
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}

function formPayload(form: FormData, keys: string[]) {
  return Object.fromEntries(keys.map((key) => [key, String(form.get(key) ?? "").trim()]));
}

function selectedFormValues(form: FormData, key: string) {
  return form.getAll(key).map((value) => String(value));
}

async function optional<T>(promise: Promise<T>) {
  try {
    return await promise;
  } catch {
    return null;
  }
}

function parseJsonValue(value: string) {
  try {
    return JSON.parse(value);
  } catch {
    return value;
  }
}

function errorMessage(error: unknown, t: TFn) {
  if (error instanceof ApiError) return `${error.code} · ${error.message}`;
  if (error instanceof Error) return error.message;
  return t("app.error");
}

function viewFromPath(path: string): View {
  if (path === "/" || path.startsWith("/public")) return "public";
  if (path.startsWith("/setup")) return "setup";
  if (path.startsWith("/login")) return "login";
  if (path.startsWith("/account")) return "account";
  return "admin";
}

function pathForView(view: View) {
  return view === "public" ? "/" : view === "admin" ? "/admin" : `/${view}`;
}

function tabTitle(tab: AdminTab, t: TFn) {
  return tab === "overview" ? t("nav.overview") : tab === "iam" ? t("nav.iam") : tab === "system" ? t("nav.system") : t("nav.assets");
}

type TFn = (key: string, params?: Record<string, string | number>) => string;

createRoot(document.getElementById("root") ?? document.body).render(<App />);
