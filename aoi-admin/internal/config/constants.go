package config

import (
	"os"
	"strings"
	"unicode"

	app "github.com/rei0721/go-scaffold/types/constants"
)

func EnvPrefix() string {
	prefix := normalizeEnvToken(app.AppPrefix)
	if prefix == "" {
		return "APP"
	}
	return prefix + "_APP"
}

func EnvConfigPathName() string {
	prefix := normalizeEnvToken(app.AppPrefix)
	if prefix == "" {
		return "CONFIG_PATH"
	}
	return prefix + "_CONFIG_PATH"
}

func ResolveConfigPath(flagValue string, flagChanged bool) string {
	if flagChanged {
		return strings.TrimSpace(flagValue)
	}
	if value := strings.TrimSpace(os.Getenv("APP_CONFIG")); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv(EnvConfigPathName())); value != "" {
		return value
	}
	return app.AppDefaultConfigPath
}

func EnvPrefixJoin(field string) string {
	field = strings.TrimSpace(field)
	prefix := EnvPrefix()
	if field == "" {
		return prefix
	}
	if prefix == "" || strings.HasPrefix(field, prefix+"_") {
		return field
	}
	return prefix + "_" + field
}

func normalizeEnvToken(value string) string {
	var builder strings.Builder
	lastUnderscore := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(unicode.ToUpper(r))
			lastUnderscore = false
			continue
		}
		if !lastUnderscore && builder.Len() > 0 {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(builder.String(), "_")
}

const (
	EnvFilePath        = ".env"
	EnvFilePathExample = ".env.example"
	DefaultSeparator   = ","
)

const (
	AppServerName    = "server"
	AppDatabaseName  = "database"
	AppCacheName     = "cache"
	AppLoggerName    = "logger"
	AppI18nName      = "i18n"
	AppBrandName     = "brand"
	AppExecutorName  = "executor"
	AppStorageName   = "storage"
	AppCORSName      = "cors"
	AppRPCName       = "rpc"
	AppAuthName      = "auth"
	AppSystemName    = "system"
	AppMigrationName = "migration"
	AppWebUIName     = "webui"
	AppPluginsName   = "plugins"
)
