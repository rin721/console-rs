import type { ReactNode } from "react";
import { useEffect } from "react";
import { useLocation, useNavigate } from "react-router";
import { useTranslation } from "react-i18next";

import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { useAuthStore } from "~/stores/auth-store";

export function RequireAuth({ children }: { children: ReactNode }) {
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const hydrated = useAuthStore((state) => state.hydrated);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);

  useEffect(() => {
    if (!hydrated || isAuthenticated) {
      return;
    }
    const next = `${location.pathname}${location.search}`;
    void navigate(`/login?next=${encodeURIComponent(next)}`, { replace: true });
  }, [hydrated, isAuthenticated, location.pathname, location.search, navigate]);

  if (!hydrated) {
    return (
      <StateBlock
        title={t("auth.guard.loadingTitle")}
        description={t("auth.guard.loadingDescription")}
      />
    );
  }

  if (!isAuthenticated) {
    return null;
  }

  return children;
}
