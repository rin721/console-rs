package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rei0721/go-scaffold/internal/app/cliapp/localization"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/cli"
	"github.com/rei0721/go-scaffold/pkg/configloader"
)

func promptDatabasePreflightRepair(ctx *cli.Context, configPath string, cfg *appconfig.Config) (bool, error) {
	localizer := localization.FromContext(ctx)
	action, err := cli.SelectKey(ctx.Context, ctx.UI, "preflight.database.action", localizer.T("cli.config.preflight.database.prompt"), []cli.SelectOption{
		{Value: preflightDatabaseActionSQLite, Label: localizer.T("cli.config.preflight.database.option.sqlite.label"), Description: localizer.T("cli.config.preflight.database.option.sqlite.description")},
		{Value: preflightDatabaseActionFile, Label: localizer.T("cli.config.preflight.database.option.file.label"), Description: localizer.T("cli.config.preflight.database.option.file.description")},
		{Value: preflightActionRuntimeEnvOnly, Label: localizer.T("cli.config.preflight.option.runtimeEnv.label"), Description: localizer.T("cli.config.preflight.option.runtimeEnv.description")},
		{Value: preflightActionSkip, Label: localizer.T("cli.config.preflight.option.skip.label"), Description: localizer.T("cli.config.preflight.option.skip.description")},
	})
	if err != nil {
		return false, err
	}
	switch action {
	case preflightDatabaseActionSQLite:
		if IsExampleConfig(configPath) {
			return false, exampleConfigWriteError(configPath)
		}
		updates := []configloader.YAMLScalarUpdate{
			{Kind: configloader.YAMLScalarString, Path: "database.driver", Value: appconfig.DatabaseDriverSQLite},
			{Kind: configloader.YAMLScalarString, Path: "database.sqlite.path", Value: "./data/app.db"},
			{Kind: configloader.YAMLScalarInt, Path: "database.pool.maxOpenConns", Value: "1"},
			{Kind: configloader.YAMLScalarInt, Path: "database.pool.maxIdleConns", Value: "1"},
		}
		paths := []string{"database.driver", "database.sqlite.path", "database.pool.maxOpenConns", "database.pool.maxIdleConns"}
		if err := applyConfigForceFileScalarUpdates(configPath, updates, paths); err != nil {
			return false, err
		}
		_ = ctx.UI.Info(localizer.T("cli.config.preflight.database.info.sqlite"))
		return true, nil
	case preflightDatabaseActionFile:
		if IsExampleConfig(configPath) {
			return false, exampleConfigWriteError(configPath)
		}
		updates, paths, err := promptDatabaseBranchUpdates(ctx, cfg)
		if err != nil {
			return false, err
		}
		if err := applyConfigForceFileScalarUpdates(configPath, updates, paths); err != nil {
			return false, err
		}
		_ = ctx.UI.Info(localizer.T("cli.config.preflight.database.info.file"))
		return true, nil
	case preflightActionRuntimeEnvOnly:
		if err := applyRuntimeEnvOnlyConfigPathsDirect(configPath, databaseRequiredPathsFor(cfg), validatePreflightRuntimeEnvValue); err != nil {
			return false, err
		}
		_ = ctx.UI.Info(localizer.T("cli.config.preflight.database.info.runtimeEnv"))
		return true, nil
	case preflightActionSkip, "":
		return false, nil
	default:
		return false, fmt.Errorf("unknown database repair action %q", action)
	}
}

func promptDatabaseBranchUpdates(ctx *cli.Context, cfg *appconfig.Config) ([]configloader.YAMLScalarUpdate, []string, error) {
	localizer := localization.FromContext(ctx)
	driver := strings.ToLower(strings.TrimSpace(cfg.Database.Driver))
	if driver == "" {
		selected, err := cli.SelectKey(ctx.Context, ctx.UI, "preflight.database.driver", localizer.T("cli.config.preflight.database.driverPrompt"), []cli.SelectOption{
			{Value: appconfig.DatabaseDriverSQLite, Label: "SQLite"},
			{Value: appconfig.DatabaseDriverMySQL, Label: "MySQL"},
			{Value: appconfig.DatabaseDriverPostgres, Label: "PostgreSQL"},
		})
		if err != nil {
			return nil, nil, err
		}
		driver = selected
	}
	switch driver {
	case appconfig.DatabaseDriverSQLite:
		path, err := promptConfigStringValue(ctx, "database.sqlite.path", cfg.Database.SQLite.Path)
		if err != nil {
			return nil, nil, err
		}
		return []configloader.YAMLScalarUpdate{
			{Kind: configloader.YAMLScalarString, Path: "database.driver", Value: appconfig.DatabaseDriverSQLite},
			{Kind: configloader.YAMLScalarString, Path: "database.sqlite.path", Value: path},
		}, []string{"database.driver", "database.sqlite.path"}, nil
	case appconfig.DatabaseDriverMySQL:
		host, err := promptConfigStringValue(ctx, "database.mysql.host", cfg.Database.MySQL.Host)
		if err != nil {
			return nil, nil, err
		}
		port, err := promptConfigIntValue(ctx, "database.mysql.port", defaultDatabasePort(driver, cfg.Database.MySQL.Port))
		if err != nil {
			return nil, nil, err
		}
		username, err := promptConfigStringValue(ctx, "database.mysql.username", cfg.Database.MySQL.Username)
		if err != nil {
			return nil, nil, err
		}
		databaseName, err := promptConfigStringValue(ctx, "database.mysql.database", cfg.Database.MySQL.Database)
		if err != nil {
			return nil, nil, err
		}
		updates := []configloader.YAMLScalarUpdate{
			{Kind: configloader.YAMLScalarString, Path: "database.driver", Value: appconfig.DatabaseDriverMySQL},
			{Kind: configloader.YAMLScalarString, Path: "database.mysql.host", Value: host},
			{Kind: configloader.YAMLScalarInt, Path: "database.mysql.port", Value: strconv.Itoa(port)},
			{Kind: configloader.YAMLScalarString, Path: "database.mysql.username", Value: username},
			{Kind: configloader.YAMLScalarString, Path: "database.mysql.database", Value: databaseName},
		}
		paths := []string{"database.driver", "database.mysql.host", "database.mysql.port", "database.mysql.username", "database.mysql.database"}
		return updates, paths, nil
	case appconfig.DatabaseDriverPostgres:
		host, err := promptConfigStringValue(ctx, "database.postgres.host", cfg.Database.Postgres.Host)
		if err != nil {
			return nil, nil, err
		}
		port, err := promptConfigIntValue(ctx, "database.postgres.port", defaultDatabasePort(driver, cfg.Database.Postgres.Port))
		if err != nil {
			return nil, nil, err
		}
		username, err := promptConfigStringValue(ctx, "database.postgres.username", cfg.Database.Postgres.Username)
		if err != nil {
			return nil, nil, err
		}
		databaseName, err := promptConfigStringValue(ctx, "database.postgres.database", cfg.Database.Postgres.Database)
		if err != nil {
			return nil, nil, err
		}
		updates := []configloader.YAMLScalarUpdate{
			{Kind: configloader.YAMLScalarString, Path: "database.driver", Value: appconfig.DatabaseDriverPostgres},
			{Kind: configloader.YAMLScalarString, Path: "database.postgres.host", Value: host},
			{Kind: configloader.YAMLScalarInt, Path: "database.postgres.port", Value: strconv.Itoa(port)},
			{Kind: configloader.YAMLScalarString, Path: "database.postgres.username", Value: username},
			{Kind: configloader.YAMLScalarString, Path: "database.postgres.database", Value: databaseName},
		}
		paths := []string{"database.driver", "database.postgres.host", "database.postgres.port", "database.postgres.username", "database.postgres.database"}
		return updates, paths, nil
	default:
		return nil, nil, fmt.Errorf("unsupported database driver %q", driver)
	}
}

func databaseRequiredPathsFor(cfg *appconfig.Config) []string {
	switch cfg.Database.Driver {
	case appconfig.DatabaseDriverMySQL:
		return []string{"database.driver", "database.mysql.host", "database.mysql.port", "database.mysql.username", "database.mysql.database"}
	case appconfig.DatabaseDriverPostgres:
		return []string{"database.driver", "database.postgres.host", "database.postgres.port", "database.postgres.username", "database.postgres.database"}
	default:
		return []string{"database.driver", "database.sqlite.path"}
	}
}

func defaultDatabasePort(driver string, current int) int {
	if current > 0 {
		return current
	}
	if strings.EqualFold(driver, appconfig.DatabaseDriverMySQL) {
		return 3306
	}
	return 5432
}
