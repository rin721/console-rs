import { useEffect, type ReactNode } from "react";
import { useLocation, useNavigate } from "react-router";

import { useSetupStatusQuery } from "~/features/setup/queries";
import { hasConfirmedSetupLanguage } from "~/features/setup/setup-progress";
import { routeSlugForKey } from "~/features/setup/wizard-helpers";
import type { InitializationStep } from "~/lib/api/setup";
import { useAuthStore } from "~/stores/auth-store";

export function SetupGate({ children }: { children: ReactNode }) {
  const location = useLocation();
  const navigate = useNavigate();
  const hydrated = useAuthStore((state) => state.hydrated);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const statusQuery = useSetupStatusQuery();
  const statusSteps = statusQuery.data?.steps;
  const setupRequired = statusQuery.data?.required;
  const currentStep = statusQuery.data?.currentStep;
  const isSetupRoute = location.pathname === "/setup" || location.pathname.startsWith("/setup/");

  useEffect(() => {
    if (setupRequired === true && !isSetupRoute) {
      void navigate(setupEntryPath(currentStep, statusSteps), { replace: true });
      return;
    }

    if (setupRequired === false && isSetupRoute && hydrated) {
      void navigate(isAuthenticated ? "/admin" : "/login", { replace: true });
    }
  }, [currentStep, hydrated, isAuthenticated, isSetupRoute, navigate, setupRequired, statusSteps]);

  return children;
}

function setupEntryPath(
  currentStep: string | undefined,
  steps: InitializationStep[] | undefined,
) {
  const stepKey = currentStep?.trim();
  if (!stepKey || !hasConfirmedSetupLanguage()) {
    return "/setup";
  }
  const routeSlug =
    steps?.find((step) => step.key === stepKey)?.schema?.routeSlug?.trim() || routeSlugForKey(stepKey);
  return `/setup/${routeSlug}`;
}
