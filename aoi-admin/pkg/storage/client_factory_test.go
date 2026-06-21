package storage

import (
	"context"
	"strings"
	"testing"
)

func TestNewManagerDisabled(t *testing.T) {
	manager, err := NewManager(context.Background(), &Config{Driver: DriverDisabled})
	if err != nil {
		t.Fatalf("NewManager(disabled) error = %v", err)
	}
	if manager.Primary() != nil || manager.Local != nil || manager.Object != nil {
		t.Fatalf("disabled manager should not create clients: %#v", manager)
	}
}

func TestLocalStorageClientExercise(t *testing.T) {
	manager, err := NewManager(context.Background(), &Config{
		Driver: DriverLocal,
		Local: LocalConfig{
			BasePath: t.TempDir(),
		},
	})
	if err != nil {
		t.Fatalf("NewManager(local) error = %v", err)
	}
	defer manager.Close()
	if manager.Local == nil || manager.Primary() == nil {
		t.Fatalf("local manager did not create local client: %#v", manager)
	}
	if err := ExerciseClient(context.Background(), manager.Local); err != nil {
		t.Fatalf("ExerciseClient(local) error = %v", err)
	}
}

func TestRemoteStorageRequiresConnectionFields(t *testing.T) {
	_, err := NewManager(context.Background(), &Config{Driver: DriverS3})
	if err == nil {
		t.Fatal("NewManager(r2 without object config) error = nil")
	}
	if !strings.Contains(err.Error(), "endpoint") {
		t.Fatalf("NewManager(r2) error = %v, want endpoint hint", err)
	}
}

func TestLocalRemoteModeCreatesLocalBeforeValidatingRemote(t *testing.T) {
	manager, err := NewManager(context.Background(), &Config{
		Driver: DriverLocalMinIO,
		Local: LocalConfig{
			BasePath: t.TempDir(),
		},
		MinIO: ObjectConfig{
			Endpoint:        "http://127.0.0.1:9000",
			Bucket:          "bucket",
			AccessKeyID:     "access",
			SecretAccessKey: "secret",
		},
	})
	if err != nil {
		t.Fatalf("NewManager(local+minio with config) error = %v", err)
	}
	defer manager.Close()
	if manager.Local == nil || manager.Object == nil {
		t.Fatalf("local+minio manager did not create both clients: %#v", manager)
	}
}
