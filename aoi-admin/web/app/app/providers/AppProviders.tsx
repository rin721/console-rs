import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect, useState, type ReactNode } from "react";
import { I18nextProvider } from "react-i18next";

import { TooltipProvider } from "~/components/aoi/primitives/Tooltip";
import { i18n } from "~/i18n/i18n";
import {
  detectPreferredLocale,
  isSupportedLocale,
  loadStoredLocale,
  persistLocale,
} from "~/i18n/locales";
import { SetupGate } from "~/features/setup/SetupGate";
import { applyResolvedThemeMode } from "~/features/preferences/theme";
import { authApi } from "~/lib/api/auth";
import { useAuthStore } from "~/stores/auth-store";
import { usePreferencesStore } from "~/stores/preferences-store";

export function AppProviders({ children }: { children: ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            refetchOnWindowFocus: false,
            retry: (failureCount, error) => {
              if (error && typeof error === "object" && "status" in error) {
                const status = Number(error.status);
                return status >= 500 && failureCount < 2;
              }
              return failureCount < 2;
            },
          },
        },
      }),
  );

  return (
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={queryClient}>
        <TooltipProvider delayDuration={320} skipDelayDuration={100}>
          <ClientBootstrap />
          <SetupGate>{children}</SetupGate>
        </TooltipProvider>
      </QueryClientProvider>
    </I18nextProvider>
  );
}

function ClientBootstrap() {
  const clearSession = useAuthStore((state) => state.clearSession);
  const markSessionHydrated = useAuthStore((state) => state.hydrateFromStorage);
  const setIdentity = useAuthStore((state) => state.setIdentity);
  const setSession = useAuthStore((state) => state.setSession);
  const hydrateThemeMode = usePreferencesStore((state) => state.hydrateThemeMode);
  const resolvedThemeMode = usePreferencesStore((state) => state.resolvedThemeMode);
  const setSystemPrefersDark = usePreferencesStore((state) => state.setSystemPrefersDark);

  useEffect(() => {
    hydrateThemeMode();

    const preferredLocale = loadStoredLocale() ?? detectPreferredLocale();
    if (isSupportedLocale(preferredLocale) && preferredLocale !== i18n.language) {
      void i18n.changeLanguage(preferredLocale);
    }

    document.documentElement.lang = i18n.language;

    const handleLanguageChange = (locale: string) => {
      if (isSupportedLocale(locale)) {
        persistLocale(locale);
        document.documentElement.lang = locale;
      }
    };

    i18n.on("languageChanged", handleLanguageChange);
    return () => {
      i18n.off("languageChanged", handleLanguageChange);
    };
  }, [hydrateThemeMode]);

  useEffect(() => {
    const controller = new AbortController();
    let active = true;

    void authApi
      .session({ signal: controller.signal })
      .then(async (session) => {
        if (!active) {
          return;
        }
        setSession(session);
        const [user, orgs] = await Promise.all([
          authApi.getMe({ signal: controller.signal }),
          authApi.listMyOrganizations({ signal: controller.signal }),
        ]);
        if (active) {
          setIdentity(user, orgs);
        }
      })
      .catch((error) => {
        if (active && !isAbortError(error)) {
          clearSession();
        }
      })
      .finally(() => {
        if (active) {
          markSessionHydrated();
        }
      });

    return () => {
      active = false;
      controller.abort();
    };
  }, [clearSession, markSessionHydrated, setIdentity, setSession]);

  useEffect(() => {
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
      return;
    }

    const query = window.matchMedia("(prefers-color-scheme: dark)");
    setSystemPrefersDark(query.matches);

    const handleChange = (event: MediaQueryListEvent) => {
      setSystemPrefersDark(event.matches);
    };

    query.addEventListener("change", handleChange);
    return () => query.removeEventListener("change", handleChange);
  }, [setSystemPrefersDark]);

  useEffect(() => {
    applyResolvedThemeMode(document.documentElement, resolvedThemeMode);
  }, [resolvedThemeMode]);

  return null;
}

function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === "AbortError";
}
