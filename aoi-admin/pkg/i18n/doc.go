// Package i18n provides namespace-aware localization primitives.
//
// The package is intentionally infrastructure-only: it loads translation
// resources, resolves locales, interpolates template data, and records missing
// keys. Product copy ownership stays in resource files and application/system
// configuration.
//
// Resource directories are grouped by namespace, for example:
//
//	configs/locales/ui/zh-CN.yaml
//	configs/locales/api/zh-CN.yaml
//	configs/locales/validation/zh-CN.yaml
//	configs/locales/system/zh-CN.yaml
//
// Application code should keep UI copy, API messages, validation messages, and
// system-derived labels in separate namespaces instead of mixing them in one
// flat file.
package i18n
