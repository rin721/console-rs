package commands

import (
	"github.com/rei0721/go-scaffold/internal/app/cliapp/localization"
	"github.com/rei0721/go-scaffold/internal/app/cliapp/handlers"
	"github.com/rei0721/go-scaffold/internal/app/cliapp/validators"
	"github.com/rei0721/go-scaffold/pkg/cli"
)

func newServiceCommandSpec(localizer *localization.Localizer) cli.CommandSpec {
	handler := handlers.NewServiceHandler()
	linesFlag := cli.FlagSpec{Name: "lines", Type: cli.FlagTypeInt, Default: 100, Description: localizer.T("cli.service.flags.lines.description")}
	followFlag := cli.FlagSpec{Name: "follow", ShortName: "f", Type: cli.FlagTypeBool, Default: false, Description: localizer.T("cli.service.flags.follow.description")}
	return cli.CommandSpec{
		Name:        "service",
		Description: localizer.T("cli.service.description"),
		HomeLabel:   localizer.T("cli.service.homeLabel"),
		HomeOrder:   20,
		Run:         handler.Execute,
		Commands: []cli.CommandSpec{
			serviceStatusSpec(handler, localizer),
			serviceInfoSpec(handler, localizer),
			{
				Name:        "logs",
				Use:         "logs [server]",
				Description: localizer.T("cli.service.logs.description"),
				Flags:       []cli.FlagSpec{linesFlag, followFlag},
				Args:        validators.ValidateOptionalServerArg,
				Run:         handler.Logs,
			},
			{
				Name:        "terminal",
				Use:         "terminal [server]",
				Description: localizer.T("cli.service.terminal.description"),
				Flags:       []cli.FlagSpec{linesFlag},
				Args:        validators.ValidateOptionalServerArg,
				Run:         handler.Terminal,
			},
			{
				Name:        "restart",
				Use:         "restart [server]",
				Description: localizer.T("cli.service.restart.description"),
				Args:        validators.ValidateOptionalServerArg,
				Run:         handler.Restart,
			},
			{
				Name:        "stop",
				Use:         "stop [server]",
				Description: localizer.T("cli.service.stop.description"),
				Args:        validators.ValidateOptionalServerArg,
				Run:         handler.Stop,
			},
		},
	}
}

func serviceStatusSpec(handler *handlers.ServiceHandler, localizer *localization.Localizer) cli.CommandSpec {
	return cli.CommandSpec{
		Name:        "status",
		Use:         "status [server]",
		Description: localizer.T("cli.service.status.description"),
		Args:        validators.ValidateOptionalServerArg,
		Run:         handler.Status,
	}
}

func serviceInfoSpec(handler *handlers.ServiceHandler, localizer *localization.Localizer) cli.CommandSpec {
	return cli.CommandSpec{
		Name:        "info",
		Use:         "info [server]",
		Description: localizer.T("cli.service.info.description"),
		Args:        validators.ValidateOptionalServerArg,
		Run:         handler.Info,
	}
}
