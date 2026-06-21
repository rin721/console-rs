package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/rei0721/go-scaffold/internal/app/cliapp/localization"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
)

// randomSecret 生成 URL-safe 的 32 字节随机密钥。
//
// 随机源失败时返回显眼的占位值，让后续配置校验和人工检查能发现问题。
func randomSecret() string {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "change-me-generated-secret-32-bytes"
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

// isCoreSecretConfigError 判断配置错误是否来自 IAM 核心密钥缺失或过短。
func isCoreSecretConfigError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	if !strings.Contains(message, "auth config:") {
		return false
	}
	for _, needle := range []string{
		"signing_key must be at least 32 bytes",
		"refresh_token_pepper is required",
		"mfa_secret_key must be at least 32 bytes",
	} {
		if strings.Contains(message, needle) {
			return true
		}
	}
	return false
}

// coreSecretConfigError 为 IAM 核心密钥错误补充可操作的 CLI 提示。
func coreSecretConfigError(configPath string, err error) error {
	if !isCoreSecretConfigError(err) {
		return err
	}
	localizer := localization.ForArgs(nil)
	return fmt.Errorf("%w; %s", err, localizer.T("cli.config.privacy.error.coreSecretMissing", map[string]any{"ConfigPath": configPath, "Env": coreSecretEnvHelp()}))
}

// coreSecretEnvHelp 返回核心密钥路径及其可用环境变量名。
func coreSecretEnvHelp() string {
	parts := make([]string, 0, len(coreSecretPaths))
	for _, path := range coreSecretPaths {
		names := appconfig.EnvNamesForPath(path)
		if len(names) == 0 {
			parts = append(parts, path)
			continue
		}
		parts = append(parts, path+" ("+strings.Join(names, " or ")+")")
	}
	return strings.Join(parts, ", ")
}
