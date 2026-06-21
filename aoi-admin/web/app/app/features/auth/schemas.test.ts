import { describe, expect, it } from "vitest";
import type { TFunction } from "i18next";

import {
  createForgotPasswordSchema,
  createInvitationAcceptSchema,
  createLoginSchema,
  createPasswordResetSchema,
} from "./schemas";
import { resources } from "~/i18n/resources";

const t = ((key: string) => key) as TFunction;

describe("auth schemas", () => {
  it("accepts login challenge fields without requiring them by default", () => {
    const schema = createLoginSchema(t);

    expect(schema.safeParse({ identifier: "owner", password: "password123" }).success).toBe(true);
    expect(
      schema.safeParse({
        captchaCode: "A1B2",
        captchaId: "captcha-id",
        identifier: "owner",
        mfaCode: "123456",
        password: "password123",
      }).success,
    ).toBe(true);
    expect(schema.safeParse({ identifier: "", password: "password123" }).success).toBe(false);
  });

  it("validates forgot password email", () => {
    const schema = createForgotPasswordSchema(t);

    expect(schema.safeParse({ email: "owner@example.com" }).success).toBe(true);
    expect(schema.safeParse({ email: "not-email" }).success).toBe(false);
  });

  it("requires reset token and a minimally strong password", () => {
    const schema = createPasswordResetSchema(t);

    expect(schema.safeParse({ newPassword: "password123", token: "reset-token" }).success).toBe(
      true,
    );
    expect(schema.safeParse({ newPassword: "short", token: "reset-token" }).success).toBe(false);
    expect(schema.safeParse({ newPassword: "password123", token: "" }).success).toBe(false);
  });

  it("accepts invitation fields without requiring display name", () => {
    const schema = createInvitationAcceptSchema(t);

    expect(schema.safeParse({ password: "password123", username: "member" }).success).toBe(true);
    expect(schema.safeParse({ password: "short", username: "member" }).success).toBe(false);
    expect(schema.safeParse({ password: "password123", username: "" }).success).toBe(false);
  });

  it("keeps validation resources available in both locales", () => {
    expect(resources.en.errors.validation.captchaCode).toBeTruthy();
    expect(resources.en.errors.validation.mfaCode).toBeTruthy();
    expect(resources.en.errors.validation.resetToken).toBeTruthy();
    expect(resources["zh-CN"].errors.validation.captchaCode).toBeTruthy();
    expect(resources["zh-CN"].errors.validation.mfaCode).toBeTruthy();
    expect(resources["zh-CN"].errors.validation.resetToken).toBeTruthy();
  });
});
