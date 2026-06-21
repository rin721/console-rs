import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { I18nextProvider } from "react-i18next";
import { MemoryRouter } from "react-router";
import type { ReactElement } from "react";

import { i18n } from "~/i18n/i18n";
import { resources } from "~/i18n/resources";
import { useAuthStore } from "~/stores/auth-store";
import SignupRoute from "./signup";
import SignupVerifyRoute from "./signup-verify";

const authApiMock = vi.hoisted(() => ({
  confirmEmailVerification: vi.fn(),
  getMe: vi.fn(),
  listMyOrganizations: vi.fn(),
  signup: vi.fn(),
}));

const publicSettingsMock = vi.hoisted(() => vi.fn());
const navigateMock = vi.hoisted(() => vi.fn());
const paramsMock = vi.hoisted(() => ({ token: "verify-token" }));

vi.mock("~/lib/api/auth", () => ({
  authApi: authApiMock,
}));

vi.mock("~/hooks/usePublicSettings", () => ({
  usePublicSettings: publicSettingsMock,
}));

vi.mock("react-router", async () => {
  const actual = await vi.importActual<typeof import("react-router")>("react-router");
  return {
    ...actual,
    useNavigate: () => navigateMock,
    useParams: () => paramsMock,
  };
});

const en = resources.en;

function publicSettings(registrationMode: string) {
  return {
    data: { auth: { registrationMode } },
    error: null,
    isError: false,
    isLoading: false,
  };
}

function renderAuthRoute(route: ReactElement) {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter>{route}</MemoryRouter>
    </I18nextProvider>,
  );
}

function fillSignupForm() {
  fireEvent.change(screen.getByLabelText(en.forms.auth.email.label), {
    target: { value: "owner@example.com" },
  });
  fireEvent.change(screen.getByLabelText(en.forms.auth.username.label), {
    target: { value: "owner" },
  });
  fireEvent.change(screen.getByLabelText(en.forms.auth.orgCode.label), {
    target: { value: "acme" },
  });
  fireEvent.change(screen.getByLabelText(en.forms.auth.orgName.label), {
    target: { value: "Acme" },
  });
  fireEvent.change(screen.getByLabelText(en.forms.auth.password.label), {
    target: { value: "password123" },
  });
}

describe("SignupRoute", () => {
  beforeEach(async () => {
    navigateMock.mockReset();
    authApiMock.confirmEmailVerification.mockReset();
    authApiMock.getMe.mockReset();
    authApiMock.listMyOrganizations.mockReset();
    authApiMock.signup.mockReset();
    publicSettingsMock.mockReset();
    useAuthStore.getState().clearSession();
    await i18n.changeLanguage("en");
  });

  it("authenticates direct signup responses", async () => {
    publicSettingsMock.mockReturnValue(publicSettings("direct"));
    authApiMock.signup.mockResolvedValue({
      session: {
        clientType: "pc_web",
        orgId: 1,
        productCode: "aoi-admin",
        sessionId: "session-1",
        userId: 1,
      },
      status: "authenticated",
    });
    authApiMock.getMe.mockResolvedValue({
      email: "owner@example.com",
      id: 1,
      username: "owner",
    });
    authApiMock.listMyOrganizations.mockResolvedValue([{ code: "acme", id: 1, name: "Acme" }]);

    renderAuthRoute(<SignupRoute />);
    fillSignupForm();
    fireEvent.click(screen.getByRole("button", { name: en.auth.signup.submit }));

    await waitFor(() => expect(authApiMock.signup).toHaveBeenCalled());
    expect(useAuthStore.getState().isAuthenticated).toBe(true);
    expect(navigateMock).toHaveBeenCalledWith("/admin");
  });

  it("shows pending state for email verification signup", async () => {
    publicSettingsMock.mockReturnValue(publicSettings("email_verification"));
    authApiMock.signup.mockResolvedValue({
      delivery: { debug: true, token: "verify-token", url: "/signup/verify/verify-token" },
      status: "verification_pending",
    });

    renderAuthRoute(<SignupRoute />);
    fillSignupForm();
    fireEvent.click(screen.getByRole("button", { name: en.auth.signup.submit }));

    expect(await screen.findByText(en.auth.signup.verificationPendingTitle)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: en.auth.signup.openVerification })).toHaveAttribute(
      "href",
      "/signup/verify/verify-token",
    );
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
  });

  it("redirects disabled signup mode to login", async () => {
    publicSettingsMock.mockReturnValue(publicSettings("disabled"));

    renderAuthRoute(<SignupRoute />);

    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith("/login", { replace: true }));
  });
});

describe("SignupVerifyRoute", () => {
  beforeEach(async () => {
    navigateMock.mockReset();
    authApiMock.confirmEmailVerification.mockReset();
    authApiMock.getMe.mockReset();
    authApiMock.listMyOrganizations.mockReset();
    paramsMock.token = "verify-token";
    useAuthStore.getState().clearSession();
    await i18n.changeLanguage("en");
  });

  it("confirms token and opens the admin console", async () => {
    authApiMock.confirmEmailVerification.mockResolvedValue({
      clientType: "pc_web",
      orgId: 1,
      productCode: "aoi-admin",
      sessionId: "session-1",
      userId: 1,
    });
    authApiMock.getMe.mockResolvedValue({
      email: "owner@example.com",
      id: 1,
      username: "owner",
    });
    authApiMock.listMyOrganizations.mockResolvedValue([{ code: "acme", id: 1, name: "Acme" }]);

    renderAuthRoute(<SignupVerifyRoute />);

    await waitFor(() =>
      expect(authApiMock.confirmEmailVerification).toHaveBeenCalledWith("verify-token"),
    );
    expect(useAuthStore.getState().isAuthenticated).toBe(true);
    expect(navigateMock).toHaveBeenCalledWith("/admin", { replace: true });
  });
});
