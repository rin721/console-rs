package commands

import (
	"github.com/rei0721/go-scaffold/internal/app/cliapp/localization"
	"github.com/rei0721/go-scaffold/internal/app/cliapp/handlers"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/cli"
	"github.com/rei0721/go-scaffold/types/constants"
)

func newRunCommandSpec(localizer *localization.Localizer) cli.CommandSpec {
	handler := handlers.NewRunHandler()
	configFlag := cli.FlagSpec{Name: "config", ShortName: "c", Type: cli.FlagTypeString, Default: constants.AppDefaultConfigPath, Description: localizer.T("cli.flags.config.description"), EnvVar: appconfig.EnvConfigPathName()}
	serviceFlag := cli.FlagSpec{Name: "service", Type: cli.FlagTypeString, Description: localizer.T("cli.run.flags.service.description")}
	yesFlag := cli.FlagSpec{Name: "yes", Type: cli.FlagTypeBool, Default: false, Description: localizer.T("cli.run.flags.yes.description")}
	return cli.CommandSpec{
		Name:        "run",
		Description: localizer.T("cli.run.description"),
		HomeLabel:   localizer.T("cli.run.homeLabel"),
		HomeOrder:   10,
		Flags:       []cli.FlagSpec{configFlag, serviceFlag, yesFlag},
		Run:         handler.Execute,
		Commands: []cli.CommandSpec{
			{
				Name:        constants.AppServerCommandName,
				Use:         "server [--config=<path>]",
				Description: localizer.T("cli.run.server.description"),
				Flags:       []cli.FlagSpec{configFlag},
				Run:         handler.StartServerDirect,
			},
		},
	}
}
