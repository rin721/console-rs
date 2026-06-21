package config

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rei0721/go-scaffold/types/constants"
)

// DiscoverConfigFiles 返回启动向导可选配置，默认配置优先。
func DiscoverConfigFiles() []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, 4)
	add := func(path string) {
		path = filepath.Clean(strings.TrimSpace(path))
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		if _, err := os.Stat(path); err != nil {
			return
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	add(constants.AppDefaultConfigPath)
	matches, _ := filepath.Glob(filepath.Join("configs", "*.yaml"))
	sort.Strings(matches)
	for _, match := range matches {
		add(match)
	}
	return out
}
