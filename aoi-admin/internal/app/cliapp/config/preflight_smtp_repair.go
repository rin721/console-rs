package config

import (
	"fmt"
	"strconv"

	"github.com/rei0721/go-scaffold/internal/app/cliapp/localization"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/cli"
	"github.com/rei0721/go-scaffold/pkg/configloader"
)

var smtpRequiredPaths = []string{
	"auth.smtp.host",
	"auth.smtp.port",
	"auth.smtp.from",
}

// promptSMTPPreflightRepair 引导用户修复 SMTP 通知配置。
//
// 切换到 debug driver 会绕过 SMTP 必填项；写文件和运行时 env 两种路径保持与数据库修复一致的持久化语义。
func promptSMTPPreflightRepair(ctx *cli.Context, configPath string, cfg *appconfig.Config) (bool, error) {
	localizer := localization.FromContext(ctx)
	action, err := cli.SelectKey(ctx.Context, ctx.UI, "preflight.smtp.action", localizer.T("cli.config.preflight.smtp.prompt"), []cli.SelectOption{
		{Value: preflightSMTPActionDebug, Label: localizer.T("cli.config.preflight.smtp.option.debug.label"), Description: localizer.T("cli.config.preflight.smtp.option.debug.description")},
		{Value: preflightSMTPActionFile, Label: localizer.T("cli.config.preflight.smtp.option.file.label"), Description: localizer.T("cli.config.preflight.smtp.option.file.description")},
		{Value: preflightActionRuntimeEnvOnly, Label: localizer.T("cli.config.preflight.option.runtimeEnv.label"), Description: localizer.T("cli.config.preflight.option.runtimeEnv.description")},
		{Value: preflightActionSkip, Label: localizer.T("cli.config.preflight.option.skip.label"), Description: localizer.T("cli.config.preflight.option.skip.description")},
	})
	if err != nil {
		return false, err
	}
	switch action {
	case preflightSMTPActionDebug:
		if IsExampleConfig(configPath) {
			return false, exampleConfigWriteError(configPath)
		}
		updates := []configloader.YAMLScalarUpdate{
			{Kind: configloader.YAMLScalarString, Path: "auth.notification_driver", Value: "debug"},
		}
		if err := applyConfigForceFileScalarUpdates(configPath, updates, []string{"auth.notification_driver"}); err != nil {
			return false, err
		}
		_ = ctx.UI.Info(localizer.T("cli.config.preflight.smtp.info.debug"))
		return true, nil
	case preflightSMTPActionFile:
		if IsExampleConfig(configPath) {
			return false, exampleConfigWriteError(configPath)
		}
		host, err := promptConfigStringValue(ctx, "auth.smtp.host", cfg.Auth.SMTP.Host)
		if err != nil {
			return false, err
		}
		port, err := promptConfigIntValue(ctx, "auth.smtp.port", defaultInt(cfg.Auth.SMTP.Port, 587))
		if err != nil {
			return false, err
		}
		from, err := promptConfigStringValue(ctx, "auth.smtp.from", cfg.Auth.SMTP.From)
		if err != nil {
			return false, err
		}
		updates := []configloader.YAMLScalarUpdate{
			{Kind: configloader.YAMLScalarString, Path: "auth.smtp.host", Value: host},
			{Kind: configloader.YAMLScalarInt, Path: "auth.smtp.port", Value: strconv.Itoa(port)},
			{Kind: configloader.YAMLScalarString, Path: "auth.smtp.from", Value: from},
		}
		if err := applyConfigForceFileScalarUpdates(configPath, updates, smtpRequiredPaths); err != nil {
			return false, err
		}
		_ = ctx.UI.Info(localizer.T("cli.config.preflight.smtp.info.file"))
		return true, nil
	case preflightActionRuntimeEnvOnly:
		if err := applyRuntimeEnvOnlyConfigPathsDirect(configPath, smtpRequiredPaths, validatePreflightRuntimeEnvValue); err != nil {
			return false, err
		}
		_ = ctx.UI.Info(localizer.T("cli.config.preflight.smtp.info.runtimeEnv"))
		return true, nil
	case preflightActionSkip, "":
		return false, nil
	default:
		return false, fmt.Errorf("unknown SMTP repair action %q", action)
	}
}
