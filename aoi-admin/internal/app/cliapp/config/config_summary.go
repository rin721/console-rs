package config

import (
	"fmt"
	"io"

	"github.com/rei0721/go-scaffold/internal/app/cliapp/localization"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
)

func PrintConfigSummary(stdout io.Writer, configPath string, localizers ...*localization.Localizer) error {
	localizer := firstLocalizer(localizers...)
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "%s: %s\n", localizer.T("cli.config.summary.configFile"), configPath)
	fmt.Fprintf(stdout, "%s: %s:%d\n", localizer.T("cli.config.summary.http"), cfg.Server.Host, cfg.Server.Port)
	fmt.Fprintf(stdout, "%s: %s %s\n", localizer.T("cli.config.summary.database"), cfg.Database.Driver, cliDatabaseTarget(cfg))
	fmt.Fprintf(stdout, "%s: %s\n", localizer.T("cli.config.summary.cache"), cfg.Cache.Driver)
	fmt.Fprintf(stdout, "%s: %s\n", localizer.T("cli.config.summary.storage"), cfg.Storage.Driver)
	fmt.Fprintf(stdout, "%s: %v\n", localizer.T("cli.config.summary.iam"), cfg.Auth.Enabled)
	if cfg.Logger.FilePath != "" {
		fmt.Fprintf(stdout, "%s: %s\n", localizer.T("cli.config.summary.appLog"), cfg.Logger.FilePath)
	}
	return nil
}

func firstLocalizer(localizers ...*localization.Localizer) *localization.Localizer {
	if len(localizers) > 0 && localizers[0] != nil {
		return localizers[0]
	}
	return localization.ForArgs(nil)
}

func cliDatabaseTarget(cfg *appconfig.Config) string {
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
