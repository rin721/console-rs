package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rei0721/go-scaffold/pkg/storage"
)

const (
	StorageDriverDisabled   = "disabled"
	StorageDriverLocal      = "local"
	StorageDriverS3         = "s3"
	StorageDriverMinIO      = "minio"
	StorageDriverLocalS3    = "local+s3"
	StorageDriverLocalMinIO = "local+minio"
)

type StorageConfig struct {
	Driver string             `mapstructure:"driver" envname:"STORAGE_DRIVER" json:"driver" yaml:"driver" toml:"driver"`
	Local  StorageLocalConfig `mapstructure:"local" json:"local" yaml:"local" toml:"local"`
	S3     StorageS3Config    `mapstructure:"s3" json:"s3" yaml:"s3" toml:"s3"`
	MinIO  StorageMinIOConfig `mapstructure:"minio" json:"minio" yaml:"minio" toml:"minio"`
}

type StorageLocalConfig struct {
	FSType          string `mapstructure:"fsType" envname:"STORAGE_LOCAL_FS_TYPE" json:"fsType" yaml:"fsType" toml:"fsType"`
	BasePath        string `mapstructure:"basePath" envname:"STORAGE_LOCAL_BASE_PATH" json:"basePath" yaml:"basePath" toml:"basePath"`
	PublicURL       string `mapstructure:"publicUrl" envname:"STORAGE_LOCAL_PUBLIC_URL" json:"publicUrl" yaml:"publicUrl" toml:"publicUrl"`
	EnableWatch     bool   `mapstructure:"enableWatch" envname:"STORAGE_LOCAL_ENABLE_WATCH" json:"enableWatch" yaml:"enableWatch" toml:"enableWatch"`
	WatchBufferSize int    `mapstructure:"watchBufferSize" envname:"STORAGE_LOCAL_WATCH_BUFFER_SIZE" json:"watchBufferSize" yaml:"watchBufferSize" toml:"watchBufferSize"`
}

type StorageS3Config struct {
	Endpoint        string `mapstructure:"endpoint" envname:"STORAGE_S3_ENDPOINT" json:"endpoint" yaml:"endpoint" toml:"endpoint"`
	Region          string `mapstructure:"region" envname:"STORAGE_S3_REGION" json:"region" yaml:"region" toml:"region"`
	Bucket          string `mapstructure:"bucket" envname:"STORAGE_S3_BUCKET" json:"bucket" yaml:"bucket" toml:"bucket"`
	AccessKeyID     string `mapstructure:"accessKeyId" envname:"STORAGE_S3_ACCESS_KEY_ID" json:"accessKeyId" yaml:"accessKeyId" toml:"accessKeyId"`
	SecretAccessKey string `mapstructure:"secretAccessKey" envname:"STORAGE_S3_SECRET_ACCESS_KEY" json:"secretAccessKey" yaml:"secretAccessKey" toml:"secretAccessKey"`
	UsePathStyle    bool   `mapstructure:"usePathStyle" envname:"STORAGE_S3_USE_PATH_STYLE" json:"usePathStyle" yaml:"usePathStyle" toml:"usePathStyle"`
	PublicBaseURL   string `mapstructure:"publicBaseUrl" envname:"STORAGE_S3_PUBLIC_BASE_URL" json:"publicBaseUrl" yaml:"publicBaseUrl" toml:"publicBaseUrl"`
}

type StorageMinIOConfig struct {
	Endpoint        string `mapstructure:"endpoint" envname:"STORAGE_MINIO_ENDPOINT" json:"endpoint" yaml:"endpoint" toml:"endpoint"`
	Region          string `mapstructure:"region" envname:"STORAGE_MINIO_REGION" json:"region" yaml:"region" toml:"region"`
	Bucket          string `mapstructure:"bucket" envname:"STORAGE_MINIO_BUCKET" json:"bucket" yaml:"bucket" toml:"bucket"`
	AccessKeyID     string `mapstructure:"accessKeyId" envname:"STORAGE_MINIO_ACCESS_KEY_ID" json:"accessKeyId" yaml:"accessKeyId" toml:"accessKeyId"`
	SecretAccessKey string `mapstructure:"secretAccessKey" envname:"STORAGE_MINIO_SECRET_ACCESS_KEY" json:"secretAccessKey" yaml:"secretAccessKey" toml:"secretAccessKey"`
	UsePathStyle    bool   `mapstructure:"usePathStyle" envname:"STORAGE_MINIO_USE_PATH_STYLE" json:"usePathStyle" yaml:"usePathStyle" toml:"usePathStyle"`
	PublicBaseURL   string `mapstructure:"publicBaseUrl" envname:"STORAGE_MINIO_PUBLIC_BASE_URL" json:"publicBaseUrl" yaml:"publicBaseUrl" toml:"publicBaseUrl"`
}

func (c *StorageConfig) ValidateName() string {
	return AppStorageName
}

func (c *StorageConfig) ValidateRequired() bool {
	return false
}

func (c *StorageConfig) Validate() error {
	c.Driver = strings.ToLower(strings.TrimSpace(c.Driver))
	switch c.Driver {
	case StorageDriverDisabled:
		return nil
	case StorageDriverLocal:
		return c.Local.Validate()
	case StorageDriverS3:
		return c.S3.Validate("s3")
	case StorageDriverMinIO:
		return c.MinIO.Validate("minio")
	case StorageDriverLocalS3:
		if err := c.Local.Validate(); err != nil {
			return err
		}
		return c.S3.Validate("s3")
	case StorageDriverLocalMinIO:
		if err := c.Local.Validate(); err != nil {
			return err
		}
		return c.MinIO.Validate("minio")
	case "":
		return errors.New("driver is required")
	default:
		return fmt.Errorf("driver must be one of: disabled, local, s3, minio, local+s3, local+minio")
	}
}

func (c StorageLocalConfig) Validate() error {
	fsType := strings.TrimSpace(c.FSType)
	if fsType == "" {
		fsType = string(storage.FSTypeOS)
	}
	if !stringInSet(fsType, "os", "memory", "readonly", "basepath") {
		return fmt.Errorf("local.fsType must be one of: os, memory, readonly, basepath")
	}
	if fsType == "basepath" && strings.TrimSpace(c.BasePath) == "" {
		return errors.New("local.basePath is required when local.fsType is basepath")
	}
	if c.WatchBufferSize < 0 {
		return errors.New("local.watchBufferSize must be non-negative")
	}
	return nil
}

func (c StorageS3Config) Validate(prefix string) error {
	return validateStorageObject(prefix, c.Endpoint, c.Bucket, c.AccessKeyID, c.SecretAccessKey)
}

func (c StorageMinIOConfig) Validate(prefix string) error {
	return validateStorageObject(prefix, c.Endpoint, c.Bucket, c.AccessKeyID, c.SecretAccessKey)
}

func validateStorageObject(prefix, endpoint, bucket, accessKeyID, secretAccessKey string) error {
	if strings.TrimSpace(endpoint) == "" {
		return fmt.Errorf("%s.endpoint is required", prefix)
	}
	if strings.TrimSpace(bucket) == "" {
		return fmt.Errorf("%s.bucket is required", prefix)
	}
	if strings.TrimSpace(accessKeyID) == "" {
		return fmt.Errorf("%s.accessKeyId is required", prefix)
	}
	if strings.TrimSpace(secretAccessKey) == "" {
		return fmt.Errorf("%s.secretAccessKey is required", prefix)
	}
	return nil
}

func (c StorageConfig) ToPkgConfig() *storage.Config {
	return &storage.Config{
		Driver: storage.Driver(c.Driver),
		Local: storage.LocalConfig{
			FSType:          storage.FSType(firstNonEmpty(c.Local.FSType, string(storage.FSTypeOS))),
			BasePath:        c.Local.BasePath,
			PublicURL:       c.Local.PublicURL,
			EnableWatch:     c.Local.EnableWatch,
			WatchBufferSize: c.Local.WatchBufferSize,
		},
		S3: storage.ObjectConfig{
			Provider:        string(storage.ObjectProviderS3),
			Endpoint:        c.S3.Endpoint,
			Region:          c.S3.Region,
			Bucket:          c.S3.Bucket,
			AccessKeyID:     c.S3.AccessKeyID,
			SecretAccessKey: c.S3.SecretAccessKey,
			PathStyle:       c.S3.UsePathStyle,
			PublicBaseURL:   c.S3.PublicBaseURL,
		},
		MinIO: storage.ObjectConfig{
			Provider:        string(storage.ObjectProviderMinIO),
			Endpoint:        c.MinIO.Endpoint,
			Region:          c.MinIO.Region,
			Bucket:          c.MinIO.Bucket,
			AccessKeyID:     c.MinIO.AccessKeyID,
			SecretAccessKey: c.MinIO.SecretAccessKey,
			PathStyle:       true,
			PublicBaseURL:   c.MinIO.PublicBaseURL,
		},
	}
}

func (c *StorageConfig) OverrideConfig() {
	overrideConfigFromEnv(c)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
