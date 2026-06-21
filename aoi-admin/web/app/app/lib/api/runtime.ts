import { useAuthStore } from "~/stores/auth-store";
import { createApiClient } from "./client";

export const apiClient = createApiClient({
  onRefresh: (session) => {
    useAuthStore.getState().setSession(session);
  },
  onUnauthorized: () => {
    useAuthStore.getState().clearSession();
  },
});
