package config

import (
	"strings"
	"testing"
)

func TestAuthConfigValidateRequiresSMTPSecurity(t *testing.T) {
	cfg := validSMTPAuthConfig()
	cfg.SMTP.Security = ""

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "smtp security is required") {
		t.Fatalf("Validate() error = %v, want missing smtp security", err)
	}
}

func TestAuthConfigValidateRejectsUnknownSMTPSecurity(t *testing.T) {
	cfg := validSMTPAuthConfig()
	cfg.SMTP.Security = "ssl"

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "smtp security must be one of none, starttls, tls") {
		t.Fatalf("Validate() error = %v, want invalid smtp security", err)
	}
}

func TestAuthConfigValidateRejectsUnknownRegistrationMode(t *testing.T) {
	cfg := validSMTPAuthConfig()
	cfg.RegistrationMode = "open"

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "registration_mode must be one of") {
		t.Fatalf("Validate() error = %v, want invalid registration mode", err)
	}
}

func validSMTPAuthConfig() AuthConfig {
	return AuthConfig{
		Enabled:                     true,
		RegistrationMode:            RegistrationModeEmailVerification,
		Issuer:                      "aoi-admin",
		Audience:                    []string{"aoi-admin-api"},
		SigningKey:                  "example-signing-key-at-least-32-bytes",
		AccessTokenTTLSeconds:       900,
		RefreshTokenTTLSeconds:      604800,
		RefreshTokenPepper:          "example-refresh-pepper-at-least-32",
		MFAIssuer:                   "aoi-admin",
		MFASecretKey:                "example-mfa-secret-key-at-least-32",
		LoginMaxFailures:            5,
		LoginLockMinutes:            15,
		InvitationTTLSeconds:        86400,
		EmailVerificationTTLSeconds: 86400,
		PasswordResetTTLSeconds:     1800,
		NotificationDriver:          "smtp",
		SMTP: SMTPConfig{
			Host:     "smtp.example.com",
			Port:     587,
			From:     "noreply@example.com",
			Security: SMTPSecurityStartTLS,
		},
		PasswordPolicy: PasswordPolicyConfig{MinLength: 8},
	}
}
