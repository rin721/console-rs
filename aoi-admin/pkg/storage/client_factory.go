package storage

import (
	"context"
	"fmt"
	"strings"

	storageclient "github.com/rei0721/go-scaffold/pkg/storage/client"
	"github.com/rei0721/go-scaffold/pkg/storage/local"
	"github.com/rei0721/go-scaffold/pkg/storage/s3compat"
)

type StorageClient = storageclient.StorageClient
type PutOptions = storageclient.PutOptions
type ObjectInfo = storageclient.ObjectInfo

type LocalConfig struct {
	FSType          FSType `mapstructure:"fsType"`
	BasePath        string `mapstructure:"basePath"`
	PublicURL       string `mapstructure:"publicUrl"`
	EnableWatch     bool   `mapstructure:"enableWatch"`
	WatchBufferSize int    `mapstructure:"watchBufferSize"`
}

type ObjectConfig struct {
	Provider        string `mapstructure:"provider"`
	Endpoint        string `mapstructure:"endpoint"`
	Region          string `mapstructure:"region"`
	Bucket          string `mapstructure:"bucket"`
	AccessKeyID     string `mapstructure:"accessKeyId"`
	SecretAccessKey string `mapstructure:"secretAccessKey"`
	PathStyle       bool   `mapstructure:"usePathStyle"`
	PublicBaseURL   string `mapstructure:"publicBaseUrl"`
}

type StorageManager struct {
	Driver Driver
	Local  StorageClient
	Object StorageClient
}

func NewClient(ctx context.Context, cfg *Config) (StorageClient, error) {
	manager, err := NewManager(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return manager.Primary(), nil
}

func NewManager(ctx context.Context, cfg *Config) (*StorageManager, error) {
	cfg = normalizeClientConfig(cfg)
	manager := &StorageManager{Driver: cfg.Driver}
	switch cfg.Driver {
	case DriverDisabled:
		return manager, nil
	case DriverLocal:
		localClient, err := newLocalClient(cfg)
		if err != nil {
			return nil, err
		}
		manager.Local = localClient
	case DriverS3:
		objectClient, err := newObjectClient(ctx, cfg.S3, ObjectProviderS3)
		if err != nil {
			return nil, err
		}
		manager.Object = objectClient
	case DriverMinIO:
		objectClient, err := newObjectClient(ctx, cfg.MinIO, ObjectProviderMinIO)
		if err != nil {
			return nil, err
		}
		manager.Object = objectClient
	case DriverLocalS3:
		localClient, err := newLocalClient(cfg)
		if err != nil {
			return nil, err
		}
		objectClient, err := newObjectClient(ctx, cfg.S3, ObjectProviderS3)
		if err != nil {
			_ = localClient.Close()
			return nil, err
		}
		manager.Local = localClient
		manager.Object = objectClient
	case DriverLocalMinIO:
		localClient, err := newLocalClient(cfg)
		if err != nil {
			return nil, err
		}
		objectClient, err := newObjectClient(ctx, cfg.MinIO, ObjectProviderMinIO)
		if err != nil {
			_ = localClient.Close()
			return nil, err
		}
		manager.Local = localClient
		manager.Object = objectClient
	default:
		return nil, fmt.Errorf("%w: unsupported storage driver %q", ErrInvalidConfig, cfg.Driver)
	}
	return manager, nil
}

func (m *StorageManager) Primary() StorageClient {
	if m == nil {
		return nil
	}
	if m.Object != nil {
		return m.Object
	}
	return m.Local
}

func (m *StorageManager) Close() error {
	if m == nil {
		return nil
	}
	var err error
	if m.Local != nil {
		err = m.Local.Close()
	}
	if m.Object != nil {
		if closeErr := m.Object.Close(); err == nil {
			err = closeErr
		}
	}
	return err
}

func ExerciseClient(ctx context.Context, client StorageClient) error {
	if client == nil {
		return nil
	}
	const key = "setup/storage-healthcheck.txt"
	if err := client.Put(ctx, key, []byte("ok"), PutOptions{ContentType: "text/plain"}); err != nil {
		return err
	}
	defer client.Delete(context.Background(), key)
	data, _, err := client.Get(ctx, key)
	if err != nil {
		return err
	}
	if string(data) != "ok" {
		return fmt.Errorf("unexpected storage healthcheck payload")
	}
	if exists, err := client.Exists(ctx, key); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("storage healthcheck object missing after write")
	}
	return nil
}

func normalizeClientConfig(cfg *Config) *Config {
	if cfg == nil {
		cfg = &Config{}
		cfg.DefaultConfig()
	}
	next := *cfg
	next.Driver = Driver(strings.ToLower(strings.TrimSpace(string(next.Driver))))
	if next.Driver == "" {
		next.Driver = DriverLocal
	}
	if next.Local.FSType == "" {
		next.Local.FSType = DefaultFSType
	}
	if next.Local.BasePath == "" {
		next.Local.BasePath = DefaultBasePath
	}
	return &next
}

func newLocalClient(cfg *Config) (StorageClient, error) {
	return local.New(local.Config{BasePath: cfg.Local.BasePath})
}

func newObjectClient(ctx context.Context, objectCfg ObjectConfig, provider ObjectProvider) (StorageClient, error) {
	if objectCfg.Provider == "" {
		objectCfg.Provider = string(provider)
	}
	return s3compat.New(ctx, s3compat.Config{
		Provider:        objectCfg.Provider,
		Endpoint:        objectCfg.Endpoint,
		Region:          objectCfg.Region,
		Bucket:          objectCfg.Bucket,
		AccessKeyID:     objectCfg.AccessKeyID,
		SecretAccessKey: objectCfg.SecretAccessKey,
		PathStyle:       objectCfg.PathStyle,
		PublicBaseURL:   objectCfg.PublicBaseURL,
	})
}
