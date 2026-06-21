package i18n

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

const (
	NamespaceUI         = "ui"
	NamespaceAPI        = "api"
	NamespaceValidation = "validation"
	NamespaceSystem     = "system"
)

var DefaultNamespaces = []string{NamespaceUI, NamespaceAPI, NamespaceValidation, NamespaceSystem}

type MissingKey struct {
	Key       string
	Locale    string
	Namespace string
}

type MissingKeyLogger func(MissingKey)

type I18n interface {
	Localize(locale string, namespace string, key string, data map[string]any) string
	ResolveLocale(candidates ...string) string
	ValidateResources() error
	IsSupported(locale string) bool
	DefaultLocale() string
	FallbackLocale() string
	LoadNamespace(namespace string, dir string) error
	MissingKeys() []MissingKey
}

type Config struct {
	DefaultLocale  string
	FallbackLocale string
	Supported      []string
	Resources      map[string]string
	MissingLogger  MissingKeyLogger
}

type namespaceBundle struct {
	bundle *goi18n.Bundle
	loaded map[string]bool
}

type manager struct {
	defaultLocale  string
	fallbackLocale string
	supported      map[string]bool
	resources      map[string]string
	namespaces     map[string]*namespaceBundle
	missingLogger  MissingKeyLogger

	mu      sync.Mutex
	missing []MissingKey
}

func New(cfg *Config) (I18n, error) {
	if cfg == nil {
		cfg = &Config{}
	}
	defaultLocale := strings.TrimSpace(cfg.DefaultLocale)
	if defaultLocale == "" {
		defaultLocale = DefaultLanguage
	}
	fallbackLocale := strings.TrimSpace(cfg.FallbackLocale)
	if fallbackLocale == "" {
		fallbackLocale = defaultLocale
	}
	supported := append([]string(nil), cfg.Supported...)
	if len(supported) == 0 {
		supported = SupportedLanguagesStringSlice
	}
	if !containsLocale(supported, defaultLocale) {
		supported = append(supported, defaultLocale)
	}
	if !containsLocale(supported, fallbackLocale) {
		supported = append(supported, fallbackLocale)
	}
	if _, err := language.Parse(defaultLocale); err != nil {
		return nil, fmt.Errorf("invalid default locale %q: %w", defaultLocale, err)
	}
	if _, err := language.Parse(fallbackLocale); err != nil {
		return nil, fmt.Errorf("invalid fallback locale %q: %w", fallbackLocale, err)
	}
	supportedSet := make(map[string]bool, len(supported))
	for _, locale := range supported {
		locale = strings.TrimSpace(locale)
		if locale == "" {
			continue
		}
		if _, err := language.Parse(locale); err != nil {
			return nil, fmt.Errorf("invalid supported locale %q: %w", locale, err)
		}
		supportedSet[locale] = true
	}
	if len(supportedSet) == 0 {
		return nil, errors.New("at least one supported locale is required")
	}
	m := &manager{
		defaultLocale:  defaultLocale,
		fallbackLocale: fallbackLocale,
		supported:      supportedSet,
		resources:      copyStringMap(cfg.Resources),
		namespaces:     map[string]*namespaceBundle{},
		missingLogger:  cfg.MissingLogger,
	}
	for namespace, dir := range cfg.Resources {
		if strings.TrimSpace(namespace) == "" || strings.TrimSpace(dir) == "" {
			continue
		}
		if err := m.LoadNamespace(namespace, dir); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func Default() I18n {
	m, err := New(&Config{
		DefaultLocale:  DefaultLanguage,
		FallbackLocale: DefaultLanguage,
		Supported:      SupportedLanguagesStringSlice,
	})
	if err != nil {
		panic(fmt.Sprintf("create default i18n: %v", err))
	}
	return m
}

func (m *manager) Localize(locale string, namespace string, key string, data map[string]any) string {
	namespace = strings.TrimSpace(namespace)
	key = strings.TrimSpace(key)
	if namespace == "" || key == "" {
		return key
	}
	resolvedLocale := m.ResolveLocale(locale)
	if msg, ok := m.localize(resolvedLocale, namespace, key, data); ok {
		return msg
	}
	if m.fallbackLocale != resolvedLocale {
		if msg, ok := m.localize(m.fallbackLocale, namespace, key, data); ok {
			m.recordMissing(MissingKey{Locale: resolvedLocale, Namespace: namespace, Key: key})
			return msg
		}
	}
	m.recordMissing(MissingKey{Locale: resolvedLocale, Namespace: namespace, Key: key})
	return key
}

func (m *manager) ResolveLocale(candidates ...string) string {
	for _, candidate := range candidates {
		for _, locale := range parseLocaleCandidates(candidate) {
			if m.IsSupported(locale) {
				return locale
			}
			if normalized := normalizeShortLocale(locale, m.supported); normalized != "" {
				return normalized
			}
		}
	}
	return m.defaultLocale
}

func (m *manager) ValidateResources() error {
	for _, namespace := range DefaultNamespaces {
		if _, ok := m.namespaces[namespace]; !ok {
			return fmt.Errorf("i18n namespace %q is not loaded", namespace)
		}
	}
	var missing []string
	for namespace, bundle := range m.namespaces {
		for locale := range m.supported {
			if !bundle.loaded[locale] {
				missing = append(missing, namespace+"/"+locale)
			}
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("missing i18n resource files: %s", strings.Join(missing, ", "))
	}
	return nil
}

func (m *manager) IsSupported(locale string) bool {
	return m.supported[strings.TrimSpace(locale)]
}

func (m *manager) DefaultLocale() string {
	return m.defaultLocale
}

func (m *manager) FallbackLocale() string {
	return m.fallbackLocale
}

func (m *manager) LoadNamespace(namespace string, dir string) error {
	namespace = strings.TrimSpace(namespace)
	dir = strings.TrimSpace(dir)
	if namespace == "" {
		return errors.New("i18n namespace is required")
	}
	if dir == "" {
		return fmt.Errorf("i18n namespace %q resource directory is required", namespace)
	}
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("i18n namespace %q directory: %w", namespace, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("i18n namespace %q path is not a directory: %s", namespace, dir)
	}
	defaultTag, err := language.Parse(m.defaultLocale)
	if err != nil {
		return err
	}
	bundle := goi18n.NewBundle(defaultTag)
	bundle.RegisterUnmarshalFunc(FilenameFormatJson, json.Unmarshal)
	bundle.RegisterUnmarshalFunc(FilenameFormatYaml, yaml.Unmarshal)
	bundle.RegisterUnmarshalFunc(FilenameFormatYml, yaml.Unmarshal)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read i18n namespace %q directory: %w", namespace, err)
	}
	loaded := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.TrimPrefix(filepath.Ext(entry.Name()), ".")
		if ext != FilenameFormatJson && ext != FilenameFormatYaml && ext != FilenameFormatYml {
			continue
		}
		locale := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if !m.IsSupported(locale) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if _, err := bundle.LoadMessageFile(path); err != nil {
			return fmt.Errorf("load i18n namespace %q file %s: %w", namespace, entry.Name(), err)
		}
		loaded[locale] = true
	}
	if len(loaded) == 0 {
		return fmt.Errorf("i18n namespace %q has no resource files in %s", namespace, dir)
	}
	m.namespaces[namespace] = &namespaceBundle{bundle: bundle, loaded: loaded}
	m.resources[namespace] = dir
	return nil
}

func (m *manager) MissingKeys() []MissingKey {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]MissingKey(nil), m.missing...)
}

func (m *manager) localize(locale string, namespace string, key string, data map[string]any) (string, bool) {
	bundle, ok := m.namespaces[namespace]
	if !ok {
		return "", false
	}
	localizer := goi18n.NewLocalizer(bundle.bundle, locale)
	message, err := localizer.Localize(&goi18n.LocalizeConfig{
		MessageID:    key,
		TemplateData: data,
	})
	if err != nil {
		return "", false
	}
	return message, true
}

func (m *manager) recordMissing(item MissingKey) {
	m.mu.Lock()
	m.missing = append(m.missing, item)
	m.mu.Unlock()
	if m.missingLogger != nil {
		m.missingLogger(item)
	}
}

func containsLocale(locales []string, needle string) bool {
	for _, locale := range locales {
		if strings.TrimSpace(locale) == needle {
			return true
		}
	}
	return false
}

func parseLocaleCandidates(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		if index := strings.Index(value, ";"); index >= 0 {
			value = strings.TrimSpace(value[:index])
		}
		if value != "" && value != "*" {
			out = append(out, value)
		}
	}
	return out
}

func normalizeShortLocale(locale string, supported map[string]bool) string {
	locale = strings.TrimSpace(locale)
	if len(locale) != 2 {
		return ""
	}
	prefix := locale + "-"
	for candidate := range supported {
		if strings.HasPrefix(candidate, prefix) {
			return candidate
		}
	}
	return ""
}

func copyStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
