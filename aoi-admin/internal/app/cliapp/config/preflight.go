package config

import (
	"strings"

	"github.com/rei0721/go-scaffold/internal/app/cliapp/localization"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/cli"
)

const (
	preflightActionSkip           = "skip"
	preflightActionRuntimeEnvOnly = "runtime_env_only"

	preflightDatabaseActionFile   = "file"
	preflightDatabaseActionSQLite = "sqlite"

	preflightSMTPActionFile  = "file"
	preflightSMTPActionDebug = "debug"
)

// preflightConfigForStart 在启动前加载配置并尝试交互式修复阻断项。
//
// repaired 表示本轮是否修改过配置或 env 管理元数据；最多重试 4 次，避免用户连续选择无效修复时陷入无限循环。
func PreflightConfigForStart(ctx *cli.Context, configPath string) (*appconfig.Config, bool, error) {
	localizer := localization.FromContext(ctx)
	repaired := false
	for attempt := 0; attempt < 4; attempt++ {
		cfg, diagnostics, err := appconfig.LoadDiagnostics(configPath)
		if err != nil {
			return nil, repaired, err
		}
		if len(diagnostics) == 0 {
			return cfg, repaired, nil
		}
		printConfigDiagnostics(ctx.Stdout, configPath, diagnostics, localizer)
		if !canPromptPreflightRepair(ctx) {
			return nil, repaired, newConfigDiagnosticsError(configPath, diagnostics, localizer)
		}
		changed, err := promptPreflightRepairs(ctx, configPath, cfg, diagnostics)
		if err != nil {
			return nil, repaired, err
		}
		if !changed {
			return nil, repaired, newConfigDiagnosticsError(configPath, diagnostics, localizer)
		}
		repaired = true
	}
	_, diagnostics, err := appconfig.LoadDiagnostics(configPath)
	if err != nil {
		return nil, repaired, err
	}
	return nil, repaired, newConfigDiagnosticsError(configPath, diagnostics, localizer)
}

// actionableConfigLoadError 将普通加载错误转换为更适合 CLI 展示的诊断错误。
//
// 如果配置仍能被诊断系统解析，则优先返回聚合诊断；否则退回核心密钥缺失的专项提示。
func ActionableConfigLoadError(configPath string, loadErr error) error {
	if loadErr == nil {
		return nil
	}
	_, diagnostics, diagErr := appconfig.LoadDiagnostics(configPath)
	if diagErr == nil && len(diagnostics) > 0 {
		return newConfigDiagnosticsError(configPath, diagnostics)
	}
	return coreSecretConfigError(configPath, loadErr)
}

// canPromptPreflightRepair 判断当前 CLI 上下文是否允许启动前交互式修复。
func canPromptPreflightRepair(ctx *cli.Context) bool {
	if ctx == nil || ctx.UI == nil || ctx.GetBool("yes") {
		return false
	}
	if value, ok := cli.PromptAnswer(ctx.UI, "privacy"); ok {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "false", "f", "no", "n", "0":
			return false
		}
	}
	return true
}

// promptPreflightRepairs 根据诊断区段分派到对应的修复向导。
//
// 只处理当前能安全自动化的修复类型；其它诊断会保留为阻断错误交给用户手动处理。
func promptPreflightRepairs(ctx *cli.Context, configPath string, cfg *appconfig.Config, diagnostics []appconfig.ConfigDiagnostic) (bool, error) {
	changed := false
	if hasSectionDiagnostics(diagnostics, appconfig.AppDatabaseName) &&
		(strings.EqualFold(cfg.Database.Driver, "postgres") || strings.EqualFold(cfg.Database.Driver, "mysql")) {
		ok, err := promptDatabasePreflightRepair(ctx, configPath, cfg)
		if err != nil {
			return false, err
		}
		changed = changed || ok
	}
	if hasAuthCoreDiagnostics(diagnostics) {
		ok, err := PromptCoreSecretRecovery(ctx, configPath)
		if err != nil {
			return false, err
		}
		changed = changed || ok
	}
	if hasSMTPDiagnostics(diagnostics) {
		ok, err := promptSMTPPreflightRepair(ctx, configPath, cfg)
		if err != nil {
			return false, err
		}
		changed = changed || ok
	}
	return changed, nil
}

func hasSectionDiagnostics(diagnostics []appconfig.ConfigDiagnostic, section string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Section == section {
			return true
		}
	}
	return false
}

func hasAuthCoreDiagnostics(diagnostics []appconfig.ConfigDiagnostic) bool {
	for _, diagnostic := range diagnostics {
		for _, path := range coreSecretPaths {
			if diagnostic.Path == path {
				return true
			}
		}
	}
	return false
}

func hasSMTPDiagnostics(diagnostics []appconfig.ConfigDiagnostic) bool {
	for _, diagnostic := range diagnostics {
		if strings.HasPrefix(diagnostic.Path, "auth.smtp.") {
			return true
		}
	}
	return false
}
