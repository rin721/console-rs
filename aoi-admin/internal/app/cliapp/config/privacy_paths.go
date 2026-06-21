package config

import (
	"sort"
	"strings"
)

// normalizePrivacyPaths 过滤、去重并排序隐私配置路径。
func normalizePrivacyPaths(paths []string) []string {
	normalized := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" || !supportedPrivacyPath(path) {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		normalized = append(normalized, path)
	}
	sort.Strings(normalized)
	return normalized
}

// isGeneratedSecretPath 判断路径是否属于可自动生成的 IAM 核心密钥。
func isGeneratedSecretPath(path string) bool {
	for _, candidate := range coreSecretPaths {
		if path == candidate {
			return true
		}
	}
	return false
}
