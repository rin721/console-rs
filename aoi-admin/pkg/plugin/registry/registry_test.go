package registry

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

func TestMemoryRegistryLeaseCapabilityAndSubscription(t *testing.T) {
	now := time.Date(2026, 6, 15, 1, 0, 0, 0, time.UTC)
	reg := NewMemory()
	registered, err := reg.RegisterInstance(context.Background(), testSnapshot(now))
	if err != nil {
		t.Fatalf("RegisterInstance() error = %v", err)
	}
	if registered.InstanceID != "demo-1" {
		t.Fatalf("instance_id = %q", registered.InstanceID)
	}
	byCapability, err := reg.ListByCapability(context.Background(), "demo.echo", InstanceFilter{Status: protocol.StatusOnline, Now: now})
	if err != nil {
		t.Fatalf("ListByCapability() error = %v", err)
	}
	if len(byCapability) != 1 {
		t.Fatalf("ListByCapability() = %#v", byCapability)
	}
	renewed, err := reg.RenewLease(context.Background(), Lease{
		PluginID:        "demo",
		InstanceID:      "demo-1",
		LeaseTTL:        time.Minute,
		LastHeartbeatAt: now.Add(10 * time.Second),
		ExpiresAt:       now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("RenewLease() error = %v", err)
	}
	if renewed.LeaseTTLSeconds != 60 {
		t.Fatalf("lease ttl = %d", renewed.LeaseTTLSeconds)
	}
	if _, err := reg.SubscribeEvent(context.Background(), Subscription{PluginID: "demo", InstanceID: "demo-1", Event: "demo.event"}); err != nil {
		t.Fatalf("SubscribeEvent() error = %v", err)
	}
	subs, err := reg.ListSubscriptions(context.Background(), SubscriptionFilter{Event: "demo.event"})
	if err != nil {
		t.Fatalf("ListSubscriptions() error = %v", err)
	}
	if len(subs) != 1 || subs[0].InstanceID != "demo-1" {
		t.Fatalf("subscriptions = %#v", subs)
	}
	expired, err := reg.ExpireLeases(context.Background(), now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("ExpireLeases() error = %v", err)
	}
	if len(expired) != 1 || expired[0].Status != protocol.StatusOffline {
		t.Fatalf("expired = %#v", expired)
	}
}

func TestMemoryRegistryWatchEmitsStateChanges(t *testing.T) {
	now := time.Now().UTC()
	reg := NewMemory()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changes, err := reg.Watch(ctx, WatchOptions{Buffer: 8})
	if err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	if _, err := reg.RegisterInstance(context.Background(), testSnapshot(now)); err != nil {
		t.Fatalf("RegisterInstance() error = %v", err)
	}
	if change := readRegistryChange(t, changes); change.Type != ChangeRegistered || change.PluginID != "demo" {
		t.Fatalf("register change = %#v", change)
	}

	if _, err := reg.SubscribeEvent(context.Background(), Subscription{PluginID: "demo", InstanceID: "demo-1", Event: "demo.event"}); err != nil {
		t.Fatalf("SubscribeEvent() error = %v", err)
	}
	if change := readRegistryChange(t, changes); change.Type != ChangeEventSubscribed || change.Subscription == nil || change.Subscription.Event != "demo.event" {
		t.Fatalf("subscription change = %#v", change)
	}

	if _, err := reg.ExpireLeases(context.Background(), now.Add(time.Minute)); err != nil {
		t.Fatalf("ExpireLeases() error = %v", err)
	}
	if change := readRegistryChange(t, changes); change.Type != ChangeLeaseExpired || change.Plugin.Status != protocol.StatusOffline {
		t.Fatalf("expire change = %#v", change)
	}
}

func TestSQLStoreSharesRegisteredInstances(t *testing.T) {
	db := openRegistryTestDB(t)
	storeA := NewSQLStore(db, WithDialect("sqlite"))
	storeB := NewSQLStore(db, WithDialect("sqlite"))
	now := time.Now().UTC()
	if _, err := storeA.RegisterInstance(context.Background(), testSnapshot(now)); err != nil {
		t.Fatalf("RegisterInstance() error = %v", err)
	}
	items, err := storeB.ListByCapability(context.Background(), "demo.echo", InstanceFilter{Status: protocol.StatusOnline, Now: now})
	if err != nil {
		t.Fatalf("ListByCapability() error = %v", err)
	}
	if len(items) != 1 || items[0].PluginID != "demo" || items[0].InstanceID != "demo-1" {
		t.Fatalf("items = %#v", items)
	}
	if _, err := storeB.RenewLease(context.Background(), Lease{
		PluginID:        "demo",
		InstanceID:      "demo-1",
		LeaseTTL:        2 * time.Minute,
		LastHeartbeatAt: now.Add(time.Minute),
		ExpiresAt:       now.Add(3 * time.Minute),
	}); err != nil {
		t.Fatalf("RenewLease() error = %v", err)
	}
	renewed, err := storeA.GetInstance(context.Background(), "demo", "demo-1")
	if err != nil {
		t.Fatalf("GetInstance() error = %v", err)
	}
	if renewed.LeaseTTLSeconds != 120 {
		t.Fatalf("lease ttl = %d", renewed.LeaseTTLSeconds)
	}
}

func TestSQLStoreWatchObservesSharedState(t *testing.T) {
	db := openRegistryTestDB(t)
	storeA := NewSQLStore(db, WithDialect("sqlite"))
	storeB := NewSQLStore(db, WithDialect("sqlite"))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changes, err := storeB.Watch(ctx, WatchOptions{Buffer: 8, InitialSnapshot: true, PollInterval: 10 * time.Millisecond})
	if err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	now := time.Date(2026, 6, 15, 1, 0, 0, 0, time.UTC)
	if _, err := storeA.RegisterInstance(context.Background(), testSnapshot(now)); err != nil {
		t.Fatalf("RegisterInstance() error = %v", err)
	}
	if change := readRegistryChange(t, changes); change.Type != ChangeSnapshotObserved || change.PluginID != "demo" {
		t.Fatalf("watched change = %#v", change)
	}
}

func openRegistryTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	paths, err := filepath.Glob(filepath.Join("..", "..", "..", "internal", "migrations", "*plugin_instance*.sql"))
	if err != nil {
		t.Fatalf("glob plugin registry migrations: %v", err)
	}
	basePath := filepath.Join("..", "..", "..", "internal", "migrations", "20260615000100_create_plugin_registry.sql")
	paths = append([]string{basePath}, paths...)
	for _, path := range paths {
		applyRegistryMigration(t, db, path)
	}
	return db
}

func applyRegistryMigration(t *testing.T, db *sql.DB, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	up, _, _ := strings.Cut(string(raw), "-- +goose Down")
	up = strings.ReplaceAll(up, "-- +goose Up", "")
	for _, statement := range strings.Split(up, ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("exec migration statement %q: %v", statement, err)
		}
	}
}

func readRegistryChange(t *testing.T, changes <-chan Change) Change {
	t.Helper()
	select {
	case change, ok := <-changes:
		if !ok {
			t.Fatal("registry watch channel closed")
		}
		return change
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for registry change")
		return Change{}
	}
}

func testSnapshot(now time.Time) protocol.PluginSnapshot {
	return protocol.PluginSnapshot{
		PluginMetadata: protocol.PluginMetadata{
			PluginID:   "demo",
			InstanceID: "demo-1",
			Name:       "Demo",
			Version:    "0.1.0",
			Protocol:   protocol.TransportHTTP,
			Endpoint:   "http://127.0.0.1:10098",
			Capabilities: []protocol.Capability{
				{Name: "demo.echo", Version: "v1"},
			},
			Permissions:   []string{"plugin:demo"},
			SchemaVersion: protocol.ProtocolVersionV1,
		},
		Status:          protocol.StatusOnline,
		RuntimeStatus:   protocol.RuntimeStatusReady,
		OwnerHost:       "host-a",
		LeaseTTLSeconds: 30,
		LeaseExpiresAt:  now.Add(30 * time.Second),
		RegisteredAt:    now,
		LastHeartbeatAt: now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}
