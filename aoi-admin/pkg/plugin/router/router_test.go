package router

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
	"github.com/rei0721/go-scaffold/pkg/plugin/registry"
)

func TestRouterInvokesRoundRobinHealthyInstances(t *testing.T) {
	now := time.Date(2026, 6, 15, 1, 0, 0, 0, time.UTC)
	reg := registry.NewMemory()
	mustRegister(t, reg, routeSnapshot("demo-1", now, now.Add(time.Minute)))
	mustRegister(t, reg, routeSnapshot("demo-2", now, now.Add(time.Minute)))
	invoker := &fakeInvoker{}
	r := New(Config{
		Registry:       reg,
		RemoteInvokers: map[string]RemoteInvoker{protocol.TransportHTTP: invoker},
		Now:            func() time.Time { return now },
	})

	for i := 0; i < 2; i++ {
		if _, err := r.Invoke(context.Background(), protocol.InvokeRequest{Capability: "demo.echo"}); err != nil {
			t.Fatalf("Invoke(%d) error = %v", i, err)
		}
	}
	if got := invoker.invoked; len(got) != 2 || got[0] != "demo-1" || got[1] != "demo-2" {
		t.Fatalf("invoked instances = %v", got)
	}
}

func TestRouterFiltersExpiredInstances(t *testing.T) {
	now := time.Date(2026, 6, 15, 1, 0, 0, 0, time.UTC)
	reg := registry.NewMemory()
	mustRegister(t, reg, routeSnapshot("expired", now, now.Add(-time.Second)))
	mustRegister(t, reg, routeSnapshot("healthy", now, now.Add(time.Minute)))
	invoker := &fakeInvoker{}
	r := New(Config{
		Registry:       reg,
		RemoteInvokers: map[string]RemoteInvoker{protocol.TransportHTTP: invoker},
		Now:            func() time.Time { return now },
	})
	if _, err := r.Invoke(context.Background(), protocol.InvokeRequest{Capability: "demo.echo"}); err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if got := invoker.invoked; len(got) != 1 || got[0] != "healthy" {
		t.Fatalf("invoked instances = %v", got)
	}
}

func TestRouterPushesSubscribedEvents(t *testing.T) {
	now := time.Date(2026, 6, 15, 1, 0, 0, 0, time.UTC)
	reg := registry.NewMemory()
	mustRegister(t, reg, routeSnapshot("demo-1", now, now.Add(time.Minute)))
	if _, err := reg.SubscribeEvent(context.Background(), registry.Subscription{PluginID: "demo", InstanceID: "demo-1", Event: "demo.event"}); err != nil {
		t.Fatalf("SubscribeEvent() error = %v", err)
	}
	invoker := &fakeInvoker{}
	r := New(Config{
		Registry:       reg,
		RemoteInvokers: map[string]RemoteInvoker{protocol.TransportHTTP: invoker},
		Now:            func() time.Time { return now },
	})
	response, err := r.PushEvent(context.Background(), protocol.PushEventRequest{Event: "demo.event"})
	if err != nil {
		t.Fatalf("PushEvent() error = %v", err)
	}
	if response.Delivered != 1 || len(invoker.events) != 1 {
		t.Fatalf("response = %#v events=%v", response, invoker.events)
	}
}

type fakeInvoker struct {
	invoked []string
	events  []string
}

func (f *fakeInvoker) Invoke(_ context.Context, remote protocol.PluginSnapshot, _ protocol.InvokeRequest) (json.RawMessage, error) {
	f.invoked = append(f.invoked, remote.InstanceID)
	return json.RawMessage(`{"ok":true}`), nil
}

func (f *fakeInvoker) PushEvent(_ context.Context, remote protocol.PluginSnapshot, req protocol.PushEventRequest) error {
	f.events = append(f.events, remote.InstanceID+":"+req.Event)
	return nil
}

func mustRegister(t *testing.T, reg registry.Registry, snapshot protocol.PluginSnapshot) {
	t.Helper()
	if _, err := reg.RegisterInstance(context.Background(), snapshot); err != nil {
		t.Fatalf("RegisterInstance() error = %v", err)
	}
}

func routeSnapshot(instanceID string, now time.Time, expires time.Time) protocol.PluginSnapshot {
	return protocol.PluginSnapshot{
		PluginMetadata: protocol.PluginMetadata{
			PluginID:   "demo",
			InstanceID: instanceID,
			Name:       "Demo",
			Version:    "0.1.0",
			Protocol:   protocol.TransportHTTP,
			Endpoint:   "http://127.0.0.1:10098",
			Capabilities: []protocol.Capability{
				{Name: "demo.echo"},
			},
		},
		Status:          protocol.StatusOnline,
		RuntimeStatus:   protocol.RuntimeStatusReady,
		LeaseTTLSeconds: 60,
		LeaseExpiresAt:  expires,
		RegisteredAt:    now,
		LastHeartbeatAt: now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}
