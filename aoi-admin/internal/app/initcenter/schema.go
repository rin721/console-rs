package initcenter

import (
	appconfig "github.com/rei0721/go-scaffold/internal/config"
	cachepkg "github.com/rei0721/go-scaffold/pkg/cache"
)

func stepSchema(key, phase string, order int, _ string, _ string, required, skippable, testable bool, dependencies []string, fields []FieldSchema) StepSchema {
	if dependencies == nil {
		dependencies = []string{}
	}
	if fields == nil {
		fields = []FieldSchema{}
	}
	return StepSchema{
		Key:            key,
		RouteSlug:      routeSlugForStep(key),
		Phase:          phase,
		Order:          order,
		TitleKey:       "ui.setup.steps." + key + ".title",
		DescriptionKey: "ui.setup.steps." + key + ".description",
		Required:       required,
		Skippable:      skippable,
		Testable:       testable,
		Dependencies:   append([]string(nil), dependencies...),
		Fields:         append([]FieldSchema(nil), fields...),
		Groups:         []FieldGroup{},
	}
}

func databaseSchema() StepSchema {
	return groupedStepSchema("database.configure", "database", 5, true, false, true, nil, []FieldGroup{
		fieldGroup("driver", []FieldSchema{
			selectField("database.driver", true, "database.driver", []Option{
				{Value: appconfig.DatabaseDriverSQLite},
				{Value: appconfig.DatabaseDriverMySQL},
				{Value: appconfig.DatabaseDriverPostgres},
			}),
		}),
		fieldGroupWhen("sqlite", visibleIn("database.driver", appconfig.DatabaseDriverSQLite), []FieldSchema{
			textField("database.sqlite.path", true, "database.sqlite.path"),
		}),
		fieldGroupWhen("mysql", visibleIn("database.driver", appconfig.DatabaseDriverMySQL), []FieldSchema{
			textField("database.mysql.host", true, "database.mysql.host"),
			numberField("database.mysql.port", true, "database.mysql.port"),
			textField("database.mysql.username", true, "database.mysql.username"),
			passwordField("database.mysql.password", false, "database.mysql.password"),
			textField("database.mysql.database", true, "database.mysql.database"),
			textField("database.mysql.charset", false, "database.mysql.charset"),
		}),
		fieldGroupWhen("postgres", visibleIn("database.driver", appconfig.DatabaseDriverPostgres), []FieldSchema{
			textField("database.postgres.host", true, "database.postgres.host"),
			numberField("database.postgres.port", true, "database.postgres.port"),
			textField("database.postgres.username", true, "database.postgres.username"),
			passwordField("database.postgres.password", false, "database.postgres.password"),
			textField("database.postgres.database", true, "database.postgres.database"),
			textField("database.postgres.sslMode", false, "database.postgres.sslMode"),
		}),
		fieldGroup("pool", []FieldSchema{
			numberField("database.pool.maxOpenConns", false, "database.pool.maxOpenConns"),
			numberField("database.pool.maxIdleConns", false, "database.pool.maxIdleConns"),
		}),
	})
}

func cacheSchema() StepSchema {
	return groupedStepSchema("cache.configure", "cache", 8, false, true, true, []string{"database.configure"}, []FieldGroup{
		fieldGroup("driver", []FieldSchema{
			selectField("cache.driver", false, "cache.driver", []Option{
				{Value: appconfig.CacheDriverLocal},
				{Value: appconfig.CacheDriverHybrid},
				{Value: appconfig.CacheDriverRedis},
				{Value: appconfig.CacheDriverDisabled},
			}),
		}),
		fieldGroupWhen("local", visibleIn("cache.driver", appconfig.CacheDriverLocal, appconfig.CacheDriverHybrid), []FieldSchema{
			numberField("cache.local.maxCost", false, "cache.local.maxCost"),
			numberField("cache.local.numCounters", false, "cache.local.numCounters"),
			numberField("cache.local.bufferItems", false, "cache.local.bufferItems"),
			numberField("cache.local.defaultTtlSeconds", false, "cache.local.defaultTtlSeconds"),
		}),
		fieldGroupWhen("redis", visibleIn("cache.driver", appconfig.CacheDriverRedis, appconfig.CacheDriverHybrid), []FieldSchema{
			textField("cache.redis.addr", true, "cache.redis.addr"),
			textField("cache.redis.username", false, "cache.redis.username"),
			passwordField("cache.redis.password", false, "cache.redis.password"),
			numberField("cache.redis.db", false, "cache.redis.db"),
			numberField("cache.redis.poolSize", false, "cache.redis.poolSize"),
			numberField("cache.redis.minIdleConns", false, "cache.redis.minIdleConns"),
			numberField("cache.redis.maxRetries", false, "cache.redis.maxRetries"),
			numberField("cache.redis.dialTimeout", false, "cache.redis.dialTimeout"),
			numberField("cache.redis.readTimeout", false, "cache.redis.readTimeout"),
			numberField("cache.redis.writeTimeout", false, "cache.redis.writeTimeout"),
		}),
	})
}

func storageSchema() StepSchema {
	return groupedStepSchema("storage.configure", "storage", 4, false, true, true, nil, []FieldGroup{
		fieldGroup("driver", []FieldSchema{
			selectField("storage.driver", false, "storage.driver", []Option{
				{Value: appconfig.StorageDriverLocal},
				{Value: appconfig.StorageDriverS3},
				{Value: appconfig.StorageDriverMinIO},
				{Value: appconfig.StorageDriverLocalS3},
				{Value: appconfig.StorageDriverLocalMinIO},
				{Value: appconfig.StorageDriverDisabled},
			}),
		}),
		fieldGroupWhen("local", visibleIn("storage.driver", appconfig.StorageDriverLocal, appconfig.StorageDriverLocalS3, appconfig.StorageDriverLocalMinIO), []FieldSchema{
			selectField("storage.local.fsType", false, "storage.local.fsType", []Option{
				{Value: "os"},
				{Value: "basepath"},
				{Value: "memory"},
				{Value: "readonly"},
			}),
			textField("storage.local.basePath", true, "storage.local.basePath"),
			textField("storage.local.publicUrl", false, "storage.local.publicUrl"),
			boolField("storage.local.enableWatch", false, "storage.local.enableWatch"),
			numberField("storage.local.watchBufferSize", false, "storage.local.watchBufferSize"),
		}),
		fieldGroupWhen("s3", visibleIn("storage.driver", appconfig.StorageDriverS3, appconfig.StorageDriverLocalS3), []FieldSchema{
			textField("storage.s3.endpoint", true, "storage.s3.endpoint"),
			textField("storage.s3.region", false, "storage.s3.region"),
			textField("storage.s3.bucket", true, "storage.s3.bucket"),
			textField("storage.s3.accessKeyId", true, "storage.s3.accessKeyId"),
			passwordField("storage.s3.secretAccessKey", true, "storage.s3.secretAccessKey"),
			boolField("storage.s3.usePathStyle", false, "storage.s3.usePathStyle"),
			textField("storage.s3.publicBaseUrl", false, "storage.s3.publicBaseUrl"),
		}),
		fieldGroupWhen("minio", visibleIn("storage.driver", appconfig.StorageDriverMinIO, appconfig.StorageDriverLocalMinIO), []FieldSchema{
			textField("storage.minio.endpoint", true, "storage.minio.endpoint"),
			textField("storage.minio.region", false, "storage.minio.region"),
			textField("storage.minio.bucket", true, "storage.minio.bucket"),
			textField("storage.minio.accessKeyId", true, "storage.minio.accessKeyId"),
			passwordField("storage.minio.secretAccessKey", true, "storage.minio.secretAccessKey"),
			boolField("storage.minio.usePathStyle", false, "storage.minio.usePathStyle"),
			textField("storage.minio.publicBaseUrl", false, "storage.minio.publicBaseUrl"),
		}),
	})
}

func systemSchema() StepSchema {
	return groupedStepSchema("system.configure", "system", 20, true, false, true, []string{"database.configure"}, []FieldGroup{
		fieldGroup("locale", []FieldSchema{
			selectField("i18n.defaultLocale", true, "i18n.defaultLocale", []Option{
				{Value: "zh-CN"},
				{Value: "en-US"},
			}),
		}),
		fieldGroup("security", []FieldSchema{
			textField("auth.issuer", true, "auth.issuer"),
			numberField("auth.password_policy.min_length", true, "auth.password_policy.min_length"),
			boolField("system.seed_defaults_on_start", false, "system.seed_defaults_on_start"),
		}),
	})
}

func siteSchema() StepSchema {
	return groupedStepSchema("site.configure", "site", 75, true, false, true, []string{"iam.owner"}, []FieldGroup{
		fieldGroup("site", []FieldSchema{
			textField("brand.productName", true, "brand.productName"),
			textField("brand.versionName", true, "brand.versionName"),
			textField("webui.public_base_url", false, "webui.public_base_url"),
		}),
	})
}

func iamOwnerSchema() StepSchema {
	return stepSchema("iam.owner", "iam", 70, "", "", true, false, false, []string{"catalog.sync"}, []FieldSchema{
		textField("orgCode", true, ""),
		textField("orgName", true, ""),
		textField("username", true, ""),
		emailField("email", true, ""),
		textField("displayName", false, ""),
		passwordField("password", true, ""),
	})
}

func optionalFinalizeSchema() StepSchema {
	return stepSchema("optional.finalize", "finalize", 80, "", "", false, true, false, []string{"site.configure"}, []FieldSchema{
		boolField("createServiceToken", false, ""),
		numberField("serviceTokenDays", false, ""),
		textField("serviceTokenRemark", false, ""),
	})
}

func groupedStepSchema(key, phase string, order int, required, skippable, testable bool, dependencies []string, groups []FieldGroup) StepSchema {
	fields := flattenGroups(groups)
	schema := stepSchema(key, phase, order, "", "", required, skippable, testable, dependencies, fields)
	schema.Groups = normalizeGroups(groups)
	return schema
}

func fieldGroup(key string, fields []FieldSchema) FieldGroup {
	if fields == nil {
		fields = []FieldSchema{}
	}
	return FieldGroup{
		Key:            key,
		TitleKey:       "ui.setup.groups." + key + ".title",
		DescriptionKey: "ui.setup.groups." + key + ".description",
		Fields:         append([]FieldSchema(nil), fields...),
	}
}

func fieldGroupWhen(key string, visibleWhen *VisibilityCondition, fields []FieldSchema) FieldGroup {
	group := fieldGroup(key, fields)
	group.VisibleWhen = visibleWhen
	for i := range group.Fields {
		if group.Fields[i].VisibleWhen == nil {
			group.Fields[i].VisibleWhen = visibleWhen
		}
	}
	return group
}

func visibleIn(field string, values ...string) *VisibilityCondition {
	return &VisibilityCondition{Field: field, In: append([]string(nil), values...)}
}

func flattenGroups(groups []FieldGroup) []FieldSchema {
	fields := []FieldSchema{}
	for _, group := range groups {
		fields = append(fields, group.Fields...)
	}
	return fields
}

func normalizeGroups(groups []FieldGroup) []FieldGroup {
	if groups == nil {
		return []FieldGroup{}
	}
	out := make([]FieldGroup, 0, len(groups))
	for _, group := range groups {
		if group.Fields == nil {
			group.Fields = []FieldSchema{}
		} else {
			group.Fields = append([]FieldSchema(nil), group.Fields...)
		}
		out = append(out, group)
	}
	return out
}

func routeSlugForStep(key string) string {
	switch key {
	case "welcome":
		return "welcome"
	case "config.source":
		return "config-source"
	case "config.diagnostics":
		return "diagnostics"
	case "dependencies.check":
		return "preflight"
	case "database.configure":
		return "database"
	case "cache.configure":
		return "cache"
	case "storage.configure":
		return "storage"
	case "system.configure":
		return "system"
	case "site.configure":
		return "site"
	case "database.migrate":
		return "migrate"
	case "system.seed":
		return "seed"
	case "catalog.sync":
		return "permissions"
	case "iam.owner":
		return "owner"
	case "optional.finalize":
		return "finalize"
	case "verify.finish":
		return "verify"
	default:
		return key
	}
}

func textField(key string, required bool, configPath string) FieldSchema {
	return withFieldDefault(FieldSchema{Key: key, Type: "text", Required: required, ConfigPath: configPath})
}

func emailField(key string, required bool, configPath string) FieldSchema {
	return withFieldDefault(FieldSchema{Key: key, Type: "email", Required: required, ConfigPath: configPath})
}

func passwordField(key string, required bool, configPath string) FieldSchema {
	return withFieldDefault(FieldSchema{Key: key, Type: "password", Required: required, Sensitive: true, ConfigPath: configPath})
}

func numberField(key string, required bool, configPath string) FieldSchema {
	return withFieldDefault(FieldSchema{Key: key, Type: "number", Required: required, ConfigPath: configPath})
}

func boolField(key string, required bool, configPath string) FieldSchema {
	return withFieldDefault(FieldSchema{Key: key, Type: "boolean", Required: required, ConfigPath: configPath})
}

func selectField(key string, required bool, configPath string, options []Option) FieldSchema {
	for index := range options {
		if options[index].LabelKey == "" {
			options[index].LabelKey = "ui.setup.options." + key + "." + sanitizeKeyPart(options[index].Value)
		}
	}
	return withFieldDefault(FieldSchema{Key: key, Type: "select", Required: required, ConfigPath: configPath, Options: options})
}

func withFieldDefault(field FieldSchema) FieldSchema {
	field.LabelKey = "ui.setup.fields." + field.Key + ".label"
	field.HelpKey = "ui.setup.fields." + field.Key + ".help"
	if value, ok := defaultValueForField(field.Key); ok {
		field.Default = value
	}
	return field
}

func sanitizeKeyPart(value string) string {
	out := make([]rune, 0, len(value))
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			out = append(out, r)
		case r >= 'A' && r <= 'Z':
			out = append(out, r+('a'-'A'))
		case r >= '0' && r <= '9':
			out = append(out, r)
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}

func defaultValueForField(key string) (any, bool) {
	defaults := map[string]any{
		"database.driver":                 appconfig.DatabaseDriverSQLite,
		"database.sqlite.path":            "./data/app.db",
		"database.mysql.host":             "127.0.0.1",
		"database.mysql.port":             3306,
		"database.mysql.username":         "root",
		"database.mysql.database":         "aoi_admin",
		"database.mysql.charset":          "utf8mb4",
		"database.postgres.host":          "127.0.0.1",
		"database.postgres.port":          5432,
		"database.postgres.username":      "postgres",
		"database.postgres.database":      "aoi_admin",
		"database.postgres.sslMode":       "disable",
		"database.pool.maxOpenConns":      1,
		"database.pool.maxIdleConns":      1,
		"cache.driver":                    appconfig.CacheDriverLocal,
		"cache.redis.addr":                "127.0.0.1:6379",
		"cache.redis.db":                  0,
		"storage.driver":                  appconfig.StorageDriverLocal,
		"storage.local.fsType":            "basepath",
		"storage.local.basePath":          "./data/uploads",
		"storage.local.publicUrl":         "/uploads",
		"storage.local.watchBufferSize":   100,
		"storage.s3.region":               "auto",
		"storage.s3.usePathStyle":         false,
		"storage.minio.endpoint":          "http://127.0.0.1:9000",
		"storage.minio.region":            "us-east-1",
		"storage.minio.usePathStyle":      true,
		"i18n.defaultLocale":              "zh-CN",
		"auth.issuer":                     "aoi-admin",
		"auth.password_policy.min_length": 8,
		"system.seed_defaults_on_start":   true,
		"orgCode":                         "default",
		"orgName":                         "Default Organization",
		"username":                        "admin",
		"email":                           "admin@example.com",
		"displayName":                     "Admin",
		"serviceTokenDays":                365,
		"serviceTokenRemark":              "initial setup",
	}
	value, ok := defaults[key]
	return value, ok
}

func configValue(cfg *appconfig.Config, path string) (any, bool) {
	if cfg == nil {
		return nil, false
	}
	switch path {
	case "database.driver":
		return cfg.Database.Driver, true
	case "database.sqlite.path":
		return cfg.Database.SQLite.Path, true
	case "database.mysql.host":
		return cfg.Database.MySQL.Host, true
	case "database.mysql.port":
		return cfg.Database.MySQL.Port, true
	case "database.mysql.username":
		return cfg.Database.MySQL.Username, true
	case "database.mysql.password":
		return cfg.Database.MySQL.Password, true
	case "database.mysql.database":
		return cfg.Database.MySQL.Database, true
	case "database.mysql.charset":
		return cfg.Database.MySQL.Charset, true
	case "database.postgres.host":
		return cfg.Database.Postgres.Host, true
	case "database.postgres.port":
		return cfg.Database.Postgres.Port, true
	case "database.postgres.username":
		return cfg.Database.Postgres.Username, true
	case "database.postgres.password":
		return cfg.Database.Postgres.Password, true
	case "database.postgres.database":
		return cfg.Database.Postgres.Database, true
	case "database.postgres.sslMode":
		return cfg.Database.Postgres.SSLMode, true
	case "database.pool.maxOpenConns":
		return cfg.Database.Pool.MaxOpenConns, true
	case "database.pool.maxIdleConns":
		return cfg.Database.Pool.MaxIdleConns, true
	case "cache.driver":
		return cfg.Cache.Driver, true
	case "cache.local.maxCost":
		if cfg.Cache.Local.MaxCost > 0 {
			return cfg.Cache.Local.MaxCost, true
		}
		return cachepkg.DefaultLocalConfig().MaxCost, true
	case "cache.local.numCounters":
		if cfg.Cache.Local.NumCounters > 0 {
			return cfg.Cache.Local.NumCounters, true
		}
		return cachepkg.DefaultLocalConfig().NumCounters, true
	case "cache.local.bufferItems":
		if cfg.Cache.Local.BufferItems > 0 {
			return cfg.Cache.Local.BufferItems, true
		}
		return cachepkg.DefaultLocalConfig().BufferItems, true
	case "cache.local.defaultTtlSeconds":
		return cfg.Cache.Local.DefaultTTLSeconds, true
	case "cache.redis.addr":
		return cfg.Cache.Redis.Addr, true
	case "cache.redis.username":
		return cfg.Cache.Redis.Username, true
	case "cache.redis.password":
		return cfg.Cache.Redis.Password, true
	case "cache.redis.db":
		return cfg.Cache.Redis.DB, true
	case "cache.redis.poolSize":
		return cfg.Cache.Redis.PoolSize, true
	case "cache.redis.minIdleConns":
		return cfg.Cache.Redis.MinIdleConns, true
	case "cache.redis.maxRetries":
		return cfg.Cache.Redis.MaxRetries, true
	case "cache.redis.dialTimeout":
		return cfg.Cache.Redis.DialTimeout, true
	case "cache.redis.readTimeout":
		return cfg.Cache.Redis.ReadTimeout, true
	case "cache.redis.writeTimeout":
		return cfg.Cache.Redis.WriteTimeout, true
	case "storage.driver":
		return cfg.Storage.Driver, true
	case "storage.local.fsType":
		return cfg.Storage.Local.FSType, true
	case "storage.local.basePath":
		return cfg.Storage.Local.BasePath, true
	case "storage.local.publicUrl":
		return cfg.Storage.Local.PublicURL, true
	case "storage.local.enableWatch":
		return cfg.Storage.Local.EnableWatch, true
	case "storage.local.watchBufferSize":
		return cfg.Storage.Local.WatchBufferSize, true
	case "storage.s3.endpoint":
		return cfg.Storage.S3.Endpoint, true
	case "storage.s3.region":
		return cfg.Storage.S3.Region, true
	case "storage.s3.bucket":
		return cfg.Storage.S3.Bucket, true
	case "storage.s3.accessKeyId":
		return cfg.Storage.S3.AccessKeyID, true
	case "storage.s3.secretAccessKey":
		return cfg.Storage.S3.SecretAccessKey, true
	case "storage.s3.usePathStyle":
		return cfg.Storage.S3.UsePathStyle, true
	case "storage.s3.publicBaseUrl":
		return cfg.Storage.S3.PublicBaseURL, true
	case "storage.minio.endpoint":
		return cfg.Storage.MinIO.Endpoint, true
	case "storage.minio.region":
		return cfg.Storage.MinIO.Region, true
	case "storage.minio.bucket":
		return cfg.Storage.MinIO.Bucket, true
	case "storage.minio.accessKeyId":
		return cfg.Storage.MinIO.AccessKeyID, true
	case "storage.minio.secretAccessKey":
		return cfg.Storage.MinIO.SecretAccessKey, true
	case "storage.minio.usePathStyle":
		return cfg.Storage.MinIO.UsePathStyle, true
	case "storage.minio.publicBaseUrl":
		return cfg.Storage.MinIO.PublicBaseURL, true
	case "i18n.defaultLocale":
		return cfg.I18n.DefaultLocale, true
	case "brand.productName":
		return cfg.Brand.ProductName, true
	case "brand.versionName":
		return cfg.Brand.VersionName, true
	case "webui.public_base_url":
		return cfg.WebUI.PublicBaseURL, true
	case "auth.issuer":
		return cfg.Auth.Issuer, true
	case "auth.password_policy.min_length":
		return cfg.Auth.PasswordPolicy.MinLength, true
	case "system.seed_defaults_on_start":
		return cfg.System.SeedDefaultsOnStartValue(), true
	default:
		return nil, false
	}
}
