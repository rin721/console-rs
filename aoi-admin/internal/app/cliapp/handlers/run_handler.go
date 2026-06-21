package handlers

import (
	"context"
	"fmt"
	"io"
	"strings"

	cliconfig "github.com/rei0721/go-scaffold/internal/app/cliapp/config"
	"github.com/rei0721/go-scaffold/internal/app/cliapp/localization"
	clioutput "github.com/rei0721/go-scaffold/internal/app/cliapp/output"
	initservice "github.com/rei0721/go-scaffold/internal/app/cliapp/services/init"
	"github.com/rei0721/go-scaffold/internal/app/cliapp/services/managed"
	"github.com/rei0721/go-scaffold/internal/app/initcenter"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/cli"
	"github.com/rei0721/go-scaffold/types/constants"
)

// RunHandler 处理 run 启动向导。
type RunHandler struct{}

func NewRunHandler() *RunHandler {
	return &RunHandler{}
}

func (h *RunHandler) Execute(ctx *cli.Context) error {
	if RunDirectRequested(ctx) {
		answers, err := RunShortcutAnswersFromContext(ctx)
		if err != nil {
			return err
		}
		ctx.UI = cli.WithPromptAnswers(ctx.UI, answers)
	}
	return h.RunStartFlow(ctx)
}

func (h *RunHandler) RunStartFlow(ctx *cli.Context) error {
	if ctx == nil {
		return fmt.Errorf("cli context is required")
	}
	if ctx.Context == nil {
		ctx.Context = context.Background()
	}
	ui, err := requireUI(ctx)
	if err != nil {
		return err
	}
	localizer := localization.FromContext(ctx)

	service, err := cli.SelectKey(ctx.Context, ui, "service", localizer.T("cli.run.prompts.service"), startServiceOptions(localizer))
	if err != nil {
		return err
	}
	service = normalizeStartService(service)
	if !isSupportedStartService(service) {
		return fmt.Errorf("unsupported service %q; expected one of: server, db, iam, cache, storage", service)
	}

	configPath, err := cliconfig.SelectConfigPath(ctx)
	if err != nil {
		return err
	}
	cfg, repairedConfig, err := cliconfig.PreflightConfigForStart(ctx, configPath)
	if err != nil {
		return err
	}
	if err := cliconfig.PrintConfigSummary(ctx.Stdout, configPath, localizer); err != nil {
		return err
	}
	if service != managed.ServiceServer {
		return clioutput.PrintDependencyServiceInfo(ctx.Stdout, service, configPath, localizer)
	}
	handled, err := h.handleInitializationBeforeServer(ctx, ui, configPath, cfg)
	if handled || err != nil {
		return err
	}
	if repairedConfig {
		return h.startServer(ctx, configPath)
	}
	ok, err := cli.ConfirmKey(ctx.Context, ui, "privacy", localizer.T("cli.run.prompts.privacy"), false)
	if err != nil {
		return err
	}
	if !ok {
		return h.startServer(ctx, configPath)
	}
	if cliconfig.IsExampleConfig(configPath) {
		return fmt.Errorf("%s", localizer.T("cli.run.errors.exampleConfigReadonly"))
	}
	updates, err := cliconfig.PromptPrivacyUpdates(ctx, configPath)
	if err != nil {
		return err
	}
	if updates.HasChanges() {
		if err := cliconfig.ApplyPrivacyRuntimeEnvOnly(configPath, updates.RuntimeEnvOnlyPaths); err != nil {
			return err
		}
		if err := cliconfig.ApplyPrivacyUpdates(configPath, updates.FileUpdates); err != nil {
			return err
		}
		if err := cliconfig.ApplyPrivacyUpdates(configPath, updates.ForceFileUpdates, appconfig.WithEnvManagedPersistMode(appconfig.EnvManagedPersistForceFile)); err != nil {
			return err
		}
		_ = ui.Info(localizer.T("cli.run.info.privacyHandled"))
	}
	return h.startServer(ctx, configPath)
}

func (h *RunHandler) handleInitializationBeforeServer(ctx *cli.Context, ui cli.PromptUI, configPath string, cfg *appconfig.Config) (bool, error) {
	localizer := localization.FromContext(ctx)
	status, err := initservice.InspectInitializationStatus(ctx.Context, configPath)
	if err != nil {
		return false, nil
	}
	if !status.Required {
		return false, nil
	}
	if ctx.GetBool("yes") {
		_ = ui.Info(localizer.T("cli.run.info.uninitializedWithYes"))
		return false, nil
	}
	choice, err := cli.SelectKey(ctx.Context, ui, "initialization.mode", localizer.T("cli.run.initialization.prompt"), []cli.SelectOption{
		{Value: "cli", Label: localizer.T("cli.run.initialization.option.cli.label"), Description: localizer.T("cli.run.initialization.option.cli.description")},
		{Value: "web", Label: localizer.T("cli.run.initialization.option.web.label"), Description: localizer.T("cli.run.initialization.option.web.description")},
		{Value: "continue", Label: localizer.T("cli.run.initialization.option.continue.label"), Description: localizer.T("cli.run.initialization.option.continue.description")},
		{Value: "status", Label: localizer.T("cli.run.initialization.option.status.label"), Description: localizer.T("cli.run.initialization.option.status.description")},
		{Value: "skip", Label: localizer.T("cli.run.initialization.option.skip.label"), Description: localizer.T("cli.run.initialization.option.skip.description")},
	})
	if err != nil {
		return true, err
	}
	switch choice {
	case "cli", "continue":
		return true, NewInitHandler().runInitializationFlowWithConfigPath(ctx, initservice.InitializationInput{ConfigPath: configPath}, configPath)
	case "web":
		state, err := newManagedManager().StartServer(ctx.Context, configPath)
		if err != nil {
			return true, err
		}
		clioutput.PrintServiceState(ctx.Stdout, state, localizer)
		_, _ = fmt.Fprintln(ctx.Stdout, localizer.T("cli.run.setupUrl", map[string]any{"URL": setupURL(cfg)}))
		return true, nil
	case "status":
		printInitializationStatus(ctx.Stdout, status, localizer)
		return true, nil
	case "skip":
		_ = ui.Info(localizer.T("cli.run.initialization.skipped"))
		return false, nil
	default:
		return true, fmt.Errorf("unsupported initialization choice %q", choice)
	}
}

func printInitializationStatus(stdout io.Writer, status initcenter.Status, localizer *localization.Localizer) {
	if stdout == nil {
		return
	}
	current := strings.TrimSpace(status.CurrentStep)
	if current == "" {
		current = localizer.T("cli.run.initialization.status.notStarted")
	}
	_, _ = fmt.Fprintln(stdout, localizer.T("cli.run.initialization.status.line", map[string]any{"Required": status.Required, "Completed": status.Completed, "CurrentStep": current}))
	if status.RestartRequired {
		_, _ = fmt.Fprintln(stdout, localizer.T("cli.run.initialization.status.restartRequired", map[string]any{"Reason": strings.TrimSpace(status.RestartReason)}))
	}
	for _, step := range status.Steps {
		if step.Status != initcenter.StepStatusFailed {
			continue
		}
		_, _ = fmt.Fprintln(stdout, "- "+localizer.T("cli.run.initialization.status.failedStep", map[string]any{"Title": step.Title, "Key": step.Key}))
		if step.ErrorMessage != "" {
			_, _ = fmt.Fprintln(stdout, "  "+localizer.T("cli.run.initialization.status.reason", map[string]any{"Reason": step.ErrorMessage}))
		}
		if step.RepairHint != "" {
			_, _ = fmt.Fprintln(stdout, "  "+localizer.T("cli.run.initialization.status.repairHint", map[string]any{"Hint": step.RepairHint}))
		}
	}
	if status.Report != nil {
		report := status.Report
		_, _ = fmt.Fprintln(stdout, localizer.T("cli.run.initialization.status.report", map[string]any{"Successful": report.Successful, "Failed": report.Failed, "Skipped": report.Skipped, "Risk": report.Risk}))
	}
}

func (h *RunHandler) startServer(ctx *cli.Context, configPath string) error {
	state, err := newManagedManager().StartServer(ctx.Context, configPath)
	if err != nil {
		return err
	}
	clioutput.PrintServiceState(ctx.Stdout, state, localization.FromContext(ctx))
	return nil
}

func (h *RunHandler) StartServerDirect(ctx *cli.Context) error {
	return h.startServer(ctx, ctx.GetString("config"))
}

func RunDirectRequested(ctx *cli.Context) bool {
	return ctx.GetBool("yes") || ctx.IsFlagChanged("service")
}

func RunShortcutAnswersFromContext(ctx *cli.Context) (map[string]string, error) {
	answers := map[string]string{}
	service := strings.ToLower(strings.TrimSpace(ctx.GetString("service")))
	if service == "" {
		service = managed.ServiceServer
	}
	switch service {
	case managed.ServiceServer, "db", "iam", "cache", "storage":
	default:
		return nil, &cli.UsageError{
			Command: ctx.CommandPath,
			Message: fmt.Sprintf("unsupported --service %q; expected one of: server, db, iam, cache, storage", ctx.GetString("service")),
		}
	}
	answers["service"] = service

	configPath := ctx.GetString("config")
	if ctx.IsFlagChanged("config") || strings.TrimSpace(ctx.GetString("config")) != constants.AppDefaultConfigPath {
		answers["config"] = configPath
	} else {
		answers["config"] = DefaultRunConfigPath()
	}
	if ctx.GetBool("yes") {
		answers["privacy"] = "false"
	}
	return answers, nil
}

func DefaultRunConfigPath() string {
	files := cliconfig.DiscoverConfigFiles()
	if len(files) > 0 {
		return files[0]
	}
	return constants.AppDefaultConfigPath
}

func setupURL(cfg *appconfig.Config) string {
	if cfg == nil {
		return "http://127.0.0.1:9999/admin/setup"
	}
	webui := cfg.WebUI
	webui.ApplyDefaults()
	host := strings.TrimSpace(cfg.Server.Host)
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("http://%s:%d%s/setup", host, cfg.Server.Port, strings.TrimRight(webui.MountPath, "/"))
}

func startServiceOptions(localizer *localization.Localizer) []cli.SelectOption {
	return []cli.SelectOption{
		{Value: managed.ServiceServer, Label: "server", Description: localizer.T("cli.run.service.server.description")},
		{Value: "db", Label: "db", Description: localizer.T("cli.run.service.db.description")},
		{Value: "iam", Label: "iam", Description: localizer.T("cli.run.service.iam.description")},
		{Value: "cache", Label: "cache", Description: localizer.T("cli.run.service.cache.description")},
		{Value: "storage", Label: "storage", Description: localizer.T("cli.run.service.storage.description")},
	}
}

func normalizeStartService(service string) string {
	return strings.ToLower(strings.TrimSpace(service))
}

func isSupportedStartService(service string) bool {
	switch normalizeStartService(service) {
	case managed.ServiceServer, "db", "iam", "cache", "storage":
		return true
	default:
		return false
	}
}
