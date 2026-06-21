package commands

import (
	"github.com/rei0721/go-scaffold/internal/app/cliapp/localization"
	"github.com/rei0721/go-scaffold/internal/app/cliapp/handlers"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/cli"
	"github.com/rei0721/go-scaffold/types/constants"
)

func newInitCommandSpec(localizer *localization.Localizer) cli.CommandSpec {
	handler := handlers.NewInitHandler()
	configFlag := cli.FlagSpec{Name: "config", ShortName: "c", Type: cli.FlagTypeString, Default: constants.AppDefaultConfigPath, Description: localizer.T("cli.flags.config.description"), EnvVar: appconfig.EnvConfigPathName()}
	return cli.CommandSpec{
		Name:        "init",
		Use:         "init [flags]",
		Description: localizer.T("cli.init.description"),
		HomeLabel:   localizer.T("cli.init.homeLabel"),
		HomeOrder:   30,
		Flags: []cli.FlagSpec{
			configFlag,
			{Name: "org-code", Type: cli.FlagTypeString, Default: "acme", Description: localizer.T("cli.init.flags.orgCode.description")},
			{Name: "org-name", Type: cli.FlagTypeString, Default: "acme", Description: localizer.T("cli.init.flags.orgName.description")},
			{Name: "admin-username", Type: cli.FlagTypeString, Default: "admin", Description: localizer.T("cli.init.flags.adminUsername.description")},
			{Name: "admin-email", Type: cli.FlagTypeString, Default: "admin@example.com", Description: localizer.T("cli.init.flags.adminEmail.description")},
			{Name: "admin-display-name", Type: cli.FlagTypeString, Default: "admin", Description: localizer.T("cli.init.flags.adminDisplayName.description")},
			{Name: "admin-password", Type: cli.FlagTypeString, Description: localizer.T("cli.init.flags.adminPassword.description")},
			{Name: "admin-password-stdin", Type: cli.FlagTypeBool, Description: localizer.T("cli.init.flags.adminPasswordStdin.description")},
			{Name: "create-service-token", Type: cli.FlagTypeBool, Description: localizer.T("cli.init.flags.createServiceToken.description")},
			{Name: "service-token-days", Type: cli.FlagTypeInt, Default: 30, Description: localizer.T("cli.init.flags.serviceTokenDays.description")},
			{Name: "service-token-remark", Type: cli.FlagTypeString, Default: localizer.T("cli.init.defaults.serviceTokenRemark"), Description: localizer.T("cli.init.flags.serviceTokenRemark.description")},
		},
		Run: handler.Execute,
	}
}
