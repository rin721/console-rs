package localization

import (
	"os"
	"path/filepath"
	"strings"

	appconfig "github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/cli"
	"github.com/rei0721/go-scaffold/pkg/i18n"
	"github.com/rei0721/go-scaffold/types/constants"
)

const (
	FlagLocale = "locale"
	EnvLocale  = "AOI_LOCALE"
)

type Localizer struct {
	manager i18n.I18n
	locale  string
}

func ForArgs(args []string) *Localizer {
	locale := localeFromArgs(args)
	if locale == "" {
		locale = strings.TrimSpace(os.Getenv(EnvLocale))
	}
	return New(locale, configPathFromArgs(args))
}

func FromContext(ctx *cli.Context) *Localizer {
	if ctx == nil {
		return New(strings.TrimSpace(os.Getenv(EnvLocale)), constants.AppDefaultConfigPath)
	}
	locale := strings.TrimSpace(ctx.GetString(FlagLocale))
	if locale == "" {
		locale = strings.TrimSpace(os.Getenv(EnvLocale))
	}
	configPath := strings.TrimSpace(ctx.GetString("config"))
	if configPath == "" {
		configPath = constants.AppDefaultConfigPath
	}
	return New(locale, configPath)
}

func New(locale string, configPath string) *Localizer {
	cfg := loadConfig(configPath)
	resources := defaultResources()
	defaultLocale := i18n.DefaultLanguage
	fallbackLocale := i18n.DefaultLanguage
	supported := i18n.SupportedLanguagesStringSlice
	if cfg != nil {
		if strings.TrimSpace(cfg.I18n.DefaultLocale) != "" {
			defaultLocale = cfg.I18n.DefaultLocale
		}
		if strings.TrimSpace(cfg.I18n.FallbackLocale) != "" {
			fallbackLocale = cfg.I18n.FallbackLocale
		}
		if len(cfg.I18n.Supported) > 0 {
			supported = append([]string(nil), cfg.I18n.Supported...)
		}
		if len(cfg.I18n.Resources) > 0 {
			resources = cfg.I18n.Resources
		}
	}
	manager, err := i18n.New(&i18n.Config{
		DefaultLocale:  defaultLocale,
		FallbackLocale: fallbackLocale,
		Supported:      supported,
		Resources:      resources,
	})
	if err != nil {
		manager, _ = i18n.New(&i18n.Config{
			DefaultLocale:  i18n.DefaultLanguage,
			FallbackLocale: i18n.DefaultLanguage,
			Supported:      i18n.SupportedLanguagesStringSlice,
			Resources:      defaultResources(),
		})
	}
	if manager == nil {
		manager = i18n.Default()
	}
	resolved := manager.ResolveLocale(locale, defaultLocale, fallbackLocale)
	return &Localizer{manager: manager, locale: resolved}
}

func (l *Localizer) Locale() string {
	if l == nil {
		return i18n.DefaultLanguage
	}
	return l.locale
}

func (l *Localizer) T(key string, data ...map[string]any) string {
	if l == nil || l.manager == nil {
		return key
	}
	args := map[string]any(nil)
	if len(data) > 0 {
		args = data[0]
	}
	return l.manager.Localize(l.locale, i18n.NamespaceUI, key, args)
}

func LocaleFlag(l *Localizer) cli.FlagSpec {
	description := "Display locale"
	if l != nil {
		description = l.T("cli.flags.locale.description")
	}
	return cli.FlagSpec{Name: FlagLocale, Type: cli.FlagTypeString, Description: description, EnvVar: EnvLocale}
}

func defaultResources() map[string]string {
	root := resourceRoot()
	return map[string]string{
		i18n.NamespaceUI:         filepath.Join(root, "configs", "locales", "ui"),
		i18n.NamespaceAPI:        filepath.Join(root, "configs", "locales", "api"),
		i18n.NamespaceValidation: filepath.Join(root, "configs", "locales", "validation"),
		i18n.NamespaceSystem:     filepath.Join(root, "configs", "locales", "system"),
	}
}

func resourceRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if info, err := os.Stat(filepath.Join(cwd, "configs", "locales", "ui")); err == nil && info.IsDir() {
			return cwd
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			return "."
		}
		cwd = parent
	}
}

func loadConfig(configPath string) *appconfig.Config {
	configPath = strings.TrimSpace(configPath)
	if configPath == "" {
		configPath = constants.AppDefaultConfigPath
	}
	manager := appconfig.NewManager()
	if err := manager.Load(configPath); err != nil {
		return nil
	}
	return manager.Get()
}

func localeFromArgs(args []string) string {
	for index, arg := range args {
		if arg == "--"+FlagLocale && index+1 < len(args) {
			return strings.TrimSpace(args[index+1])
		}
		if value, ok := strings.CutPrefix(arg, "--"+FlagLocale+"="); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func configPathFromArgs(args []string) string {
	for index, arg := range args {
		switch {
		case arg == "--config" || arg == "-c":
			if index+1 < len(args) {
				return strings.TrimSpace(args[index+1])
			}
		case strings.HasPrefix(arg, "--config="):
			return strings.TrimSpace(strings.TrimPrefix(arg, "--config="))
		case strings.HasPrefix(arg, "-c="):
			return strings.TrimSpace(strings.TrimPrefix(arg, "-c="))
		}
	}
	return constants.AppDefaultConfigPath
}
