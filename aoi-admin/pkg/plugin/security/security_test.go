package security

import (
	"context"
	"errors"
	"testing"

	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

func TestSharedSecretAuthenticator(t *testing.T) {
	authenticator := SharedSecretAuthenticator{Mode: "shared_secret", Secret: "secret"}
	principal, err := authenticator.Authenticate(context.Background(), Operation{Name: protocol.OperationRegister, PluginID: "demo", InstanceID: "demo-1"}, &protocol.Auth{
		Mode:  "shared_secret",
		Token: "secret",
	})
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if principal.PluginID != "demo" || principal.InstanceID != "demo-1" || principal.AuthMode != "shared_secret" {
		t.Fatalf("principal = %#v", principal)
	}
	if _, err := authenticator.Authenticate(context.Background(), Operation{}, &protocol.Auth{Mode: "shared_secret", Token: "bad"}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Authenticate(bad) error = %v, want ErrUnauthorized", err)
	}
}

func TestSignatureAuthenticatorIsExplicitlyUnsupported(t *testing.T) {
	_, err := SharedSecretAuthenticator{Mode: "signature"}.Authenticate(context.Background(), Operation{}, &protocol.Auth{Mode: "signature"})
	if !errors.Is(err, ErrUnsupportedAuth) {
		t.Fatalf("Authenticate(signature) error = %v, want ErrUnsupportedAuth", err)
	}
}

func TestScopeAuthorizerRejectsDisallowedPermission(t *testing.T) {
	authorizer := ScopeAuthorizer{AllowedPermissions: []string{"plugin:read"}}
	_, err := authorizer.Authorize(context.Background(), Principal{}, PermissionRequest{Permissions: []string{"plugin:write"}})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Authorize() error = %v, want ErrUnauthorized", err)
	}
	decision, err := authorizer.Authorize(context.Background(), Principal{}, PermissionRequest{Permissions: []string{"plugin:read"}})
	if err != nil {
		t.Fatalf("Authorize(allowed) error = %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("decision = %#v", decision)
	}
}
