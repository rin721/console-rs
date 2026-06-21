package config

import (
	"errors"
	"strings"
)

type I18nConfig struct {
	DefaultLocale  string            `mapstructure:"defaultLocale" envname:"I18N_DEFAULT_LOCALE" json:"defaultLocale" yaml:"defaultLocale" toml:"defaultLocale"`
	FallbackLocale string            `mapstructure:"fallbackLocale" envname:"I18N_FALLBACK_LOCALE" json:"fallbackLocale" yaml:"fallbackLocale" toml:"fallbackLocale"`
	Supported      []string          `mapstructure:"supportedLocales" envname:"I18N_SUPPORTED_LOCALES" json:"supportedLocales" yaml:"supportedLocales" toml:"supportedLocales"`
	Resources      map[string]string `mapstructure:"resources" envname:"-" json:"resources" yaml:"resources" toml:"resources"`
}

func (c *I18nConfig) ValidateName() string {
	return AppI18nName
}

func (c *I18nConfig) ValidateRequired() bool {
	return true
}

func (c *I18nConfig) Validate() error {
	if strings.TrimSpace(c.DefaultLocale) == "" {
		return errors.New("defaultLocale is required")
	}
	if strings.TrimSpace(c.FallbackLocale) == "" {
		return errors.New("fallbackLocale is required")
	}
	if len(c.Supported) == 0 {
		return errors.New("supportedLocales must not be empty")
	}
	supported := map[string]struct{}{}
	for _, locale := range c.Supported {
		locale = strings.TrimSpace(locale)
		if locale == "" {
			return errors.New("supportedLocales contains an empty locale")
		}
		supported[locale] = struct{}{}
	}
	if _, ok := supported[c.DefaultLocale]; !ok {
		return errors.New("defaultLocale must be listed in supportedLocales")
	}
	if _, ok := supported[c.FallbackLocale]; !ok {
		return errors.New("fallbackLocale must be listed in supportedLocales")
	}
	required := []string{"ui", "api", "validation", "system"}
	for _, namespace := range required {
		if strings.TrimSpace(c.Resources[namespace]) == "" {
			return errors.New("i18n.resources." + namespace + " is required")
		}
	}
	return nil
}

func overrideI18nConfig(cfg *I18nConfig) {
	overrideConfigFromEnv(cfg)
}
