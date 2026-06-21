import { create } from "zustand";

import type { CurrentUser, Organization, SessionSnapshot } from "~/lib/api/types";

type AuthState = {
  accessExpiresAt: string;
  clearSession: () => void;
  clientType: string;
  currentOrgId: string | null;
  currentSessionId: string | null;
  hydrateFromStorage: () => void;
  hydrated: boolean;
  isAuthenticated: boolean;
  orgs: Organization[];
  productCode: string;
  refreshExpiresAt: string;
  setIdentity: (user: CurrentUser | null, orgs: Organization[]) => void;
  setSession: (session: SessionSnapshot | null) => void;
  user: CurrentUser | null;
};

export const useAuthStore = create<AuthState>((set, get) => ({
  accessExpiresAt: "",
  clearSession: () => {
    set({
      accessExpiresAt: "",
      clientType: "",
      currentOrgId: null,
      currentSessionId: null,
      isAuthenticated: false,
      orgs: [],
      productCode: "",
      refreshExpiresAt: "",
      user: null,
    });
  },
  clientType: "",
  currentOrgId: null,
  currentSessionId: null,
  hydrateFromStorage: () => {
    set({ hydrated: true });
  },
  hydrated: false,
  isAuthenticated: false,
  orgs: [],
  productCode: "",
  refreshExpiresAt: "",
  setIdentity: (user, orgs) => {
    set({
      currentOrgId: get().currentOrgId || String(orgs[0]?.id ?? "") || null,
      isAuthenticated: Boolean(get().currentSessionId && user),
      orgs,
      user,
    });
  },
  setSession: (session) => {
    if (!session) {
      get().clearSession();
      return;
    }
    set({
      accessExpiresAt: session.accessExpiresAt ?? "",
      clientType: session.clientType,
      currentOrgId: stringID(session.orgId),
      currentSessionId: stringID(session.sessionId),
      isAuthenticated: Boolean(session.sessionId),
      productCode: session.productCode,
      refreshExpiresAt: session.refreshExpiresAt ?? "",
    });
  },
  user: null,
}));

function stringID(value: number | string | undefined) {
  if (value === undefined || value === null || value === "") {
    return null;
  }
  return String(value);
}
