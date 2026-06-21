import type { TFunction } from "i18next";
import { z } from "zod";

export function createLoginSchema(t: TFunction) {
  return z.object({
    captchaCode: z.string().optional(),
    captchaId: z.string().optional(),
    identifier: z.string().min(1, t("errors.validation.identifier")),
    mfaCode: z.string().optional(),
    password: z.string().min(8, t("errors.validation.password")),
  });
}

export function createSignupSchema(t: TFunction) {
  return z.object({
    displayName: z.string().optional(),
    email: z.string().email(t("errors.validation.email")),
    orgCode: z.string().min(1, t("errors.validation.orgCode")),
    orgName: z.string().min(1, t("errors.validation.orgName")),
    password: z.string().min(8, t("errors.validation.password")),
    username: z.string().min(1, t("errors.validation.username")),
  });
}

export function createForgotPasswordSchema(t: TFunction) {
  return z.object({
    email: z.string().email(t("errors.validation.email")),
  });
}

export function createPasswordResetSchema(t: TFunction) {
  return z.object({
    newPassword: z.string().min(8, t("errors.validation.password")),
    token: z.string().min(1, t("errors.validation.resetToken")),
  });
}

export function createInvitationAcceptSchema(t: TFunction) {
  return z.object({
    displayName: z.string().optional(),
    password: z.string().min(8, t("errors.validation.password")),
    username: z.string().min(1, t("errors.validation.username")),
  });
}

export type LoginFormValues = z.infer<ReturnType<typeof createLoginSchema>>;
export type SignupFormValues = z.infer<ReturnType<typeof createSignupSchema>>;
export type ForgotPasswordFormValues = z.infer<ReturnType<typeof createForgotPasswordSchema>>;
export type PasswordResetFormValues = z.infer<ReturnType<typeof createPasswordResetSchema>>;
export type InvitationAcceptFormValues = z.infer<ReturnType<typeof createInvitationAcceptSchema>>;
