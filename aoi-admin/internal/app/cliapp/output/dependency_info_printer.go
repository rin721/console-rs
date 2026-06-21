package output

import (
	"fmt"
	"io"
	"strings"

	cliconfig "github.com/rei0721/go-scaffold/internal/app/cliapp/config"
	"github.com/rei0721/go-scaffold/internal/app/cliapp/localization"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
)

func PrintDependencyServiceInfo(w io.Writer, service string, configPath string, localizers ...*localization.Localizer) error {
	localizer := firstLocalizer(localizers...)
	cfg, err := cliconfig.LoadConfig(configPath)
	if err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(service)) {
	case "db":
		fmt.Fprintf(w, "db: driver=%s target=%s\n", cfg.Database.Driver, databaseTarget(cfg))
	case "iam":
		fmt.Fprintf(w, "iam: enabled=%v issuer=%s\n", cfg.Auth.Enabled, cfg.Auth.Issuer)
	case "cache":
		fmt.Fprintf(w, "cache: driver=%s redis=%s\n", cfg.Cache.Driver, cfg.Cache.Redis.Addr)
	case "storage":
		fmt.Fprintf(w, "storage: driver=%s local=%s s3=%s minio=%s\n", cfg.Storage.Driver, cfg.Storage.Local.BasePath, cfg.Storage.S3.Bucket, cfg.Storage.MinIO.Bucket)
	}
	fmt.Fprintln(w, localizer.T("cli.run.dependencyInfo.note"))
	return nil
}

func databaseTarget(cfg *appconfig.Config) string {
	switch cfg.Database.Driver {
	case appconfig.DatabaseDriverSQLite:
		return cfg.Database.SQLite.Path
	case appconfig.DatabaseDriverMySQL:
		return cfg.Database.MySQL.Database + "@" + cfg.Database.MySQL.Host
	case appconfig.DatabaseDriverPostgres:
		return cfg.Database.Postgres.Database + "@" + cfg.Database.Postgres.Host
	default:
		return ""
	}
}
