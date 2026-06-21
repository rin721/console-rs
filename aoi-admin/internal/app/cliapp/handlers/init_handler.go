package handlers

import (
	"context"
	"fmt"
	"io"
	"strings"

	cliconfig "github.com/rei0721/go-scaffold/internal/app/cliapp/config"
	"github.com/rei0721/go-scaffold/internal/app/cliapp/localization"
	initservice "github.com/rei0721/go-scaffold/internal/app/cliapp/services/init"
	"github.com/rei0721/go-scaffold/internal/app/initcenter"
	"github.com/rei0721/go-scaffold/pkg/cli"
)

type InitHandler struct{}

var executeInitialization = initservice.ExecuteInitialization

func SetExecuteInitializationForTest(
	fn func(ctx context.Context, stdout io.Writer, input initservice.InitializationInput) error,
) func() {
	previous := executeInitialization
	executeInitialization = fn
	return func() {
		executeInitialization = previous
	}
}

func NewInitHandler() *InitHandler {
	return &InitHandler{}
}

func (h *InitHandler) Execute(ctx *cli.Context) error {
	input, err := InitializationInputFromContext(ctx)
	if err != nil {
		return err
	}
	return h.RunInitializationFlow(ctx, input)
}

func (h *InitHandler) RunInitializationFlow(ctx *cli.Context, input initservice.InitializationInput) error {
	configPath, err := cliconfig.SelectConfigPath(ctx)
	if err != nil {
		return err
	}
	return h.runInitializationFlowWithConfigPath(ctx, input, configPath)
}

func (h *InitHandler) runInitializationFlowWithConfigPath(ctx *cli.Context, input initservice.InitializationInput, configPath string) error {
	input.ConfigPath = configPath
	cfg, err := cliconfig.LoadConfig(configPath)
	if err != nil {
		return err
	}
	ui, err := requireUI(ctx)
	if err != nil {
		return err
	}
	localizer := localization.FromContext(ctx)
	stop, err := h.runSetupConfigPrompts(ctx, ui, configPath, isCLIPreAccountConfigStep)
	if err != nil || stop {
		return err
	}
	cfg, err = cliconfig.LoadConfig(configPath)
	if err != nil {
		return err
	}
	if cfg.Auth.Enabled && input.AdminPassword == "" {
		if !ctx.IsFlagChanged("org-code") {
			input.OrgCode, err = cli.InputKey(ctx.Context, ui, "org-code", localizer.T("cli.init.prompts.orgCode"), defaultString(input.OrgCode, "acme"))
			if err != nil {
				return err
			}
		}
		if !ctx.IsFlagChanged("org-name") {
			input.OrgName, err = cli.InputKey(ctx.Context, ui, "org-name", localizer.T("cli.init.prompts.orgName"), defaultString(input.OrgName, input.OrgCode))
			if err != nil {
				return err
			}
		}
		if !ctx.IsFlagChanged("admin-username") {
			input.AdminUsername, err = cli.InputKey(ctx.Context, ui, "admin-username", localizer.T("cli.init.prompts.adminUsername"), defaultString(input.AdminUsername, "admin"))
			if err != nil {
				return err
			}
		}
		if !ctx.IsFlagChanged("admin-email") {
			input.AdminEmail, err = cli.InputKey(ctx.Context, ui, "admin-email", localizer.T("cli.init.prompts.adminEmail"), defaultString(input.AdminEmail, "admin@example.com"))
			if err != nil {
				return err
			}
		}
		if !ctx.IsFlagChanged("admin-display-name") {
			input.AdminDisplayName, err = cli.InputKey(ctx.Context, ui, "admin-display-name", localizer.T("cli.init.prompts.adminDisplayName"), defaultString(input.AdminDisplayName, input.AdminUsername))
			if err != nil {
				return err
			}
		}
		input.AdminPassword, err = cli.PasswordKey(ctx.Context, ui, "admin-password", localizer.T("cli.init.prompts.adminPassword"))
		if err != nil {
			return err
		}
	}
	stop, err = h.runSetupConfigPrompts(ctx, ui, configPath, isCLIPostAccountConfigStep)
	if err != nil || stop {
		return err
	}
	if cfg.Auth.Enabled && input.AdminPassword != "" && !ctx.IsFlagChanged("create-service-token") {
		input.CreateServiceToken, err = cli.ConfirmKey(ctx.Context, ui, "create-service-token", localizer.T("cli.init.prompts.createServiceToken"), false)
		if err != nil {
			return err
		}
	}
	if input.CreateServiceToken {
		if !ctx.IsFlagChanged("service-token-days") {
			days, err := cli.InputKey(ctx.Context, ui, "service-token-days", localizer.T("cli.init.prompts.serviceTokenDays"), fmt.Sprint(defaultInt(input.ServiceTokenDays, 30)))
			if err != nil {
				return err
			}
			_, _ = fmt.Sscanf(days, "%d", &input.ServiceTokenDays)
		}
		if !ctx.IsFlagChanged("service-token-remark") {
			input.ServiceTokenRemark, err = cli.InputKey(ctx.Context, ui, "service-token-remark", localizer.T("cli.init.prompts.serviceTokenRemark"), defaultString(input.ServiceTokenRemark, localizer.T("cli.init.defaults.serviceTokenRemark")))
			if err != nil {
				return err
			}
		}
	}
	if err := executeInitialization(ctx.Context, ctx.Stdout, input); err != nil {
		return err
	}
	return initservice.OfferManagedServerRestartAfterInit(ctx, input.ConfigPath)
}

func (h *InitHandler) runSetupConfigPrompts(ctx *cli.Context, ui cli.PromptUI, configPath string, includeStep func(string) bool) (bool, error) {
	localizer := localization.FromContext(ctx)
	schema, err := initservice.SetupSchema(ctx.Context, configPath)
	if err != nil {
		return false, err
	}
	for _, step := range schema.Steps {
		if !includeStep(step.Key) {
			continue
		}
		configure, err := cli.ConfirmKey(ctx.Context, ui, setupConfigurePromptKey(step.Key), localizer.T("cli.init.prompts.configureStep", map[string]any{"Step": setupStepTitle(step)}), false)
		if err != nil {
			return false, err
		}
		if !configure {
			continue
		}
		values, err := promptSetupStepValues(ctx, ui, step)
		if err != nil {
			return false, err
		}
		result, err := initservice.SaveSetupConfig(ctx.Context, configPath, step.Key, values)
		if err != nil {
			return false, err
		}
		if result.Test != nil && result.Test.Summary != "" {
			_ = ui.Info(result.Test.Summary)
		}
		if result.RestartRequired {
			reason := result.RestartReason
			if reason == "" {
				reason = localizer.T("cli.init.restartRequired")
			}
			_ = ui.Info(reason)
			return true, nil
		}
	}
	return false, nil
}

func setupConfigurePromptKey(stepKey string) string {
	if strings.HasSuffix(stepKey, ".configure") {
		return "setup." + stepKey
	}
	return "setup." + stepKey + ".configure"
}

func isCLIPreAccountConfigStep(key string) bool {
	switch key {
	case "database.configure", "cache.configure", "storage.configure", "system.configure":
		return true
	default:
		return false
	}
}

func isCLIPostAccountConfigStep(key string) bool {
	switch key {
	case "site.configure":
		return true
	default:
		return false
	}
}

func setupStepTitle(step initcenter.StepSchema) string {
	return defaultString(step.Title, step.Key)
}

func promptSetupStepValues(ctx *cli.Context, ui cli.PromptUI, step initcenter.StepSchema) (map[string]any, error) {
	values := map[string]any{}
	if len(step.Groups) > 0 {
		for _, group := range step.Groups {
			if !setupVisible(group.VisibleWhen, values) {
				continue
			}
			if group.Title != "" {
				_ = ui.Info(group.Title)
			}
			for _, field := range group.Fields {
				if !setupVisible(field.VisibleWhen, values) {
					continue
				}
				value, ok, err := promptSetupField(ctx, ui, step, field)
				if err != nil {
					return nil, err
				}
				if ok {
					values[field.Key] = value
				}
			}
		}
		return values, nil
	}
	for _, field := range step.Fields {
		if !setupVisible(field.VisibleWhen, values) {
			continue
		}
		value, ok, err := promptSetupField(ctx, ui, step, field)
		if err != nil {
			return nil, err
		}
		if ok {
			values[field.Key] = value
		}
	}
	return values, nil
}

func setupVisible(condition *initcenter.VisibilityCondition, values map[string]any) bool {
	if condition == nil || condition.Field == "" {
		return true
	}
	value := fmt.Sprint(values[condition.Field])
	if len(condition.In) > 0 {
		for _, candidate := range condition.In {
			if value == candidate {
				return true
			}
		}
		return false
	}
	if condition.Equals != nil {
		return value == fmt.Sprint(condition.Equals)
	}
	return true
}

func promptSetupField(ctx *cli.Context, ui cli.PromptUI, step initcenter.StepSchema, field initcenter.FieldSchema) (any, bool, error) {
	localizer := localization.FromContext(ctx)
	promptKey := "setup." + step.Key + "." + field.Key
	label := setupFieldLabel(field)
	defaultValue := fieldDefaultString(field)
	switch field.Type {
	case "boolean":
		value, err := cli.ConfirmKey(ctx.Context, ui, promptKey, label, fieldDefaultBool(field))
		return value, true, err
	case "select":
		options := make([]cli.SelectOption, 0, len(field.Options))
		for _, option := range field.Options {
			options = append(options, cli.SelectOption{Value: option.Value, Label: option.Label})
		}
		value, err := cli.SelectKey(ctx.Context, ui, promptKey, label, options)
		return value, true, err
	case "password":
		value, err := cli.PasswordKey(ctx.Context, ui, promptKey, localizer.T("cli.init.prompts.secretKeepCurrent", map[string]any{"Label": label}))
		if err != nil {
			return nil, false, err
		}
		if value == "" && !field.Required {
			return nil, false, nil
		}
		return value, true, nil
	default:
		value, err := cli.InputKey(ctx.Context, ui, promptKey, label, defaultValue)
		if err != nil {
			return nil, false, err
		}
		return value, true, nil
	}
}

func setupFieldLabel(field initcenter.FieldSchema) string {
	return defaultString(field.Label, field.Key)
}

func fieldDefaultString(field initcenter.FieldSchema) string {
	if field.Value != nil {
		return fmt.Sprint(field.Value)
	}
	if field.Default != nil {
		return fmt.Sprint(field.Default)
	}
	return ""
}

func fieldDefaultBool(field initcenter.FieldSchema) bool {
	switch value := field.Value.(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(value, "true")
	default:
		return false
	}
}

func InitializationInputFromContext(ctx *cli.Context) (initservice.InitializationInput, error) {
	password := ctx.GetString("admin-password")
	if ctx.GetBool("admin-password-stdin") {
		raw, err := io.ReadAll(ctx.Stdin)
		if err != nil {
			return initservice.InitializationInput{}, err
		}
		password = strings.TrimSpace(string(raw))
	}
	return initservice.InitializationInput{
		ConfigPath:         ctx.GetString("config"),
		OrgCode:            ctx.GetString("org-code"),
		OrgName:            ctx.GetString("org-name"),
		AdminUsername:      ctx.GetString("admin-username"),
		AdminEmail:         ctx.GetString("admin-email"),
		AdminDisplayName:   ctx.GetString("admin-display-name"),
		AdminPassword:      password,
		CreateServiceToken: ctx.GetBool("create-service-token"),
		ServiceTokenDays:   ctx.GetInt("service-token-days"),
		ServiceTokenRemark: ctx.GetString("service-token-remark"),
	}, nil
}
