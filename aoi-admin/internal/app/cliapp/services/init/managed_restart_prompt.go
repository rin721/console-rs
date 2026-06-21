package initservice

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rei0721/go-scaffold/internal/app/cliapp/localization"
	clioutput "github.com/rei0721/go-scaffold/internal/app/cliapp/output"
	"github.com/rei0721/go-scaffold/internal/app/cliapp/services/managed"
	"github.com/rei0721/go-scaffold/pkg/cli"
)

var newManagedManager = managed.NewManager

// SetManagedManagerFactoryForTest 临时替换托管服务管理器工厂，供测试注入假进程运行器。
func SetManagedManagerFactoryForTest(factory func() *managed.Manager) func() {
	previous := newManagedManager
	newManagedManager = factory
	return func() {
		newManagedManager = previous
	}
}

// OfferManagedServerRestartAfterInit 在初始化完成后询问是否启动或重启托管 server。
func OfferManagedServerRestartAfterInit(ctx *cli.Context, configPath string) error {
	if ctx == nil {
		return nil
	}
	manager := newManagedManager()
	localizer := localization.FromContext(ctx)
	state, err := manager.Status(ctx.Context, managed.ServiceServer)
	if err != nil {
		fmt.Fprintln(ctx.Stdout, localizer.T("cli.init.post.statusCheckFailed", map[string]any{"Error": err.Error()}))
		return nil
	}
	if managed.ActiveStatus(state.Status) && sameConfigPath(state.ConfigPath, configPath) {
		return offerManagedServerRestart(ctx, manager)
	}
	return offerManagedServerStart(ctx, manager, configPath)
}

func offerManagedServerRestart(ctx *cli.Context, manager *managed.Manager) error {
	localizer := localization.FromContext(ctx)
	command := executableCommandName()
	if shouldAvoidPostInitPrompt(ctx) {
		fmt.Fprintln(ctx.Stdout, localizer.T("cli.init.post.restartCommand", map[string]any{"Command": command, "Service": managed.ServiceServer}))
		return nil
	}
	restart, err := cli.ConfirmKey(ctx.Context, ctx.UI, "init.restart-server", localizer.T("cli.init.post.restartPrompt"), false)
	if err != nil {
		fmt.Fprintln(ctx.Stdout, localizer.T("cli.init.post.restartCommand", map[string]any{"Command": command, "Service": managed.ServiceServer}))
		return nil
	}
	if !restart {
		fmt.Fprintln(ctx.Stdout, localizer.T("cli.init.post.restartSkipped", map[string]any{"Command": command, "Service": managed.ServiceServer}))
		return nil
	}
	restarted, err := manager.RestartServer(ctx.Context)
	if err != nil {
		return fmt.Errorf("restart managed server after init: %w", err)
	}
	clioutput.PrintServiceState(ctx.Stdout, restarted, localizer)
	return nil
}

func offerManagedServerStart(ctx *cli.Context, manager *managed.Manager, configPath string) error {
	localizer := localization.FromContext(ctx)
	command := executableCommandName()
	if shouldAvoidPostInitPrompt(ctx) {
		fmt.Fprintln(ctx.Stdout, localizer.T("cli.init.post.startCommand", map[string]any{"Command": command, "ConfigPath": configPath}))
		return nil
	}
	start, err := cli.ConfirmKey(ctx.Context, ctx.UI, "init.start-server", localizer.T("cli.init.post.startPrompt"), false)
	if err != nil || !start {
		fmt.Fprintln(ctx.Stdout, localizer.T("cli.init.post.startSkipped", map[string]any{"Command": command, "ConfigPath": configPath}))
		return nil
	}
	started, err := manager.StartServer(ctx.Context, configPath)
	if err != nil {
		return fmt.Errorf("start managed server after init: %w", err)
	}
	clioutput.PrintServiceState(ctx.Stdout, started, localizer)
	return nil
}

func executableCommandName() string {
	name := strings.TrimSpace(filepath.Base(os.Args[0]))
	if name == "" {
		return "aoi"
	}
	return name
}

func shouldAvoidPostInitPrompt(ctx *cli.Context) bool {
	if ctx == nil || ctx.GetBool("yes") || ctx.UI == nil {
		return true
	}
	if _, ok := cli.PromptAnswer(ctx.UI, "init.restart-server"); ok {
		return false
	}
	if _, ok := cli.PromptAnswer(ctx.UI, "init.start-server"); ok {
		return false
	}
	return ctx.IsFlagChanged("admin-password") || ctx.GetBool("admin-password-stdin")
}

func sameConfigPath(left string, right string) bool {
	if strings.TrimSpace(left) == "" || strings.TrimSpace(right) == "" {
		return false
	}
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr == nil && rightErr == nil {
		left = leftAbs
		right = rightAbs
	}
	return strings.EqualFold(left, right)
}
