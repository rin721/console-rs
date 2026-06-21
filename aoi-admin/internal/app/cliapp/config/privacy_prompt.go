package config

import (
	"fmt"
	"strings"

	"github.com/rei0721/go-scaffold/internal/app/cliapp/localization"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/cli"
)

const (
	privacyCoreActionGenerateFile = "generate_file"
	privacyCoreActionManual       = "manual"
)

// PromptCoreSecretRecovery 引导用户修复 IAM 启动必需的核心密钥。
func PromptCoreSecretRecovery(ctx *cli.Context, configPath string) (bool, error) {
	localizer := localization.FromContext(ctx)
	if IsExampleConfig(configPath) {
		return false, fmt.Errorf("%s", localizer.T("cli.config.privacy.error.exampleReadonly", map[string]any{"Env": coreSecretEnvHelp()}))
	}
	if coreSecretValueAnswersProvided(ctx.UI) {
		return promptAndWriteCoreSecrets(ctx, configPath)
	}
	action, err := cli.SelectKey(ctx.Context, ctx.UI, "privacy.core_secrets.action", localizer.T("cli.config.privacy.core.prompt"), []cli.SelectOption{
		{Value: privacyCoreActionGenerateFile, Label: localizer.T("cli.config.privacy.core.option.generateFile.label"), Description: localizer.T("cli.config.privacy.core.option.generateFile.description")},
		{Value: privacyActionRuntimeEnvOnly, Label: localizer.T("cli.config.privacy.option.runtimeEnv.label"), Description: localizer.T("cli.config.privacy.option.runtimeEnv.description")},
		{Value: privacyCoreActionManual, Label: localizer.T("cli.config.privacy.core.option.manual.label"), Description: localizer.T("cli.config.privacy.core.option.manual.description")},
		{Value: privacyActionSkip, Label: localizer.T("cli.config.privacy.option.skip.label"), Description: localizer.T("cli.config.privacy.option.skip.description")},
	})
	if err != nil {
		return false, err
	}
	switch action {
	case privacyCoreActionGenerateFile:
		updates := map[string]string{}
		for _, path := range coreSecretPaths {
			updates[path] = randomSecret()
		}
		if err := applyPrivacyForceFileUpdates(configPath, updates); err != nil {
			return false, err
		}
		_ = ctx.UI.Info(localizer.T("cli.config.privacy.core.info.generated"))
		return true, nil
	case privacyActionRuntimeEnvOnly:
		if err := applyPrivacyRuntimeEnvOnlyDirect(configPath, coreSecretPaths); err != nil {
			return false, err
		}
		_ = ctx.UI.Info(localizer.T("cli.config.privacy.core.info.runtimeEnv"))
		return true, nil
	case privacyCoreActionManual:
		return promptAndWriteCoreSecrets(ctx, configPath)
	case privacyActionSkip, "":
		return false, nil
	default:
		return false, fmt.Errorf("unknown IAM core secret repair action %q", action)
	}
}

func coreSecretValueAnswersProvided(ui cli.PromptUI) bool {
	for _, path := range coreSecretPaths {
		if _, ok := cli.PromptAnswer(ui, "privacy."+path+".value"); ok {
			return true
		}
	}
	return false
}

func promptAndWriteCoreSecrets(ctx *cli.Context, configPath string) (bool, error) {
	localizer := localization.FromContext(ctx)
	updates := map[string]string{}
	for _, path := range coreSecretPaths {
		value, err := promptPrivacyValue(ctx, path)
		if err != nil {
			return false, err
		}
		if value != "" {
			updates[path] = value
		}
	}
	if len(updates) == 0 {
		return false, nil
	}
	if err := applyPrivacyForceFileUpdates(configPath, updates); err != nil {
		return false, err
	}
	_ = ctx.UI.Info(localizer.T("cli.config.privacy.core.info.written"))
	return true, nil
}

// PromptPrivacyUpdates 收集隐私配置的持久化计划。
func PromptPrivacyUpdates(ctx *cli.Context, configPath string) (PrivacyPersistPlan, error) {
	localizer := localization.FromContext(ctx)
	paths, err := privacyPaths(configPath)
	if err != nil {
		return PrivacyPersistPlan{}, err
	}
	updates := newPrivacyPersistPlan()
	for _, path := range paths {
		envManaged, err := privacyPathIsEnvManaged(configPath, path)
		if err != nil {
			return PrivacyPersistPlan{}, err
		}
		if envManaged {
			action, err := cli.SelectKey(ctx.Context, ctx.UI, "privacy."+path+".action", localizer.T("cli.config.privacy.envManagedPrompt", map[string]any{"Path": path}), []cli.SelectOption{
				{Value: privacyActionForceFile, Label: localizer.T("cli.config.privacy.option.forceFile.label"), Description: localizer.T("cli.config.privacy.option.forceFile.description")},
				{Value: privacyActionRuntimeEnvOnly, Label: localizer.T("cli.config.privacy.option.runtimeEnvRestore.label"), Description: localizer.T("cli.config.privacy.option.runtimeEnvRestore.description")},
				{Value: privacyActionSkip, Label: localizer.T("cli.config.privacy.option.skip.label"), Description: localizer.T("cli.config.privacy.option.skip.description")},
			})
			if err != nil {
				return PrivacyPersistPlan{}, err
			}
			switch action {
			case privacyActionRuntimeEnvOnly:
				updates.RuntimeEnvOnlyPaths = append(updates.RuntimeEnvOnlyPaths, path)
				continue
			case privacyActionSkip, "":
				continue
			case privacyActionForceFile:
				value, err := promptPrivacyValue(ctx, path)
				if err != nil {
					return PrivacyPersistPlan{}, err
				}
				if value != "" {
					updates.ForceFileUpdates[path] = value
				}
				continue
			default:
				return PrivacyPersistPlan{}, fmt.Errorf("unknown privacy config action %q", action)
			}
		}

		value, err := promptPrivacyValue(ctx, path)
		if err != nil {
			return PrivacyPersistPlan{}, err
		}
		if value == "" {
			continue
		}
		updates.FileUpdates[path] = value
	}
	return updates, nil
}

func promptPrivacyValue(ctx *cli.Context, path string) (string, error) {
	localizer := localization.FromContext(ctx)
	hint := localizer.T("cli.config.privacy.value.hint.skip")
	if isGeneratedSecretPath(path) {
		hint = localizer.T("cli.config.privacy.value.hint.generate")
	}
	value, err := cli.InputKey(ctx.Context, ctx.UI, "privacy."+path+".value", localizer.T("cli.config.privacy.value.prompt", map[string]any{"Path": path, "Hint": hint}), "")
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if strings.EqualFold(value, "generate") && isGeneratedSecretPath(path) {
		return randomSecret(), nil
	}
	return value, nil
}

func privacyPaths(configPath string) ([]string, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	paths := append([]string(nil), coreSecretPaths...)
	switch cfg.Database.Driver {
	case appconfig.DatabaseDriverMySQL:
		paths = append(paths, "database.mysql.password")
	case appconfig.DatabaseDriverPostgres:
		paths = append(paths, "database.postgres.password")
	}
	switch cfg.Cache.Driver {
	case appconfig.CacheDriverRedis, appconfig.CacheDriverHybrid:
		paths = append(paths, "cache.redis.password")
	}
	if strings.EqualFold(cfg.Auth.NotificationDriver, "smtp") {
		paths = append(paths, "auth.smtp.password")
	}
	return paths, nil
}
