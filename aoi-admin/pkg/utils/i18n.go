package utils

import "github.com/rei0721/go-scaffold/pkg/i18n"

type I18nUtils struct {
	_i18n         i18n.I18n
	defaultLocale string
}

func NewI18nUtils(_i18n i18n.I18n, defaultLocale string) *I18nUtils {
	return &I18nUtils{_i18n: _i18n, defaultLocale: defaultLocale}
}

func (i I18nUtils) Localize(namespace string, key string, data map[string]any) string {
	if i._i18n == nil {
		return key
	}
	return i._i18n.Localize(i.defaultLocale, namespace, key, data)
}

func (i I18nUtils) UI(key string, data map[string]any) string {
	return i.Localize(i18n.NamespaceUI, key, data)
}

func (i I18nUtils) API(key string, data map[string]any) string {
	return i.Localize(i18n.NamespaceAPI, key, data)
}

func (i I18nUtils) Validation(key string, data map[string]any) string {
	return i.Localize(i18n.NamespaceValidation, key, data)
}

func (i I18nUtils) System(key string, data map[string]any) string {
	return i.Localize(i18n.NamespaceSystem, key, data)
}
