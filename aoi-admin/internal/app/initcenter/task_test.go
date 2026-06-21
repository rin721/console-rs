package initcenter

import (
	"context"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/rei0721/go-scaffold/internal/app/initapp"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/database"
	"github.com/rei0721/go-scaffold/pkg/utils"
)

func TestInitTaskRegistryResolveOrdersDependencies(t *testing.T) {
	registry := NewInitTaskRegistry()
	for _, def := range []stepDefinition{
		{Key: "verify", Order: 30, Dependencies: []string{"migrate"}},
		{Key: "config", Order: 10},
		{Key: "migrate", Order: 20, Dependencies: []string{"config"}},
	} {
		if err := registry.Register(taskAdapter{def: def}); err != nil {
			t.Fatalf("register %s: %v", def.Key, err)
		}
	}

	defs, err := registry.Resolve()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	got := make([]string, 0, len(defs))
	for _, def := range defs {
		got = append(got, def.Key)
	}
	want := []string{"config", "migrate", "verify"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolved order = %v, want %v", got, want)
	}
}

func TestInitTaskRegistryRejectsDuplicate(t *testing.T) {
	registry := NewInitTaskRegistry()
	if err := registry.Register(taskAdapter{def: stepDefinition{Key: "config"}}); err != nil {
		t.Fatalf("register first task: %v", err)
	}
	if err := registry.Register(taskAdapter{def: stepDefinition{Key: "config"}}); err == nil {
		t.Fatalf("expected duplicate task error")
	}
}

func TestBaseDefinitionsOrderSetupConfigurationSteps(t *testing.T) {
	service := New(initapp.Core{}, initapp.Infrastructure{}, initapp.Modules{}, "", nil)
	defs := service.definitions()
	index := map[string]int{}
	for position, def := range defs {
		index[def.Key] = position
	}
	assertBefore := func(left, right string) {
		t.Helper()
		leftIndex, leftOK := index[left]
		rightIndex, rightOK := index[right]
		if !leftOK || !rightOK {
			t.Fatalf("missing setup steps %s=%v %s=%v", left, leftOK, right, rightOK)
		}
		if leftIndex >= rightIndex {
			t.Fatalf("setup step %s index=%d must be before %s index=%d; order=%v", left, leftIndex, right, rightIndex, stepKeys(defs))
		}
	}

	assertBefore("storage.configure", "database.configure")
	assertBefore("database.configure", "cache.configure")
	assertBefore("database.configure", "system.configure")
	assertBefore("catalog.sync", "iam.owner")
	assertBefore("iam.owner", "site.configure")
	assertBefore("site.configure", "optional.finalize")
	assertBefore("site.configure", "verify.finish")
}

func TestInitDependencyResolverRejectsMissingDependency(t *testing.T) {
	resolver := InitDependencyResolver{tasks: map[string]stepDefinition{
		"migrate": {Key: "migrate", Dependencies: []string{"config"}},
	}}
	if _, err := resolver.Resolve(); err == nil {
		t.Fatalf("expected missing dependency error")
	}
}

func TestInitDependencyResolverRejectsCycle(t *testing.T) {
	resolver := InitDependencyResolver{tasks: map[string]stepDefinition{
		"a": {Key: "a", Dependencies: []string{"b"}},
		"b": {Key: "b", Dependencies: []string{"a"}},
	}}
	if _, err := resolver.Resolve(); err == nil {
		t.Fatalf("expected cyclic dependency error")
	}
}

func stepKeys(defs []stepDefinition) []string {
	out := make([]string, 0, len(defs))
	for _, def := range defs {
		out = append(out, def.Key)
	}
	return out
}

func TestStatusWithoutIAMModuleUsesBootstrapUserCheck(t *testing.T) {
	db, err := database.New(&database.Config{
		Driver: database.DriverSQLite,
		DBName: filepath.Join(t.TempDir(), "app.db"),
		Silent: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	service := New(
		initapp.Core{
			Config: &appconfig.Config{
				Auth: appconfig.AuthConfig{
					Enabled: true,
					PasswordPolicy: appconfig.PasswordPolicyConfig{
						MinLength: 8,
					},
				},
			},
			IDGenerator: utils.DefaultSnowflake(),
		},
		initapp.Infrastructure{Database: db},
		initapp.Modules{},
		"",
		nil,
	)
	status, err := service.Status(context.Background())
	if err != nil {
		t.Fatalf("status on empty bootstrap database: %v", err)
	}
	if !status.Required {
		t.Fatalf("status.Required = false, want true for empty IAM users table")
	}
	if status.PasswordPolicy.MinLength != 8 {
		t.Fatalf("password policy min length = %d, want 8", status.PasswordPolicy.MinLength)
	}
}

func TestSetupSchemaUsesI18nKeysWithoutDisplayFallbacks(t *testing.T) {
	schemas := []StepSchema{
		databaseSchema(),
		cacheSchema(),
		storageSchema(),
		systemSchema(),
		siteSchema(),
		iamOwnerSchema(),
		optionalFinalizeSchema(),
	}

	for _, schema := range schemas {
		if schema.Title != "" || schema.Description != "" {
			t.Fatalf("%s contains display fallback title=%q description=%q", schema.Key, schema.Title, schema.Description)
		}
		if schema.TitleKey == "" || schema.DescriptionKey == "" {
			t.Fatalf("%s missing title/description keys", schema.Key)
		}
		assertFieldsUseI18nKeys(t, schema.Fields)
		for _, group := range schema.Groups {
			if group.Title != "" || group.Description != "" {
				t.Fatalf("%s group %s contains display fallback title=%q description=%q", schema.Key, group.Key, group.Title, group.Description)
			}
			if group.TitleKey == "" || group.DescriptionKey == "" {
				t.Fatalf("%s group %s missing title/description keys", schema.Key, group.Key)
			}
			assertFieldsUseI18nKeys(t, group.Fields)
		}
	}
}

func TestSiteConfigureOwnsOnlySiteDisplayConfigPaths(t *testing.T) {
	values := map[string]any{
		"auth.issuer":                   "should-stay-system",
		"brand.productName":             "Aoi",
		"brand.versionName":             "Community",
		"i18n.defaultLocale":            "en-US",
		"webui.public_base_url":         "https://admin.example.com",
		"system.seed_defaults_on_start": false,
	}

	sitePaths := configPathsForStep("site.configure", values)
	sort.Strings(sitePaths)
	if !reflect.DeepEqual(sitePaths, []string{"brand.productName", "brand.versionName", "webui.public_base_url"}) {
		t.Fatalf("site.configure paths = %v", sitePaths)
	}
	systemPaths := configPathsForStep("system.configure", values)
	for _, path := range systemPaths {
		switch path {
		case "brand.productName", "brand.versionName", "webui.public_base_url":
			t.Fatalf("system.configure still accepts site path %s: %v", path, systemPaths)
		}
	}

	cfg := &appconfig.Config{}
	for _, path := range sitePaths {
		if err := setConfigPath(cfg, path, values[path]); err != nil {
			t.Fatalf("setConfigPath(%s): %v", path, err)
		}
	}
	if cfg.Brand.ProductName != "Aoi" || cfg.Brand.VersionName != "Community" || cfg.WebUI.PublicBaseURL != "https://admin.example.com" {
		t.Fatalf("site fields not written: brand=%#v webui=%#v", cfg.Brand, cfg.WebUI)
	}
}

func TestSiteConfigureValidatorChecksVisibleFieldsOnly(t *testing.T) {
	validator := NewInitValidator(&Service{core: initapp.Core{Config: &appconfig.Config{}}})
	failed := validator.Test(context.Background(), "site.configure", map[string]any{
		"brand.productName": "",
		"brand.versionName": "Community",
	})
	if failed.Status != "failed" || !strings.Contains(failed.Error, "productName") {
		t.Fatalf("site.configure failed result = %#v", failed)
	}

	passed := validator.Test(context.Background(), "site.configure", map[string]any{
		"brand.productName":     "Aoi Admin",
		"brand.versionName":     "Community",
		"webui.public_base_url": "https://admin.example.com",
	})
	if passed.Status != "succeeded" {
		t.Fatalf("site.configure status = %s error=%s hint=%s", passed.Status, passed.Error, passed.RepairHint)
	}
}

func assertFieldsUseI18nKeys(t *testing.T, fields []FieldSchema) {
	t.Helper()
	for _, field := range fields {
		if field.Label != "" || field.Help != "" || field.Placeholder != "" {
			t.Fatalf("%s contains display fallback label=%q help=%q placeholder=%q", field.Key, field.Label, field.Help, field.Placeholder)
		}
		if field.LabelKey == "" || field.HelpKey == "" {
			t.Fatalf("%s missing label/help keys", field.Key)
		}
		for _, option := range field.Options {
			if option.Label != "" {
				t.Fatalf("%s option %s contains display fallback label=%q", field.Key, option.Value, option.Label)
			}
			if option.LabelKey == "" {
				t.Fatalf("%s option %s missing label key", field.Key, option.Value)
			}
		}
	}
}
