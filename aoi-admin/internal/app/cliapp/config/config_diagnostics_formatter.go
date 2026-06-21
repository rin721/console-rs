package config

import (
	"fmt"
	"strings"

	"github.com/rei0721/go-scaffold/internal/app/cliapp/localization"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
)

// configDiagnosticsError 将多条配置诊断包装为一个 CLI 可直接展示的错误。
type configDiagnosticsError struct {
	configPath  string
	diagnostics []appconfig.ConfigDiagnostic
	localizer   *localization.Localizer
}

// newConfigDiagnosticsError 创建聚合诊断错误。
func newConfigDiagnosticsError(configPath string, diagnostics []appconfig.ConfigDiagnostic, localizers ...*localization.Localizer) error {
	return configDiagnosticsError{configPath: configPath, diagnostics: diagnostics, localizer: firstConfigLocalizer(localizers...)}
}

func (e configDiagnosticsError) Error() string {
	return formatConfigDiagnostics(e.configPath, e.diagnostics, true, e.localizer)
}

// printConfigDiagnostics 输出不带最终建议的 preflight 诊断摘要。
func printConfigDiagnostics(w interface{ Write([]byte) (int, error) }, configPath string, diagnostics []appconfig.ConfigDiagnostic, localizers ...*localization.Localizer) {
	if w == nil || len(diagnostics) == 0 {
		return
	}
	_, _ = fmt.Fprint(w, formatConfigDiagnostics(configPath, diagnostics, false, firstConfigLocalizer(localizers...)))
}

// formatConfigDiagnostics 将诊断按配置区段分组，便于 CLI 用户一次性定位所有阻断项。
func formatConfigDiagnostics(configPath string, diagnostics []appconfig.ConfigDiagnostic, includeAdvice bool, localizer *localization.Localizer) string {
	if localizer == nil {
		localizer = localization.ForArgs(nil)
	}
	var builder strings.Builder
	builder.WriteString(localizer.T("cli.config.diagnostics.summary", map[string]any{"Count": len(diagnostics), "ConfigPath": configPath}))
	builder.WriteByte('\n')
	lastSection := ""
	for _, diagnostic := range diagnostics {
		section := diagnostic.Section
		if section == "" {
			section = "config"
		}
		if section != lastSection {
			fmt.Fprintf(&builder, "[%s]\n", section)
			lastSection = section
		}
		path := diagnostic.Path
		if path == "" {
			path = section
		}
		fmt.Fprintf(&builder, "  - %s: %s", path, diagnostic.Message)
		if len(diagnostic.EnvNames) > 0 {
			fmt.Fprintf(&builder, " (env: %s)", strings.Join(diagnostic.EnvNames, " or "))
		}
		builder.WriteByte('\n')
	}
	if includeAdvice {
		builder.WriteString(localizer.T("cli.config.diagnostics.advice"))
	}
	return builder.String()
}

func firstConfigLocalizer(localizers ...*localization.Localizer) *localization.Localizer {
	for _, localizer := range localizers {
		if localizer != nil {
			return localizer
		}
	}
	return localization.ForArgs(nil)
}
