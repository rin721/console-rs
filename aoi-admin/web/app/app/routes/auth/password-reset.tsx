import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { Link, useSearchParams } from "react-router";
import { useTranslation } from "react-i18next";

import { Button } from "~/components/aoi/primitives/Button";
import { AoiForm, AoiTextField } from "~/components/aoi/patterns/Form";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import {
  createPasswordResetSchema,
  type PasswordResetFormValues,
} from "~/features/auth/schemas";
import { useDocumentMeta } from "~/hooks/useDocumentMeta";
import { authApi } from "~/lib/api/auth";
import { ApiError } from "~/lib/api/client";

export default function PasswordResetRoute() {
  const { t } = useTranslation();
  const [searchParams] = useSearchParams();
  const schema = useMemo(() => createPasswordResetSchema(t), [t]);
  const [apiError, setApiError] = useState("");
  const [success, setSuccess] = useState(false);
  useDocumentMeta("seo.passwordReset.title", "seo.passwordReset.description", {
    canonicalPath: "/password/reset",
    ogDescriptionKey: "seo.passwordReset.ogDescription",
    ogTitleKey: "seo.passwordReset.ogTitle",
  });

  const form = useForm<PasswordResetFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      newPassword: "",
      token: searchParams.get("token") ?? "",
    },
  });
  const {
    formState: { isSubmitting },
    resetField,
    setValue,
  } = form;

  useEffect(() => {
    const token = searchParams.get("token");
    if (token) {
      setValue("token", token);
    }
  }, [searchParams, setValue]);

  async function onSubmit(values: PasswordResetFormValues) {
    setApiError("");
    setSuccess(false);
    try {
      await authApi.resetPassword({
        newPassword: values.newPassword,
        token: values.token.trim(),
      });
      resetField("newPassword");
      setSuccess(true);
    } catch (error) {
      setApiError(error instanceof ApiError ? error.message : t("errors.api.requestFailed"));
    }
  }

  return (
    <main className="aoi-auth-page">
      <section className="aoi-auth-card" aria-labelledby="password-reset-title">
        <h1 id="password-reset-title">{t("auth.passwordReset.title")}</h1>
        <p>{t("auth.passwordReset.description")}</p>
        {apiError ? (
          <StateBlock
            intent="danger"
            title={t("errors.api.requestFailed")}
            description={apiError}
          />
        ) : null}
        {success ? (
          <StateBlock
            title={t("auth.passwordReset.resetComplete")}
            description={t("auth.passwordReset.successDescription")}
          />
        ) : null}
        <AoiForm form={form} onSubmit={onSubmit}>
          <AoiTextField<PasswordResetFormValues>
            autoComplete="off"
            help={t("forms.auth.resetToken.help")}
            label={t("forms.auth.resetToken.label")}
            name="token"
            placeholder={t("forms.auth.resetToken.placeholder")}
          />
          <AoiTextField<PasswordResetFormValues>
            autoComplete="new-password"
            help={t("forms.auth.newPassword.help")}
            label={t("forms.auth.newPassword.label")}
            name="newPassword"
            placeholder={t("forms.auth.newPassword.placeholder")}
            type="password"
          />
          <Button loading={isSubmitting} type="submit">
            {isSubmitting ? t("loading.submit") : t("auth.passwordReset.submit")}
          </Button>
        </AoiForm>
        <p className="aoi-auth-links">
          <Link to="/login">{t("auth.links.backToLogin")}</Link>
          <Link to="/password/forgot">{t("auth.links.forgotPassword")}</Link>
        </p>
      </section>
    </main>
  );
}
