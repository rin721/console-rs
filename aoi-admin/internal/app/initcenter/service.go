package initcenter

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rei0721/go-scaffold/internal/app/initapp"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
	iammodel "github.com/rei0721/go-scaffold/internal/modules/iam/model"
	iamservice "github.com/rei0721/go-scaffold/internal/modules/iam/service"
)

var (
	ErrSetupUnauthorized       = errors.New("setup token is required after initial setup")
	ErrInitializationRunAbsent = errors.New("initialization run not found")
)

type Service struct {
	core       initapp.Core
	infra      initapp.Infrastructure
	modules    initapp.Modules
	configPath string
	stdout     io.Writer
	store      *stateStore
	registry   *InitTaskRegistry
	mu         sync.Mutex
}

func New(core initapp.Core, infra initapp.Infrastructure, modules initapp.Modules, configPath string, stdout io.Writer) *Service {
	service := &Service{
		core:       core,
		infra:      infra,
		modules:    modules,
		configPath: configPath,
		stdout:     stdout,
		store:      newStateStore(infra.Database, core.IDGenerator),
	}
	registry := NewInitTaskRegistry()
	for _, def := range service.baseDefinitions() {
		_ = registry.Register(taskAdapter{def: def})
	}
	service.registry = registry
	return service
}

func (s *Service) Status(ctx context.Context) (Status, error) {
	diagnostics := s.configDiagnostics()
	iamStatus, err := s.iamSetupStatus(ctx)
	if err != nil {
		return Status{}, err
	}
	var latest *runRecord
	var records []stepRecord
	if s.store != nil {
		latest, records, err = s.store.latestRun(ctx)
		if err != nil {
			return Status{}, err
		}
	}
	steps := s.stepReports(ctx, Input{}, records)
	required := len(diagnostics) > 0 || iamStatus.Required
	completed := !required
	currentStep := firstPendingStep(steps)
	if latest != nil && latest.Status == string(RunStatusFailed) {
		currentStep = latest.CurrentStep
	}
	bootstrap, restartRequired := s.pendingRestart()
	restartReason := ""
	if restartRequired {
		restartReason = normalizeRestartReason(bootstrap.RestartReason)
		if currentStep == "" {
			currentStep = bootstrap.CurrentStep
		}
	}
	report := reportFromSteps(steps, restartRequired, restartReason)
	return Status{
		Required:        required,
		Completed:       completed,
		CurrentStep:     currentStep,
		AllowedActions:  allowedActions(required, completed),
		Diagnostics:     diagnostics,
		RestartRequired: restartRequired,
		RestartReason:   restartReason,
		PasswordPolicy:  iamStatus.PasswordPolicy,
		Steps:           steps,
		LastRun:         optionalRunReport(latest),
		Report:          &report,
	}, nil
}

func (s *Service) Schema(ctx context.Context) (SetupSchema, error) {
	steps := make([]StepSchema, 0, len(s.definitions()))
	for _, def := range s.definitions() {
		schema := def.Schema
		if schema.Key == "" {
			schema = stepSchema(def.Key, def.Phase, def.Order, def.Title, def.Goal, def.Required, def.Skippable, def.Testable, def.Dependencies, nil)
		}
		if schema.RouteSlug == "" {
			schema.RouteSlug = routeSlugForStep(schema.Key)
		}
		schema.Required = def.Required
		schema.Skippable = def.Skippable
		schema.Testable = def.Testable
		schema.Dependencies = nonNilStrings(def.Dependencies)
		schema.Fields = s.hydrateSchemaFields(schema.Fields)
		schema.Groups = s.hydrateSchemaGroups(schema.Groups)
		schema.InputFingerprint = inputFingerprintFor(schema.Key, schemaVisibleValues(schema))
		steps = append(steps, schema)
	}
	return SetupSchema{Steps: steps}, nil
}

func (s *Service) Run(ctx context.Context, input Input) (RunResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	input.Source = defaultSource(input.Source)
	input.Mode = defaultMode(input.Mode)
	if err := s.authorizeRun(ctx, input); err != nil {
		return RunResult{}, err
	}
	if s.store != nil {
		if err := s.store.ensure(ctx); err != nil {
			return RunResult{}, err
		}
	}

	var run *runRecord
	if s.store != nil {
		var err error
		run, err = s.store.createRun(ctx, input, s.configPath, s.configFingerprint())
		if err != nil {
			return RunResult{}, err
		}
	}
	exec := &execution{service: s, input: input, run: run}
	for _, def := range s.definitions() {
		if err := s.runStep(ctx, exec, def); err != nil {
			if run != nil {
				now := time.Now().UTC()
				run.Status = string(RunStatusFailed)
				run.CurrentStep = def.Key
				run.LastError = err.Error()
				run.FinishedAt = &now
				_ = s.store.saveRun(ctx, run)
			}
			return s.result(ctx, exec), err
		}
	}
	if run != nil {
		now := time.Now().UTC()
		run.Status = string(RunStatusSucceeded)
		run.CurrentStep = ""
		run.LastError = ""
		run.FinishedAt = &now
		if err := s.store.saveRun(ctx, run); err != nil {
			return s.result(ctx, exec), err
		}
	}
	return s.result(ctx, exec), nil
}

func (s *Service) Retry(ctx context.Context, runKey string, input Input) (RunResult, error) {
	if s.store != nil {
		if err := s.store.requireRun(ctx, runKey); err != nil {
			return RunResult{}, err
		}
	}
	input.Mode = ModeRepair
	if input.Source == "" {
		input.Source = SourceWeb
	}
	return s.Run(ctx, input)
}

func (s *Service) Logs(ctx context.Context, runKey string) ([]StepReport, error) {
	if s.store == nil {
		return nil, nil
	}
	if err := s.store.ensure(ctx); err != nil {
		return nil, err
	}
	var records []stepRecord
	if strings.TrimSpace(runKey) == "" {
		_, latest, err := s.store.latestRun(ctx)
		if err != nil {
			return nil, err
		}
		records = latest
	} else {
		_, found, err := s.store.findRun(ctx, runKey)
		if err != nil {
			return nil, err
		}
		records = found
	}
	return s.stepReports(ctx, Input{}, records), nil
}

func (s *Service) SaveConfig(ctx context.Context, stepKey string, input Input, values map[string]any, persist bool, runTest bool) (ConfigSaveResult, error) {
	if err := s.authorizeRun(ctx, input); err != nil {
		return ConfigSaveResult{}, err
	}
	def, ok := s.definition(stepKey)
	if !ok {
		return ConfigSaveResult{}, fmt.Errorf("unknown initialization step %s", stepKey)
	}
	var test *TestResult
	if runTest || def.Testable {
		result, err := s.TestConfig(ctx, stepKey, input, values)
		if err != nil {
			return ConfigSaveResult{}, err
		}
		test = &result
		if result.Status == "failed" {
			return ConfigSaveResult{
				StepKey:          stepKey,
				InputSummary:     summarizeInput(stepKey, values),
				InputFingerprint: result.InputFingerprint,
				Test:             test,
			}, errors.New(result.Error)
		}
	}
	store := NewInitConfigStore(s.core, s.configPath)
	saved, err := store.Save(ctx, stepKey, values, persist)
	if err != nil {
		return ConfigSaveResult{}, err
	}
	saved.Test = test
	if saved.InputFingerprint == "" {
		saved.InputFingerprint = inputFingerprintFor(stepKey, values)
	}
	if s.store != nil {
		if err := s.store.ensure(ctx); err != nil {
			return ConfigSaveResult{}, err
		}
		run, err := s.store.latestOrCreateRun(ctx, input, s.configPath, s.configFingerprint())
		if err != nil {
			return ConfigSaveResult{}, err
		}
		_, _ = s.store.upsertStep(ctx, run, def, StepStatusPending, 0, saved.InputSummary, nil)
		records, _ := s.store.steps(ctx, run.ID)
		saved.Steps = s.stepReports(ctx, input, records)
	}
	return saved, nil
}

func (s *Service) TestConfig(ctx context.Context, stepKey string, input Input, values map[string]any) (TestResult, error) {
	if err := s.authorizeRun(ctx, input); err != nil {
		return TestResult{}, err
	}
	def, ok := s.definition(stepKey)
	if !ok {
		return TestResult{}, fmt.Errorf("unknown initialization step %s", stepKey)
	}
	if !def.Testable {
		return TestResult{StepKey: stepKey, Status: "skipped", SummaryKey: "ui.setup.tests.skipped", InputFingerprint: inputFingerprintFor(stepKey, values), TestedAt: time.Now().UTC()}, nil
	}
	test := NewInitValidator(s).Test(ctx, stepKey, values)
	test.InputFingerprint = inputFingerprintFor(stepKey, values)
	if s.store != nil {
		if err := s.store.ensure(ctx); err != nil {
			return TestResult{}, err
		}
		run, err := s.store.latestOrCreateRun(ctx, input, s.configPath, s.configFingerprint())
		if err != nil {
			return TestResult{}, err
		}
		if err := s.store.updateStepTest(ctx, run, def, test); err != nil {
			return TestResult{}, err
		}
	}
	return test, nil
}

func (s *Service) SkipStep(ctx context.Context, runKey string, stepKey string, reason string, input Input) (RunResult, error) {
	if err := s.authorizeRun(ctx, input); err != nil {
		return RunResult{}, err
	}
	def, ok := s.definition(stepKey)
	if !ok {
		return RunResult{}, fmt.Errorf("unknown initialization step %s", stepKey)
	}
	if !def.Skippable {
		return RunResult{}, fmt.Errorf("initialization step %s cannot be skipped", stepKey)
	}
	if s.store == nil {
		return RunResult{Steps: s.stepReports(ctx, input, nil)}, nil
	}
	if err := s.store.ensure(ctx); err != nil {
		return RunResult{}, err
	}
	var run *runRecord
	var err error
	if strings.TrimSpace(runKey) != "" {
		run, _, err = s.store.findRun(ctx, runKey)
	} else {
		run, err = s.store.latestOrCreateRun(ctx, input, s.configPath, s.configFingerprint())
	}
	if err != nil {
		return RunResult{}, err
	}
	if err := s.store.skipStep(ctx, run, def, reason); err != nil {
		return RunResult{}, err
	}
	return s.result(ctx, &execution{service: s, input: input, run: run}), nil
}

func (s *Service) Complete(ctx context.Context, input Input) (CompleteResult, error) {
	if err := s.authorizeRun(ctx, input); err != nil {
		return CompleteResult{}, err
	}
	var records []stepRecord
	var latest *runRecord
	var err error
	if s.store != nil {
		if err := s.store.ensure(ctx); err != nil {
			return CompleteResult{}, err
		}
		latest, records, err = s.store.latestRun(ctx)
		if err != nil {
			return CompleteResult{}, err
		}
		if latest != nil {
			now := time.Now().UTC()
			latest.Status = string(RunStatusSucceeded)
			latest.CurrentStep = ""
			latest.LastError = ""
			latest.FinishedAt = &now
			if err := s.store.saveRun(ctx, latest); err != nil {
				return CompleteResult{}, err
			}
		}
	}
	steps := s.stepReports(ctx, input, records)
	bootstrap, restartRequired := s.pendingRestart()
	if !restartRequired {
		clearBootstrapState()
	}
	report := reportFromSteps(steps, restartRequired, normalizeRestartReason(bootstrap.RestartReason))
	return CompleteResult{Completed: report.Failed == 0 && !report.RestartRequired, Report: report, Steps: steps}, nil
}

func (s *Service) IAMSetupService() IAMSetupService {
	return iamSetupAdapter{center: s}
}

func (s *Service) AuthorizeSetupRead(ctx context.Context, setupToken string) error {
	return s.authorizeSetupToken(ctx, setupToken)
}

func (s *Service) runStep(ctx context.Context, exec *execution, def stepDefinition) error {
	if s.store != nil && exec.run != nil {
		exec.run.CurrentStep = def.Key
		_ = s.store.saveRun(ctx, exec.run)
	}
	if s.store != nil {
		if _, err := s.store.upsertStep(ctx, exec.run, def, StepStatusRunning, 1, "", nil); err != nil {
			return err
		}
	}

	failStep := func(summary string, stepErr error) {
		if s.store != nil {
			_, _ = s.store.upsertStep(ctx, exec.run, def, StepStatusFailed, 0, summary, stepErr)
		}
	}
	finishStep := func(status StepStatus, summary string) error {
		if s.store == nil {
			return nil
		}
		_, err := s.store.upsertStep(ctx, exec.run, def, status, 0, summary, nil)
		return err
	}

	if def.Check != nil {
		status, summary, err := def.Check(ctx, exec)
		if err != nil {
			failStep(summary, err)
			return err
		}
		if status == StepStatusSucceeded || status == StepStatusSkipped {
			err = finishStep(status, summary)
			s.printf("%s: %s\n", def.Title, summary)
			return err
		}
	}

	outcome, err := def.Apply(ctx, exec)
	status := outcome.Status
	if status == "" {
		status = StepStatusSucceeded
	}
	if err != nil {
		failStep(outcome.Summary, err)
		return err
	}
	err = finishStep(status, outcome.Summary)
	s.printf("%s: %s\n", def.Title, outcome.Summary)
	return err
}

func (s *Service) result(ctx context.Context, exec *execution) RunResult {
	var steps []StepReport
	if s.store != nil && exec.run != nil {
		records, _ := s.store.steps(ctx, exec.run.ID)
		steps = s.stepReports(ctx, exec.input, records)
	}
	bootstrap, restartRequired := s.pendingRestart()
	restartReason := ""
	if restartRequired {
		restartReason = normalizeRestartReason(bootstrap.RestartReason)
	}
	return RunResult{
		Run:                runReport(exec.run),
		LoginTokens:        exec.loginTokens.SessionSnapshot(),
		LoginTokensIssued:  exec.loginTokensIssued,
		ServiceToken:       exec.serviceToken,
		ServiceTokenIssued: exec.serviceTokenIssued,
		RestartRequired:    restartRequired,
		RestartReason:      restartReason,
		Report:             reportFromSteps(steps, restartRequired, restartReason),
		Steps:              steps,
		RawLoginTokens:     exec.loginTokens,
	}
}

func (s *Service) definitions() []stepDefinition {
	if s.registry != nil {
		defs, err := s.registry.Resolve()
		if err == nil {
			return defs
		}
		if s.core.Logger != nil {
			s.core.Logger.Error("resolve initialization tasks failed", "error", err)
		}
	}
	return s.baseDefinitions()
}

func (s *Service) definition(key string) (stepDefinition, bool) {
	if s.registry != nil {
		if def, ok := s.registry.Get(key); ok {
			return def, true
		}
	}
	for _, def := range s.baseDefinitions() {
		if def.Key == key {
			return def, true
		}
	}
	return stepDefinition{}, false
}

func (s *Service) baseDefinitions() []stepDefinition {
	return []stepDefinition{
		{
			Key:              "welcome",
			Phase:            "welcome",
			Order:            1,
			Title:            "欢迎与初始化说明",
			Goal:             "说明初始化流程、必要配置和可选择的 Web/CLI 初始化方式。",
			AutomaticActions: []string{"读取初始化状态", "展示初始化步骤"},
			CompletionMark:   "用户已进入统一初始化入口",
			Recovery:         "可继续使用 Web 向导，或在 CLI 中运行 aoi init。",
			Required:         true,
			Retryable:        true,
			Idempotent:       true,
			Schema:           stepSchema("welcome", "welcome", 1, "欢迎与初始化说明", "当前系统尚未完成初始化，向导会依次完成语言、文件存储、数据库、缓存、管理员账号和官网信息配置。", true, false, false, nil, nil),
			Check: func(context.Context, *execution) (StepStatus, string, error) {
				return StepStatusSucceeded, "setup wizard ready", nil
			},
			Apply: noOpApply,
		},
		{
			Key:              "database.configure",
			Phase:            "database",
			Order:            5,
			Title:            "数据库配置与连接测试",
			Goal:             "确认数据库配置、连接权限和迁移可用性；数据库配置变更后需要重启继续。",
			AutomaticActions: []string{"测试数据库连接", "检查 SQLite 目录权限", "保存数据库配置时生成重启提示"},
			CompletionMark:   "数据库连接可用",
			Recovery:         "修正数据库配置后重新测试；若已保存新数据库配置，请重启进程后继续。",
			Required:         true,
			Retryable:        true,
			Idempotent:       true,
			Testable:         true,
			Schema:           databaseSchema(),
			Check: func(ctx context.Context, exec *execution) (StepStatus, string, error) {
				if exec.service.infra.Database == nil {
					return StepStatusPending, "database is not connected", nil
				}
				if err := exec.service.infra.Database.Ping(ctx); err != nil {
					return StepStatusPending, "database ping failed", nil
				}
				return StepStatusSucceeded, "current database connection ok", nil
			},
			Apply: noOpApply,
		},
		optionalConfigDefinition("storage.configure", "storage", 4, "文件存储", "配置文件存储并执行读写健康检查；未启用时可跳过。", storageSchema()),
		optionalConfigDefinition("cache.configure", "cache", 8, "缓存方案", "配置 disabled/local/redis/hybrid 缓存模式，并执行读写健康检查。", cacheSchema()),
		{
			Key:              "system.configure",
			Phase:            "system",
			Order:            20,
			Title:            "系统基础配置",
			Goal:             "确认默认语言、IAM issuer、密码策略和默认数据策略。",
			AutomaticActions: []string{"校验系统配置", "应用必要默认值"},
			CompletionMark:   "系统基础配置有效",
			Recovery:         "修正语言、IAM 和密码策略配置后重试。",
			Required:         true,
			Retryable:        true,
			Idempotent:       true,
			Testable:         true,
			Dependencies:     []string{"database.configure"},
			Schema:           systemSchema(),
			Apply: func(ctx context.Context, exec *execution) (stepOutcome, error) {
				test := NewInitValidator(exec.service).Test(ctx, "system.configure", nil)
				if test.Status == "failed" {
					return stepOutcome{Summary: test.Error}, errors.New(test.Error)
				}
				return stepOutcome{Summary: test.Summary}, nil
			},
		},
		{
			Key:              "config.source",
			Phase:            "config",
			Order:            2,
			Title:            "配置来源",
			Goal:             "确认当前进程使用的配置文件和运行时配置快照。",
			AutomaticActions: []string{"读取配置文件路径", "生成配置指纹"},
			CompletionMark:   "配置快照已加载",
			Recovery:         "检查 --config 参数或 RIN_CONFIG_PATH 环境变量。",
			Required:         true,
			Retryable:        true,
			Idempotent:       true,
			Check: func(_ context.Context, exec *execution) (StepStatus, string, error) {
				if exec.service.core.Config == nil {
					return StepStatusPending, "configuration is not loaded", nil
				}
				return StepStatusSucceeded, "configuration loaded from " + fallback(exec.service.configPath, "runtime manager"), nil
			},
			Apply: noOpApply,
		},
		{
			Key:              "config.diagnostics",
			Phase:            "config",
			Order:            30,
			Title:            "配置诊断",
			Goal:             "集中检查服务、数据库、认证、安全、WebUI、存储和插件配置。",
			AutomaticActions: []string{"执行配置诊断", "返回阻塞项和环境变量候选名"},
			CompletionMark:   "无阻塞配置诊断",
			Recovery:         "运行 aoi init 或 aoi run 根据提示修复配置。",
			Required:         true,
			Retryable:        true,
			Idempotent:       true,
			Dependencies:     []string{"system.configure"},
			Check: func(_ context.Context, exec *execution) (StepStatus, string, error) {
				diagnostics := exec.service.configDiagnostics()
				if len(diagnostics) > 0 {
					return StepStatusPending, fmt.Sprintf("%d blocking diagnostic(s)", len(diagnostics)), nil
				}
				return StepStatusSucceeded, "no blocking diagnostics", nil
			},
			Apply: func(_ context.Context, exec *execution) (stepOutcome, error) {
				diagnostics := exec.service.configDiagnostics()
				if len(diagnostics) > 0 {
					return stepOutcome{Summary: fmt.Sprintf("%d blocking diagnostic(s)", len(diagnostics))}, fmt.Errorf("configuration diagnostics failed")
				}
				return stepOutcome{Summary: "no blocking diagnostics"}, nil
			},
		},
		{
			Key:              "dependencies.check",
			Phase:            "preflight",
			Order:            40,
			Title:            "依赖检查",
			Goal:             "确认数据库和已启用的可选依赖具备运行条件。",
			AutomaticActions: []string{"Ping 数据库", "检查 Redis/存储/WebUI/插件启用状态"},
			CompletionMark:   "必要依赖可用",
			Recovery:         "修复数据库连接；可选依赖可关闭或修复配置后重试。",
			Required:         true,
			Retryable:        true,
			Idempotent:       true,
			Dependencies:     []string{"config.diagnostics"},
			Apply:            dependencyCheck,
		},
		{
			Key:              "database.migrate",
			Phase:            "database",
			Order:            50,
			Title:            "数据库迁移",
			Goal:             "显式应用 goose 迁移，使数据库结构完整。",
			AutomaticActions: []string{"运行所有未应用迁移", "不自动创建远程数据库"},
			CompletionMark:   "迁移已应用到最新版本",
			Recovery:         "修复数据库权限或迁移 SQL 后从失败步骤重试。",
			Required:         true,
			Retryable:        true,
			Idempotent:       true,
			Dependencies:     []string{"dependencies.check"},
			Apply: func(ctx context.Context, exec *execution) (stepOutcome, error) {
				if err := initapp.ApplyExplicitMigrations(ctx, exec.service.core, exec.service.infra); err != nil {
					return stepOutcome{Summary: "database migration failed"}, err
				}
				return stepOutcome{Summary: "database migrations applied"}, nil
			},
		},
		{
			Key:              "system.seed",
			Phase:            "system",
			Order:            60,
			Title:            "系统默认数据",
			Goal:             "初始化字典、参数等系统默认数据。",
			AutomaticActions: []string{"执行系统默认数据种子", "保持幂等写入"},
			CompletionMark:   "默认数据已补齐",
			Recovery:         "检查 system_* 表结构和数据库写权限后重试。",
			Required:         true,
			Retryable:        true,
			Idempotent:       true,
			Dependencies:     []string{"database.migrate"},
			Apply:            seedSystemDefaults,
		},
		{
			Key:              "catalog.sync",
			Phase:            "system",
			Order:            65,
			Title:            "API 与权限同步",
			Goal:             "同步当前路由目录，并补齐 IAM 权限定义。",
			AutomaticActions: []string{"同步 API catalog", "同步 permission catalog"},
			CompletionMark:   "路由和权限目录已同步",
			Recovery:         "确认 HTTP 路由已装配且 IAM 权限仓储可用。",
			Required:         true,
			Retryable:        true,
			Idempotent:       true,
			Dependencies:     []string{"system.seed"},
			Apply:            syncCatalog,
		},
		{
			Key:              "iam.owner",
			Phase:            "iam",
			Order:            70,
			Title:            "首个管理员",
			Goal:             "创建或引导平台 owner 管理员，并加载授权策略。",
			UserInputs:       []string{"组织 code", "组织名称", "管理员用户名", "邮箱", "显示名", "密码"},
			AutomaticActions: []string{"创建组织/用户/成员关系", "绑定 platform owner 角色", "Web setup 签发登录令牌"},
			CompletionMark:   "platform owner 管理员可登录并拥有平台权限",
			Recovery:         "确认用户表为空或使用 CLI 幂等 bootstrap 模式重试。",
			Required:         true,
			Retryable:        true,
			Idempotent:       true,
			Dependencies:     []string{"catalog.sync"},
			Schema:           iamOwnerSchema(),
			Check:            iamOwnerCheck,
			Apply:            iamOwnerApply,
		},
		{
			Key:              "site.configure",
			Phase:            "site",
			Order:            75,
			Title:            "官网信息",
			Goal:             "确认产品名称、版本展示名和公开访问地址。",
			AutomaticActions: []string{"校验官网信息配置", "保存品牌和公开访问配置"},
			CompletionMark:   "官网信息配置有效",
			Recovery:         "修正产品名称、版本展示名或公开访问地址后重试。",
			Required:         true,
			Retryable:        true,
			Idempotent:       true,
			Testable:         true,
			Dependencies:     []string{"iam.owner"},
			Schema:           siteSchema(),
			Apply: func(ctx context.Context, exec *execution) (stepOutcome, error) {
				test := NewInitValidator(exec.service).Test(ctx, "site.configure", nil)
				if test.Status == "failed" {
					return stepOutcome{Summary: test.Error}, errors.New(test.Error)
				}
				return stepOutcome{Summary: test.Summary}, nil
			},
		},
		{
			Key:              "optional.finalize",
			Phase:            "finalize",
			Order:            80,
			Title:            "可选能力收尾",
			Goal:             "处理服务 API Token 等可选初始化输出。",
			UserInputs:       []string{"是否创建服务 API Token", "Token 有效期", "Token 备注"},
			AutomaticActions: []string{"按需创建 API Token", "跳过未启用的可选能力"},
			CompletionMark:   "可选输出已处理",
			Recovery:         "若 Token 创建失败，确认 IAM platform owner 已完成后重试。",
			Required:         false,
			Retryable:        true,
			Idempotent:       false,
			Skippable:        true,
			Dependencies:     []string{"site.configure"},
			Schema:           optionalFinalizeSchema(),
			Apply:            finalizeOptional,
		},
		{
			Key:              "verify.finish",
			Phase:            "verify",
			Order:            90,
			Title:            "最终验证",
			Goal:             "验证初始化后的系统已经具备正常运行条件。",
			AutomaticActions: []string{"Ping 数据库", "重新加载 IAM policies", "汇总初始化状态"},
			CompletionMark:   "系统初始化流程完成",
			Recovery:         "查看失败步骤日志，修复后从失败步骤重试。",
			Required:         true,
			Retryable:        true,
			Idempotent:       true,
			Dependencies:     []string{"optional.finalize"},
			Apply:            verifyFinish,
		},
	}
}

func (s *Service) stepReports(ctx context.Context, input Input, records []stepRecord) []StepReport {
	recordByKey := map[string]stepRecord{}
	for _, record := range records {
		recordByKey[record.StepKey] = record
	}
	reports := make([]StepReport, 0, len(s.definitions()))
	exec := &execution{service: s, input: input}
	for _, def := range s.definitions() {
		report := def.report()
		if record, ok := recordByKey[def.Key]; ok {
			report.Status = StepStatus(record.Status)
			report.Attempt = record.Attempt
			report.OutputSummary = record.OutputSummary
			report.ErrorCode = record.ErrorCode
			report.ErrorMessage = record.ErrorMessage
			report.TestStatus = record.TestStatus
			report.TestSummary = record.TestSummary
			report.TestError = record.TestError
			report.TestInputFingerprint = record.TestInputFingerprint
			report.SkippedReason = record.SkippedReason
			report.RepairHint = record.RepairHint
			report.RestartRequired = record.RestartRequired
			report.StartedAt = record.StartedAt
			report.FinishedAt = record.FinishedAt
		} else if def.Check != nil {
			status, summary, err := def.Check(ctx, exec)
			if err == nil && status != "" {
				report.Status = status
				report.OutputSummary = summary
			}
		}
		reports = append(reports, report)
	}
	return reports
}

func (s *Service) hydrateSchemaFields(fields []FieldSchema) []FieldSchema {
	if len(fields) == 0 {
		return []FieldSchema{}
	}
	cfg := s.core.Config
	if s.core.ConfigManager != nil {
		cfg = s.core.ConfigManager.Get()
	}
	out := make([]FieldSchema, 0, len(fields))
	for _, field := range fields {
		defaultValue := field.Default
		if cfg != nil && field.ConfigPath != "" {
			if value, ok := configValue(cfg, field.ConfigPath); ok {
				if field.Sensitive && stringValue(value) != "" {
					field.Value = ""
					field.Help = fallback(field.Help, "已配置，出于安全原因不会回显。")
				} else if field.Required && defaultValue != nil && stringValue(value) == "" {
					field.Value = defaultValue
				} else {
					field.Value = value
				}
			}
		}
		out = append(out, field)
	}
	return out
}

func (s *Service) hydrateSchemaGroups(groups []FieldGroup) []FieldGroup {
	if len(groups) == 0 {
		return []FieldGroup{}
	}
	out := make([]FieldGroup, 0, len(groups))
	for _, group := range groups {
		group.Fields = s.hydrateSchemaFields(group.Fields)
		out = append(out, group)
	}
	return out
}

func schemaVisibleValues(schema StepSchema) map[string]any {
	all := map[string]any{}
	for _, field := range schema.Fields {
		if value, ok := schemaFieldValue(field); ok {
			all[field.Key] = value
		}
	}
	values := map[string]any{}
	if len(schema.Groups) > 0 {
		for _, group := range schema.Groups {
			if !visibilityMatches(group.VisibleWhen, all) {
				continue
			}
			for _, field := range group.Fields {
				if !visibilityMatches(field.VisibleWhen, all) {
					continue
				}
				if value, ok := schemaFieldValue(field); ok {
					values[field.Key] = value
				}
			}
		}
		return values
	}
	for _, field := range schema.Fields {
		if !visibilityMatches(field.VisibleWhen, all) {
			continue
		}
		if value, ok := schemaFieldValue(field); ok {
			values[field.Key] = value
		}
	}
	return values
}

func schemaFieldValue(field FieldSchema) (any, bool) {
	if field.Value != nil {
		return field.Value, true
	}
	if field.Default != nil {
		return field.Default, true
	}
	return nil, false
}

func visibilityMatches(condition *VisibilityCondition, values map[string]any) bool {
	if condition == nil || condition.Field == "" {
		return true
	}
	value := stringValue(values[condition.Field])
	if len(condition.In) > 0 {
		for _, candidate := range condition.In {
			if value == candidate {
				return true
			}
		}
		return false
	}
	if condition.Equals != nil {
		return value == stringValue(condition.Equals)
	}
	return true
}

func (s *Service) configDiagnostics() []appconfig.ConfigDiagnostic {
	if s.core.Config == nil {
		return []appconfig.ConfigDiagnostic{{Section: "config", Path: "", Message: "config is required", Severity: appconfig.ConfigDiagnosticError}}
	}
	return s.core.Config.Diagnostics()
}

func (s *Service) iamSetupStatus(ctx context.Context) (iamservice.SetupStatus, error) {
	if s.modules.IAM.Service == nil {
		return s.bootstrapIAMSetupStatus(ctx)
	}
	return s.modules.IAM.Service.SetupStatus(ctx)
}

type bootstrapIAMUser struct{}

func (bootstrapIAMUser) TableName() string { return "iam_users" }

func (s *Service) bootstrapIAMSetupStatus(ctx context.Context) (iamservice.SetupStatus, error) {
	status := iamservice.SetupStatus{Required: false, PasswordPolicy: s.passwordPolicyFromConfig()}
	if s.core.Config != nil && !s.core.Config.Auth.Enabled {
		return status, nil
	}
	if s.infra.Database == nil {
		status.Required = true
		return status, nil
	}
	hasUsersTable, err := s.infra.Database.HasTable(ctx, bootstrapIAMUser{})
	if err != nil {
		return iamservice.SetupStatus{}, err
	}
	if !hasUsersTable {
		status.Required = true
		return status, nil
	}
	count, err := s.infra.Database.Count(ctx, &bootstrapIAMUser{})
	if err != nil {
		return iamservice.SetupStatus{}, err
	}
	status.Required = count == 0
	return status, nil
}

func (s *Service) passwordPolicyFromConfig() iamservice.PasswordPolicy {
	if s.core.Config == nil {
		return iamservice.PasswordPolicy{MinLength: 8}
	}
	policy := s.core.Config.Auth.PasswordPolicy
	minLength := policy.MinLength
	if minLength <= 0 {
		minLength = 8
	}
	return iamservice.PasswordPolicy{
		MinLength:     minLength,
		RequireLower:  policy.RequireLower,
		RequireUpper:  policy.RequireUpper,
		RequireNumber: policy.RequireNumber,
		RequireSymbol: policy.RequireSymbol,
	}
}

func (s *Service) authorizeRun(ctx context.Context, input Input) error {
	if input.Source != SourceWeb {
		return nil
	}
	return s.authorizeSetupToken(ctx, input.SetupToken)
}

func (s *Service) authorizeSetupToken(ctx context.Context, setupToken string) error {
	if setupTokenValid(setupToken) {
		return nil
	}
	status, err := s.iamSetupStatus(ctx)
	if err != nil {
		return err
	}
	if status.Required {
		return nil
	}
	return ErrSetupUnauthorized
}

func (s *Service) configFingerprint() string {
	return configFingerprintFor(s.effectiveConfig())
}

func (s *Service) effectiveConfig() *appconfig.Config {
	if s.core.ConfigManager != nil {
		if cfg := s.core.ConfigManager.Get(); cfg != nil {
			return cfg
		}
	}
	return s.core.Config
}

func (s *Service) databaseFingerprint() string {
	return databaseFingerprintFor(s.effectiveConfig())
}

func (s *Service) pendingRestart() (bootstrapState, bool) {
	bootstrap, ok := readBootstrapState()
	if !ok || !bootstrap.RestartRequired {
		return bootstrapState{}, false
	}
	if bootstrap.TargetFingerprint != "" && bootstrap.TargetFingerprint == s.configFingerprint() {
		clearBootstrapState()
		return bootstrap, false
	}
	if bootstrap.CurrentStep == "database.configure" && s.databaseRestartApplied() {
		clearBootstrapState()
		return bootstrap, false
	}
	return bootstrap, true
}

func (s *Service) databaseRestartApplied() bool {
	persisted, err := loadConfigFromPath(s.configPath)
	if err != nil {
		return false
	}
	return databaseFingerprintFor(persisted) == s.databaseFingerprint()
}

func normalizeRestartReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" || strings.Contains(strings.ToLower(reason), "database configuration changed") {
		return "数据库配置已保存。请重启服务，让当前进程加载新的数据库配置后继续初始化。"
	}
	return reason
}

func (s *Service) printf(format string, args ...any) {
	if s.stdout == nil {
		return
	}
	_, _ = fmt.Fprintf(s.stdout, format, args...)
}

type execution struct {
	service            *Service
	input              Input
	run                *runRecord
	admin              *iamservice.Principal
	loginTokens        iamservice.TokenPair
	loginTokensIssued  bool
	serviceToken       iamservice.CreateAPITokenResult
	serviceTokenIssued bool
}

type stepDefinition struct {
	Key              string
	Phase            string
	Order            int
	Title            string
	Goal             string
	Prerequisites    []string
	UserInputs       []string
	AutomaticActions []string
	CompletionMark   string
	Recovery         string
	Required         bool
	Retryable        bool
	Idempotent       bool
	Skippable        bool
	Testable         bool
	Dependencies     []string
	Schema           StepSchema
	Check            func(context.Context, *execution) (StepStatus, string, error)
	Apply            func(context.Context, *execution) (stepOutcome, error)
}

type stepOutcome struct {
	Status  StepStatus
	Summary string
}

func (d stepDefinition) report() StepReport {
	schema := d.Schema
	if schema.Key == "" {
		schema = stepSchema(d.Key, d.Phase, d.Order, d.Title, d.Goal, d.Required, d.Skippable, d.Testable, d.Dependencies, nil)
	}
	if schema.RouteSlug == "" {
		schema.RouteSlug = routeSlugForStep(schema.Key)
	}
	if schema.Fields == nil {
		schema.Fields = []FieldSchema{}
	}
	if schema.Groups == nil {
		schema.Groups = []FieldGroup{}
	}
	prerequisites := nonNilStrings(d.Prerequisites)
	userInputs := nonNilStrings(d.UserInputs)
	automaticActions := nonNilStrings(d.AutomaticActions)
	detailPrefix := "ui.setup.details." + sanitizeKeyPart(d.Key) + "."
	return StepReport{
		Key:                 d.Key,
		Phase:               d.Phase,
		Order:               d.Order,
		Title:               d.Title,
		TitleKey:            "ui.setup.steps." + d.Key + ".title",
		Goal:                d.Goal,
		GoalKey:             "ui.setup.steps." + d.Key + ".description",
		Prerequisites:       prerequisites,
		PrerequisiteKeys:    detailKeyList(detailPrefix+"prerequisites", len(prerequisites)),
		UserInputs:          userInputs,
		UserInputKeys:       detailKeyList(detailPrefix+"userInputs", len(userInputs)),
		AutomaticActions:    automaticActions,
		AutomaticActionKeys: detailKeyList(detailPrefix+"automaticActions", len(automaticActions)),
		CompletionMark:      d.CompletionMark,
		CompletionMarkKey:   detailPrefix + "completionMark",
		Recovery:            d.Recovery,
		RecoveryKey:         detailPrefix + "recovery",
		Required:            d.Required,
		Retryable:           d.Retryable,
		Idempotent:          d.Idempotent,
		Skippable:           d.Skippable,
		Testable:            d.Testable,
		Dependencies:        nonNilStrings(d.Dependencies),
		Schema:              schema,
		Status:              StepStatusPending,
	}
}

func detailKeyList(prefix string, count int) []string {
	if count <= 0 {
		return []string{}
	}
	out := make([]string, count)
	for index := range out {
		out[index] = fmt.Sprintf("%s.%d", prefix, index+1)
	}
	return out
}

func noOpApply(context.Context, *execution) (stepOutcome, error) {
	return stepOutcome{Summary: "already satisfied"}, nil
}

func optionalConfigDefinition(key, phase string, order int, title, goal string, schema StepSchema) stepDefinition {
	return stepDefinition{
		Key:              key,
		Phase:            phase,
		Order:            order,
		Title:            title,
		Goal:             goal,
		AutomaticActions: []string{"保存配置草稿", "执行即时可用性测试", "记录跳过或降级原因"},
		CompletionMark:   "当前可选能力已确认",
		Recovery:         "修正配置后重新测试；未启用能力可以跳过，显式启用失败会阻塞继续。",
		Required:         false,
		Retryable:        true,
		Idempotent:       true,
		Skippable:        true,
		Testable:         true,
		Dependencies:     nonNilStrings(schema.Dependencies),
		Schema:           schema,
		Apply: func(ctx context.Context, exec *execution) (stepOutcome, error) {
			test := NewInitValidator(exec.service).Test(ctx, key, nil)
			if test.Status == "failed" {
				return stepOutcome{Summary: test.Error}, errors.New(test.Error)
			}
			return stepOutcome{Summary: test.Summary}, nil
		},
	}
}

func dependencyCheck(ctx context.Context, exec *execution) (stepOutcome, error) {
	parts := []string{}
	db := exec.service.infra.Database
	if db == nil {
		return stepOutcome{Summary: "database is missing"}, fmt.Errorf("database is missing")
	}
	if err := db.Ping(ctx); err != nil {
		return stepOutcome{Summary: "database ping failed"}, err
	}
	parts = append(parts, "database=ok")
	cfg := exec.service.core.Config
	if cfg != nil {
		cacheDriver := cfg.Cache.Driver
		if cacheDriver != appconfig.CacheDriverDisabled && exec.service.infra.Cache == nil {
			parts = append(parts, "cache="+cacheDriver+" degraded")
		} else if cacheDriver != appconfig.CacheDriverDisabled {
			parts = append(parts, "cache="+cacheDriver+" ok")
		} else {
			parts = append(parts, "cache=disabled")
		}
		storageDriver := cfg.Storage.Driver
		if storageDriver != appconfig.StorageDriverDisabled && exec.service.infra.Storage == nil {
			parts = append(parts, "storage=degraded")
		} else if storageDriver != appconfig.StorageDriverDisabled {
			parts = append(parts, "storage="+storageDriver+" ok")
		} else {
			parts = append(parts, "storage=disabled")
		}
		webui := cfg.WebUI
		webui.ApplyDefaults()
		if webui.EnabledValue() {
			if _, err := os.Stat(filepath.Join(webui.DistDir, "index.html")); err == nil {
				parts = append(parts, "webui=ok")
			} else {
				parts = append(parts, "webui=missing")
			}
		} else {
			parts = append(parts, "webui=disabled")
		}
		if cfg.Plugins.Enabled {
			parts = append(parts, "plugins=enabled")
		} else {
			parts = append(parts, "plugins=disabled")
		}
	}
	return stepOutcome{Summary: strings.Join(parts, ", ")}, nil
}

func seedSystemDefaults(ctx context.Context, exec *execution) (stepOutcome, error) {
	if exec.service.modules.System.Service == nil {
		return stepOutcome{Status: StepStatusSkipped, Summary: "system module disabled"}, nil
	}
	seed, err := exec.service.modules.System.Service.SeedDefaults(ctx)
	if err != nil {
		return stepOutcome{Summary: "system seed failed"}, err
	}
	return stepOutcome{Summary: fmt.Sprintf("dictionaries=%d items=%d parameters=%d storage=%s", seed.DictionariesCreated, seed.DictionaryItemsCreated, seed.ParametersCreated, seed.StorageStatus)}, nil
}

func syncCatalog(ctx context.Context, exec *execution) (stepOutcome, error) {
	if exec.service.modules.System.Service == nil {
		return stepOutcome{Status: StepStatusSkipped, Summary: "system module disabled"}, nil
	}
	apiSync, err := exec.service.modules.System.Service.SyncAPIs(ctx)
	if err != nil {
		return stepOutcome{Summary: "api catalog sync failed"}, err
	}
	permissionSync, err := exec.service.modules.System.Service.SyncPermissions(ctx)
	if err != nil {
		return stepOutcome{Summary: "permission sync failed"}, err
	}
	return stepOutcome{Summary: fmt.Sprintf("apis=%d permissions=%d skipped=%d", apiSync.Total, permissionSync.Created, permissionSync.Skipped)}, nil
}

func iamOwnerCheck(ctx context.Context, exec *execution) (StepStatus, string, error) {
	if exec.service.modules.IAM.Service == nil {
		if exec.service.core.Config != nil && !exec.service.core.Config.Auth.Enabled {
			return StepStatusSkipped, "iam disabled", nil
		}
		status, err := exec.service.bootstrapIAMSetupStatus(ctx)
		if err != nil {
			return "", "", err
		}
		if status.Required {
			return StepStatusPending, "platform owner admin is required", nil
		}
		return StepStatusSucceeded, "platform owner admin already exists", nil
	}
	if exec.input.IssueLoginTokens {
		return StepStatusPending, "web initial-admin request must execute setup", nil
	}
	if exec.input.AdminPassword != "" {
		return StepStatusPending, "admin bootstrap requested", nil
	}
	status, err := exec.service.modules.IAM.Service.SetupStatus(ctx)
	if err != nil {
		return "", "", err
	}
	if status.Required {
		return StepStatusPending, "platform owner admin is required", nil
	}
	return StepStatusSucceeded, "platform owner admin already exists", nil
}

func iamOwnerApply(ctx context.Context, exec *execution) (stepOutcome, error) {
	iam := exec.service.modules.IAM.Service
	if iam == nil {
		return stepOutcome{Status: StepStatusSkipped, Summary: "iam disabled"}, nil
	}
	if strings.TrimSpace(exec.input.AdminPassword) == "" {
		if err := iam.LoadPolicies(ctx); err != nil {
			return stepOutcome{Summary: "iam policy reload failed"}, err
		}
		return stepOutcome{Status: StepStatusSkipped, Summary: "admin password empty; admin creation skipped"}, nil
	}
	if exec.input.IssueLoginTokens {
		pair, err := iam.InitialAdminSetup(ctx, iamservice.InitialAdminSetupInput{
			OrgCode:     exec.input.OrgCode,
			OrgName:     exec.input.OrgName,
			Username:    exec.input.AdminUsername,
			Email:       exec.input.AdminEmail,
			DisplayName: exec.input.AdminDisplayName,
			Password:    exec.input.AdminPassword,
			ProductCode: exec.input.ProductCode,
			ClientType:  exec.input.ClientType,
			UserAgent:   exec.input.UserAgent,
			IPAddress:   exec.input.IPAddress,
		})
		if err != nil {
			return stepOutcome{Summary: "initial admin setup failed"}, err
		}
		exec.loginTokens = pair
		exec.loginTokensIssued = true
		if err := iam.LoadPolicies(ctx); err != nil {
			return stepOutcome{Summary: "iam policy reload failed"}, err
		}
		return stepOutcome{Summary: "initial platform owner admin created and login tokens issued"}, nil
	}
	admin, err := iam.BootstrapAdmin(ctx, iamservice.BootstrapAdminInput{
		OrgCode:     exec.input.OrgCode,
		OrgName:     exec.input.OrgName,
		Username:    exec.input.AdminUsername,
		Email:       exec.input.AdminEmail,
		DisplayName: exec.input.AdminDisplayName,
		Password:    exec.input.AdminPassword,
	})
	if err != nil {
		return stepOutcome{Summary: "admin bootstrap failed"}, err
	}
	exec.admin = admin
	return stepOutcome{Summary: fmt.Sprintf("platform owner admin ready: user=%s org=%d", admin.Username, admin.OrgID)}, nil
}

func finalizeOptional(ctx context.Context, exec *execution) (stepOutcome, error) {
	if !exec.input.CreateServiceToken {
		return stepOutcome{Status: StepStatusSkipped, Summary: "service token not requested"}, nil
	}
	if exec.service.modules.IAM.Service == nil {
		return stepOutcome{Summary: "iam disabled; service token unavailable"}, fmt.Errorf("iam disabled")
	}
	if exec.admin == nil {
		return stepOutcome{Summary: "platform owner principal unavailable for service token"}, fmt.Errorf("platform owner principal unavailable")
	}
	created, err := exec.service.modules.IAM.Service.CreateAPIToken(ctx, iamservice.CreateAPITokenInput{
		Principal: *exec.admin,
		UserID:    exec.admin.UserID,
		RoleCode:  iammodel.RolePlatformOwner,
		Days:      exec.input.ServiceTokenDays,
		Remark:    exec.input.ServiceTokenRemark,
	})
	if err != nil {
		return stepOutcome{Summary: "service token creation failed"}, err
	}
	exec.serviceToken = created
	exec.serviceTokenIssued = true
	return stepOutcome{Summary: "service token created with prefix " + created.Item.TokenPrefix}, nil
}

func verifyFinish(ctx context.Context, exec *execution) (stepOutcome, error) {
	if exec.service.infra.Database != nil {
		if err := exec.service.infra.Database.Ping(ctx); err != nil {
			return stepOutcome{Summary: "database verification failed"}, err
		}
	}
	if exec.service.modules.IAM.Service != nil {
		if err := exec.service.modules.IAM.Service.LoadPolicies(ctx); err != nil {
			return stepOutcome{Summary: "iam policy reload failed"}, err
		}
		status, err := exec.service.modules.IAM.Service.SetupStatus(ctx)
		if err != nil {
			return stepOutcome{Summary: "iam setup status failed"}, err
		}
		if status.Required {
			return stepOutcome{Summary: "system data initialized; platform owner admin still required"}, nil
		}
	}
	return stepOutcome{Summary: "system initialization verified"}, nil
}

func setupTokenValid(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, key := range []string{"AOI_SETUP_TOKEN", "RIN_APP_SETUP_TOKEN"} {
		expected := strings.TrimSpace(os.Getenv(key))
		if expected != "" && subtle.ConstantTimeCompare([]byte(value), []byte(expected)) == 1 {
			return true
		}
	}
	return false
}

func defaultSource(source Source) Source {
	if source == "" {
		return SourceCLI
	}
	return source
}

func defaultMode(mode Mode) Mode {
	if mode == "" {
		return ModeFirstRun
	}
	return mode
}

func optionalRunReport(run *runRecord) *RunReport {
	if run == nil {
		return nil
	}
	report := runReport(run)
	return &report
}

func firstPendingStep(steps []StepReport) string {
	for _, step := range steps {
		if step.Status == StepStatusPending || step.Status == StepStatusFailed || step.Status == StepStatusRunning {
			return step.Key
		}
	}
	return ""
}

func allowedActions(required bool, completed bool) []string {
	if completed {
		return []string{"verify", "repair"}
	}
	if required {
		return []string{"run", "retry"}
	}
	return []string{"verify"}
}

func fallback(value string, fallbackValue string) string {
	if strings.TrimSpace(value) == "" {
		return fallbackValue
	}
	return value
}

func nonNilStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

func shortID(id int64) string {
	return fmt.Sprintf("%d", id)
}

func inputSummary(key string) string {
	switch key {
	case "iam.owner":
		return "org/admin fields captured; password redacted"
	case "optional.finalize":
		return "optional flags captured; token value redacted"
	default:
		return "no user input"
	}
}

func reportFromSteps(steps []StepReport, restartRequired bool, restartReason string) InitReport {
	report := InitReport{GeneratedAt: time.Now().UTC(), RestartRequired: restartRequired, RestartReason: restartReason}
	for _, step := range steps {
		switch step.Status {
		case StepStatusSucceeded:
			report.Successful++
		case StepStatusFailed:
			report.Failed++
		case StepStatusSkipped:
			report.Skipped++
			if step.Required {
				report.Risk++
			}
		default:
			if step.Required {
				report.Risk++
			}
		}
		if step.TestStatus == "failed" {
			report.Risk++
		}
	}
	switch {
	case report.RestartRequired:
		report.Summary = "restart required before continuing initialization"
	case report.Failed > 0:
		report.Summary = fmt.Sprintf("%d initialization step(s) failed", report.Failed)
	case report.Risk > 0:
		report.Summary = fmt.Sprintf("%d initialization item(s) still need attention", report.Risk)
	default:
		report.Summary = "initialization checks passed"
	}
	return report
}
