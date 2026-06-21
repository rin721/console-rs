import { zodResolver } from "@hookform/resolvers/zod";
import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { Link } from "react-router";
import { useTranslation } from "react-i18next";

import { Button } from "~/components/aoi/primitives/Button";
import { AoiForm, AoiTextField } from "~/components/aoi/patterns/Form";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import {
  createForgotPasswordSchema,
  type ForgotPasswordFormValues,
} from "~/features/auth/schemas";
import { useDocumentMeta } from "~/hooks/useDocumentMeta";
import { authApi } from "~/lib/api/auth";
import { ApiError } from "~/lib/api/client";
import type { NotificationDelivery } from "~/lib/api/types";

export default function PasswordForgotRoute() {
  const { t } = useTranslation();
  const schema = useMemo(() => createForgotPasswordSchema(t), [t]);
  const [apiError, setApiError] = useState("");
  const [delivery, setDelivery] = useState<NotificationDelivery | null>(null);
  useDocumentMeta("seo.passwordForgot.title", "seo.passwordForgot.description", {
    canonicalPath: "/password/forgot",
    ogDescriptionKey: "seo.passwordForgot.ogDescription",
    ogTitleKey: "seo.passwordForgot.ogTitle",
  });

  const form = useForm<ForgotPasswordFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      email: "",
    },
  });
  const {
    formState: { isSubmitting },
  } = form;

  async function onSubmit(values: ForgotPasswordFormValues) {
    setApiError("");
    setDelivery(null);
    try {
      setDelivery(await authApi.forgotPassword({ email: values.email.trim() }));
    } catch (error) {
      setApiError(error instanceof ApiError ? error.message : t("errors.api.requestFailed"));
    }
  }

  const debugDelivery = delivery?.debug === true;

  return (
    <main className="aoi-auth-page">
      <section className="aoi-auth-card" aria-labelledby="password-forgot-title">
        <h1 id="password-forgot-title">{t("auth.passwordForgot.title")}</h1>
        <p>{t("auth.passwordForgot.description")}</p>
        {apiError ? (
          <StateBlock
            intent="danger"
            title={t("errors.api.requestFailed")}
            description={apiError}
          />
        ) : null}
        {delivery ? (
          <StateBlock
            title={t(
              debugDelivery
                ? "auth.passwordForgot.tokenCreated"
                : "auth.passwordForgot.emailSentIfExists",
            )}
            description={t("auth.passwordForgot.successDescription")}
          />
        ) : null}
        {debugDelivery ? <PasswordDebugDelivery delivery={delivery} /> : null}
        <AoiForm form={form} onSubmit={onSubmit}>
          <AoiTextField<ForgotPasswordFormValues>
            autoComplete="email"
            help={t("forms.auth.email.help")}
            label={t("forms.auth.email.label")}
            name="email"
            placeholder={t("forms.auth.email.placeholder")}
            type="email"
          />
          <Button loading={isSubmitting} type="submit">
            {isSubmitting ? t("loading.submit") : t("auth.passwordForgot.submit")}
          </Button>
        </AoiForm>
        <p className="aoi-auth-links">
          <Link to="/login">{t("auth.links.backToLogin")}</Link>
          <Link to="/password/reset">{t("auth.links.resetPassword")}</Link>
        </p>
      </section>
    </main>
  );
}

function PasswordDebugDelivery({ delivery }: { delivery: NotificationDelivery }) {
  const { t } = useTranslation();

  return (
    <div className="aoi-auth-debug" aria-label={t("auth.passwordForgot.debugTitle")}>
      {delivery.token ? (
        <div>
          <span>{t("auth.passwordForgot.debugTokenLabel")}</span>
          <code>{delivery.token}</code>
        </div>
      ) : null}
      {delivery.url ? (
        <div>
          <span>{t("auth.passwordForgot.debugUrlLabel")}</span>
          <code>{delivery.url}</code>
        </div>
      ) : null}
      {delivery.token ? (
        <Button asChild appearance="secondary">
          <Link to={`/password/reset?token=${encodeURIComponent(delivery.token)}`}>
            {t("auth.passwordForgot.openReset")}
          </Link>
        </Button>
      ) : null}
    </div>
  );
}
