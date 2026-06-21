package commands

import (
	"github.com/rei0721/go-scaffold/internal/app/cliapp/handlers"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/cli"
	"github.com/rei0721/go-scaffold/types/constants"
)

// IAMCommand 提供 IAM 相关的运维子命令。
type IAMCommand struct {
	BootstrapHandler *handlers.IAMBootstrapHandler
}

func NewIAMCommand() *IAMCommand {
	return &IAMCommand{BootstrapHandler: handlers.NewIAMBootstrapHandler()}
}

func (c *IAMCommand) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:        "iam",
		Use:         "iam",
		Description: "Manage IAM users, organizations, roles, and bootstrap tasks",
		Commands: []cli.CommandSpec{
			c.BootstrapAdminSpec(),
		},
	}
}

func (c *IAMCommand) BootstrapAdminSpec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:        "bootstrap-admin",
		Use:         "bootstrap-admin [flags]",
		Description: "Create the initial IAM organization owner",
		Flags: []cli.FlagSpec{
			{Name: "config", Type: cli.FlagTypeString, Default: constants.AppDefaultConfigPath, Description: "Config file path", EnvVar: appconfig.EnvConfigPathName()},
			{Name: "org-code", Type: cli.FlagTypeString, Required: true, Description: "Organization code"},
			{Name: "org-name", Type: cli.FlagTypeString, Description: "Organization name"},
			{Name: "username", Type: cli.FlagTypeString, Required: true, Description: "Admin username"},
			{Name: "email", Type: cli.FlagTypeString, Required: true, Description: "Admin email"},
			{Name: "display-name", Type: cli.FlagTypeString, Description: "Admin display name"},
			{Name: "password", Type: cli.FlagTypeString, Description: "Admin password; prefer --password-stdin in automation"},
			{Name: "password-stdin", Type: cli.FlagTypeBool, Description: "Read admin password from stdin"},
		},
		Run: c.BootstrapHandler.Execute,
	}
}
