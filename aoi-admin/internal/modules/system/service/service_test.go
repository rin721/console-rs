package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rei0721/go-scaffold/internal/modules/system/model"
)

func TestSyncAPIsPersistsCurrentRoutesAndMarksStaleRecords(t *testing.T) {
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	repo := newMemoryAPIRepo([]model.APIRecord{
		{
			ID:          1,
			Code:        "get /api/v1/system/apis",
			Group:       "system",
			Method:      "GET",
			Path:        "/api/v1/system/apis",
			Description: "old",
			Permission:  "permission:read",
			Status:      model.APIStatusActive,
			Source:      "router",
			SyncedAt:    now.Add(-time.Hour),
			CreatedAt:   now.Add(-time.Hour),
			UpdatedAt:   now.Add(-time.Hour),
		},
		{
			ID:          2,
			Code:        "get /api/v1/old",
			Group:       "system",
			Method:      "GET",
			Path:        "/api/v1/old",
			Description: "old route",
			Status:      model.APIStatusActive,
			Source:      "router",
			SyncedAt:    now.Add(-time.Hour),
			CreatedAt:   now.Add(-time.Hour),
			UpdatedAt:   now.Add(-time.Hour),
		},
	})
	svc := New(Config{Now: func() time.Time { return now }},
		WithRepository(repo),
		WithIDGenerator(&sequenceIDGenerator{next: 100}),
	)
	svc.RegisterAPIs([]model.APIEntry{
		{Code: "get /api/v1/system/apis", Group: "system", Method: "GET", Path: "/api/v1/system/apis", Description: "catalog", Permission: "permission:read", Order: 10},
		{Code: "post /api/v1/system/apis/sync", Group: "system", Method: "POST", Path: "/api/v1/system/apis/sync", Description: "sync", Permission: "permission:read", Order: 20},
	})

	result, err := svc.SyncAPIs(context.Background())
	if err != nil {
		t.Fatalf("SyncAPIs() error = %v", err)
	}
	if !result.Persisted || result.StorageStatus != "persisted" {
		t.Fatalf("expected persisted sync result, got %#v", result)
	}
	if result.Total != 2 || result.Created != 1 || result.Updated != 1 || result.Stale != 1 {
		t.Fatalf("unexpected sync counters: %#v", result)
	}

	oldRecord, ok := repo.record("GET", "/api/v1/old")
	if !ok || oldRecord.Status != model.APIStatusStale {
		t.Fatalf("expected old route to be stale, got %#v", oldRecord)
	}
	created, ok := repo.record("POST", "/api/v1/system/apis/sync")
	if !ok || created.ID != 100 || created.Status != model.APIStatusActive {
		t.Fatalf("expected sync route to be created, got %#v", created)
	}

	groups, err := svc.ListAPIs(context.Background())
	if err != nil {
		t.Fatalf("ListAPIs() error = %v", err)
	}
	if !apiEntrySynced(groups, "GET", "/api/v1/system/apis") || !apiEntrySynced(groups, "POST", "/api/v1/system/apis/sync") {
		t.Fatalf("expected listed API entries to include sync metadata, got %#v", groups)
	}
}

func TestSyncPermissionsRegistersRoutePermissionsAndAnnotatesCatalog(t *testing.T) {
	now := time.Date(2026, 6, 11, 13, 0, 0, 0, time.UTC)
	permissions := newMemoryPermissionStore([]model.PermissionEntry{
		{Code: "permission:read", Scope: "platform", Name: "Read permissions", Description: "Read permissions"},
	})
	svc := New(Config{Now: func() time.Time { return now }},
		WithPermissionStore(permissions),
	)
	svc.RegisterAPIs([]model.APIEntry{
		{Code: "get /api/v1/system/apis", Group: "system", Method: "GET", Path: "/api/v1/system/apis", Description: "catalog", Permission: "permission:read", Scope: "platform", Order: 10},
		{Code: "post /api/v1/system/apis/permissions/sync", Group: "system", Method: "POST", Path: "/api/v1/system/apis/permissions/sync", Description: "sync permissions", Permission: "permission:sync", Scope: "platform", Order: 20},
		{Code: "get /api/v1/system/menus", Group: "system", Method: "GET", Path: "/api/v1/system/menus", Description: "menus", Order: 30},
		{Code: "get /api/v1/broken", Group: "system", Method: "GET", Path: "/api/v1/broken", Description: "broken", Permission: "broken", Order: 40},
	})

	result, err := svc.SyncPermissions(context.Background())
	if err != nil {
		t.Fatalf("SyncPermissions() error = %v", err)
	}
	if !result.Persisted || result.StorageStatus != "persisted" {
		t.Fatalf("expected persisted permission sync, got %#v", result)
	}
	if result.Total != 2 || result.Created != 1 || result.Skipped != 1 {
		t.Fatalf("unexpected permission sync counters: %#v", result)
	}
	if !permissions.has("permission:sync") {
		t.Fatalf("expected permission:sync to be created, got %#v", permissions.records)
	}

	groups, err := svc.ListAPIs(context.Background())
	if err != nil {
		t.Fatalf("ListAPIs() error = %v", err)
	}
	if !apiEntryPermissionRegistered(groups, "GET", "/api/v1/system/apis") {
		t.Fatalf("expected permission:read route to be marked registered: %#v", groups)
	}
	if !apiEntryPermissionRegistered(groups, "POST", "/api/v1/system/apis/permissions/sync") {
		t.Fatalf("expected permission:sync route to be marked registered: %#v", groups)
	}
}

func TestListMenusIncludesSystemMenuCatalog(t *testing.T) {
	svc := New(Config{})
	groups, err := svc.ListMenus(context.Background())
	if err != nil {
		t.Fatalf("ListMenus() error = %v", err)
	}
	if !menuItemExists(groups, "system", "menus", "/menus", "permission:read") {
		t.Fatalf("expected system menu catalog entry, got %#v", groups)
	}
	if !menuItemExists(groups, "system", "apis", "/apis", "permission:read") {
		t.Fatalf("expected system API catalog entry, got %#v", groups)
	}
	if !menuItemExists(groups, "system", "dictionaries", "/dictionaries", "dictionary:read") {
		t.Fatalf("expected system dictionary catalog entry, got %#v", groups)
	}
	if !menuItemExists(groups, "system", "operation-records", "/operation-records", "operation:read") {
		t.Fatalf("expected operation history entry, got %#v", groups)
	}
	if !menuItemExists(groups, "system", "parameters", "/parameters", "parameter:read") {
		t.Fatalf("expected system parameter management entry, got %#v", groups)
	}
	if !menuItemExists(groups, "system", "system-config", "/system", "config:read") {
		t.Fatalf("expected system config entry, got %#v", groups)
	}
	if !menuItemExists(groups, "security", "login-logs", "/login-logs", "audit:read") {
		t.Fatalf("expected login log entry, got %#v", groups)
	}
	if !menuItemExists(groups, "security", "error-logs", "/error-logs", "operation:read") {
		t.Fatalf("expected error log entry, got %#v", groups)
	}
}

func TestListConfigUsesProviderAndReturnsSnapshotClone(t *testing.T) {
	current := model.ConfigSnapshot{
		Sections: []model.ConfigSection{
			{
				Code:  "server",
				Label: "System",
				Items: []model.ConfigItem{
					{Key: "server.port", Label: "Port", Value: 9999},
				},
			},
		},
	}
	svc := New(Config{
		ConfigProvider: func() model.ConfigSnapshot { return current },
	})

	snapshot, err := svc.ListConfig(context.Background())
	if err != nil {
		t.Fatalf("ListConfig() error = %v", err)
	}
	if len(snapshot.Sections) != 1 || len(snapshot.Sections[0].Items) != 1 {
		t.Fatalf("unexpected config snapshot: %#v", snapshot)
	}
	snapshot.Sections[0].Items[0].Value = 10000
	snapshot.Sections[0].Items = append(snapshot.Sections[0].Items, model.ConfigItem{Key: "server.mode"})

	again, err := svc.ListConfig(context.Background())
	if err != nil {
		t.Fatalf("ListConfig() second error = %v", err)
	}
	if again.Sections[0].Items[0].Value != 9999 || len(again.Sections[0].Items) != 1 {
		t.Fatalf("expected stored snapshot to remain unchanged, got %#v", again)
	}

	current = model.ConfigSnapshot{
		Sections: []model.ConfigSection{
			{
				Code:  "server",
				Label: "System",
				Items: []model.ConfigItem{
					{Key: "server.port", Label: "Port", Value: 10000},
				},
			},
		},
	}
	latest, err := svc.ListConfig(context.Background())
	if err != nil {
		t.Fatalf("ListConfig() latest error = %v", err)
	}
	if latest.Sections[0].Items[0].Value != 10000 {
		t.Fatalf("expected provider refresh to be reflected, got %#v", latest)
	}
}

func TestGetPublicSettingsReadsBrandAndLocalesFromConfigSnapshot(t *testing.T) {
	current := model.ConfigSnapshot{
		Sections: []model.ConfigSection{
			{
				Code: "brand",
				Items: []model.ConfigItem{
					{Key: "brand.productName", Value: "Aoi Admin"},
					{Key: "brand.productCode", Value: "aoi-admin"},
					{Key: "brand.versionName", Value: "Community"},
				},
			},
			{
				Code: "runtime",
				Items: []model.ConfigItem{
					{Key: "i18n.defaultLocale", Value: "zh-CN"},
					{Key: "i18n.fallbackLocale", Value: "zh-CN"},
					{Key: "i18n.supportedLocales", Value: []string{"zh-CN", "en-US"}},
				},
			},
			{
				Code: "auth",
				Items: []model.ConfigItem{
					{Key: "auth.registration_mode", Value: "email_verification"},
					{Key: "auth.csrf.enabled", Value: true},
					{Key: "auth.csrf.cookie_name", Value: "aoi_csrf"},
					{Key: "auth.csrf.header_name", Value: "X-CSRF-Token"},
					{Key: "auth.session.product_header", Value: "X-Aoi-Product-Code"},
					{Key: "auth.session.client_type_header", Value: "X-Aoi-Client-Type"},
					{Key: "auth.session.default_client_type", Value: "pc_web"},
				},
			},
		},
	}
	svc := New(Config{
		ConfigProvider: func() model.ConfigSnapshot { return current },
	})

	settings, err := svc.GetPublicSettings(context.Background())
	if err != nil {
		t.Fatalf("GetPublicSettings() error = %v", err)
	}
	if settings.Brand.ProductName != "Aoi Admin" || settings.Brand.ProductCode != "aoi-admin" || settings.Brand.VersionName != "Community" {
		t.Fatalf("unexpected public brand settings: %#v", settings.Brand)
	}
	if settings.DefaultLocale != "zh-CN" || settings.FallbackLocale != "zh-CN" {
		t.Fatalf("unexpected public locale settings: %#v", settings)
	}
	if len(settings.SupportedLocales) != 2 || settings.SupportedLocales[0] != "zh-CN" || settings.SupportedLocales[1] != "en-US" {
		t.Fatalf("unexpected supported locales: %#v", settings.SupportedLocales)
	}
	if settings.Auth.RegistrationMode != "email_verification" {
		t.Fatalf("registration mode = %q, want email_verification", settings.Auth.RegistrationMode)
	}
}

func TestUpdateConfigUsesUpdaterAndReturnsSnapshotClone(t *testing.T) {
	var calls int
	current := model.ConfigSnapshot{
		Sections: []model.ConfigSection{
			{
				Code:  "server",
				Label: "System",
				Items: []model.ConfigItem{
					{Key: "server.port", Label: "Port", Value: 9999},
				},
			},
		},
	}
	svc := New(Config{
		ConfigUpdater: func(_ context.Context, input UpdateConfigInput) (model.ConfigSnapshot, error) {
			calls++
			if !input.Persist || len(input.Items) != 1 || input.Items[0].Key != "server.port" || input.Items[0].Value != 10000 {
				t.Fatalf("unexpected update input: %#v", input)
			}
			current.Sections[0].Items[0].Value = input.Items[0].Value
			return current, nil
		},
	})

	snapshot, err := svc.UpdateConfig(context.Background(), UpdateConfigInput{
		Items:   []UpdateConfigItem{{Key: " server.port ", Value: 10000}},
		Persist: true,
	})
	if err != nil {
		t.Fatalf("UpdateConfig() error = %v", err)
	}
	if calls != 1 || snapshot.Sections[0].Items[0].Value != 10000 {
		t.Fatalf("expected updater result, calls=%d snapshot=%#v", calls, snapshot)
	}
	snapshot.Sections[0].Items[0].Value = 18081
	if current.Sections[0].Items[0].Value != 10000 {
		t.Fatalf("expected UpdateConfig() to return clone, current=%#v", current)
	}
}

func TestUpdateConfigWithoutUpdaterReturnsUnavailable(t *testing.T) {
	svc := New(Config{})

	if _, err := svc.UpdateConfig(context.Background(), UpdateConfigInput{
		Items: []UpdateConfigItem{{Key: "server.port", Value: 10000}},
	}); !errors.Is(err, ErrConfigUnavailable) {
		t.Fatalf("UpdateConfig() error = %v, want ErrConfigUnavailable", err)
	}
}

func TestGetServerInfoReportsRuntimeAndMemory(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	svc := New(Config{
		Now:       func() time.Time { return now },
		StartTime: now.Add(-time.Hour),
	}, WithHostMetrics(fakeHostMetricsCollector{metrics: HostMetrics{
		CPU: CPUInfo{Cores: 4, Percent: []float64{10.5}},
		RAM: RAMInfo{TotalMB: 8192, UsedMB: 4096, UsedPercent: 50},
		Disk: []DiskInfo{{
			FSType:      "ext4",
			MountPoint:  "/",
			TotalGB:     100,
			TotalMB:     102400,
			UsedGB:      50,
			UsedMB:      51200,
			UsedPercent: 50,
		}},
	}}))

	info, err := svc.GetServerInfo(context.Background())
	if err != nil {
		t.Fatalf("GetServerInfo() error = %v", err)
	}
	if info.OS.GoOS == "" || info.OS.GoArch == "" || info.OS.GoVersion == "" || info.OS.NumCPU <= 0 {
		t.Fatalf("expected runtime OS fields, got %#v", info.OS)
	}
	if info.Runtime.StartTime != now.Add(-time.Hour) || info.Runtime.UptimeSeconds != 3600 || info.Runtime.Uptime == "" {
		t.Fatalf("unexpected runtime info: %#v", info.Runtime)
	}
	if info.Memory.SysMB == 0 || info.Memory.HeapObjects == 0 {
		t.Fatalf("expected memory stats, got %#v", info.Memory)
	}
	if info.CPU.Cores <= 0 {
		t.Fatalf("expected host CPU core count, got %#v", info.CPU)
	}
	for _, value := range info.CPU.Percent {
		if value < 0 {
			t.Fatalf("expected non-negative CPU percentage, got %#v", info.CPU)
		}
	}
	if info.RAM.TotalMB > 0 && info.RAM.UsedMB > info.RAM.TotalMB {
		t.Fatalf("expected RAM usage to fit total, got %#v", info.RAM)
	}
	for _, item := range info.Disk {
		if item.MountPoint == "" || item.TotalMB == 0 || item.UsedPercent < 0 {
			t.Fatalf("expected valid disk entry, got %#v", item)
		}
	}
	if info.Build.GoVersion == "" {
		t.Fatalf("expected build info go version, got %#v", info.Build)
	}
	if !info.RefreshedAt.Equal(now) {
		t.Fatalf("expected refreshedAt %s, got %s", now, info.RefreshedAt)
	}
}

func TestGetServerMetricsHistoryReturnsProviderSamples(t *testing.T) {
	sampledAt := time.Date(2026, 6, 12, 12, 1, 0, 0, time.UTC)
	svc := New(Config{}, WithMetricsHistory(fakeMetricsHistoryProvider{history: MetricsHistory{
		IntervalSeconds: 5,
		WindowSeconds:   300,
		Samples: []MetricsSample{{
			SampledAt:             sampledAt,
			CPUUsedPercent:        12.5,
			RAMUsedPercent:        45.2,
			DiskMaxUsedPercent:    67.8,
			DiskReadMBPerSecond:   1.2,
			DiskWriteMBPerSecond:  0.8,
			DiskReadOpsPerSecond:  3.4,
			DiskWriteOpsPerSecond: 2.1,
			DiskIOLatencyMs:       4.5,
			DiskIO: []DiskIOSample{{
				Name:              "disk0",
				ReadMBPerSecond:   1.2,
				WriteMBPerSecond:  0.8,
				ReadOpsPerSecond:  3.4,
				WriteOpsPerSecond: 2.1,
				IOLatencyMs:       4.5,
			}},
			HeapAllocMB:                32,
			Goroutines:                 27,
			NetworkReceiveKBPerSecond:  4.7,
			NetworkTransmitKBPerSecond: 5.4,
		}},
	}}))

	history, err := svc.GetServerMetricsHistory(context.Background())
	if err != nil {
		t.Fatalf("GetServerMetricsHistory() error = %v", err)
	}
	if history.IntervalSeconds != 5 || history.WindowSeconds != 300 || len(history.Samples) != 1 {
		t.Fatalf("unexpected history metadata: %#v", history)
	}
	sample := history.Samples[0]
	if !sample.SampledAt.Equal(sampledAt) ||
		sample.CPUUsedPercent != 12.5 ||
		sample.RAMUsedPercent != 45.2 ||
		sample.DiskMaxUsedPercent != 67.8 ||
		sample.DiskReadMBPerSecond != 1.2 ||
		sample.DiskWriteMBPerSecond != 0.8 ||
		sample.DiskReadOpsPerSecond != 3.4 ||
		sample.DiskWriteOpsPerSecond != 2.1 ||
		sample.DiskIOLatencyMs != 4.5 ||
		sample.HeapAllocMB != 32 ||
		sample.Goroutines != 27 ||
		sample.NetworkReceiveKBPerSecond != 4.7 ||
		sample.NetworkTransmitKBPerSecond != 5.4 {
		t.Fatalf("unexpected sample: %#v", sample)
	}
	if len(sample.DiskIO) != 1 ||
		sample.DiskIO[0].Name != "disk0" ||
		sample.DiskIO[0].ReadMBPerSecond != 1.2 ||
		sample.DiskIO[0].WriteMBPerSecond != 0.8 ||
		sample.DiskIO[0].ReadOpsPerSecond != 3.4 ||
		sample.DiskIO[0].WriteOpsPerSecond != 2.1 ||
		sample.DiskIO[0].IOLatencyMs != 4.5 {
		t.Fatalf("unexpected disk IO sample: %#v", sample.DiskIO)
	}
}

type fakeHostMetricsCollector struct {
	metrics HostMetrics
}

func (c fakeHostMetricsCollector) Collect(context.Context) HostMetrics {
	return c.metrics
}

type fakeMetricsHistoryProvider struct {
	history MetricsHistory
}

func (p fakeMetricsHistoryProvider) History(context.Context) MetricsHistory {
	out := p.history
	out.Samples = append([]MetricsSample(nil), p.history.Samples...)
	return out
}

func TestDictionaryManagementCreatesUpdatesAndDeletesDictionariesAndItems(t *testing.T) {
	now := time.Date(2026, 6, 12, 9, 0, 0, 0, time.UTC)
	repo := newMemoryAPIRepo(nil)
	svc := New(Config{Now: func() time.Time { return now }},
		WithRepository(repo),
		WithIDGenerator(&sequenceIDGenerator{next: 100}),
	)

	dictionary, err := svc.CreateDictionary(context.Background(), CreateDictionaryInput{
		Code:        " Status ",
		Description: "Workflow status",
		Name:        "Status",
	})
	if err != nil {
		t.Fatalf("CreateDictionary() error = %v", err)
	}
	if dictionary.ID != 100 || dictionary.Code != "status" || dictionary.Status != model.DictionaryStatusActive {
		t.Fatalf("unexpected dictionary: %#v", dictionary)
	}

	item, err := svc.CreateDictionaryItem(context.Background(), dictionary.ID, CreateDictionaryItemInput{
		Label: "Enabled",
		Sort:  20,
		Value: "enabled",
	})
	if err != nil {
		t.Fatalf("CreateDictionaryItem() error = %v", err)
	}
	if item.ID != 101 || item.DictionaryID != dictionary.ID || item.Status != model.DictionaryStatusActive {
		t.Fatalf("unexpected dictionary item: %#v", item)
	}

	catalog, err := svc.ListDictionaries(context.Background())
	if err != nil {
		t.Fatalf("ListDictionaries() error = %v", err)
	}
	if catalog.StorageStatus != "persisted" || catalog.Total != 1 || len(catalog.Items[0].Items) != 1 {
		t.Fatalf("unexpected dictionary catalog: %#v", catalog)
	}

	name := "Status Dictionary"
	status := model.DictionaryStatusDisabled
	updated, err := svc.UpdateDictionary(context.Background(), dictionary.ID, UpdateDictionaryInput{Name: &name, Status: &status})
	if err != nil {
		t.Fatalf("UpdateDictionary() error = %v", err)
	}
	if updated.Name != name || updated.Status != model.DictionaryStatusDisabled {
		t.Fatalf("unexpected updated dictionary: %#v", updated)
	}

	label := "Active"
	sortOrder := 5
	updatedItem, err := svc.UpdateDictionaryItem(context.Background(), item.ID, UpdateDictionaryItemInput{Label: &label, Sort: &sortOrder})
	if err != nil {
		t.Fatalf("UpdateDictionaryItem() error = %v", err)
	}
	if updatedItem.Label != label || updatedItem.Sort != sortOrder {
		t.Fatalf("unexpected updated item: %#v", updatedItem)
	}

	if err := svc.DeleteDictionaryItem(context.Background(), item.ID); err != nil {
		t.Fatalf("DeleteDictionaryItem() error = %v", err)
	}
	catalog, err = svc.ListDictionaries(context.Background())
	if err != nil {
		t.Fatalf("ListDictionaries() after item delete error = %v", err)
	}
	if len(catalog.Items[0].Items) != 0 {
		t.Fatalf("expected item to be removed from catalog, got %#v", catalog.Items[0].Items)
	}

	if err := svc.DeleteDictionary(context.Background(), dictionary.ID); err != nil {
		t.Fatalf("DeleteDictionary() error = %v", err)
	}
	catalog, err = svc.ListDictionaries(context.Background())
	if err != nil {
		t.Fatalf("ListDictionaries() after dictionary delete error = %v", err)
	}
	if catalog.Total != 0 {
		t.Fatalf("expected dictionary to be removed from catalog, got %#v", catalog)
	}
}

func TestOperationRecordManagementPersistsFiltersAndDeletesRecords(t *testing.T) {
	now := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)
	repo := newMemoryAPIRepo(nil)
	svc := New(Config{Now: func() time.Time { return now }},
		WithRepository(repo),
		WithIDGenerator(&sequenceIDGenerator{next: 300}),
	)

	if err := svc.RecordOperation(context.Background(), OperationRecordInput{
		Body:      `{"name":"aoi"}`,
		IPAddress: "127.0.0.1",
		LatencyMs: 32,
		Method:    "get",
		Path:      "/api/v1/system/menus",
		Status:    200,
		TraceID:   "trace-1",
		UserAgent: "test-agent",
		UserID:    1,
		Username:  "admin",
	}); err != nil {
		t.Fatalf("RecordOperation() error = %v", err)
	}
	if err := svc.RecordOperation(context.Background(), OperationRecordInput{
		IPAddress:    "127.0.0.1",
		Method:       "delete",
		Path:         "/api/v1/system/operation-records",
		Response:     `{"deleted":true}`,
		Status:       204,
		UserID:       1,
		Username:     "admin",
		ErrorMessage: strings.Repeat("x", 9000),
	}); err != nil {
		t.Fatalf("RecordOperation() second error = %v", err)
	}

	page, err := svc.ListOperationRecords(context.Background(), OperationRecordFilter{
		Method:   "DELETE",
		Page:     1,
		PageSize: 10,
		Path:     "operation-records",
		Status:   204,
	})
	if err != nil {
		t.Fatalf("ListOperationRecords() error = %v", err)
	}
	if page.StorageStatus != "persisted" || page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("unexpected operation record page: %#v", page)
	}
	record := page.Items[0]
	if record.ID != 301 || record.Method != "DELETE" || record.Username != "admin" || record.ErrorMessage == "" {
		t.Fatalf("unexpected operation record: %#v", record)
	}
	if len(record.ErrorMessage) <= 8192 || !strings.Contains(record.ErrorMessage, "truncated") {
		t.Fatalf("expected long operation payload to be truncated, got len=%d", len(record.ErrorMessage))
	}

	if err := svc.DeleteOperationRecords(context.Background(), []int64{record.ID}); err != nil {
		t.Fatalf("DeleteOperationRecords() error = %v", err)
	}
	page, err = svc.ListOperationRecords(context.Background(), OperationRecordFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListOperationRecords() after delete error = %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != 300 {
		t.Fatalf("expected only first operation record to remain, got %#v", page)
	}
}

func TestOperationRecordStatusClassFilters(t *testing.T) {
	now := time.Date(2026, 6, 12, 10, 30, 0, 0, time.UTC)
	repo := newMemoryAPIRepo(nil)
	svc := New(Config{Now: func() time.Time { return now }},
		WithRepository(repo),
		WithIDGenerator(&sequenceIDGenerator{next: 400}),
	)
	for _, item := range []OperationRecordInput{
		{IPAddress: "127.0.0.1", Method: "GET", Path: "/api/v1/ok", Status: 200, UserID: 1, Username: "admin"},
		{IPAddress: "127.0.0.1", Method: "GET", Path: "/api/v1/not-found", Status: 404, UserID: 1, Username: "admin"},
		{ErrorMessage: "boom", IPAddress: "127.0.0.1", Method: "POST", Path: "/api/v1/error", Status: 503, UserID: 1, Username: "admin"},
	} {
		if err := svc.RecordOperation(context.Background(), item); err != nil {
			t.Fatalf("RecordOperation() error = %v", err)
		}
	}

	page, err := svc.ListOperationRecords(context.Background(), OperationRecordFilter{Page: 1, PageSize: 10, StatusClass: "5xx"})
	if err != nil {
		t.Fatalf("ListOperationRecords(5xx) error = %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Status != 503 {
		t.Fatalf("expected only 5xx records, got %#v", page)
	}

	page, err = svc.ListOperationRecords(context.Background(), OperationRecordFilter{Page: 1, PageSize: 10, StatusClass: "error"})
	if err != nil {
		t.Fatalf("ListOperationRecords(error) error = %v", err)
	}
	if page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("expected all error records, got %#v", page)
	}

	page, err = svc.ListOperationRecords(context.Background(), OperationRecordFilter{Page: 1, PageSize: 10, StatusClass: "4xx"})
	if err != nil {
		t.Fatalf("ListOperationRecords(4xx) error = %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Status != 404 {
		t.Fatalf("expected only 4xx records, got %#v", page)
	}

	page, err = svc.ListOperationRecords(context.Background(), OperationRecordFilter{Page: 1, PageSize: 10, Status: 404, StatusClass: "5xx"})
	if err != nil {
		t.Fatalf("ListOperationRecords(exact status) error = %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Status != 404 {
		t.Fatalf("expected exact status to win over status class, got %#v", page)
	}

	if _, err = svc.ListOperationRecords(context.Background(), OperationRecordFilter{Page: 1, PageSize: 10, StatusClass: "2xx"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid status class error, got %v", err)
	}
}

func TestParameterManagementCreatesFiltersUpdatesFindsAndDeletes(t *testing.T) {
	now := time.Date(2026, 6, 12, 11, 0, 0, 0, time.UTC)
	repo := newMemoryAPIRepo(nil)
	svc := New(Config{Now: func() time.Time { return now }},
		WithRepository(repo),
		WithIDGenerator(&sequenceIDGenerator{next: 500}),
	)

	parameter, err := svc.CreateParameter(context.Background(), CreateParameterInput{
		Description: "Local site name",
		Key:         "site.name",
		Name:        "Site Name",
		Value:       "Aoi Admin",
	})
	if err != nil {
		t.Fatalf("CreateParameter() error = %v", err)
	}
	if parameter.ID != 500 || parameter.Key != "site.name" || parameter.Value != "Aoi Admin" {
		t.Fatalf("unexpected parameter: %#v", parameter)
	}
	if _, err := svc.CreateParameter(context.Background(), CreateParameterInput{Name: "Duplicate", Key: "site.name", Value: "x"}); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected duplicate error, got %v", err)
	}

	page, err := svc.ListParameters(context.Background(), ParameterFilter{
		Key:            "site",
		Name:           "Site",
		Page:           1,
		PageSize:       10,
		StartCreatedAt: ptrTime(now.Add(-time.Minute)),
		EndCreatedAt:   ptrTime(now.Add(time.Minute)),
	})
	if err != nil {
		t.Fatalf("ListParameters() error = %v", err)
	}
	if page.StorageStatus != "persisted" || page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("unexpected parameter page: %#v", page)
	}

	found, err := svc.FindParameterByKey(context.Background(), "site.name")
	if err != nil {
		t.Fatalf("FindParameterByKey() error = %v", err)
	}
	if found.ID != parameter.ID {
		t.Fatalf("expected parameter by key, got %#v", found)
	}

	newValue := "Aoi Console"
	newKey := "app.name"
	updated, err := svc.UpdateParameter(context.Background(), parameter.ID, UpdateParameterInput{Key: &newKey, Value: &newValue})
	if err != nil {
		t.Fatalf("UpdateParameter() error = %v", err)
	}
	if updated.Key != newKey || updated.Value != newValue {
		t.Fatalf("unexpected updated parameter: %#v", updated)
	}

	if err := svc.DeleteParameters(context.Background(), []int64{parameter.ID}); err != nil {
		t.Fatalf("DeleteParameters() error = %v", err)
	}
	page, err = svc.ListParameters(context.Background(), ParameterFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListParameters() after delete error = %v", err)
	}
	if page.Total != 0 || len(page.Items) != 0 {
		t.Fatalf("expected parameter to be soft deleted, got %#v", page)
	}
}

func TestVersionManagementExportsAndImportsReleasePackages(t *testing.T) {
	now := time.Date(2026, 6, 12, 15, 0, 0, 0, time.UTC)
	repo := newMemoryAPIRepo(nil)
	svc := New(Config{Now: func() time.Time { return now }},
		WithRepository(repo),
		WithIDGenerator(&sequenceIDGenerator{next: 600}),
	)
	svc.RegisterAPIs([]model.APIEntry{
		{Code: "get /api/v1/system/menus", Group: "system", Method: "GET", Path: "/api/v1/system/menus", Description: "menus", Order: 10},
		{Code: "get /api/v1/system/apis", Group: "system", Method: "GET", Path: "/api/v1/system/apis", Description: "apis", Permission: "permission:read", Order: 20},
	})
	dictionary, err := svc.CreateDictionary(context.Background(), CreateDictionaryInput{
		Code: "release.status",
		Name: "Release Status",
	})
	if err != nil {
		t.Fatalf("CreateDictionary() error = %v", err)
	}
	if _, err := svc.CreateDictionaryItem(context.Background(), dictionary.ID, CreateDictionaryItemInput{
		Label: "Ready",
		Value: "ready",
	}); err != nil {
		t.Fatalf("CreateDictionaryItem() error = %v", err)
	}

	detail, err := svc.ExportVersion(context.Background(), ExportVersionInput{
		APICodes:        []string{"get /api/v1/system/menus"},
		CreatedBy:       1,
		CreatorUsername: "admin",
		Description:     "Release package",
		DictionaryCodes: []string{"release.status"},
		MenuCodes:       []string{"system:menus"},
		VersionCode:     "v2026.06.12",
		VersionName:     "June Release",
	})
	if err != nil {
		t.Fatalf("ExportVersion() error = %v", err)
	}
	if detail.Item.ID != 602 || detail.Item.Source != model.VersionSourceExport || detail.Item.MenuCount != 1 || detail.Item.APICount != 1 || detail.Item.DictionaryCount != 1 {
		t.Fatalf("unexpected exported version detail: %#v", detail)
	}
	if detail.Package.Version.Code != "v2026.06.12" || len(detail.Package.APIs) != 1 || len(detail.Package.Dictionaries) != 1 || countMenus(detail.Package.Menus) != 1 {
		t.Fatalf("unexpected exported package: %#v", detail.Package)
	}

	page, err := svc.ListVersions(context.Background(), VersionFilter{Page: 1, PageSize: 10, VersionCode: "2026"})
	if err != nil {
		t.Fatalf("ListVersions() error = %v", err)
	}
	if page.StorageStatus != "persisted" || page.Total != 1 || page.Items[0].ID != detail.Item.ID {
		t.Fatalf("unexpected version page: %#v", page)
	}
	downloaded, err := svc.GetVersionPackage(context.Background(), detail.Item.ID)
	if err != nil {
		t.Fatalf("GetVersionPackage() error = %v", err)
	}
	if downloaded.Version.Name != "June Release" || len(downloaded.Dictionaries) != 1 {
		t.Fatalf("unexpected downloaded package: %#v", downloaded)
	}

	raw, err := json.Marshal(downloaded)
	if err != nil {
		t.Fatalf("marshal package: %v", err)
	}
	importRepo := newMemoryAPIRepo(nil)
	importSvc := New(Config{Now: func() time.Time { return now.Add(time.Hour) }},
		WithRepository(importRepo),
		WithIDGenerator(&sequenceIDGenerator{next: 700}),
	)
	imported, err := importSvc.ImportVersion(context.Background(), ImportVersionInput{
		CreatedBy:       2,
		CreatorUsername: "operator",
		VersionData:     string(raw),
	})
	if err != nil {
		t.Fatalf("ImportVersion() error = %v", err)
	}
	if imported.Item.ID != 702 || imported.Item.Source != model.VersionSourceImport || imported.DictionariesCreated != 1 || imported.DictionaryItemsCreated != 1 {
		t.Fatalf("unexpected import result: %#v", imported)
	}
	if imported.MenusSkipped != 1 || imported.APIsSkipped != 1 {
		t.Fatalf("expected menu/API entries to be recorded as skipped, got %#v", imported)
	}
	catalog, err := importSvc.ListDictionaries(context.Background())
	if err != nil {
		t.Fatalf("ListDictionaries() after import error = %v", err)
	}
	if catalog.Total != 1 || !dictionaryItemExists(catalog, "release.status", "ready") {
		t.Fatalf("expected imported dictionary and item, got %#v", catalog)
	}

	if err := svc.DeleteVersions(context.Background(), []int64{detail.Item.ID}); err != nil {
		t.Fatalf("DeleteVersions() error = %v", err)
	}
	page, err = svc.ListVersions(context.Background(), VersionFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListVersions() after delete error = %v", err)
	}
	if page.Total != 0 {
		t.Fatalf("expected deleted version to be hidden, got %#v", page)
	}
}

func TestMediaLibraryManagesCategoriesUploadsURLsAndDownloads(t *testing.T) {
	now := time.Date(2026, 6, 12, 16, 0, 0, 0, time.UTC)
	repo := newMemoryAPIRepo(nil)
	store := newMemoryMediaStore()
	svc := New(Config{Now: func() time.Time { return now }},
		WithRepository(repo),
		WithStorage(store),
		WithIDGenerator(&sequenceIDGenerator{next: 800}),
	)

	category, err := svc.UpsertMediaCategory(context.Background(), UpsertMediaCategoryInput{
		Name: "Images",
		Sort: 10,
	})
	if err != nil {
		t.Fatalf("UpsertMediaCategory() error = %v", err)
	}
	if category.ID != 800 || category.ParentID != 0 {
		t.Fatalf("unexpected category: %#v", category)
	}

	catalog, err := svc.ListMediaCategories(context.Background())
	if err != nil {
		t.Fatalf("ListMediaCategories() error = %v", err)
	}
	if catalog.StorageStatus != "persisted" || catalog.Total != 1 || catalog.Items[0].Name != "Images" {
		t.Fatalf("unexpected category catalog: %#v", catalog)
	}

	asset, err := svc.UploadMediaAsset(context.Background(), UploadMediaAssetInput{
		CategoryID:         category.ID,
		Filename:           `..\avatar.png`,
		Reader:             strings.NewReader("hello-media"),
		Size:               int64(len("hello-media")),
		UploadedBy:         1,
		UploadedByUsername: "admin",
	})
	if err != nil {
		t.Fatalf("UploadMediaAsset() error = %v", err)
	}
	if asset.ID != 801 || asset.OriginalName != "avatar.png" || !strings.HasPrefix(asset.StorageKey, "media/2026/06/") {
		t.Fatalf("unexpected uploaded asset: %#v", asset)
	}
	if string(store.files[asset.StorageKey]) != "hello-media" {
		t.Fatalf("expected object storage write, got %#v", store.files)
	}

	download, err := svc.DownloadMediaAsset(context.Background(), asset.ID)
	if err != nil {
		t.Fatalf("DownloadMediaAsset() error = %v", err)
	}
	if string(download.Data) != "hello-media" || download.Filename != "avatar.png" {
		t.Fatalf("unexpected download: %#v", download)
	}

	renamed, err := svc.UpdateMediaAsset(context.Background(), asset.ID, UpdateMediaAssetInput{DisplayName: "Login Logo"})
	if err != nil {
		t.Fatalf("UpdateMediaAsset() error = %v", err)
	}
	if renamed.DisplayName != "Login Logo" {
		t.Fatalf("unexpected renamed asset: %#v", renamed)
	}

	imported, err := svc.ImportMediaURLs(context.Background(), ImportMediaURLsInput{
		CategoryID:         category.ID,
		Items:              []MediaURLImportItem{{Name: "remote.png", URL: "https://example.com/assets/remote.png"}},
		UploadedBy:         1,
		UploadedByUsername: "admin",
	})
	if err != nil {
		t.Fatalf("ImportMediaURLs() error = %v", err)
	}
	if imported.StorageStatus != "persisted" || imported.Imported != 1 || !imported.Items[0].External {
		t.Fatalf("unexpected import result: %#v", imported)
	}

	page, err := svc.ListMediaAssets(context.Background(), MediaAssetFilter{CategoryID: category.ID, Keyword: "Logo", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListMediaAssets() error = %v", err)
	}
	if page.StorageStatus != "persisted" || page.ObjectStorage != "enabled" || page.Total != 1 || page.Items[0].DisplayName != "Login Logo" {
		t.Fatalf("unexpected media page: %#v", page)
	}

	if err := svc.DeleteMediaAsset(context.Background(), asset.ID); err != nil {
		t.Fatalf("DeleteMediaAsset(uploaded) error = %v", err)
	}
	if _, ok := store.files[asset.StorageKey]; ok {
		t.Fatalf("expected uploaded object to be removed")
	}
	if err := svc.DeleteMediaAsset(context.Background(), imported.Items[0].ID); err != nil {
		t.Fatalf("DeleteMediaAsset(imported) error = %v", err)
	}
	if err := svc.DeleteMediaCategory(context.Background(), category.ID); err != nil {
		t.Fatalf("DeleteMediaCategory() error = %v", err)
	}
	catalog, err = svc.ListMediaCategories(context.Background())
	if err != nil {
		t.Fatalf("ListMediaCategories() after delete error = %v", err)
	}
	if catalog.Total != 0 {
		t.Fatalf("expected category to be deleted, got %#v", catalog)
	}
}

func TestMediaResumableUploadCreatesAssetFromChunks(t *testing.T) {
	now := time.Date(2026, 6, 12, 16, 0, 0, 0, time.UTC)
	repo := newMemoryAPIRepo(nil)
	store := newMemoryMediaStore()
	svc := New(Config{
		MediaMaxBytes: 512 * 1024,
		MediaPrefix:   "media",
		Now:           func() time.Time { return now },
	}, WithRepository(repo), WithStorage(store), WithIDGenerator(&sequenceIDGenerator{next: 900}))

	data := bytes.Repeat([]byte("aoi-admin-resumable-upload\n"), 6000)
	chunkSize := minMediaChunkSize
	chunkTotal := expectedMediaChunkTotal(int64(len(data)), chunkSize)
	check, err := svc.CheckMediaResumableUpload(context.Background(), CheckMediaResumableUploadInput{
		ChunkSize:          chunkSize,
		ChunkTotal:         chunkTotal,
		FileHash:           sha256Hex(data),
		Filename:           "report.txt",
		SizeBytes:          int64(len(data)),
		UploadedBy:         7,
		UploadedByUsername: "admin",
	})
	if err != nil {
		t.Fatalf("CheckMediaResumableUpload() error = %v", err)
	}
	if check.Session.Status != model.MediaUploadStatusActive || len(check.MissingChunks) != chunkTotal {
		t.Fatalf("unexpected check result: %#v", check)
	}

	for index := 0; index < chunkTotal; index++ {
		start := int64(index) * chunkSize
		end := start + chunkSize
		if end > int64(len(data)) {
			end = int64(len(data))
		}
		chunkData := data[int(start):int(end)]
		chunk, err := svc.UploadMediaChunk(context.Background(), UploadMediaChunkInput{
			ChunkHash:  sha256Hex(chunkData),
			ChunkIndex: index,
			ChunkTotal: chunkTotal,
			FileHash:   sha256Hex(data),
			Reader:     bytes.NewReader(chunkData),
			SessionID:  check.Session.ID,
			Size:       int64(len(chunkData)),
			UploadedBy: 7,
		})
		if err != nil {
			t.Fatalf("UploadMediaChunk(%d) error = %v", index, err)
		}
		if chunk.Progress <= 0 {
			t.Fatalf("expected progress after chunk %d, got %#v", index, chunk)
		}
	}

	complete, err := svc.CompleteMediaResumableUpload(context.Background(), CompleteMediaResumableUploadInput{
		FileHash:   sha256Hex(data),
		SessionID:  check.Session.ID,
		UploadedBy: 7,
	})
	if err != nil {
		t.Fatalf("CompleteMediaResumableUpload() error = %v", err)
	}
	if complete.Asset.Source != model.MediaSourceResumable || complete.Asset.SizeBytes != int64(len(data)) {
		t.Fatalf("unexpected completed asset: %#v", complete.Asset)
	}
	stored, err := store.ReadFile(complete.Asset.StorageKey)
	if err != nil {
		t.Fatalf("stored asset missing: %v", err)
	}
	if !bytes.Equal(stored, data) {
		t.Fatalf("stored asset content mismatch")
	}
	if len(repo.mediaChunks) != 0 {
		t.Fatalf("expected chunk records to be cleaned, got %#v", repo.mediaChunks)
	}

	again, err := svc.CheckMediaResumableUpload(context.Background(), CheckMediaResumableUploadInput{
		ChunkSize:  chunkSize,
		ChunkTotal: chunkTotal,
		FileHash:   sha256Hex(data),
		Filename:   "report.txt",
		SizeBytes:  int64(len(data)),
		UploadedBy: 7,
	})
	if err != nil {
		t.Fatalf("CheckMediaResumableUpload() completed error = %v", err)
	}
	if again.Session.Status != model.MediaUploadStatusCompleted || again.Asset == nil || again.Progress != 100 {
		t.Fatalf("expected instant completed session, got %#v", again)
	}
}

func TestSeedDefaultsCreatesSystemDataIdempotently(t *testing.T) {
	now := time.Date(2026, 6, 12, 14, 0, 0, 0, time.UTC)
	repo := newMemoryAPIRepo(nil)
	svc := New(Config{Now: func() time.Time { return now }},
		WithRepository(repo),
		WithIDGenerator(&sequenceIDGenerator{next: 700}),
	)

	result, err := svc.SeedDefaults(context.Background())
	if err != nil {
		t.Fatalf("SeedDefaults() error = %v", err)
	}
	if result.StorageStatus != "persisted" || result.DictionariesCreated != 3 || result.DictionaryItemsCreated != 9 || result.ParametersCreated != 3 {
		t.Fatalf("unexpected seed result: %#v", result)
	}

	catalog, err := svc.ListDictionaries(context.Background())
	if err != nil {
		t.Fatalf("ListDictionaries() error = %v", err)
	}
	if catalog.Total != 3 || !dictionaryItemExists(catalog, "system.status", model.DictionaryStatusActive) || !dictionaryItemExists(catalog, "http.method", "DELETE") {
		t.Fatalf("expected seeded dictionaries and items, got %#v", catalog)
	}
	title, err := svc.FindParameterByKey(context.Background(), "admin.title")
	if err != nil {
		t.Fatalf("FindParameterByKey(admin.title) error = %v", err)
	}
	customTitle := "Custom Admin"
	if _, err := svc.UpdateParameter(context.Background(), title.ID, UpdateParameterInput{Value: &customTitle}); err != nil {
		t.Fatalf("UpdateParameter(admin.title) error = %v", err)
	}

	again, err := svc.SeedDefaults(context.Background())
	if err != nil {
		t.Fatalf("SeedDefaults() second error = %v", err)
	}
	if again.DictionariesCreated != 0 || again.DictionaryItemsCreated != 0 || again.ParametersCreated != 0 {
		t.Fatalf("expected second seed to be idempotent, got %#v", again)
	}
	title, err = svc.FindParameterByKey(context.Background(), "admin.title")
	if err != nil {
		t.Fatalf("FindParameterByKey(admin.title) second error = %v", err)
	}
	if title.Value != customTitle {
		t.Fatalf("expected seed to preserve customized parameter, got %#v", title)
	}
}

func TestSeedDefaultsWithoutRepositoryReportsUnavailable(t *testing.T) {
	svc := New(Config{})

	result, err := svc.SeedDefaults(context.Background())
	if err != nil {
		t.Fatalf("SeedDefaults() error = %v", err)
	}
	if result.StorageStatus != "unavailable" || result.DictionariesCreated != 0 || result.ParametersCreated != 0 {
		t.Fatalf("unexpected seed result without repository: %#v", result)
	}
}

type memoryAPIRepo struct {
	dictionaries     map[int64]model.Dictionary
	items            map[int64]model.DictionaryItem
	mediaAssets      map[int64]model.MediaAsset
	mediaCategories  map[int64]model.MediaCategory
	mediaChunks      map[int64]model.MediaUploadChunk
	mediaSessions    map[int64]model.MediaUploadSession
	operationRecords map[int64]model.OperationRecord
	parameters       map[int64]model.Parameter
	records          map[string]model.APIRecord
	versions         map[int64]model.Version
}

func newMemoryAPIRepo(records []model.APIRecord) *memoryAPIRepo {
	repo := &memoryAPIRepo{
		dictionaries:     make(map[int64]model.Dictionary),
		items:            make(map[int64]model.DictionaryItem),
		mediaAssets:      make(map[int64]model.MediaAsset),
		mediaCategories:  make(map[int64]model.MediaCategory),
		mediaChunks:      make(map[int64]model.MediaUploadChunk),
		mediaSessions:    make(map[int64]model.MediaUploadSession),
		operationRecords: make(map[int64]model.OperationRecord),
		parameters:       make(map[int64]model.Parameter),
		records:          make(map[string]model.APIRecord, len(records)),
		versions:         make(map[int64]model.Version),
	}
	for _, record := range records {
		repo.records[memoryAPIKey(record.Method, record.Path)] = record
	}
	return repo
}

func (r *memoryAPIRepo) CreateAPI(_ context.Context, api *model.APIRecord) error {
	r.records[memoryAPIKey(api.Method, api.Path)] = *api
	return nil
}

func (r *memoryAPIRepo) CreateDictionary(_ context.Context, dictionary *model.Dictionary) error {
	r.dictionaries[dictionary.ID] = *dictionary
	return nil
}

func (r *memoryAPIRepo) CreateDictionaryItem(_ context.Context, item *model.DictionaryItem) error {
	r.items[item.ID] = *item
	return nil
}

func (r *memoryAPIRepo) CreateMediaAsset(_ context.Context, asset *model.MediaAsset) error {
	r.mediaAssets[asset.ID] = *asset
	return nil
}

func (r *memoryAPIRepo) CreateMediaCategory(_ context.Context, category *model.MediaCategory) error {
	r.mediaCategories[category.ID] = *category
	return nil
}

func (r *memoryAPIRepo) CreateMediaUploadChunk(_ context.Context, chunk *model.MediaUploadChunk) error {
	r.mediaChunks[chunk.ID] = *chunk
	return nil
}

func (r *memoryAPIRepo) CreateMediaUploadSession(_ context.Context, session *model.MediaUploadSession) error {
	r.mediaSessions[session.ID] = *session
	return nil
}

func (r *memoryAPIRepo) CreateOperationRecord(_ context.Context, record *model.OperationRecord) error {
	r.operationRecords[record.ID] = *record
	return nil
}

func (r *memoryAPIRepo) CreateParameter(_ context.Context, parameter *model.Parameter) error {
	r.parameters[parameter.ID] = *parameter
	return nil
}

func (r *memoryAPIRepo) CreateVersion(_ context.Context, version *model.Version) error {
	r.versions[version.ID] = *version
	return nil
}

func (r *memoryAPIRepo) DeleteDictionary(_ context.Context, id int64, deletedAt time.Time) error {
	dictionary, ok := r.dictionaries[id]
	if !ok || dictionary.DeletedAt != nil {
		return ErrNotFound
	}
	dictionary.DeletedAt = &deletedAt
	dictionary.UpdatedAt = deletedAt
	r.dictionaries[id] = dictionary
	for itemID, item := range r.items {
		if item.DictionaryID != id || item.DeletedAt != nil {
			continue
		}
		item.DeletedAt = &deletedAt
		item.UpdatedAt = deletedAt
		r.items[itemID] = item
	}
	return nil
}

func (r *memoryAPIRepo) DeleteDictionaryItem(_ context.Context, id int64, deletedAt time.Time) error {
	item, ok := r.items[id]
	if !ok || item.DeletedAt != nil {
		return ErrNotFound
	}
	item.DeletedAt = &deletedAt
	item.UpdatedAt = deletedAt
	r.items[id] = item
	return nil
}

func (r *memoryAPIRepo) DeleteMediaAsset(_ context.Context, id int64, deletedAt time.Time) error {
	asset, ok := r.mediaAssets[id]
	if !ok || asset.DeletedAt != nil {
		return ErrNotFound
	}
	asset.DeletedAt = &deletedAt
	asset.UpdatedAt = deletedAt
	r.mediaAssets[id] = asset
	return nil
}

func (r *memoryAPIRepo) DeleteMediaCategory(_ context.Context, id int64, deletedAt time.Time) error {
	category, ok := r.mediaCategories[id]
	if !ok || category.DeletedAt != nil {
		return ErrNotFound
	}
	category.DeletedAt = &deletedAt
	category.UpdatedAt = deletedAt
	r.mediaCategories[id] = category
	return nil
}

func (r *memoryAPIRepo) DeleteMediaUploadChunks(_ context.Context, sessionID int64) error {
	for id, chunk := range r.mediaChunks {
		if chunk.SessionID == sessionID {
			delete(r.mediaChunks, id)
		}
	}
	return nil
}

func (r *memoryAPIRepo) DeleteOperationRecords(_ context.Context, ids []int64) error {
	for _, id := range ids {
		delete(r.operationRecords, id)
	}
	return nil
}

func (r *memoryAPIRepo) DeleteParameter(_ context.Context, id int64, deletedAt time.Time) error {
	parameter, ok := r.parameters[id]
	if !ok || parameter.DeletedAt != nil {
		return ErrNotFound
	}
	parameter.DeletedAt = &deletedAt
	parameter.UpdatedAt = deletedAt
	r.parameters[id] = parameter
	return nil
}

func (r *memoryAPIRepo) DeleteParameters(_ context.Context, ids []int64, deletedAt time.Time) error {
	for _, id := range ids {
		parameter, ok := r.parameters[id]
		if !ok || parameter.DeletedAt != nil {
			continue
		}
		parameter.DeletedAt = &deletedAt
		parameter.UpdatedAt = deletedAt
		r.parameters[id] = parameter
	}
	return nil
}

func (r *memoryAPIRepo) DeleteVersion(_ context.Context, id int64, deletedAt time.Time) error {
	version, ok := r.versions[id]
	if !ok || version.DeletedAt != nil {
		return ErrNotFound
	}
	version.DeletedAt = &deletedAt
	version.UpdatedAt = deletedAt
	r.versions[id] = version
	return nil
}

func (r *memoryAPIRepo) DeleteVersions(_ context.Context, ids []int64, deletedAt time.Time) error {
	for _, id := range ids {
		version, ok := r.versions[id]
		if !ok || version.DeletedAt != nil {
			continue
		}
		version.DeletedAt = &deletedAt
		version.UpdatedAt = deletedAt
		r.versions[id] = version
	}
	return nil
}

func (r *memoryAPIRepo) FindAPI(_ context.Context, method string, path string) (*model.APIRecord, error) {
	record, ok := r.record(method, path)
	if !ok {
		return nil, errors.New("not found")
	}
	return &record, nil
}

func (r *memoryAPIRepo) FindDictionaryByCode(_ context.Context, code string) (*model.Dictionary, error) {
	for _, dictionary := range r.dictionaries {
		if dictionary.Code == code && dictionary.DeletedAt == nil {
			return &dictionary, nil
		}
	}
	return nil, ErrNotFound
}

func (r *memoryAPIRepo) FindDictionaryByID(_ context.Context, id int64) (*model.Dictionary, error) {
	dictionary, ok := r.dictionaries[id]
	if !ok || dictionary.DeletedAt != nil {
		return nil, ErrNotFound
	}
	return &dictionary, nil
}

func (r *memoryAPIRepo) FindDictionaryItemByID(_ context.Context, id int64) (*model.DictionaryItem, error) {
	item, ok := r.items[id]
	if !ok || item.DeletedAt != nil {
		return nil, ErrNotFound
	}
	return &item, nil
}

func (r *memoryAPIRepo) FindMediaAssetByID(_ context.Context, id int64) (*model.MediaAsset, error) {
	asset, ok := r.mediaAssets[id]
	if !ok || asset.DeletedAt != nil {
		return nil, ErrNotFound
	}
	return &asset, nil
}

func (r *memoryAPIRepo) FindMediaCategoryByID(_ context.Context, id int64) (*model.MediaCategory, error) {
	category, ok := r.mediaCategories[id]
	if !ok || category.DeletedAt != nil {
		return nil, ErrNotFound
	}
	return &category, nil
}

func (r *memoryAPIRepo) FindMediaUploadChunk(_ context.Context, sessionID int64, chunkIndex int) (*model.MediaUploadChunk, error) {
	for _, chunk := range r.mediaChunks {
		if chunk.SessionID == sessionID && chunk.ChunkIndex == chunkIndex {
			return &chunk, nil
		}
	}
	return nil, ErrNotFound
}

func (r *memoryAPIRepo) FindMediaUploadSessionByHash(_ context.Context, fileHash string, fileName string, categoryID int64, uploadedBy int64) (*model.MediaUploadSession, error) {
	var found *model.MediaUploadSession
	for _, session := range r.mediaSessions {
		if session.DeletedAt != nil || session.FileHash != fileHash || session.FileName != fileName || session.CategoryID != categoryID || session.UploadedBy != uploadedBy {
			continue
		}
		session := session
		if found == nil || session.CreatedAt.After(found.CreatedAt) || (session.CreatedAt.Equal(found.CreatedAt) && session.ID > found.ID) {
			found = &session
		}
	}
	if found == nil {
		return nil, ErrNotFound
	}
	return found, nil
}

func (r *memoryAPIRepo) FindMediaUploadSessionByID(_ context.Context, id int64) (*model.MediaUploadSession, error) {
	session, ok := r.mediaSessions[id]
	if !ok || session.DeletedAt != nil {
		return nil, ErrNotFound
	}
	return &session, nil
}

func (r *memoryAPIRepo) FindParameterByID(_ context.Context, id int64) (*model.Parameter, error) {
	parameter, ok := r.parameters[id]
	if !ok || parameter.DeletedAt != nil {
		return nil, ErrNotFound
	}
	return &parameter, nil
}

func (r *memoryAPIRepo) FindParameterByKey(_ context.Context, key string) (*model.Parameter, error) {
	for _, parameter := range r.parameters {
		if parameter.Key == key && parameter.DeletedAt == nil {
			return &parameter, nil
		}
	}
	return nil, ErrNotFound
}

func (r *memoryAPIRepo) FindVersionByID(_ context.Context, id int64) (*model.Version, error) {
	version, ok := r.versions[id]
	if !ok || version.DeletedAt != nil {
		return nil, ErrNotFound
	}
	return &version, nil
}

func (r *memoryAPIRepo) ListAPIs(context.Context) ([]model.APIRecord, error) {
	records := make([]model.APIRecord, 0, len(r.records))
	for _, record := range r.records {
		records = append(records, record)
	}
	return records, nil
}

func (r *memoryAPIRepo) ListDictionaries(context.Context) ([]model.Dictionary, error) {
	dictionaries := make([]model.Dictionary, 0, len(r.dictionaries))
	for _, dictionary := range r.dictionaries {
		if dictionary.DeletedAt != nil {
			continue
		}
		dictionaries = append(dictionaries, dictionary)
	}
	sort.SliceStable(dictionaries, func(i, j int) bool {
		return dictionaries[i].Code < dictionaries[j].Code
	})
	return dictionaries, nil
}

func (r *memoryAPIRepo) ListDictionaryItems(_ context.Context, dictionaryID int64) ([]model.DictionaryItem, error) {
	items := make([]model.DictionaryItem, 0, len(r.items))
	for _, item := range r.items {
		if item.DictionaryID != dictionaryID || item.DeletedAt != nil {
			continue
		}
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Sort == items[j].Sort {
			return items[i].Value < items[j].Value
		}
		return items[i].Sort < items[j].Sort
	})
	return items, nil
}

func (r *memoryAPIRepo) ListMediaCategories(context.Context) ([]model.MediaCategory, error) {
	categories := make([]model.MediaCategory, 0, len(r.mediaCategories))
	for _, category := range r.mediaCategories {
		if category.DeletedAt != nil {
			continue
		}
		categories = append(categories, category)
	}
	sort.SliceStable(categories, func(i, j int) bool {
		if categories[i].Sort == categories[j].Sort {
			if categories[i].Name == categories[j].Name {
				return categories[i].ID < categories[j].ID
			}
			return categories[i].Name < categories[j].Name
		}
		return categories[i].Sort < categories[j].Sort
	})
	return categories, nil
}

func (r *memoryAPIRepo) ListMediaAssets(_ context.Context, filter model.MediaAssetFilter) ([]model.MediaAsset, int64, error) {
	assets := make([]model.MediaAsset, 0, len(r.mediaAssets))
	keyword := strings.TrimSpace(filter.Keyword)
	for _, asset := range r.mediaAssets {
		if asset.DeletedAt != nil {
			continue
		}
		if filter.CategoryID > 0 && asset.CategoryID != filter.CategoryID {
			continue
		}
		if keyword != "" && !strings.Contains(asset.DisplayName, keyword) && !strings.Contains(asset.OriginalName, keyword) && !strings.Contains(asset.URL, keyword) {
			continue
		}
		assets = append(assets, asset)
	}
	sort.SliceStable(assets, func(i, j int) bool {
		if assets[i].CreatedAt.Equal(assets[j].CreatedAt) {
			return assets[i].ID > assets[j].ID
		}
		return assets[i].CreatedAt.After(assets[j].CreatedAt)
	})
	total := int64(len(assets))
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	if start >= len(assets) {
		return []model.MediaAsset{}, total, nil
	}
	end := start + pageSize
	if end > len(assets) {
		end = len(assets)
	}
	return assets[start:end], total, nil
}

func (r *memoryAPIRepo) ListMediaUploadChunks(_ context.Context, sessionID int64) ([]model.MediaUploadChunk, error) {
	chunks := make([]model.MediaUploadChunk, 0, len(r.mediaChunks))
	for _, chunk := range r.mediaChunks {
		if chunk.SessionID == sessionID {
			chunks = append(chunks, chunk)
		}
	}
	sort.SliceStable(chunks, func(i, j int) bool {
		return chunks[i].ChunkIndex < chunks[j].ChunkIndex
	})
	return chunks, nil
}

func (r *memoryAPIRepo) ListOperationRecords(_ context.Context, filter model.OperationRecordFilter) ([]model.OperationRecord, int64, error) {
	records := make([]model.OperationRecord, 0, len(r.operationRecords))
	method := strings.ToUpper(strings.TrimSpace(filter.Method))
	path := strings.TrimSpace(filter.Path)
	for _, record := range r.operationRecords {
		if method != "" && record.Method != method {
			continue
		}
		if path != "" && !strings.Contains(record.Path, path) {
			continue
		}
		if filter.Status > 0 {
			if record.Status != filter.Status {
				continue
			}
		} else {
			switch strings.ToLower(strings.TrimSpace(filter.StatusClass)) {
			case "4xx":
				if record.Status < 400 || record.Status >= 500 {
					continue
				}
			case "5xx":
				if record.Status < 500 || record.Status >= 600 {
					continue
				}
			case "error":
				if record.Status < 400 {
					continue
				}
			}
		}
		records = append(records, record)
	}
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].CreatedAt.Equal(records[j].CreatedAt) {
			return records[i].ID > records[j].ID
		}
		return records[i].CreatedAt.After(records[j].CreatedAt)
	})
	total := int64(len(records))
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	if start >= len(records) {
		return []model.OperationRecord{}, total, nil
	}
	end := start + pageSize
	if end > len(records) {
		end = len(records)
	}
	return records[start:end], total, nil
}

func (r *memoryAPIRepo) ListParameters(_ context.Context, filter model.ParameterFilter) ([]model.Parameter, int64, error) {
	parameters := make([]model.Parameter, 0, len(r.parameters))
	name := strings.TrimSpace(filter.Name)
	key := strings.TrimSpace(filter.Key)
	for _, parameter := range r.parameters {
		if parameter.DeletedAt != nil {
			continue
		}
		if name != "" && !strings.Contains(parameter.Name, name) {
			continue
		}
		if key != "" && !strings.Contains(parameter.Key, key) {
			continue
		}
		if filter.StartCreatedAt != nil && parameter.CreatedAt.Before(*filter.StartCreatedAt) {
			continue
		}
		if filter.EndCreatedAt != nil && !parameter.CreatedAt.Before(*filter.EndCreatedAt) {
			continue
		}
		parameters = append(parameters, parameter)
	}
	sort.SliceStable(parameters, func(i, j int) bool {
		if parameters[i].CreatedAt.Equal(parameters[j].CreatedAt) {
			return parameters[i].ID > parameters[j].ID
		}
		return parameters[i].CreatedAt.After(parameters[j].CreatedAt)
	})
	total := int64(len(parameters))
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	if start >= len(parameters) {
		return []model.Parameter{}, total, nil
	}
	end := start + pageSize
	if end > len(parameters) {
		end = len(parameters)
	}
	return parameters[start:end], total, nil
}

func (r *memoryAPIRepo) ListVersions(_ context.Context, filter model.VersionFilter) ([]model.Version, int64, error) {
	versions := make([]model.Version, 0, len(r.versions))
	name := strings.TrimSpace(filter.VersionName)
	code := strings.TrimSpace(filter.VersionCode)
	for _, version := range r.versions {
		if version.DeletedAt != nil {
			continue
		}
		if name != "" && !strings.Contains(version.VersionName, name) {
			continue
		}
		if code != "" && !strings.Contains(version.VersionCode, code) {
			continue
		}
		if filter.StartCreatedAt != nil && version.CreatedAt.Before(*filter.StartCreatedAt) {
			continue
		}
		if filter.EndCreatedAt != nil && !version.CreatedAt.Before(*filter.EndCreatedAt) {
			continue
		}
		versions = append(versions, version)
	}
	sort.SliceStable(versions, func(i, j int) bool {
		if versions[i].CreatedAt.Equal(versions[j].CreatedAt) {
			return versions[i].ID > versions[j].ID
		}
		return versions[i].CreatedAt.After(versions[j].CreatedAt)
	})
	total := int64(len(versions))
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	if start >= len(versions) {
		return []model.Version{}, total, nil
	}
	end := start + pageSize
	if end > len(versions) {
		end = len(versions)
	}
	return versions[start:end], total, nil
}

func (r *memoryAPIRepo) SaveAPI(_ context.Context, api *model.APIRecord) error {
	r.records[memoryAPIKey(api.Method, api.Path)] = *api
	return nil
}

func (r *memoryAPIRepo) SaveDictionary(_ context.Context, dictionary *model.Dictionary) error {
	if _, ok := r.dictionaries[dictionary.ID]; !ok {
		return ErrNotFound
	}
	r.dictionaries[dictionary.ID] = *dictionary
	return nil
}

func (r *memoryAPIRepo) SaveDictionaryItem(_ context.Context, item *model.DictionaryItem) error {
	if _, ok := r.items[item.ID]; !ok {
		return ErrNotFound
	}
	r.items[item.ID] = *item
	return nil
}

func (r *memoryAPIRepo) SaveMediaAsset(_ context.Context, asset *model.MediaAsset) error {
	if _, ok := r.mediaAssets[asset.ID]; !ok {
		return ErrNotFound
	}
	r.mediaAssets[asset.ID] = *asset
	return nil
}

func (r *memoryAPIRepo) SaveMediaCategory(_ context.Context, category *model.MediaCategory) error {
	if _, ok := r.mediaCategories[category.ID]; !ok {
		return ErrNotFound
	}
	r.mediaCategories[category.ID] = *category
	return nil
}

func (r *memoryAPIRepo) SaveMediaUploadChunk(_ context.Context, chunk *model.MediaUploadChunk) error {
	if _, ok := r.mediaChunks[chunk.ID]; !ok {
		return ErrNotFound
	}
	r.mediaChunks[chunk.ID] = *chunk
	return nil
}

func (r *memoryAPIRepo) SaveMediaUploadSession(_ context.Context, session *model.MediaUploadSession) error {
	if _, ok := r.mediaSessions[session.ID]; !ok {
		return ErrNotFound
	}
	r.mediaSessions[session.ID] = *session
	return nil
}

func (r *memoryAPIRepo) SaveParameter(_ context.Context, parameter *model.Parameter) error {
	if _, ok := r.parameters[parameter.ID]; !ok {
		return ErrNotFound
	}
	r.parameters[parameter.ID] = *parameter
	return nil
}

func (r *memoryAPIRepo) record(method string, path string) (model.APIRecord, bool) {
	record, ok := r.records[memoryAPIKey(method, path)]
	return record, ok
}

type sequenceIDGenerator struct {
	next int64
}

func (g *sequenceIDGenerator) NextID() int64 {
	id := g.next
	g.next++
	return id
}

func (g *sequenceIDGenerator) NextIDString() string {
	return strconv.FormatInt(g.NextID(), 10)
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func apiEntrySynced(groups []model.APIGroup, method string, path string) bool {
	for _, group := range groups {
		for _, entry := range group.Items {
			if entry.Method == method && entry.Path == path && entry.Synced && entry.SyncedAt != nil {
				return true
			}
		}
	}
	return false
}

func memoryAPIKey(method string, path string) string {
	return method + " " + path
}

type memoryPermissionStore struct {
	records map[string]model.PermissionEntry
}

func newMemoryPermissionStore(records []model.PermissionEntry) *memoryPermissionStore {
	store := &memoryPermissionStore{records: make(map[string]model.PermissionEntry, len(records))}
	for _, record := range records {
		store.records[permissionKey(record.ProductCode, record.Scope, record.Code)] = record
	}
	return store
}

func (s *memoryPermissionStore) CreatePermission(_ context.Context, permission model.PermissionEntry) error {
	s.records[permissionKey(permission.ProductCode, permission.Scope, permission.Code)] = permission
	return nil
}

func (s *memoryPermissionStore) ListPermissions(context.Context) ([]model.PermissionEntry, error) {
	records := make([]model.PermissionEntry, 0, len(s.records))
	for _, record := range s.records {
		records = append(records, record)
	}
	return records, nil
}

func (s *memoryPermissionStore) has(code string) bool {
	for _, record := range s.records {
		if record.Code == code {
			return true
		}
	}
	return false
}

func apiEntryPermissionRegistered(groups []model.APIGroup, method string, path string) bool {
	for _, group := range groups {
		for _, entry := range group.Items {
			if entry.Method == method && entry.Path == path && entry.PermissionRegistered {
				return true
			}
		}
	}
	return false
}

func menuItemExists(groups []model.MenuGroup, groupCode string, itemCode string, path string, permission string) bool {
	for _, group := range groups {
		if group.Code != groupCode {
			continue
		}
		for _, item := range group.Items {
			if item.Code == itemCode && item.Path == path && item.Permission == permission {
				return true
			}
		}
	}
	return false
}

func dictionaryItemExists(catalog model.DictionaryCatalog, code string, value string) bool {
	for _, dictionary := range catalog.Items {
		if dictionary.Code != code {
			continue
		}
		for _, item := range dictionary.Items {
			if item.Value == value {
				return true
			}
		}
	}
	return false
}

func (r *memoryAPIRepo) CreateTrafficHijackEvent(_ context.Context, _ *model.TrafficHijackEvent) error {
	return nil
}

func (r *memoryAPIRepo) CreateTrafficProbeResult(_ context.Context, _ *model.TrafficProbeResult) error {
	return nil
}

func (r *memoryAPIRepo) CreateTrafficProbeTarget(_ context.Context, _ *model.TrafficProbeTarget) error {
	return nil
}

func (r *memoryAPIRepo) DeleteOldTrafficProbeResults(_ context.Context, _ int64, _ int) error {
	return nil
}

func (r *memoryAPIRepo) DeleteTrafficProbeTarget(_ context.Context, _ int64, _ time.Time) error {
	return nil
}

func (r *memoryAPIRepo) FindTrafficHijackEvent(_ context.Context, _ int64) (*model.TrafficHijackEvent, error) {
	return nil, ErrNotFound
}

func (r *memoryAPIRepo) FindTrafficProbeTargetByID(_ context.Context, _ int64) (*model.TrafficProbeTarget, error) {
	return nil, ErrNotFound
}

func (r *memoryAPIRepo) FindOpenTrafficHijackEvent(_ context.Context, _ int64, _ string, _ string) (*model.TrafficHijackEvent, error) {
	return nil, ErrNotFound
}

func (r *memoryAPIRepo) ListTrafficHijackEvents(_ context.Context, _ model.TrafficHijackEventFilter) ([]model.TrafficHijackEvent, int64, error) {
	return nil, 0, nil
}

func (r *memoryAPIRepo) ListTrafficProbeResults(_ context.Context, _ model.TrafficProbeResultFilter) ([]model.TrafficProbeResult, error) {
	return nil, nil
}

func (r *memoryAPIRepo) ListTrafficProbeTargets(context.Context) ([]model.TrafficProbeTarget, error) {
	return nil, nil
}

func (r *memoryAPIRepo) SaveTrafficHijackEvent(_ context.Context, _ *model.TrafficHijackEvent) error {
	return nil
}

func (r *memoryAPIRepo) SaveTrafficProbeTarget(_ context.Context, _ *model.TrafficProbeTarget) error {
	return nil
}

type memoryMediaStore struct {
	files map[string][]byte
	dirs  map[string]struct{}
}

func newMemoryMediaStore() *memoryMediaStore {
	return &memoryMediaStore{
		files: make(map[string][]byte),
		dirs:  make(map[string]struct{}),
	}
}

func (s *memoryMediaStore) ReadFile(path string) ([]byte, error) {
	data, ok := s.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), data...), nil
}

func (s *memoryMediaStore) WriteFile(path string, data []byte, _ os.FileMode) error {
	s.files[path] = append([]byte(nil), data...)
	return nil
}

func (s *memoryMediaStore) Remove(path string) error {
	if _, ok := s.files[path]; !ok {
		return os.ErrNotExist
	}
	delete(s.files, path)
	return nil
}

func (s *memoryMediaStore) RemoveAll(path string) error {
	delete(s.dirs, path)
	prefix := strings.TrimRight(path, "/") + "/"
	for key := range s.files {
		if key == path || strings.HasPrefix(key, prefix) {
			delete(s.files, key)
		}
	}
	return nil
}

func (s *memoryMediaStore) MkdirAll(path string, _ os.FileMode) error {
	s.dirs[path] = struct{}{}
	return nil
}

func (s *memoryMediaStore) DetectMIMEFromBytes(data []byte) (string, error) {
	return http.DetectContentType(data), nil
}
