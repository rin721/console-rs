package commands

import (
	"fmt"

	"github.com/rei0721/go-scaffold/internal/app/cliapp/handlers"
	servicedb "github.com/rei0721/go-scaffold/internal/app/cliapp/services/db"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/cli"
	"github.com/rei0721/go-scaffold/types/constants"
)

const dbCommandName = "db"

// DBCommand 暴露数据库 DDL 和迁移运维命令。
type DBCommand struct {
	Handler          *handlers.DBHandler
	MigrationHandler *handlers.DBMigrationHandler
}

func NewDBCommand() *DBCommand {
	return &DBCommand{
		Handler:          handlers.NewDBHandler(),
		MigrationHandler: handlers.NewDBMigrationHandler(),
	}
}

func (c *DBCommand) Name() string {
	return dbCommandName
}

func (c *DBCommand) Description() string {
	return "Run sqlgen-powered database DDL and migrations"
}

func (c *DBCommand) Usage() string {
	return fmt.Sprintf("%s [--operation=<database>] [flags]", dbCommandName)
}

func (c *DBCommand) Flags() []cli.FlagSpec {
	return []cli.FlagSpec{
		{Name: "config", Type: cli.FlagTypeString, Default: constants.AppDefaultConfigPath, Description: "Config file path", EnvVar: appconfig.EnvConfigPathName()},
		{Name: "operation", Type: cli.FlagTypeString, Default: servicedb.DefaultOperation, Description: "Database operation"},
		{Name: "apply", Type: cli.FlagTypeBool, Default: false, Description: "Apply generated SQL instead of printing it"},
		{Name: "print-sql", Type: cli.FlagTypeBool, Default: false, Description: "Print generated SQL after executing the operation"},
	}
}

func (c *DBCommand) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:        c.Name(),
		Use:         c.Usage(),
		Description: c.Description(),
		Flags:       c.Flags(),
		Run:         c.Handler.Execute,
		Commands: []cli.CommandSpec{
			c.MigrateSpec(),
		},
	}
}

func (c *DBCommand) MigrateSpec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:        "migrate",
		Use:         "migrate <up|down|status>",
		Description: "Run database migrations",
		Flags: []cli.FlagSpec{
			{Name: "config", Type: cli.FlagTypeString, Default: constants.AppDefaultConfigPath, Description: "Config file path", EnvVar: appconfig.EnvConfigPathName()},
		},
		Args: func(ctx *cli.Context) error {
			if len(ctx.Args) != 1 {
				return &cli.UsageError{Command: ctx.CommandPath, Message: "expected migrate operation: up, down, or status"}
			}
			switch ctx.Args[0] {
			case "up", "down", "status":
				return nil
			default:
				return &cli.UsageError{Command: ctx.CommandPath, Message: "unsupported migrate operation: " + ctx.Args[0]}
			}
		},
		Run: c.MigrationHandler.Execute,
	}
}
