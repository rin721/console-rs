import { zodResolver } from "@hookform/resolvers/zod";
import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { Link, useParams } from "react-router";
import { useTranslation } from "react-i18next";

import { Button } from "~/components/aoi/primitives/Button";
import { AoiForm, AoiTextField } from "~/components/aoi/patterns/Form";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import {
  createInvitationAcceptSchema,
  type InvitationAcceptFormValues,
} from "~/features/auth/schemas";
import { useDocumentMeta } from "~/hooks/useDocumentMeta";
import { authApi } from "~/lib/api/auth";
import { ApiError } from "~/lib/api/client";

export default function InvitationRoute() {
  const { token = "" } = useParams();
  const { t } = useTranslation();
  const schema = useMemo(() => createInvitationAcceptSchema(t), [t]);
  const [apiError, setApiError] = useState("");
  const [success, setSuccess] = useState(false);
  useDocumentMeta("seo.invitation.title", "seo.invitation.description", {
    canonicalPath: "/invitations",
    ogDescriptionKey: "seo.invitation.ogDescription",
    ogTitleKey: "seo.invitation.ogTitle",
  });

  const form = useForm<InvitationAcceptFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      displayName: "",
      password: "",
      username: "",
    },
  });
  const {
    formState: { isSubmitting },
    resetField,
  } = form;

  async function onSubmit(values: InvitationAcceptFormValues) {
    setApiError("");
    setSuccess(false);
    try {
      await authApi.acceptInvitation(token, {
        displayName: values.displayName?.trim() || undefined,
        password: values.password,
        username: values.username.trim(),
      });
      resetField("password");
      setSuccess(true);
    } catch (error) {
      setApiError(error instanceof ApiError ? error.message : t("errors.api.requestFailed"));
    }
  }

  return (
    <main className="aoi-auth-page">
      <section className="aoi-auth-card" aria-labelledby="invitation-title">
        <h1 id="invitation-title">{t("auth.invitation.title")}</h1>
        <p>{t("auth.invitation.description")}</p>
        {apiError ? (
          <StateBlock
            intent="danger"
            title={t("errors.api.requestFailed")}
            description={apiError}
          />
        ) : null}
        {!token ? (
          <StateBlock
            intent="danger"
            title={t("auth.invitation.missingTokenTitle")}
            description={t("auth.invitation.missingTokenDescription")}
          />
        ) : null}
        {success ? (
          <StateBlock
            title={t("auth.invitation.accepted")}
            description={t("auth.invitation.successDescription")}
          />
        ) : null}
        <AoiForm form={form} onSubmit={onSubmit}>
          <AoiTextField<InvitationAcceptFormValues>
            autoComplete="username"
            help={t("forms.auth.username.help")}
            label={t("forms.auth.username.label")}
            name="username"
            placeholder={t("forms.auth.username.placeholder")}
          />
          <AoiTextField<InvitationAcceptFormValues>
            autoComplete="name"
            help={t("forms.auth.displayName.help")}
            label={t("forms.auth.displayName.label")}
            name="displayName"
            placeholder={t("forms.auth.displayName.placeholder")}
          />
          <AoiTextField<InvitationAcceptFormValues>
            autoComplete="new-password"
            help={t("forms.auth.password.help")}
            label={t("forms.auth.password.label")}
            name="password"
            placeholder={t("forms.auth.password.placeholder")}
            type="password"
          />
          <Button disabled={!token} loading={isSubmitting} type="submit">
            {isSubmitting ? t("loading.submit") : t("auth.invitation.submit")}
          </Button>
        </AoiForm>
        <p className="aoi-auth-links">
          <Link to="/login">{t("auth.links.backToLogin")}</Link>
        </p>
      </section>
    </main>
  );
}
