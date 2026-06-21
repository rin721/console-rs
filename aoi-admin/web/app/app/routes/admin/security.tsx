import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Building2,
  CalendarClock,
  Check,
  Clock3,
  Copy,
  Fingerprint,
  LogOut,
  Mail,
  MonitorCheck,
  ShieldCheck,
  ShieldPlus,
  UserRoundCheck,
} from "lucide-react";
import { useEffect, useMemo, useState, type ReactNode } from "react";
import { useForm } from "react-hook-form";
import { Link, useNavigate } from "react-router";
import { useTranslation } from "react-i18next";
import { z } from "zod";

import { AoiForm, AoiTextField } from "~/components/aoi/patterns/Form";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { Badge } from "~/components/aoi/primitives/Badge";
import { Button } from "~/components/aoi/primitives/Button";
import { clientTypeLabel } from "~/features/admin/PlatformTag";
import { authApi } from "~/lib/api/auth";
import { ApiError } from "~/lib/api/client";
import { queryKeys } from "~/lib/api/query-keys";
import type { MFASetupPayload } from "~/lib/api/types";
import { useAuthStore } from "~/stores/auth-store";

type MFAFormValues = {
  code: string;
};

type Notice = {
  description: string;
  intent?: "danger" | "info";
  title: string;
};

type SecurityItem = {
  icon: ReactNode;
  label: string;
  monospace?: boolean;
  value: string;
};

export default function AdminSecurityRoute() {
  const { i18n, t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const orgs = useAuthStore((state) => state.orgs);
  const clientType = useAuthStore((state) => state.clientType);
  const currentOrgId = useAuthStore((state) => state.currentOrgId);
  const currentSessionId = useAuthStore((state) => state.currentSessionId);
  const productCode = useAuthStore((state) => state.productCode);
  const accessExpiresAt = useAuthStore((state) => state.accessExpiresAt);
  const refreshExpiresAt = useAuthStore((state) => state.refreshExpiresAt);
  const clearSession = useAuthStore((state) => state.clearSession);
  const setIdentity = useAuthStore((state) => state.setIdentity);
  const [setupPayload, setSetupPayload] = useState<MFASetupPayload | null>(null);
  const [notice, setNotice] = useState<Notice | null>(null);
  const schema = useMemo(() => createMFASchema(t), [t]);
  const identityQuery = useQuery({
    queryFn: async ({ signal }) => {
      const [nextUser, nextOrganizations] = await Promise.all([
        authApi.getMe({ signal }),
        authApi.listMyOrganizations({ signal }),
      ]);
      return { orgs: nextOrganizations, user: nextUser };
    },
    queryKey: queryKeys.auth.identity,
  });
  const form = useForm<MFAFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      code: "",
    },
  });

  useEffect(() => {
    if (identityQuery.data) {
      setIdentity(identityQuery.data.user, identityQuery.data.orgs);
    }
  }, [identityQuery.data, setIdentity]);

  const currentOrganization = useMemo(
    () => orgs.find((organization) => sameID(organization.id, currentOrgId)) ?? orgs[0],
    [currentOrgId, orgs],
  );
  const accountName = user?.displayName || user?.username || t("common.labels.none");
  const mfaEnabled = Boolean(user?.mfaEnabled);
  const accountItems = useMemo<SecurityItem[]>(
    () => [
      {
        icon: <UserRoundCheck size={18} />,
        label: t("admin.security.fields.account"),
        value: accountName,
      },
      {
        icon: <Mail size={18} />,
        label: t("admin.security.fields.email"),
        value: user?.email || t("common.labels.none"),
      },
      {
        icon: <Building2 size={18} />,
        label: t("admin.security.fields.organization"),
        value: currentOrganization?.name || t("common.labels.none"),
      },
      {
        icon: <Fingerprint size={18} />,
        label: t("admin.security.fields.session"),
        monospace: true,
        value: currentSessionId || t("common.labels.none"),
      },
      {
        icon: <MonitorCheck size={18} />,
        label: t("admin.security.fields.platform"),
        value:
          clientType || productCode
            ? t("admin.security.values.platform", {
                platform: clientTypeLabel(clientType, t),
                product: productCode || t("common.labels.none"),
              })
            : t("common.labels.none"),
      },
      {
        icon: <Clock3 size={18} />,
        label: t("admin.security.fields.accessExpires"),
        value: formatDate(accessExpiresAt, i18n.language, t),
      },
      {
        icon: <CalendarClock size={18} />,
        label: t("admin.security.fields.refreshExpires"),
        value: formatDate(refreshExpiresAt, i18n.language, t),
      },
    ],
    [
      accessExpiresAt,
      accountName,
      clientType,
      currentOrganization?.name,
      currentSessionId,
      i18n.language,
      productCode,
      refreshExpiresAt,
      t,
      user?.email,
    ],
  );

  const setupMFAMutation = useMutation({
    mutationFn: authApi.setupMFA,
    onError: (error) => {
      setSetupPayload(null);
      setNotice({
        description: errorDescription(error, t),
        intent: "danger",
        title: t("admin.security.messages.setupFailedTitle"),
      });
    },
    onMutate: () => {
      setNotice(null);
      setSetupPayload(null);
    },
    onSuccess: (payload) => {
      setSetupPayload(payload);
      form.reset({ code: "" });
      setNotice({
        description: t("admin.security.messages.secretGeneratedDescription"),
        title: t("admin.security.messages.secretGeneratedTitle"),
      });
    },
  });

  const verifyMFAMutation = useMutation({
    mutationFn: (code: string) => authApi.verifyMFA(code),
    onError: (error) => {
      setNotice({
        description: errorDescription(error, t),
        intent: "danger",
        title: t("admin.security.messages.verifyFailedTitle"),
      });
    },
    onMutate: () => {
      setNotice(null);
    },
    onSuccess: async () => {
      const [nextUser, nextOrganizations] = await Promise.all([
        authApi.getMe(),
        authApi.listMyOrganizations(),
      ]);
      setIdentity(nextUser, nextOrganizations);
      queryClient.setQueryData(queryKeys.auth.identity, {
        orgs: nextOrganizations,
        user: nextUser,
      });
      queryClient.setQueryData(queryKeys.auth.me, nextUser);
      queryClient.setQueryData(queryKeys.auth.organizations, nextOrganizations);
      setSetupPayload(null);
      form.reset({ code: "" });
      setNotice({
        description: t("admin.security.messages.enabledDescription"),
        title: t("admin.security.messages.enabledTitle"),
      });
    },
  });

  const logoutMutation = useMutation({
    mutationFn: authApi.logout,
    onSettled: () => {
      clearSession();
      void navigate("/login");
    },
  });

  async function copyMFAValue(value: string, label: string) {
    if (!value) {
      return;
    }
    try {
      if (!navigator.clipboard?.writeText) {
        throw new Error("clipboard unsupported");
      }
      await navigator.clipboard.writeText(value);
      setNotice({
        description: t("admin.security.messages.copiedDescription", { label }),
        title: t("admin.security.messages.copiedTitle"),
      });
    } catch {
      setNotice({
        description: t("admin.security.messages.copyDeniedDescription", { label }),
        intent: "danger",
        title: t("admin.security.messages.copyDeniedTitle"),
      });
    }
  }

  function submitMFA(values: MFAFormValues) {
    verifyMFAMutation.mutate(values.code.trim());
  }

  return (
    <section className="aoi-admin-dashboard" aria-labelledby="admin-security-title">
      <div className="aoi-admin-page-header">
        <div>
          <Badge>{t("admin.security.badge")}</Badge>
          <h1 id="admin-security-title">{t("admin.security.title")}</h1>
          <p>{t("admin.security.description")}</p>
        </div>
        <Button asChild appearance="secondary">
          <Link to="/admin/sessions">
            <MonitorCheck aria-hidden="true" size={17} />
            <span>{t("admin.security.actions.manageSessions")}</span>
          </Link>
        </Button>
      </div>

      {notice ? (
        <StateBlock description={notice.description} intent={notice.intent} title={notice.title} />
      ) : null}

      {identityQuery.isLoading ? (
        <StateBlock
          description={t("admin.security.states.identityLoadingDescription")}
          title={t("admin.security.states.identityLoadingTitle")}
        />
      ) : null}

      {identityQuery.error ? (
        <StateBlock
          description={errorDescription(identityQuery.error, t)}
          intent="danger"
          title={t("admin.security.states.identityFailedTitle")}
        />
      ) : null}

      <div className="aoi-security-grid">
        <section className="aoi-admin-panel">
          <header className="aoi-security-panel-header">
            <div>
              <h2>{t("admin.security.accountTitle")}</h2>
              <p>{user?.username || t("common.labels.none")}</p>
            </div>
            <span className="aoi-iam-status" data-status={mfaEnabled ? "active" : "pending"}>
              {t(
                mfaEnabled ? "admin.security.mfa.enabledBadge" : "admin.security.mfa.disabledBadge",
              )}
            </span>
          </header>
          <SecurityKeyValueList
            ariaLabel={t("admin.security.accountSummaryLabel")}
            items={accountItems}
          />
          <div className="aoi-security-actions">
            <Button asChild appearance="secondary">
              <Link to="/admin/sessions">
                <MonitorCheck aria-hidden="true" size={17} />
                <span>{t("admin.security.actions.viewSessions")}</span>
              </Link>
            </Button>
            <Button
              appearance="secondary"
              icon={<LogOut size={17} />}
              loading={logoutMutation.isPending}
              onClick={() => logoutMutation.mutate()}
            >
              {t("admin.security.actions.logout")}
            </Button>
          </div>
        </section>

        <section className="aoi-admin-panel">
          <header className="aoi-security-panel-header">
            <div>
              <h2>{t("admin.security.mfa.title")}</h2>
              <p>
                {t(
                  mfaEnabled
                    ? "admin.security.mfa.enabledDescription"
                    : "admin.security.mfa.disabledDescription",
                )}
              </p>
            </div>
            <ShieldCheck aria-hidden="true" size={22} />
          </header>

          <Button
            appearance="secondary"
            icon={<ShieldPlus size={17} />}
            loading={setupMFAMutation.isPending}
            onClick={() => setupMFAMutation.mutate()}
          >
            {t(
              mfaEnabled
                ? "admin.security.actions.rotateSecret"
                : "admin.security.actions.generateSecret",
            )}
          </Button>

          {setupPayload ? (
            <div className="aoi-mfa-setup">
              <ReadonlySecretField
                copyLabel={t("admin.security.actions.copyUrl")}
                id="mfa-otpauth-url"
                label={t("admin.security.mfa.otpauthLabel")}
                multiline
                value={setupPayload.otpauthUrl}
                onCopy={() =>
                  void copyMFAValue(setupPayload.otpauthUrl, t("admin.security.mfa.otpauthLabel"))
                }
              />
              <ReadonlySecretField
                copyLabel={t("admin.security.actions.copySecret")}
                id="mfa-secret"
                label={t("admin.security.mfa.secretLabel")}
                value={setupPayload.secret}
                onCopy={() =>
                  void copyMFAValue(setupPayload.secret, t("admin.security.mfa.secretLabel"))
                }
              />
              <AoiForm form={form} onSubmit={submitMFA}>
                <AoiTextField<MFAFormValues>
                  autoComplete="one-time-code"
                  help={t("admin.security.fields.mfaCodeHelp")}
                  inputMode="numeric"
                  label={t("admin.security.fields.mfaCode")}
                  maxLength={6}
                  name="code"
                  pattern="[0-9]*"
                  placeholder={t("admin.security.fields.mfaCodePlaceholder")}
                />
                <Button
                  icon={<Check size={17} />}
                  loading={verifyMFAMutation.isPending}
                  type="submit"
                >
                  {t("admin.security.actions.verifyAndEnable")}
                </Button>
              </AoiForm>
            </div>
          ) : (
            <StateBlock
              description={t("admin.security.mfa.setupEmptyDescription")}
              title={t("admin.security.mfa.setupEmptyTitle")}
            />
          )}
        </section>
      </div>
    </section>
  );
}

function createMFASchema(t: ReturnType<typeof useTranslation>["t"]) {
  return z.object({
    code: z
      .string()
      .trim()
      .min(1, t("admin.security.validation.mfaCodeRequired"))
      .regex(/^\d{6}$/, t("admin.security.validation.mfaCodeFormat")),
  });
}

function SecurityKeyValueList({ ariaLabel, items }: { ariaLabel: string; items: SecurityItem[] }) {
  return (
    <dl className="aoi-security-key-values" aria-label={ariaLabel}>
      {items.map((item) => (
        <div key={item.label}>
          <dt>
            <span aria-hidden="true">{item.icon}</span>
            {item.label}
          </dt>
          <dd>{item.monospace ? <code>{item.value}</code> : <span>{item.value}</span>}</dd>
        </div>
      ))}
    </dl>
  );
}

function ReadonlySecretField({
  copyLabel,
  id,
  label,
  multiline,
  onCopy,
  value,
}: {
  copyLabel: string;
  id: string;
  label: string;
  multiline?: boolean;
  onCopy: () => void;
  value: string;
}) {
  return (
    <div className="aoi-security-secret-field">
      <div className="aoi-form-field">
        <label htmlFor={id}>{label}</label>
        {multiline ? (
          <textarea id={id} aria-readonly="true" readOnly spellCheck={false} value={value} />
        ) : (
          <input id={id} aria-readonly="true" readOnly spellCheck={false} value={value} />
        )}
      </div>
      <Button appearance="secondary" icon={<Copy size={17} />} onClick={onCopy}>
        {copyLabel}
      </Button>
    </div>
  );
}

function formatDate(
  value: string | null | undefined,
  locale: string,
  t: ReturnType<typeof useTranslation>["t"],
) {
  if (!value) {
    return t("common.labels.none");
  }
  const timestamp = Date.parse(value);
  if (Number.isNaN(timestamp)) {
    return value;
  }
  return new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(timestamp);
}

function sameID(
  left: number | string | null | undefined,
  right: number | string | null | undefined,
) {
  if (left === null || left === undefined || right === null || right === undefined) {
    return false;
  }
  return String(left) === String(right);
}

function errorDescription(error: unknown, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError) {
    return error.message;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return t("errors.api.requestFailed");
}
