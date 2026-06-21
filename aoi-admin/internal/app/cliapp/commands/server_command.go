package commands

import (
	"fmt"

	"github.com/rei0721/go-scaffold/internal/app/cliapp/handlers"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/cli"
	"github.com/rei0721/go-scaffold/types/constants"
)

// ServerCommand 表示启动 HTTP 服务的 CLI 子命令。
type ServerCommand struct {
	Handler *handlers.ServerHandler
}

func NewServerCommand() *ServerCommand {
	return &ServerCommand{Handler: handlers.NewServerHandler()}
}

func (c *ServerCommand) Name() string {
	return constants.AppServerCommandName
}

func (c *ServerCommand) Description() string {
	return "Run server"
}

func (c *ServerCommand) Usage() string {
	return fmt.Sprintf("%s [--config=<name>]", constants.AppServerCommandName)
}

func (c *ServerCommand) Flags() []cli.FlagSpec {
	return []cli.FlagSpec{
		{
			Name:        "config",
			ShortName:   "c",
			Type:        cli.FlagTypeString,
			Required:    false,
			Default:     constants.AppDefaultConfigPath,
			Description: "Config file path",
			EnvVar:      appconfig.EnvConfigPathName(),
		},
	}
}

func (c *ServerCommand) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:        c.Name(),
		Use:         c.Usage(),
		Description: c.Description(),
		Flags:       c.Flags(),
		Run:         c.Handler.Execute,
	}
}
