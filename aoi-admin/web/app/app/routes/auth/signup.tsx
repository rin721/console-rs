import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { Link, useNavigate } from "react-router";
import { useTranslation } from "react-i18next";

import { Button } from "~/components/aoi/primitives/Button";
import { AoiForm, AoiTextField } from "~/components/aoi/patterns/Form";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { createSignupSchema, type SignupFormValues } from "~/features/auth/schemas";
import { useDocumentMeta } from "~/hooks/useDocumentMeta";
import { usePublicSettings } from "~/hooks/usePublicSettings";
import { authApi } from "~/lib/api/auth";
import { ApiError } from "~/lib/api/client";
import type { NotificationDelivery } from "~/lib/api/types";
import { useAuthStore } from "~/stores/auth-store";

export default function SignupRoute() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const setSession = useAuthStore((state) => state.setSession);
  const setIdentity = useAuthStore((state) => state.setIdentity);
  const schema = useMemo(() => createSignupSchema(t), [t]);
  const [apiError, setApiError] = useState("");
  const [delivery, setDelivery] = useState<NotificationDelivery | null>(null);
  const [verificationPending, setVerificationPending] = useState(false);
  const publicSettings = usePublicSettings();
  useDocumentMeta("seo.signup.title", "seo.signup.description", {
    canonicalPath: "/signup",
    ogDescriptionKey: "seo.signup.ogDescription",
    ogTitleKey: "seo.signup.ogTitle",
  });

  const form = useForm<SignupFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      displayName: "",
      email: "",
      orgCode: "",
      orgName: "",
      password: "",
      username: "",
    },
  });
  const {
    formState: { isSubmitting },
  } = form;
  const registrationMode = publicSettings.data?.auth.registrationMode ?? "disabled";
  const signupAvailable =
    registrationMode === "direct" || registrationMode === "email_verification";

  useEffect(() => {
    if (!publicSettings.isLoading && !publicSettings.isError && !signupAvailable) {
      void navigate("/login", { replace: true });
    }
  }, [navigate, publicSettings.isError, publicSettings.isLoading, signupAvailable]);

  async function onSubmit(values: SignupFormValues) {
    setApiError("");
    setDelivery(null);
    setVerificationPending(false);
    try {
      const signup = await authApi.signup(values);
      if (signup.status === "verification_pending") {
        setDelivery(signup.delivery ?? null);
        setVerificationPending(true);
        form.reset();
        scrollAuthPageTop();
        return;
      }
      if (signup.status !== "authenticated" || !signup.session) {
        setApiError(t("errors.api.requestFailed"));
        return;
      }
      setSession(signup.session);
      const [user, orgs] = await Promise.all([authApi.getMe(), authApi.listMyOrganizations()]);
      setIdentity(user, orgs);
      await navigate("/admin");
    } catch (error) {
      setApiError(error instanceof ApiError ? error.message : t("errors.api.requestFailed"));
    }
  }

  if (publicSettings.isLoading) {
    return (
      <main className="aoi-auth-page">
        <section className="aoi-auth-card" aria-labelledby="signup-loading-title">
          <StateBlock
            title={t("auth.signup.loadingTitle")}
            description={t("auth.signup.loadingDescription")}
          />
        </section>
      </main>
    );
  }

  if (publicSettings.isError) {
    return (
      <main className="aoi-auth-page">
        <section className="aoi-auth-card" aria-labelledby="signup-error-title">
          <StateBlock
            intent="danger"
            title={t("errors.api.requestFailed")}
            description={
              publicSettings.error instanceof ApiError
                ? publicSettings.error.message
                : t("errors.api.requestFailed")
            }
          />
          <p className="aoi-auth-links">
            <Link to="/login">{t("auth.links.backToLogin")}</Link>
          </p>
        </section>
      </main>
    );
  }

  if (!signupAvailable) {
    return null;
  }

  return (
    <main className="aoi-auth-page">
      <section className="aoi-auth-card" aria-labelledby="signup-title">
        <h1 id="signup-title">{t("auth.signup.title")}</h1>
        <p>{t("auth.signup.description")}</p>
        {apiError ? (
          <StateBlock
            intent="danger"
            title={t("errors.api.requestFailed")}
            description={apiError}
          />
        ) : null}
        {verificationPending ? (
          <>
            <StateBlock
              title={t("auth.signup.verificationPendingTitle")}
              description={t("auth.signup.verificationPendingDescription")}
            />
            {delivery?.debug ? <SignupDebugDelivery delivery={delivery} /> : null}
          </>
        ) : (
          <AoiForm form={form} onSubmit={onSubmit}>
            <AoiTextField<SignupFormValues>
              autoComplete="email"
              help={t("forms.auth.email.help")}
              label={t("forms.auth.email.label")}
              name="email"
              placeholder={t("forms.auth.email.placeholder")}
              type="email"
            />
            <AoiTextField<SignupFormValues>
              autoComplete="username"
              help={t("forms.auth.username.help")}
              label={t("forms.auth.username.label")}
              name="username"
              placeholder={t("forms.auth.username.placeholder")}
            />
            <AoiTextField<SignupFormValues>
              autoComplete="name"
              help={t("forms.auth.displayName.help")}
              label={t("forms.auth.displayName.label")}
              name="displayName"
              placeholder={t("forms.auth.displayName.placeholder")}
            />
            <AoiTextField<SignupFormValues>
              help={t("forms.auth.orgCode.help")}
              label={t("forms.auth.orgCode.label")}
              name="orgCode"
              placeholder={t("forms.auth.orgCode.placeholder")}
            />
            <AoiTextField<SignupFormValues>
              help={t("forms.auth.orgName.help")}
              label={t("forms.auth.orgName.label")}
              name="orgName"
              placeholder={t("forms.auth.orgName.placeholder")}
            />
            <AoiTextField<SignupFormValues>
              autoComplete="new-password"
              help={t("forms.auth.password.help")}
              label={t("forms.auth.password.label")}
              name="password"
              placeholder={t("forms.auth.password.placeholder")}
              type="password"
            />
            <Button loading={isSubmitting} type="submit">
              {isSubmitting ? t("loading.submit") : t("auth.signup.submit")}
            </Button>
          </AoiForm>
        )}
        <p>
          {t("auth.signup.loginHint")} <Link to="/login">{t("common.actions.login")}</Link>
        </p>
      </section>
    </main>
  );
}

function scrollAuthPageTop() {
  if (typeof window === "undefined") {
    return;
  }
  const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  window.requestAnimationFrame(() => {
    window.scrollTo({ behavior: reduceMotion ? "auto" : "smooth", top: 0 });
  });
}

function SignupDebugDelivery({ delivery }: { delivery: NotificationDelivery }) {
  const { t } = useTranslation();

  return (
    <div className="aoi-auth-debug" aria-label={t("auth.signup.debugTitle")}>
      {delivery.token ? (
        <div>
          <span>{t("auth.signup.debugTokenLabel")}</span>
          <code>{delivery.token}</code>
        </div>
      ) : null}
      {delivery.url ? (
        <div>
          <span>{t("auth.signup.debugUrlLabel")}</span>
          <code>{delivery.url}</code>
        </div>
      ) : null}
      {delivery.token ? (
        <Button asChild appearance="secondary">
          <Link to={`/signup/verify/${encodeURIComponent(delivery.token)}`}>
            {t("auth.signup.openVerification")}
          </Link>
        </Button>
      ) : null}
    </div>
  );
}
