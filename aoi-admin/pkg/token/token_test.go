package token

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestManagerIssueParseAndTypeCheck(t *testing.T) {
	now := time.Now().Add(-time.Second).UTC()
	manager, err := New(Config{
		Issuer:        "go-scaffold",
		Audience:      []string{"api"},
		SigningKey:    "01234567890123456789012345678901",
		AccessTTL:     10 * time.Minute,
		RefreshTTL:    time.Hour,
		RefreshPepper: "refresh-pepper",
		Now:           func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	pair, err := manager.IssuePair(context.Background(), Subject{UserID: 1, OrgID: 2, SessionID: 3, ProductCode: "platform"})
	if err != nil {
		t.Fatalf("IssuePair() failed: %v", err)
	}
	if pair.RefreshToken == "" || pair.RefreshTokenHash == "" || pair.RefreshToken == pair.RefreshTokenHash {
		t.Fatalf("refresh token/hash not generated correctly: %#v", pair)
	}

	claims, err := manager.Parse(context.Background(), pair.AccessToken, TokenTypeAccess)
	if err != nil {
		t.Fatalf("Parse(access) failed: %v", err)
	}
	if claims.UserID != 1 || claims.OrgID != 2 || claims.SessionID != 3 || claims.ProductCode != "platform" {
		t.Fatalf("unexpected claims: %#v", claims)
	}

	if _, err := manager.Parse(context.Background(), pair.AccessToken, TokenTypeRefresh); !errors.Is(err, ErrWrongType) {
		t.Fatalf("expected ErrWrongType, got %v", err)
	}
}

func TestRefreshHashIsStableAndPeppered(t *testing.T) {
	base := Config{
		Issuer:        "go-scaffold",
		SigningKey:    "01234567890123456789012345678901",
		AccessTTL:     time.Minute,
		RefreshTTL:    time.Hour,
		RefreshPepper: "pepper-a",
	}
	a, err := New(base)
	if err != nil {
		t.Fatalf("New(a) failed: %v", err)
	}
	base.RefreshPepper = "pepper-b"
	b, err := New(base)
	if err != nil {
		t.Fatalf("New(b) failed: %v", err)
	}

	if a.HashRefreshToken("token") != a.HashRefreshToken("token") {
		t.Fatal("same token and pepper should hash consistently")
	}
	if a.HashRefreshToken("token") == b.HashRefreshToken("token") {
		t.Fatal("different pepper should produce different hash")
	}
}
