package handlers

import (
	"encoding/json"

	"github.com/rei0721/go-scaffold/internal/app/cliapp/localization"
	"github.com/rei0721/go-scaffold/internal/app/cliapp/output"
	"github.com/rei0721/go-scaffold/internal/app/cliapp/services/managed"
	"github.com/rei0721/go-scaffold/pkg/cli"
)

// ServiceHandler 处理 service 交互菜单和直接子命令。
type ServiceHandler struct{}

func NewServiceHandler() *ServiceHandler {
	return &ServiceHandler{}
}

func (h *ServiceHandler) Execute(ctx *cli.Context) error {
	ui, err := requireUI(ctx)
	if err != nil {
		return err
	}
	localizer := localization.FromContext(ctx)
	manager := newManagedManager()
	_, singleAction := cli.PromptAnswer(ui, "action")
	for {
		action, err := cli.SelectKey(ctx.Context, ui, "action", localizer.T("cli.service.prompts.action"), []cli.SelectOption{
			{Value: "status", Label: localizer.T("cli.service.actions.status")},
			{Value: "info", Label: localizer.T("cli.service.actions.info")},
			{Value: "logs", Label: localizer.T("cli.service.actions.logs")},
			{Value: "terminal", Label: localizer.T("cli.service.actions.terminal")},
			{Value: "restart", Label: localizer.T("cli.service.actions.restart")},
			{Value: "stop", Label: localizer.T("cli.service.actions.stop")},
			{Value: "back", Label: localizer.T("cli.service.actions.back")},
		})
		if err != nil {
			return err
		}
		switch action {
		case "status":
			state, err := manager.Status(ctx.Context, managed.ServiceServer)
			if err != nil {
				return err
			}
			output.PrintServiceState(ctx.Stdout, state, localizer)
		case "info":
			state, err := manager.Status(ctx.Context, managed.ServiceServer)
			if err != nil {
				return err
			}
			output.PrintServiceState(ctx.Stdout, state, localizer)
		case "logs":
			state, err := manager.Status(ctx.Context, managed.ServiceServer)
			if err != nil {
				return err
			}
			follow, err := cli.ConfirmKey(ctx.Context, ui, "logs.follow", localizer.T("cli.service.prompts.followLogs"), false)
			if err != nil {
				return err
			}
			if err := output.PrintServiceLogs(ctx.Context, ctx.Stdout, state, 100, follow); err != nil {
				return err
			}
		case "terminal":
			state, err := manager.Status(ctx.Context, managed.ServiceServer)
			if err != nil {
				return err
			}
			if err := output.PrintServiceLogs(ctx.Context, ctx.Stdout, state, 100, true); err != nil {
				return err
			}
		case "restart":
			state, err := manager.RestartServer(ctx.Context)
			if err != nil {
				return err
			}
			output.PrintServiceState(ctx.Stdout, state, localizer)
		case "stop":
			state, err := manager.StopServer(ctx.Context)
			if err != nil {
				return err
			}
			output.PrintServiceState(ctx.Stdout, state, localizer)
		case "back":
			return nil
		}
		if singleAction {
			return nil
		}
	}
}

func (h *ServiceHandler) Status(ctx *cli.Context) error {
	localizer := localization.FromContext(ctx)
	state, err := newManagedManager().Status(ctx.Context, managed.ServiceServer)
	if err != nil {
		return err
	}
	output.PrintServiceState(ctx.Stdout, state, localizer)
	return nil
}

func (h *ServiceHandler) Info(ctx *cli.Context) error {
	state, err := newManagedManager().Status(ctx.Context, managed.ServiceServer)
	if err != nil {
		return err
	}
	return json.NewEncoder(ctx.Stdout).Encode(state)
}

func (h *ServiceHandler) Logs(ctx *cli.Context) error {
	state, err := newManagedManager().Status(ctx.Context, managed.ServiceServer)
	if err != nil {
		return err
	}
	return output.PrintServiceLogs(ctx.Context, ctx.Stdout, state, ctx.GetInt("lines"), ctx.GetBool("follow"))
}

func (h *ServiceHandler) Terminal(ctx *cli.Context) error {
	state, err := newManagedManager().Status(ctx.Context, managed.ServiceServer)
	if err != nil {
		return err
	}
	return output.PrintServiceLogs(ctx.Context, ctx.Stdout, state, ctx.GetInt("lines"), true)
}

func (h *ServiceHandler) Restart(ctx *cli.Context) error {
	localizer := localization.FromContext(ctx)
	state, err := newManagedManager().RestartServer(ctx.Context)
	if err != nil {
		return err
	}
	output.PrintServiceState(ctx.Stdout, state, localizer)
	return nil
}

func (h *ServiceHandler) Stop(ctx *cli.Context) error {
	localizer := localization.FromContext(ctx)
	state, err := newManagedManager().StopServer(ctx.Context)
	if err != nil {
		return err
	}
	output.PrintServiceState(ctx.Stdout, state, localizer)
	return nil
}
