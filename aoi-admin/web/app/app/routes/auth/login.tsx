import { zodResolver } from "@hookform/resolvers/zod";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { Link, useNavigate, useSearchParams } from "react-router";
import { useTranslation } from "react-i18next";

import { Button } from "~/components/aoi/primitives/Button";
import { AoiForm, AoiTextField } from "~/components/aoi/patterns/Form";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { createLoginSchema, type LoginFormValues } from "~/features/auth/schemas";
import { useDocumentMeta } from "~/hooks/useDocumentMeta";
import { usePublicSettings } from "~/hooks/usePublicSettings";
import { authApi } from "~/lib/api/auth";
import { ApiError } from "~/lib/api/client";
import type { CaptchaChallenge, LoginRequest } from "~/lib/api/types";
import { useAuthStore } from "~/stores/auth-store";

type LoginChallenge = "captcha" | "mfa" | "none";

export default function LoginRoute() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const setSession = useAuthStore((state) => state.setSession);
  const setIdentity = useAuthStore((state) => state.setIdentity);
  const schema = useMemo(() => createLoginSchema(t), [t]);
  const [apiError, setApiError] = useState("");
  const [captcha, setCaptcha] = useState<CaptchaChallenge | null>(null);
  const [captchaLoading, setCaptchaLoading] = useState(false);
  const [challenge, setChallenge] = useState<LoginChallenge>("none");
  const captchaControllerRef = useRef<AbortController | null>(null);
  const publicSettings = usePublicSettings();
  useDocumentMeta("seo.login.title", "seo.login.description", {
    canonicalPath: "/login",
    ogDescriptionKey: "seo.login.ogDescription",
    ogTitleKey: "seo.login.ogTitle",
  });

  const form = useForm<LoginFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      captchaCode: "",
      captchaId: "",
      identifier: "",
      mfaCode: "",
      password: "",
    },
  });
  const {
    formState: { isSubmitting },
  } = form;
  const registrationMode = publicSettings.data?.auth.registrationMode;
  const signupVisible =
    registrationMode === "direct" || registrationMode === "email_verification";

  useEffect(() => {
    return () => captchaControllerRef.current?.abort();
  }, []);

  const loadCaptchaChallenge = useCallback(async () => {
    captchaControllerRef.current?.abort();
    const controller = new AbortController();
    captchaControllerRef.current = controller;
    setCaptchaLoading(true);
    try {
      const nextCaptcha = await authApi.captcha({ signal: controller.signal });
      if (captchaControllerRef.current !== controller) {
        return;
      }
      setCaptcha(nextCaptcha.enabled ? nextCaptcha : null);
      form.setValue("captchaCode", "");
      form.setValue("captchaId", nextCaptcha.captchaId ?? "");
      if (!nextCaptcha.enabled) {
        setChallenge("none");
      }
    } catch (error) {
      if (isAbortError(error)) {
        return;
      }
      setApiError(error instanceof ApiError ? error.message : t("errors.api.requestFailed"));
    } finally {
      if (captchaControllerRef.current === controller) {
        captchaControllerRef.current = null;
        setCaptchaLoading(false);
      }
    }
  }, [form, t]);

  async function onSubmit(values: LoginFormValues) {
    setApiError("");
    const payload = loginPayload(values, challenge, captcha);
    if (!payload) {
      if (challenge === "captcha") {
        form.setError("captchaCode", { message: t("errors.validation.captchaCode") });
      }
      if (challenge === "mfa") {
        form.setError("mfaCode", { message: t("errors.validation.mfaCode") });
      }
      return;
    }

    try {
      const pair = await authApi.login(payload);
      setSession(pair);
      const [user, orgs] = await Promise.all([authApi.getMe(), authApi.listMyOrganizations()]);
      setIdentity(user, orgs);
      await navigate(safeInternalRedirect(searchParams.get("next")));
    } catch (error) {
      if (error instanceof ApiError) {
        const messageKey = apiMessageKey(error);
        setApiError(error.message);
        if (messageKey === "api.auth.captchaRequired" || messageKey === "api.auth.captchaInvalid") {
          setChallenge("captcha");
          await loadCaptchaChallenge();
          return;
        }
        if (messageKey === "api.auth.mfaRequired") {
          setChallenge("mfa");
          form.setValue("mfaCode", "");
          return;
        }
        return;
      }
      setApiError(t("errors.api.requestFailed"));
    }
  }

  return (
    <main className="aoi-auth-page">
      <section className="aoi-auth-card" aria-labelledby="login-title">
        <h1 id="login-title">{t("auth.login.title")}</h1>
        <p>{t("auth.login.description")}</p>
        {apiError ? (
          <StateBlock
            intent="danger"
            title={t("errors.api.requestFailed")}
            description={apiError}
          />
        ) : null}
        <AoiForm form={form} onSubmit={onSubmit}>
          <AoiTextField<LoginFormValues>
            autoComplete="username"
            help={t("forms.auth.identifier.help")}
            label={t("forms.auth.identifier.label")}
            name="identifier"
            placeholder={t("forms.auth.identifier.placeholder")}
          />
          <AoiTextField<LoginFormValues>
            autoComplete="current-password"
            help={t("forms.auth.password.help")}
            label={t("forms.auth.password.label")}
            name="password"
            placeholder={t("forms.auth.password.placeholder")}
            type="password"
          />
          {challenge === "captcha" ? (
            <div className="aoi-auth-challenge">
              <div className="aoi-auth-challenge__header">
                <h2 id="login-captcha-title">{t("auth.login.captchaTitle")}</h2>
                <p>{t("auth.login.captchaDescription")}</p>
              </div>
              <div className="aoi-auth-captcha">
                {captcha?.image ? (
                  <img
                    alt={t("auth.login.captchaImageAlt")}
                    className="aoi-auth-captcha__image"
                    src={captcha.image}
                  />
                ) : null}
                <Button
                  appearance="secondary"
                  loading={captchaLoading}
                  onClick={() => void loadCaptchaChallenge()}
                  type="button"
                >
                  {captchaLoading ? t("loading.submit") : t("auth.login.captchaRefresh")}
                </Button>
              </div>
              <AoiTextField<LoginFormValues>
                autoComplete="off"
                help={t("forms.auth.captchaCode.help")}
                label={t("forms.auth.captchaCode.label")}
                name="captchaCode"
                placeholder={t("forms.auth.captchaCode.placeholder")}
              />
            </div>
          ) : null}
          {challenge === "mfa" ? (
            <div className="aoi-auth-challenge">
              <div className="aoi-auth-challenge__header">
                <h2 id="login-mfa-title">{t("auth.login.mfaTitle")}</h2>
                <p>{t("auth.login.mfaDescription")}</p>
              </div>
              <AoiTextField<LoginFormValues>
                autoComplete="one-time-code"
                help={t("forms.auth.mfaCode.help")}
                inputMode="numeric"
                label={t("forms.auth.mfaCode.label")}
                name="mfaCode"
                placeholder={t("forms.auth.mfaCode.placeholder")}
              />
            </div>
          ) : null}
          <Button loading={isSubmitting} type="submit">
            {isSubmitting ? t("loading.submit") : t("auth.login.submit")}
          </Button>
        </AoiForm>
        <p className="aoi-auth-links">
          {signupVisible ? (
            <span>
              {t("auth.login.signupHint")} <Link to="/signup">{t("common.actions.signup")}</Link>
            </span>
          ) : null}
          <Link to="/password/forgot">{t("auth.links.forgotPassword")}</Link>
          <Link to="/password/reset">{t("auth.links.resetPassword")}</Link>
        </p>
      </section>
    </main>
  );
}

function safeInternalRedirect(value: string | null) {
  if (value?.startsWith("/") && !value.startsWith("//")) {
    return value;
  }
  return "/admin";
}

function loginPayload(
  values: LoginFormValues,
  challenge: LoginChallenge,
  captcha: CaptchaChallenge | null,
): LoginRequest | null {
  const payload: LoginRequest = {
    identifier: values.identifier,
    password: values.password,
  };

  if (challenge === "captcha") {
    const captchaCode = values.captchaCode?.trim();
    const captchaId = captcha?.captchaId || values.captchaId?.trim();
    if (!captchaCode || !captchaId) {
      return null;
    }
    payload.captchaCode = captchaCode;
    payload.captchaId = captchaId;
  }

  if (challenge === "mfa") {
    const mfaCode = values.mfaCode?.trim();
    if (!mfaCode) {
      return null;
    }
    payload.mfaCode = mfaCode;
  }

  return payload;
}

function apiMessageKey(error: ApiError) {
  const payload = error.payload;
  if (payload && typeof payload === "object" && "messageKey" in payload) {
    const messageKey = (payload as { messageKey?: unknown }).messageKey;
    return typeof messageKey === "string" ? messageKey : "";
  }
  return "";
}

function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === "AbortError";
}
