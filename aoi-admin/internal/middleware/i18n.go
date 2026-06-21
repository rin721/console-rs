package middleware

import (
	"strings"

	"github.com/rei0721/go-scaffold/internal/ports"
)

const (
	LocaleContextKey = "locale"
	I18nContextKey   = "i18n"
	headerLocale     = "X-Locale"
	headerLanguage   = "Accept-Language"
	defaultLocale    = "zh-CN"
)

func I18n(i18nApp ports.I18n) ports.HTTPHandlerFunc {
	return func(c ports.HTTPContext) {
		locale := i18nApp.ResolveLocale(
			c.GetHeader(headerLocale),
			c.GetHeader(headerLanguage),
		)
		c.Set(LocaleContextKey, locale)
		c.Set(I18nContextKey, i18nApp)
		c.Next()
	}
}

func LocaleFromContext(c ports.HTTPContext, i18nApp ports.I18n) string {
	if c != nil {
		if value, ok := c.Get(LocaleContextKey); ok {
			if locale, ok := value.(string); ok && strings.TrimSpace(locale) != "" {
				return locale
			}
		}
	}
	if i18nApp != nil {
		return i18nApp.DefaultLocale()
	}
	return defaultLocale
}
