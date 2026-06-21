import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { useTranslation } from "react-i18next";

import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { useDocumentMeta } from "~/hooks/useDocumentMeta";
import { authApi } from "~/lib/api/auth";
import { ApiError } from "~/lib/api/client";
import { useAuthStore } from "~/stores/auth-store";

type VerificationState = "loading" | "success" | "error";

export default function SignupVerifyRoute() {
  const { token = "" } = useParams();
  const { t } = useTranslation();
  const navigate = useNavigate();
  const setSession = useAuthStore((state) => state.setSession);
  const setIdentity = useAuthStore((state) => state.setIdentity);
  const [state, setState] = useState<VerificationState>("loading");
  const [apiError, setApiError] = useState("");
  useDocumentMeta("seo.signupVerify.title", "seo.signupVerify.description", {
    canonicalPath: "/signup/verify",
    ogDescriptionKey: "seo.signupVerify.ogDescription",
    ogTitleKey: "seo.signupVerify.ogTitle",
  });

  useEffect(() => {
    let active = true;

    async function confirmEmail() {
      if (!token) {
        setState("error");
        setApiError(t("auth.signupVerify.missingTokenDescription"));
        return;
      }
      setState("loading");
      setApiError("");
      try {
        const session = await authApi.confirmEmailVerification(token);
        if (!active) {
          return;
        }
        setSession(session);
        const [user, orgs] = await Promise.all([authApi.getMe(), authApi.listMyOrganizations()]);
        if (!active) {
          return;
        }
        setIdentity(user, orgs);
        setState("success");
        await navigate("/admin", { replace: true });
      } catch (error) {
        if (!active) {
          return;
        }
        setState("error");
        setApiError(error instanceof ApiError ? error.message : t("errors.api.requestFailed"));
      }
    }

    void confirmEmail();

    return () => {
      active = false;
    };
  }, [navigate, setIdentity, setSession, t, token]);

  return (
    <main className="aoi-auth-page">
      <section className="aoi-auth-card" aria-labelledby="signup-verify-title">
        <h1 id="signup-verify-title">{t("auth.signupVerify.title")}</h1>
        <p>{t("auth.signupVerify.description")}</p>
        {state === "loading" ? (
          <StateBlock
            title={t("auth.signupVerify.loadingTitle")}
            description={t("auth.signupVerify.loadingDescription")}
          />
        ) : null}
        {state === "success" ? (
          <StateBlock
            title={t("auth.signupVerify.successTitle")}
            description={t("auth.signupVerify.successDescription")}
          />
        ) : null}
        {state === "error" ? (
          <StateBlock
            intent="danger"
            title={t("auth.signupVerify.errorTitle")}
            description={apiError}
          />
        ) : null}
        <p className="aoi-auth-links">
          <Link to="/login">{t("auth.links.backToLogin")}</Link>
        </p>
      </section>
    </main>
  );
}
