package authorization

import (
	"context"
	"testing"
)

func TestDomainRBAC(t *testing.T) {
	enforcer, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	ctx := context.Background()

	if _, err := enforcer.AddPolicy(ctx, "role:admin", "100", "core", "tenant", "user", "read|invite"); err != nil {
		t.Fatalf("AddPolicy() failed: %v", err)
	}
	if _, err := enforcer.AddRoleForUser(ctx, "user:1", "role:admin", "100"); err != nil {
		t.Fatalf("AddRoleForUser() failed: %v", err)
	}

	allowed, err := enforcer.Enforce(ctx, "user:1", "100", "core", "tenant", "user", "read")
	if err != nil {
		t.Fatalf("Enforce(read) failed: %v", err)
	}
	if !allowed {
		t.Fatal("expected user:1 to read users in org 100")
	}

	allowed, err = enforcer.Enforce(ctx, "user:1", "200", "core", "tenant", "user", "read")
	if err != nil {
		t.Fatalf("Enforce(other org) failed: %v", err)
	}
	if allowed {
		t.Fatal("domain isolation failed")
	}

	allowed, err = enforcer.Enforce(ctx, "user:1", "100", "core", "platform", "user", "read")
	if err != nil {
		t.Fatalf("Enforce(other scope) failed: %v", err)
	}
	if allowed {
		t.Fatal("scope isolation failed")
	}

	allowed, err = enforcer.Enforce(ctx, "user:1", "100", "other-product", "tenant", "user", "read")
	if err != nil {
		t.Fatalf("Enforce(other product) failed: %v", err)
	}
	if allowed {
		t.Fatal("product isolation failed")
	}

	if _, err := enforcer.AddPolicy(ctx, "role:wide", "100", "*", "tenant", "user", "read"); err != nil {
		t.Fatalf("AddPolicy(product wildcard) failed: %v", err)
	}
	if _, err := enforcer.AddRoleForUser(ctx, "user:2", "role:wide", "100"); err != nil {
		t.Fatalf("AddRoleForUser(product wildcard) failed: %v", err)
	}
	allowed, err = enforcer.Enforce(ctx, "user:2", "100", "other-product", "tenant", "user", "read")
	if err != nil {
		t.Fatalf("Enforce(product wildcard) failed: %v", err)
	}
	if allowed {
		t.Fatal("product wildcard policy should not bypass product isolation")
	}
}

func TestLoadRules(t *testing.T) {
	enforcer, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	err = enforcer.LoadRules(context.Background(), []Rule{
		{PType: "p", Values: []string{"role:owner", "1", "core", "tenant", "audit", "*"}},
		{PType: "g", Values: []string{"user:9", "role:owner", "1"}},
	})
	if err != nil {
		t.Fatalf("LoadRules() failed: %v", err)
	}
	allowed, err := enforcer.Enforce(context.Background(), "user:9", "1", "core", "tenant", "audit", "read")
	if err != nil {
		t.Fatalf("Enforce() failed: %v", err)
	}
	if !allowed {
		t.Fatal("owner wildcard policy should allow audit read")
	}

	allowed, err = enforcer.Enforce(context.Background(), "user:9", "1", "core", "platform", "audit", "read")
	if err != nil {
		t.Fatalf("Enforce(platform scope) failed: %v", err)
	}
	if allowed {
		t.Fatal("tenant policy should not allow platform audit read")
	}
}
