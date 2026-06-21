package mfa

import (
	"strings"
	"testing"
	"time"
)

func TestTOTPGenerateAndValidate(t *testing.T) {
	key, err := GenerateTOTP("go-scaffold", "admin@example.com")
	if err != nil {
		t.Fatalf("GenerateTOTP() failed: %v", err)
	}
	if key.Secret == "" || !strings.HasPrefix(key.URL, "otpauth://totp/") {
		t.Fatalf("unexpected key: %#v", key)
	}

	code, err := GenerateTOTPCode(key.Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateTOTPCode() failed: %v", err)
	}
	if !ValidateTOTP(code, key.Secret) {
		t.Fatal("ValidateTOTP() should accept generated code")
	}
	if ValidateTOTP("000000", key.Secret) {
		t.Fatal("ValidateTOTP() should reject wrong code")
	}
}
