package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	cliappadapters "github.com/rei0721/go-scaffold/internal/app/cliapp/adapters"
	initservice "github.com/rei0721/go-scaffold/internal/app/cliapp/services/init"
	"github.com/rei0721/go-scaffold/internal/app/cliapp/services/managed"
	"github.com/rei0721/go-scaffold/pkg/cli"
)

type InitializationInput = initservice.InitializationInput
type ProcessInfo = cliappadapters.ProcessInfo
type ProcessStartRequest = cliappadapters.ProcessStartRequest
type ServiceState = managed.ServiceState

const (
	ServiceServer = managed.ServiceServer
	StatusRunning = managed.StatusRunning
	StatusStopped = managed.StatusStopped
)

// TestRunStartFlowUsesStdinBackedCLIUI 固定启动业务流程可以由 pkg/cli 的 stdin UI 驱动。
func TestRunStartFlowUsesStdinBackedCLIUI(t *testing.T) {
	configPath := copyExampleConfig(t)
	var stdout bytes.Buffer
	ctx := &cli.Context{
		Context: context.Background(),
		Flags: map[string]interface{}{
			"config": configPath,
		},
		ChangedFlags: map[string]bool{"config": true},
		Stdout:       &stdout,
		UI:           cli.NewPromptUI(strings.NewReader("2\n"), &stdout),
	}

	if err := RunStartFlow(ctx); err != nil {
		t.Fatalf("RunStartFlow() error = %v", err)
	}
	out := stdout.String()
	for _, want := range []string{"db", "sqlite", "v1"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdin-backed start flow output missing %q:\n%s", want, out)
		}
	}
}

// TestRunServiceFlowUsesStdinBackedCLIUI 固定服务管理流程可以由 pkg/cli 的 stdin UI 驱动。
func TestRunServiceFlowUsesStdinBackedCLIUI(t *testing.T) {
	manager := testManager(t, &fakeProcessRunner{})
	restore := SetManagedManagerFactoryForTest(func() *managed.Manager {
		return manager
	})
	t.Cleanup(restore)

	var stdout bytes.Buffer
	ctx := &cli.Context{
		Context: context.Background(),
		Stdout:  &stdout,
		UI:      cli.NewPromptUI(strings.NewReader("1\n7\n"), &stdout),
	}

	if err := RunServiceFlow(ctx); err != nil {
		t.Fatalf("RunServiceFlow() error = %v", err)
	}
	out := stdout.String()
	for _, want := range []string{ServiceServer, StatusStopped} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdin-backed service flow output missing %q:\n%s", want, out)
		}
	}
}

// TestRunInitializationFlowUsesStdinBackedCLIUI 固定初始化业务流程可以由 pkg/cli 的 stdin UI 收集入参。
func TestRunInitializationFlowUsesStdinBackedCLIUI(t *testing.T) {
	configPath := copyExampleConfig(t)
	oldExecuteInitialization := executeInitialization
	var captured InitializationInput
	executeInitialization = func(_ context.Context, _ io.Writer, input InitializationInput) error {
		captured = input
		return nil
	}
	t.Cleanup(func() {
		executeInitialization = oldExecuteInitialization
	})

	stdin := strings.NewReader(strings.Join([]string{
		"n",
		"n",
		"n",
		"n",
		"orgx",
		"Org X",
		"root",
		"root@example.com",
		"Root User",
		"secret-password",
		"n",
		"y",
		"14",
		"bootstrap token",
	}, "\n") + "\n")
	var stdout bytes.Buffer
	ctx := &cli.Context{
		Context: context.Background(),
		Flags: map[string]interface{}{
			"config": configPath,
		},
		ChangedFlags: map[string]bool{"config": true},
		Stdout:       &stdout,
		UI:           cli.NewPromptUI(stdin, &stdout),
	}

	if err := RunInitializationFlow(ctx, InitializationInput{}); err != nil {
		t.Fatalf("RunInitializationFlow() error = %v", err)
	}
	if captured.ConfigPath != configPath ||
		captured.OrgCode != "orgx" ||
		captured.OrgName != "Org X" ||
		captured.AdminUsername != "root" ||
		captured.AdminEmail != "root@example.com" ||
		captured.AdminDisplayName != "Root User" ||
		captured.AdminPassword != "secret-password" ||
		!captured.CreateServiceToken ||
		captured.ServiceTokenDays != 14 ||
		captured.ServiceTokenRemark != "bootstrap token" {
		t.Fatalf("captured initialization input = %#v", captured)
	}
}

func TestRunInitializationFlowUsesChainAnswers(t *testing.T) {
	configPath := copyExampleConfig(t)
	oldExecuteInitialization := executeInitialization
	var captured InitializationInput
	executeInitialization = func(_ context.Context, _ io.Writer, input InitializationInput) error {
		captured = input
		return nil
	}
	t.Cleanup(func() {
		executeInitialization = oldExecuteInitialization
	})

	var stdout bytes.Buffer
	ctx := &cli.Context{
		Context: context.Background(),
		Stdout:  &stdout,
		UI: cli.WithPromptAnswers(cli.NewPromptUI(strings.NewReader(""), &stdout), map[string]string{
			"config":                   configPath,
			"setup.database.configure": "false",
			"setup.cache.configure":    "false",
			"setup.storage.configure":  "false",
			"setup.system.configure":   "false",
			"setup.site.configure":     "false",
			"org-code":                 "chain-org",
			"org-name":                 "Chain Org",
			"admin-username":           "chain-admin",
			"admin-email":              "chain@example.com",
			"admin-display-name":       "Chain Admin",
			"admin-password":           "chain-password",
			"create-service-token":     "true",
			"service-token-days":       "21",
			"service-token-remark":     "chain token",
		}),
	}

	if err := RunInitializationFlow(ctx, InitializationInput{}); err != nil {
		t.Fatalf("RunInitializationFlow() error = %v", err)
	}
	if captured.ConfigPath != configPath ||
		captured.OrgCode != "chain-org" ||
		captured.OrgName != "Chain Org" ||
		captured.AdminUsername != "chain-admin" ||
		captured.AdminEmail != "chain@example.com" ||
		captured.AdminDisplayName != "Chain Admin" ||
		captured.AdminPassword != "chain-password" ||
		!captured.CreateServiceToken ||
		captured.ServiceTokenDays != 21 ||
		captured.ServiceTokenRemark != "chain token" {
		t.Fatalf("captured initialization input = %#v", captured)
	}
}

func TestRunInitializationFlowCanRestartManagedServerAfterInit(t *testing.T) {
	configPath := copyExampleConfig(t)
	oldExecuteInitialization := executeInitialization
	executeInitialization = func(context.Context, io.Writer, InitializationInput) error {
		return nil
	}
	t.Cleanup(func() {
		executeInitialization = oldExecuteInitialization
	})

	runner := &fakeProcessRunner{
		startInfos:     []ProcessInfo{{PID: 456, ProcessStartTime: 6789}},
		runningResults: []bool{true, true, true, false, true},
	}
	manager := testManager(t, runner)
	if err := writeManagedState(t, manager, ServiceState{Service: ServiceServer, Status: StatusRunning, PID: 123, ProcessStartTime: 4567, ConfigPath: configPath}); err != nil {
		t.Fatalf("write running state: %v", err)
	}
	restoreFlowManager(t, manager)

	var stdout bytes.Buffer
	ctx := &cli.Context{
		Context: context.Background(),
		Stdout:  &stdout,
		UI: cli.WithPromptAnswers(cli.NewPromptUI(strings.NewReader(""), &stdout), map[string]string{
			"config":                   configPath,
			"setup.database.configure": "false",
			"setup.cache.configure":    "false",
			"setup.storage.configure":  "false",
			"setup.system.configure":   "false",
			"setup.site.configure":     "false",
			"org-code":                 "chain-org",
			"org-name":                 "Chain Org",
			"admin-username":           "chain-admin",
			"admin-email":              "chain@example.com",
			"admin-display-name":       "Chain Admin",
			"admin-password":           "chain-password",
			"create-service-token":     "false",
			"init.restart-server":      "true",
		}),
	}

	if err := RunInitializationFlow(ctx, InitializationInput{}); err != nil {
		t.Fatalf("RunInitializationFlow() error = %v", err)
	}
	if len(runner.starts) != 1 {
		t.Fatalf("managed server starts = %d, want 1", len(runner.starts))
	}
}

func TestRunInitializationFlowWarnsInsteadOfPromptingForNonInteractiveInit(t *testing.T) {
	configPath := copyExampleConfig(t)
	oldExecuteInitialization := executeInitialization
	executeInitialization = func(context.Context, io.Writer, InitializationInput) error {
		return nil
	}
	t.Cleanup(func() {
		executeInitialization = oldExecuteInitialization
	})

	runner := &fakeProcessRunner{runningResults: []bool{true}}
	manager := testManager(t, runner)
	if err := writeManagedState(t, manager, ServiceState{Service: ServiceServer, Status: StatusRunning, PID: 123, ProcessStartTime: 4567, ConfigPath: configPath}); err != nil {
		t.Fatalf("write running state: %v", err)
	}
	restoreFlowManager(t, manager)

	var stdout bytes.Buffer
	ctx := &cli.Context{
		Context:      context.Background(),
		Flags:        map[string]interface{}{"config": configPath, "admin-password-stdin": false},
		ChangedFlags: map[string]bool{"config": true, "admin-password": true},
		Stdout:       &stdout,
		UI:           cli.NewPromptUI(strings.NewReader(""), &stdout),
	}

	if err := RunInitializationFlow(ctx, InitializationInput{AdminPassword: "chain-password"}); err != nil {
		t.Fatalf("RunInitializationFlow() error = %v", err)
	}
	if len(runner.starts) != 0 {
		t.Fatalf("managed server starts = %d, want 0", len(runner.starts))
	}
	if !strings.Contains(stdout.String(), "restart") {
		t.Fatalf("expected restart warning, got:\n%s", stdout.String())
	}
}

func RunStartFlow(ctx *cli.Context) error {
	return NewRunHandler().RunStartFlow(ctx)
}

func RunServiceFlow(ctx *cli.Context) error {
	return NewServiceHandler().Execute(ctx)
}

func RunInitializationFlow(ctx *cli.Context, input InitializationInput) error {
	return NewInitHandler().RunInitializationFlow(ctx, input)
}

func testManager(t *testing.T, runner *fakeProcessRunner) *managed.Manager {
	t.Helper()
	return &managed.Manager{
		RuntimeDir: filepath.Join(t.TempDir(), "runtime"),
		Executable: filepath.Join(t.TempDir(), "bin-test"),
		WorkDir:    t.TempDir(),
		Runner:     runner,
		Now: func() time.Time {
			return time.Date(2026, 6, 13, 1, 2, 3, 0, time.UTC)
		},
	}
}

func writeManagedState(t *testing.T, manager *managed.Manager, state ServiceState) error {
	t.Helper()
	path := filepath.Join(manager.RuntimeDir, state.Service, "state.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func copyExampleConfig(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
	raw, err := os.ReadFile(filepath.Join(root, "configs", "config.example.yaml"))
	if err != nil {
		t.Fatalf("read config example: %v", err)
	}
	dir := t.TempDir()
	dbPath := filepath.ToSlash(filepath.Join(dir, "app.db"))
	content := strings.ReplaceAll(string(raw), "  dbname: ./data/app.db", "  dbname: \""+dbPath+"\"")
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

func restoreFlowManager(t *testing.T, manager *managed.Manager) {
	t.Helper()
	restoreHandlers := SetManagedManagerFactoryForTest(func() *managed.Manager {
		return manager
	})
	restoreInit := initservice.SetManagedManagerFactoryForTest(func() *managed.Manager {
		return manager
	})
	t.Cleanup(func() {
		restoreInit()
		restoreHandlers()
	})
}

type fakeProcessRunner struct {
	startInfos     []ProcessInfo
	runningResults []bool
	starts         []ProcessStartRequest
	checks         []ProcessInfo
	kills          []ProcessInfo
}

func (f *fakeProcessRunner) StartProcess(req ProcessStartRequest) (ProcessInfo, error) {
	f.starts = append(f.starts, req)
	if len(f.startInfos) == 0 {
		return ProcessInfo{PID: 100 + len(f.starts), ProcessStartTime: int64(1000 + len(f.starts))}, nil
	}
	info := f.startInfos[0]
	f.startInfos = f.startInfos[1:]
	return info, nil
}

func (f *fakeProcessRunner) IsProcessRunning(info ProcessInfo) (bool, error) {
	f.checks = append(f.checks, info)
	if len(f.runningResults) == 0 {
		return true, nil
	}
	running := f.runningResults[0]
	f.runningResults = f.runningResults[1:]
	return running, nil
}

func (f *fakeProcessRunner) KillProcess(info ProcessInfo) error {
	f.kills = append(f.kills, info)
	return nil
}

func (f *fakeProcessRunner) FindTCPListener(string) (cliappadapters.ProcessDetails, bool, error) {
	return cliappadapters.ProcessDetails{}, false, nil
}
