package service

import (
	"context"
	"errors"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rei0721/go-scaffold/internal/modules/system/model"
)

const (
	permissionScopePlatform = "platform"
	permissionScopeTenant   = "tenant"
	permissionScopeProduct  = "product"
)

// Config 描述系统服务的运行时依赖和默认策略。
// ConfigProvider/ConfigUpdater 用于对接外部配置管理；Media* 控制媒体上传边界；
// Now 和 StartTime 允许测试或组合根注入稳定时间。
type Config struct {
	MediaMaxBytes  int64
	MediaPrefix    string
	RuntimeConfig  model.ConfigSnapshot
	ConfigProvider func() model.ConfigSnapshot
	ConfigUpdater  func(context.Context, UpdateConfigInput) (model.ConfigSnapshot, error)
	Now            func() time.Time
	StartTime      time.Time
}

// Service 定义系统模块对 handler 暴露的应用层能力。
// 接口聚合菜单、配置、字典、参数、媒体、版本、API 同步和操作记录等管理功能。
type Service interface {
	CreateDictionary(context.Context, CreateDictionaryInput) (*model.Dictionary, error)
	CreateDictionaryItem(context.Context, int64, CreateDictionaryItemInput) (*model.DictionaryItem, error)
	CreateParameter(context.Context, CreateParameterInput) (*model.Parameter, error)
	DeleteVersion(context.Context, int64) error
	DeleteVersions(context.Context, []int64) error
	DeleteDictionary(context.Context, int64) error
	DeleteDictionaryItem(context.Context, int64) error
	DeleteMediaAsset(context.Context, int64) error
	DeleteMediaCategory(context.Context, int64) error
	DeleteOperationRecords(context.Context, []int64) error
	DeleteParameter(context.Context, int64) error
	DeleteParameters(context.Context, []int64) error
	DownloadMediaAsset(context.Context, int64) (MediaDownload, error)
	AbortMediaResumableUpload(context.Context, AbortMediaResumableUploadInput) (model.MediaResumableAbortResult, error)
	CheckMediaResumableUpload(context.Context, CheckMediaResumableUploadInput) (model.MediaResumableCheckResult, error)
	CompleteMediaResumableUpload(context.Context, CompleteMediaResumableUploadInput) (model.MediaResumableCompleteResult, error)
	ExportVersion(context.Context, ExportVersionInput) (*model.VersionDetail, error)
	FindParameter(context.Context, int64) (*model.Parameter, error)
	FindParameterByKey(context.Context, string) (*model.Parameter, error)
	GetPublicSettings(context.Context) (model.PublicSettings, error)
	FindVersion(context.Context, int64) (*model.VersionDetail, error)
	GetServerInfo(context.Context) (model.ServerInfo, error)
	GetServerMetricsHistory(context.Context) (model.ServerMetricsHistory, error)
	GetTrafficHijackOverview(context.Context) (model.TrafficHijackOverview, error)
	GetVersionPackage(context.Context, int64) (model.VersionPackage, error)
	ImportMediaURLs(context.Context, ImportMediaURLsInput) (model.MediaURLImportResult, error)
	ImportVersion(context.Context, ImportVersionInput) (model.VersionImportResult, error)
	ListAPIs(context.Context) ([]model.APIGroup, error)
	ListConfig(context.Context) (model.ConfigSnapshot, error)
	ListDictionaries(context.Context) (model.DictionaryCatalog, error)
	ListMediaAssets(context.Context, MediaAssetFilter) (model.MediaAssetPage, error)
	ListMediaCategories(context.Context) (model.MediaCategoryCatalog, error)
	ListMenus(context.Context) ([]model.MenuGroup, error)
	ListOperationRecords(context.Context, OperationRecordFilter) (model.OperationRecordPage, error)
	ListParameters(context.Context, ParameterFilter) (model.ParameterPage, error)
	ListTrafficHijackEvents(context.Context, TrafficHijackEventFilter) (model.TrafficHijackEventPage, error)
	ListTrafficProbeResults(context.Context, TrafficProbeResultFilter) (model.TrafficProbeResultPage, error)
	ListTrafficProbeTargets(context.Context) ([]model.TrafficProbeTarget, error)
	ListVersionSources(context.Context) (model.VersionSourceCatalog, error)
	ListVersions(context.Context, VersionFilter) (model.VersionPage, error)
	RecordOperation(context.Context, OperationRecordInput) error
	RegisterAPIs([]model.APIEntry)
	RunDueTrafficProbes(context.Context) (int, error)
	RunTrafficProbe(context.Context, int64) (model.TrafficProbeResult, error)
	SeedDefaults(context.Context) (SeedResult, error)
	SubscribeTrafficHijack(context.Context) (<-chan model.TrafficHijackStreamEvent, func())
	SyncAPIs(context.Context) (model.APISyncResult, error)
	SyncPermissions(context.Context) (model.PermissionSyncResult, error)
	CreateTrafficProbeTarget(context.Context, CreateTrafficProbeTargetInput) (*model.TrafficProbeTarget, error)
	UpdateConfig(context.Context, UpdateConfigInput) (model.ConfigSnapshot, error)
	UpdateDictionary(context.Context, int64, UpdateDictionaryInput) (*model.Dictionary, error)
	UpdateDictionaryItem(context.Context, int64, UpdateDictionaryItemInput) (*model.DictionaryItem, error)
	UpdateMediaAsset(context.Context, int64, UpdateMediaAssetInput) (*model.MediaAsset, error)
	UpdateParameter(context.Context, int64, UpdateParameterInput) (*model.Parameter, error)
	UpdateTrafficProbeTarget(context.Context, int64, UpdateTrafficProbeTargetInput) (*model.TrafficProbeTarget, error)
	UploadMediaChunk(context.Context, UploadMediaChunkInput) (model.MediaResumableChunkResult, error)
	UploadMediaAsset(context.Context, UploadMediaAssetInput) (*model.MediaAsset, error)
	UpsertMediaCategory(context.Context, UpsertMediaCategoryInput) (*model.MediaCategory, error)
	DeleteTrafficProbeTarget(context.Context, int64) error
	ResolveTrafficHijackEvent(context.Context, int64) (*model.TrafficHijackEvent, error)
}

// Option 用于向系统服务注入仓储、ID 生成器、对象存储等可替换依赖。
type Option func(*service)

// Repository 是系统服务需要的持久化端口。
// 服务层只依赖该接口，具体 GORM/SQL 实现留在 repository 或 infrastructure 包中。
type Repository interface {
	CreateAPI(context.Context, *model.APIRecord) error
	CreateDictionary(context.Context, *model.Dictionary) error
	CreateDictionaryItem(context.Context, *model.DictionaryItem) error
	CreateMediaAsset(context.Context, *model.MediaAsset) error
	CreateMediaCategory(context.Context, *model.MediaCategory) error
	CreateMediaUploadChunk(context.Context, *model.MediaUploadChunk) error
	CreateMediaUploadSession(context.Context, *model.MediaUploadSession) error
	CreateOperationRecord(context.Context, *model.OperationRecord) error
	CreateParameter(context.Context, *model.Parameter) error
	CreateTrafficHijackEvent(context.Context, *model.TrafficHijackEvent) error
	CreateTrafficProbeResult(context.Context, *model.TrafficProbeResult) error
	CreateTrafficProbeTarget(context.Context, *model.TrafficProbeTarget) error
	CreateVersion(context.Context, *model.Version) error
	DeleteDictionary(context.Context, int64, time.Time) error
	DeleteDictionaryItem(context.Context, int64, time.Time) error
	DeleteMediaAsset(context.Context, int64, time.Time) error
	DeleteMediaCategory(context.Context, int64, time.Time) error
	DeleteMediaUploadChunks(context.Context, int64) error
	DeleteOperationRecords(context.Context, []int64) error
	DeleteParameter(context.Context, int64, time.Time) error
	DeleteParameters(context.Context, []int64, time.Time) error
	DeleteOldTrafficProbeResults(context.Context, int64, int) error
	DeleteTrafficProbeTarget(context.Context, int64, time.Time) error
	DeleteVersion(context.Context, int64, time.Time) error
	DeleteVersions(context.Context, []int64, time.Time) error
	FindAPI(context.Context, string, string) (*model.APIRecord, error)
	FindDictionaryByCode(context.Context, string) (*model.Dictionary, error)
	FindDictionaryByID(context.Context, int64) (*model.Dictionary, error)
	FindDictionaryItemByID(context.Context, int64) (*model.DictionaryItem, error)
	FindMediaAssetByID(context.Context, int64) (*model.MediaAsset, error)
	FindMediaCategoryByID(context.Context, int64) (*model.MediaCategory, error)
	FindMediaUploadChunk(context.Context, int64, int) (*model.MediaUploadChunk, error)
	FindMediaUploadSessionByHash(context.Context, string, string, int64, int64) (*model.MediaUploadSession, error)
	FindMediaUploadSessionByID(context.Context, int64) (*model.MediaUploadSession, error)
	FindParameterByID(context.Context, int64) (*model.Parameter, error)
	FindParameterByKey(context.Context, string) (*model.Parameter, error)
	FindTrafficHijackEvent(context.Context, int64) (*model.TrafficHijackEvent, error)
	FindTrafficProbeTargetByID(context.Context, int64) (*model.TrafficProbeTarget, error)
	FindOpenTrafficHijackEvent(context.Context, int64, string, string) (*model.TrafficHijackEvent, error)
	FindVersionByID(context.Context, int64) (*model.Version, error)
	ListAPIs(context.Context) ([]model.APIRecord, error)
	ListDictionaries(context.Context) ([]model.Dictionary, error)
	ListDictionaryItems(context.Context, int64) ([]model.DictionaryItem, error)
	ListMediaAssets(context.Context, model.MediaAssetFilter) ([]model.MediaAsset, int64, error)
	ListMediaCategories(context.Context) ([]model.MediaCategory, error)
	ListMediaUploadChunks(context.Context, int64) ([]model.MediaUploadChunk, error)
	ListOperationRecords(context.Context, model.OperationRecordFilter) ([]model.OperationRecord, int64, error)
	ListParameters(context.Context, model.ParameterFilter) ([]model.Parameter, int64, error)
	ListTrafficHijackEvents(context.Context, model.TrafficHijackEventFilter) ([]model.TrafficHijackEvent, int64, error)
	ListTrafficProbeResults(context.Context, model.TrafficProbeResultFilter) ([]model.TrafficProbeResult, error)
	ListTrafficProbeTargets(context.Context) ([]model.TrafficProbeTarget, error)
	ListVersions(context.Context, model.VersionFilter) ([]model.Version, int64, error)
	SaveAPI(context.Context, *model.APIRecord) error
	SaveDictionary(context.Context, *model.Dictionary) error
	SaveDictionaryItem(context.Context, *model.DictionaryItem) error
	SaveMediaAsset(context.Context, *model.MediaAsset) error
	SaveMediaCategory(context.Context, *model.MediaCategory) error
	SaveMediaUploadChunk(context.Context, *model.MediaUploadChunk) error
	SaveMediaUploadSession(context.Context, *model.MediaUploadSession) error
	SaveParameter(context.Context, *model.Parameter) error
	SaveTrafficHijackEvent(context.Context, *model.TrafficHijackEvent) error
	SaveTrafficProbeTarget(context.Context, *model.TrafficProbeTarget) error
}

// PermissionStore 抽象权限表的最小写入与查询能力。
// API 权限同步只需要列出已有权限并创建缺失权限。
type PermissionStore interface {
	CreatePermission(context.Context, model.PermissionEntry) error
	ListPermissions(context.Context) ([]model.PermissionEntry, error)
}

// IDGenerator 为系统资源生成数字和字符串 ID。
// 测试可注入确定性实现，生产环境可替换为雪花或数据库序列。
type IDGenerator interface {
	NextID() int64
	NextIDString() string
}

// HostMetricsCollector 采集主机 CPU、内存和磁盘指标。
// 服务层会把采集结果映射为管理端模型，不直接依赖具体平台实现。
type HostMetricsCollector interface {
	Collect(context.Context) HostMetrics
}

// MetricsHistoryProvider 返回由应用装配层维护的短窗口运行指标历史。
type MetricsHistoryProvider interface {
	History(context.Context) MetricsHistory
}

// HostMetrics 是平台采集器返回的原始主机指标集合。
type HostMetrics struct {
	CPU     CPUInfo
	RAM     RAMInfo
	Disk    []DiskInfo
	DiskIO  []DiskIOInfo
	Network NetworkInfo
}

type MetricsHistory struct {
	IntervalSeconds int
	WindowSeconds   int
	Samples         []MetricsSample
}

type MetricsSample struct {
	SampledAt                  time.Time
	CPUUsedPercent             float64
	RAMUsedPercent             float64
	DiskMaxUsedPercent         float64
	DiskReadMBPerSecond        float64
	DiskWriteMBPerSecond       float64
	DiskReadOpsPerSecond       float64
	DiskWriteOpsPerSecond      float64
	DiskIOLatencyMs            float64
	DiskIO                     []DiskIOSample
	HeapAllocMB                uint64
	Goroutines                 int
	NetworkReceiveKBPerSecond  float64
	NetworkTransmitKBPerSecond float64
}

// CPUInfo 描述 CPU 核心数和各核心或整体使用率。
type CPUInfo struct {
	Cores   int
	Percent []float64
}

// RAMInfo 描述主机内存容量和使用情况。
type RAMInfo struct {
	TotalMB     uint64
	UsedMB      uint64
	UsedPercent float64
}

// NetworkInfo 描述网络累计收发字节数，用于短窗口趋势换算。
type NetworkInfo struct {
	ReceiveBytes  uint64
	TransmitBytes uint64
}

// DiskInfo 描述单个挂载点的容量和使用情况。
type DiskInfo struct {
	FSType      string
	MountPoint  string
	TotalGB     uint64
	TotalMB     uint64
	UsedGB      uint64
	UsedMB      uint64
	UsedPercent float64
}

type DiskIOInfo struct {
	Name        string
	ReadBytes   uint64
	WriteBytes  uint64
	ReadCount   uint64
	WriteCount  uint64
	ReadTimeMs  uint64
	WriteTimeMs uint64
}

type DiskIOSample struct {
	Name              string
	ReadMBPerSecond   float64
	WriteMBPerSecond  float64
	ReadOpsPerSecond  float64
	WriteOpsPerSecond float64
	IOLatencyMs       float64
}

// CreateDictionaryInput 描述创建字典时需要的业务字段。
type CreateDictionaryInput struct {
	Code        string
	Description string
	Name        string
	Status      string
}

// UpdateDictionaryInput 使用指针区分“未更新”和“更新为空值”。
type UpdateDictionaryInput struct {
	Description *string
	Name        *string
	Status      *string
}

// CreateDictionaryItemInput 描述创建字典项的字段。
type CreateDictionaryItemInput struct {
	Extra  string
	Label  string
	Sort   int
	Status string
	Value  string
}

// UpdateDictionaryItemInput 使用指针表达局部更新语义。
type UpdateDictionaryItemInput struct {
	Extra  *string
	Label  *string
	Sort   *int
	Status *string
	Value  *string
}

// CreateParameterInput 描述创建系统参数所需字段。
type CreateParameterInput struct {
	Description string
	Key         string
	Name        string
	Value       string
}

// UpdateParameterInput 使用指针表达参数的局部更新语义。
type UpdateParameterInput struct {
	Description *string
	Key         *string
	Name        *string
	Value       *string
}

// UpdateConfigInput 描述运行时配置批量更新请求。
// Persist 控制是否要求外部配置管理器把变更写回持久化配置。
type UpdateConfigInput struct {
	Items   []UpdateConfigItem
	Persist bool
}

// UpdateConfigItem 表示单个配置键值更新。
type UpdateConfigItem struct {
	Key   string
	Value any
}

// OperationRecordInput 描述一次 HTTP 操作审计记录。
// Body、Response、UserAgent 等大字段会在入库前截断，避免日志表被异常请求撑大。
type OperationRecordInput struct {
	Body         string
	ErrorMessage string
	IPAddress    string
	LatencyMs    int64
	Method       string
	Path         string
	Response     string
	Status       int
	TraceID      string
	UserAgent    string
	UserID       int64
	Username     string
}

// OperationRecordFilter 描述操作记录列表过滤条件。
// Status 优先级高于 StatusClass，指定精确状态码时会忽略状态分类。
type OperationRecordFilter struct {
	Method      string
	Page        int
	PageSize    int
	Path        string
	Status      int
	StatusClass string
}

type CreateTrafficProbeTargetInput struct {
	AlertChannels          []string
	AllowPrivateNetwork    bool
	EmailRecipients        []string
	Enabled                bool
	ExpectedContentKeyword string
	ExpectedFinalHost      string
	ExpectedIPCIDRs        []string
	ExpectedStatusCodes    string
	ExpectedTLSFingerprint string
	IntervalSeconds        int
	Method                 string
	Name                   string
	TimeoutSeconds         int
	URL                    string
}

type UpdateTrafficProbeTargetInput struct {
	AlertChannels          *[]string
	AllowPrivateNetwork    *bool
	EmailRecipients        *[]string
	Enabled                *bool
	ExpectedContentKeyword *string
	ExpectedFinalHost      *string
	ExpectedIPCIDRs        *[]string
	ExpectedStatusCodes    *string
	ExpectedTLSFingerprint *string
	IntervalSeconds        *int
	Method                 *string
	Name                   *string
	TimeoutSeconds         *int
	URL                    *string
}

type TrafficProbeResultFilter = model.TrafficProbeResultFilter
type TrafficHijackEventFilter = model.TrafficHijackEventFilter

type TrafficProbeRunner interface {
	Probe(context.Context, model.TrafficProbeTarget) model.TrafficProbeResult
}

type TrafficAlertSink interface {
	NotifyTrafficHijack(context.Context, model.TrafficProbeTarget, model.TrafficHijackEvent, model.TrafficProbeResult, string) string
}

// ParameterFilter 描述系统参数列表过滤条件。
// 时间范围需要满足 StartCreatedAt 早于 EndCreatedAt。
type ParameterFilter struct {
	EndCreatedAt   *time.Time
	Key            string
	Name           string
	Page           int
	PageSize       int
	StartCreatedAt *time.Time
}

var (
	ErrDuplicate          = errors.New("system resource already exists")
	ErrExternalMedia      = errors.New("external media can be opened by url")
	ErrConfigUnavailable  = errors.New("runtime config manager unavailable")
	ErrInvalidInput       = errors.New("invalid system input")
	ErrNotFound           = errors.New("system resource not found")
	ErrStorageUnavailable = errors.New("system storage unavailable")
)

func isStorageUnavailable(err error) bool {
	return errors.Is(err, ErrStorageUnavailable)
}

// service 是系统模块的应用层实现。
// 可变的 API 注册表由 mu 保护；其余依赖在构造后作为组合根注入的端口使用。
type service struct {
	cfg             Config
	ids             IDGenerator
	hostMetrics     HostMetricsCollector
	metricsHistory  MetricsHistoryProvider
	trafficRunner   TrafficProbeRunner
	trafficAlert    TrafficAlertSink
	mu              sync.RWMutex
	apis            []model.APIEntry
	trafficMu       sync.RWMutex
	trafficNextSub  int64
	trafficSubs     map[int64]chan model.TrafficHijackStreamEvent
	objectStore     MediaObjectStorage
	repo            Repository
	permissionStore PermissionStore
}

// WithRepository 注入系统模块持久化仓储。
func WithRepository(repo Repository) Option {
	return func(s *service) {
		s.repo = repo
	}
}

// WithIDGenerator 注入资源 ID 生成器。
func WithIDGenerator(ids IDGenerator) Option {
	return func(s *service) {
		s.ids = ids
	}
}

// WithHostMetrics 注入主机指标采集器。
func WithHostMetrics(collector HostMetricsCollector) Option {
	return func(s *service) {
		s.hostMetrics = collector
	}
}

// WithMetricsHistory 注入短窗口指标历史提供器。
func WithMetricsHistory(provider MetricsHistoryProvider) Option {
	return func(s *service) {
		s.metricsHistory = provider
	}
}

func WithTrafficProbeRunner(runner TrafficProbeRunner) Option {
	return func(s *service) {
		s.trafficRunner = runner
	}
}

func WithTrafficAlertSink(sink TrafficAlertSink) Option {
	return func(s *service) {
		s.trafficAlert = sink
	}
}

// WithPermissionStore 注入权限同步使用的权限存储端口。
func WithPermissionStore(store PermissionStore) Option {
	return func(s *service) {
		s.permissionStore = store
	}
}

// WithStorage 注入媒体对象存储端口。
func WithStorage(store MediaObjectStorage) Option {
	return func(s *service) {
		s.objectStore = store
	}
}

// New 创建系统服务并补齐默认依赖。
// 未注入 ID 生成器或指标采集器时使用进程内默认实现；媒体前缀和大小限制也会在这里归一化。
func New(cfg Config, options ...Option) Service {
	s := &service{cfg: cfg, trafficSubs: map[int64]chan model.TrafficHijackStreamEvent{}}
	for _, option := range options {
		option(s)
	}
	if s.ids == nil {
		s.ids = &sequentialIDGenerator{}
	}
	if s.hostMetrics == nil {
		s.hostMetrics = noopHostMetricsCollector{}
	}
	if s.metricsHistory == nil {
		s.metricsHistory = noopMetricsHistoryProvider{}
	}
	if s.trafficRunner == nil {
		s.trafficRunner = noopTrafficProbeRunner{}
	}
	if s.trafficAlert == nil {
		s.trafficAlert = noopTrafficAlertSink{}
	}
	if strings.TrimSpace(s.cfg.MediaPrefix) == "" {
		s.cfg.MediaPrefix = "media"
	}
	if s.cfg.MediaMaxBytes <= 0 {
		s.cfg.MediaMaxBytes = 20 * 1024 * 1024
	}
	return s
}

// sequentialIDGenerator 是测试和默认场景下使用的简单递增 ID 生成器。
// 生产环境如需全局唯一性，应通过 WithIDGenerator 替换。
type sequentialIDGenerator struct {
	mu   sync.Mutex
	next int64
}

// NextID 返回进程内递增的数字 ID。
func (g *sequentialIDGenerator) NextID() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.next++
	return g.next
}

// NextIDString 返回字符串形式的递增 ID。
func (g *sequentialIDGenerator) NextIDString() string {
	return strconv.FormatInt(g.NextID(), 10)
}

// noopHostMetricsCollector 是默认空指标采集器。
type noopHostMetricsCollector struct{}

// Collect 返回空指标，确保未配置平台采集器时服务器信息接口仍可用。
func (noopHostMetricsCollector) Collect(context.Context) HostMetrics {
	return HostMetrics{}
}

type noopMetricsHistoryProvider struct{}

func (noopMetricsHistoryProvider) History(context.Context) MetricsHistory {
	return MetricsHistory{}
}

type noopTrafficProbeRunner struct{}

func (noopTrafficProbeRunner) Probe(_ context.Context, target model.TrafficProbeTarget) model.TrafficProbeResult {
	return model.TrafficProbeResult{
		TargetID:     target.ID,
		TargetName:   target.Name,
		URL:          target.URL,
		Method:       target.Method,
		Status:       model.TrafficProbeStatusCritical,
		Severity:     model.TrafficProbeSeverityHigh,
		Reason:       "probe runner unavailable",
		Stage:        "probe",
		ErrorMessage: "traffic probe runner is not configured",
	}
}

type noopTrafficAlertSink struct{}

func (noopTrafficAlertSink) NotifyTrafficHijack(context.Context, model.TrafficProbeTarget, model.TrafficHijackEvent, model.TrafficProbeResult, string) string {
	return "skipped"
}

// ListMenus 返回系统内置菜单目录。
// 返回前会深拷贝并排序，避免调用方修改 baseMenus 全局定义。
func (s *service) ListMenus(context.Context) ([]model.MenuGroup, error) {
	groups := cloneGroups(baseMenus)
	sortMenus(groups)
	return groups, nil
}

// ListConfig 返回当前运行时配置快照。
// 优先使用 ConfigProvider 动态读取；返回值会克隆，避免调用方修改服务内部或外部 provider 状态。
func (s *service) ListConfig(context.Context) (model.ConfigSnapshot, error) {
	if s.cfg.ConfigProvider != nil {
		return cloneConfigSnapshot(s.cfg.ConfigProvider()), nil
	}
	return cloneConfigSnapshot(s.cfg.RuntimeConfig), nil
}

func (s *service) GetPublicSettings(ctx context.Context) (model.PublicSettings, error) {
	snapshot, err := s.ListConfig(ctx)
	if err != nil {
		return model.PublicSettings{}, err
	}
	return model.PublicSettings{
		Brand: model.PublicBrandSettings{
			ProductName: stringConfigValue(snapshot, "brand.productName"),
			ProductCode: stringConfigValue(snapshot, "brand.productCode"),
			VersionName: stringConfigValue(snapshot, "brand.versionName"),
		},
		Auth: model.PublicAuthSettings{
			CSRFEnabled:        boolConfigValue(snapshot, "auth.csrf.enabled"),
			CSRFCookieName:     stringConfigValue(snapshot, "auth.csrf.cookie_name"),
			CSRFHeaderName:     stringConfigValue(snapshot, "auth.csrf.header_name"),
			RegistrationMode:   stringConfigValue(snapshot, "auth.registration_mode"),
			ProductHeader:      stringConfigValue(snapshot, "auth.session.product_header"),
			ClientTypeHeader:   stringConfigValue(snapshot, "auth.session.client_type_header"),
			DefaultProductCode: stringConfigValue(snapshot, "brand.productCode"),
			DefaultClientType:  stringConfigValue(snapshot, "auth.session.default_client_type"),
		},
		DefaultLocale:    stringConfigValue(snapshot, "i18n.defaultLocale"),
		FallbackLocale:   stringConfigValue(snapshot, "i18n.fallbackLocale"),
		SupportedLocales: stringSliceConfigValue(snapshot, "i18n.supportedLocales"),
	}, nil
}

// UpdateConfig 通过外部配置管理器批量更新运行时配置。
// input.Items 会先校验并清理 key；返回值是更新后的配置快照副本。
func (s *service) UpdateConfig(ctx context.Context, input UpdateConfigInput) (model.ConfigSnapshot, error) {
	if s.cfg.ConfigUpdater == nil {
		return model.ConfigSnapshot{}, ErrConfigUnavailable
	}
	items := make([]UpdateConfigItem, 0, len(input.Items))
	for _, item := range input.Items {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			return model.ConfigSnapshot{}, ErrInvalidInput
		}
		items = append(items, UpdateConfigItem{
			Key:   key,
			Value: item.Value,
		})
	}
	if len(items) == 0 {
		return model.ConfigSnapshot{}, ErrInvalidInput
	}
	snapshot, err := s.cfg.ConfigUpdater(ctx, UpdateConfigInput{Items: items, Persist: input.Persist})
	if err != nil {
		return model.ConfigSnapshot{}, err
	}
	return cloneConfigSnapshot(snapshot), nil
}

// GetServerInfo 汇总 Go runtime、构建信息和主机指标。
// StartTime 为空或晚于当前时间时会回退到当前时间，避免展示负 uptime。
func (s *service) GetServerInfo(ctx context.Context) (model.ServerInfo, error) {
	now := s.now()
	start := s.cfg.StartTime
	if !start.IsZero() {
		start = start.UTC()
	}
	if start.IsZero() || start.After(now) {
		start = now
	}
	uptime := now.Sub(start)
	if uptime < 0 {
		uptime = 0
	}

	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	host := s.hostMetrics.Collect(ctx)

	info := model.ServerInfo{
		Build: buildInfo(),
		CPU:   mapServerCPU(host.CPU),
		Disk:  mapServerDisks(host.Disk),
		GC: model.ServerGCInfo{
			NextGCMB:     bytesToMB(stats.NextGC),
			NumGC:        stats.NumGC,
			PauseTotalNs: stats.PauseTotalNs,
		},
		Memory: model.ServerMemoryInfo{
			AllocMB:        bytesToMB(stats.Alloc),
			HeapAllocMB:    bytesToMB(stats.HeapAlloc),
			HeapIdleMB:     bytesToMB(stats.HeapIdle),
			HeapInuseMB:    bytesToMB(stats.HeapInuse),
			HeapObjects:    stats.HeapObjects,
			HeapReleasedMB: bytesToMB(stats.HeapReleased),
			HeapSysMB:      bytesToMB(stats.HeapSys),
			StackInuseMB:   bytesToMB(stats.StackInuse),
			StackSysMB:     bytesToMB(stats.StackSys),
			SysMB:          bytesToMB(stats.Sys),
			TotalAllocMB:   bytesToMB(stats.TotalAlloc),
		},
		OS: model.ServerOSInfo{
			Compiler:     runtime.Compiler,
			GoArch:       runtime.GOARCH,
			GoOS:         runtime.GOOS,
			GoVersion:    runtime.Version(),
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
		},
		RAM:         mapServerRAM(host.RAM),
		RefreshedAt: now,
		Runtime: model.ServerRuntimeInfo{
			StartTime:     start,
			Uptime:        formatRuntimeDuration(uptime),
			UptimeSeconds: int64(uptime.Seconds()),
		},
	}
	if stats.LastGC > 0 {
		lastGCAt := time.Unix(0, int64(stats.LastGC)).UTC()
		info.GC.LastGCAt = &lastGCAt
	}
	return info, nil
}

// GetServerMetricsHistory 返回应用装配层维护的短窗口运行指标历史。
func (s *service) GetServerMetricsHistory(ctx context.Context) (model.ServerMetricsHistory, error) {
	history := s.metricsHistory.History(ctx)
	samples := make([]model.ServerMetricsSample, 0, len(history.Samples))
	for _, sample := range history.Samples {
		samples = append(samples, model.ServerMetricsSample{
			CPUUsedPercent:             sample.CPUUsedPercent,
			DiskMaxUsedPercent:         sample.DiskMaxUsedPercent,
			DiskReadMBPerSecond:        sample.DiskReadMBPerSecond,
			DiskWriteMBPerSecond:       sample.DiskWriteMBPerSecond,
			DiskReadOpsPerSecond:       sample.DiskReadOpsPerSecond,
			DiskWriteOpsPerSecond:      sample.DiskWriteOpsPerSecond,
			DiskIOLatencyMs:            sample.DiskIOLatencyMs,
			DiskIO:                     mapServerDiskIOSamples(sample.DiskIO),
			Goroutines:                 sample.Goroutines,
			HeapAllocMB:                sample.HeapAllocMB,
			NetworkReceiveKBPerSecond:  sample.NetworkReceiveKBPerSecond,
			NetworkTransmitKBPerSecond: sample.NetworkTransmitKBPerSecond,
			RAMUsedPercent:             sample.RAMUsedPercent,
			SampledAt:                  sample.SampledAt,
		})
	}
	return model.ServerMetricsHistory{
		IntervalSeconds: history.IntervalSeconds,
		Samples:         samples,
		WindowSeconds:   history.WindowSeconds,
	}, nil
}

// ListDictionaries 返回字典及其字典项。
// 仓储不可用时返回 unavailable 状态的空目录，便于管理端降级展示。
func (s *service) ListDictionaries(ctx context.Context) (model.DictionaryCatalog, error) {
	catalog := model.DictionaryCatalog{StorageStatus: "unavailable"}
	if s.repo == nil {
		return catalog, nil
	}
	dictionaries, err := s.repo.ListDictionaries(ctx)
	if err != nil {
		if isStorageUnavailable(err) {
			return catalog, nil
		}
		return catalog, err
	}
	for i := range dictionaries {
		items, err := s.repo.ListDictionaryItems(ctx, dictionaries[i].ID)
		if err != nil {
			if isStorageUnavailable(err) {
				return model.DictionaryCatalog{StorageStatus: "unavailable"}, nil
			}
			return catalog, err
		}
		dictionaries[i].Items = items
	}
	catalog.Items = dictionaries
	catalog.StorageStatus = "persisted"
	catalog.Total = len(dictionaries)
	return catalog, nil
}

// CreateDictionary 创建字典主记录。
// Code 会被归一化并校验唯一性；返回值包含空 Items，方便前端直接使用统一结构。
func (s *service) CreateDictionary(ctx context.Context, input CreateDictionaryInput) (*model.Dictionary, error) {
	if s.repo == nil {
		return nil, ErrStorageUnavailable
	}
	code := normalizeDictionaryCode(input.Code)
	name := strings.TrimSpace(input.Name)
	if !validDictionaryCode(code) || name == "" {
		return nil, ErrInvalidInput
	}
	status, err := normalizeDictionaryStatus(input.Status)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.FindDictionaryByCode(ctx, code); err == nil {
		return nil, ErrDuplicate
	} else if !errors.Is(err, ErrNotFound) {
		if isStorageUnavailable(err) {
			return nil, ErrStorageUnavailable
		}
		return nil, err
	}
	now := s.now()
	dictionary := &model.Dictionary{
		ID:          s.ids.NextID(),
		Code:        code,
		Description: strings.TrimSpace(input.Description),
		Name:        name,
		Status:      status,
		CreatedAt:   now,
		UpdatedAt:   now,
		Items:       []model.DictionaryItem{},
	}
	if err := s.repo.CreateDictionary(ctx, dictionary); err != nil {
		if isStorageUnavailable(err) {
			return nil, ErrStorageUnavailable
		}
		return nil, err
	}
	return dictionary, nil
}

// UpdateDictionary 局部更新字典主记录。
// 只有非 nil 字段会被应用；保存后会尽力加载字典项以返回完整字典结构。
func (s *service) UpdateDictionary(ctx context.Context, id int64, input UpdateDictionaryInput) (*model.Dictionary, error) {
	if s.repo == nil {
		return nil, ErrStorageUnavailable
	}
	dictionary, err := s.repo.FindDictionaryByID(ctx, id)
	if err != nil {
		return nil, mapDictionaryLookupError(err)
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrInvalidInput
		}
		dictionary.Name = name
	}
	if input.Description != nil {
		dictionary.Description = strings.TrimSpace(*input.Description)
	}
	if input.Status != nil {
		status, err := normalizeDictionaryStatus(*input.Status)
		if err != nil {
			return nil, err
		}
		dictionary.Status = status
	}
	if err := s.repo.SaveDictionary(ctx, dictionary); err != nil {
		if isStorageUnavailable(err) {
			return nil, ErrStorageUnavailable
		}
		return nil, err
	}
	items, err := s.repo.ListDictionaryItems(ctx, dictionary.ID)
	if err != nil && !isStorageUnavailable(err) {
		return nil, err
	}
	dictionary.Items = items
	return dictionary, nil
}

// DeleteDictionary 软删除字典主记录。
// 删除前先查找记录，用统一错误语义区分不存在和仓储不可用。
func (s *service) DeleteDictionary(ctx context.Context, id int64) error {
	if s.repo == nil {
		return ErrStorageUnavailable
	}
	if _, err := s.repo.FindDictionaryByID(ctx, id); err != nil {
		return mapDictionaryLookupError(err)
	}
	if err := s.repo.DeleteDictionary(ctx, id, s.now()); err != nil {
		if isStorageUnavailable(err) {
			return ErrStorageUnavailable
		}
		return err
	}
	return nil
}

// CreateDictionaryItem 在指定字典下创建字典项。
// 创建前会确认父字典存在，避免产生孤立字典项。
func (s *service) CreateDictionaryItem(ctx context.Context, dictionaryID int64, input CreateDictionaryItemInput) (*model.DictionaryItem, error) {
	if s.repo == nil {
		return nil, ErrStorageUnavailable
	}
	if _, err := s.repo.FindDictionaryByID(ctx, dictionaryID); err != nil {
		return nil, mapDictionaryLookupError(err)
	}
	label := strings.TrimSpace(input.Label)
	value := strings.TrimSpace(input.Value)
	if label == "" || value == "" {
		return nil, ErrInvalidInput
	}
	status, err := normalizeDictionaryStatus(input.Status)
	if err != nil {
		return nil, err
	}
	now := s.now()
	item := &model.DictionaryItem{
		ID:           s.ids.NextID(),
		DictionaryID: dictionaryID,
		Extra:        strings.TrimSpace(input.Extra),
		Label:        label,
		Sort:         input.Sort,
		Status:       status,
		Value:        value,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.repo.CreateDictionaryItem(ctx, item); err != nil {
		if isStorageUnavailable(err) {
			return nil, ErrStorageUnavailable
		}
		return nil, err
	}
	return item, nil
}

// UpdateDictionaryItem 局部更新单个字典项。
// Label 和 Value 更新为空会被拒绝，避免枚举项失去可展示或可匹配字段。
func (s *service) UpdateDictionaryItem(ctx context.Context, id int64, input UpdateDictionaryItemInput) (*model.DictionaryItem, error) {
	if s.repo == nil {
		return nil, ErrStorageUnavailable
	}
	item, err := s.repo.FindDictionaryItemByID(ctx, id)
	if err != nil {
		return nil, mapDictionaryLookupError(err)
	}
	if input.Label != nil {
		label := strings.TrimSpace(*input.Label)
		if label == "" {
			return nil, ErrInvalidInput
		}
		item.Label = label
	}
	if input.Value != nil {
		value := strings.TrimSpace(*input.Value)
		if value == "" {
			return nil, ErrInvalidInput
		}
		item.Value = value
	}
	if input.Extra != nil {
		item.Extra = strings.TrimSpace(*input.Extra)
	}
	if input.Sort != nil {
		item.Sort = *input.Sort
	}
	if input.Status != nil {
		status, err := normalizeDictionaryStatus(*input.Status)
		if err != nil {
			return nil, err
		}
		item.Status = status
	}
	if err := s.repo.SaveDictionaryItem(ctx, item); err != nil {
		if isStorageUnavailable(err) {
			return nil, ErrStorageUnavailable
		}
		return nil, err
	}
	return item, nil
}

// DeleteDictionaryItem 软删除单个字典项。
func (s *service) DeleteDictionaryItem(ctx context.Context, id int64) error {
	if s.repo == nil {
		return ErrStorageUnavailable
	}
	if _, err := s.repo.FindDictionaryItemByID(ctx, id); err != nil {
		return mapDictionaryLookupError(err)
	}
	if err := s.repo.DeleteDictionaryItem(ctx, id, s.now()); err != nil {
		if isStorageUnavailable(err) {
			return ErrStorageUnavailable
		}
		return err
	}
	return nil
}

// ListParameters 分页查询系统参数。
// 仓储不可用时返回 unavailable 空页；时间范围必须按创建时间正向排列。
func (s *service) ListParameters(ctx context.Context, input ParameterFilter) (model.ParameterPage, error) {
	page := normalizePage(input.Page)
	pageSize := normalizePageSize(input.PageSize)
	result := model.ParameterPage{Page: page, PageSize: pageSize, StorageStatus: "unavailable"}
	if s.repo == nil {
		return result, nil
	}
	if input.StartCreatedAt != nil && input.EndCreatedAt != nil && !input.StartCreatedAt.Before(*input.EndCreatedAt) {
		return result, ErrInvalidInput
	}
	parameters, total, err := s.repo.ListParameters(ctx, model.ParameterFilter{
		EndCreatedAt:   input.EndCreatedAt,
		Key:            strings.TrimSpace(input.Key),
		Name:           strings.TrimSpace(input.Name),
		Page:           page,
		PageSize:       pageSize,
		StartCreatedAt: input.StartCreatedAt,
	})
	if err != nil {
		if isStorageUnavailable(err) {
			return result, nil
		}
		return result, err
	}
	result.Items = parameters
	result.StorageStatus = "persisted"
	result.Total = total
	return result, nil
}

// CreateParameter 创建系统参数。
// Key 在写入前会校验唯一性，避免多个参数争用同一个配置语义。
func (s *service) CreateParameter(ctx context.Context, input CreateParameterInput) (*model.Parameter, error) {
	if s.repo == nil {
		return nil, ErrStorageUnavailable
	}
	name := strings.TrimSpace(input.Name)
	key := strings.TrimSpace(input.Key)
	value := strings.TrimSpace(input.Value)
	if name == "" || key == "" || value == "" {
		return nil, ErrInvalidInput
	}
	if _, err := s.repo.FindParameterByKey(ctx, key); err == nil {
		return nil, ErrDuplicate
	} else if !errors.Is(err, ErrNotFound) {
		return nil, mapParameterLookupError(err)
	}
	now := s.now()
	parameter := &model.Parameter{
		ID:          s.ids.NextID(),
		CreatedAt:   now,
		Description: strings.TrimSpace(input.Description),
		Key:         key,
		Name:        name,
		UpdatedAt:   now,
		Value:       value,
	}
	if err := s.repo.CreateParameter(ctx, parameter); err != nil {
		if isStorageUnavailable(err) {
			return nil, ErrStorageUnavailable
		}
		return nil, err
	}
	return parameter, nil
}

// UpdateParameter 局部更新系统参数。
// Key 变更时会重新查重，允许保持原 key 不触发重复错误。
func (s *service) UpdateParameter(ctx context.Context, id int64, input UpdateParameterInput) (*model.Parameter, error) {
	if s.repo == nil {
		return nil, ErrStorageUnavailable
	}
	parameter, err := s.repo.FindParameterByID(ctx, id)
	if err != nil {
		return nil, mapParameterLookupError(err)
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrInvalidInput
		}
		parameter.Name = name
	}
	if input.Key != nil {
		key := strings.TrimSpace(*input.Key)
		if key == "" {
			return nil, ErrInvalidInput
		}
		if key != parameter.Key {
			existing, err := s.repo.FindParameterByKey(ctx, key)
			if err == nil && existing.ID != parameter.ID {
				return nil, ErrDuplicate
			}
			if err != nil && !errors.Is(err, ErrNotFound) {
				return nil, mapParameterLookupError(err)
			}
		}
		parameter.Key = key
	}
	if input.Value != nil {
		value := strings.TrimSpace(*input.Value)
		if value == "" {
			return nil, ErrInvalidInput
		}
		parameter.Value = value
	}
	if input.Description != nil {
		parameter.Description = strings.TrimSpace(*input.Description)
	}
	if err := s.repo.SaveParameter(ctx, parameter); err != nil {
		if isStorageUnavailable(err) {
			return nil, ErrStorageUnavailable
		}
		return nil, err
	}
	return parameter, nil
}

// DeleteParameter 软删除单个系统参数。
func (s *service) DeleteParameter(ctx context.Context, id int64) error {
	if s.repo == nil {
		return ErrStorageUnavailable
	}
	if _, err := s.repo.FindParameterByID(ctx, id); err != nil {
		return mapParameterLookupError(err)
	}
	if err := s.repo.DeleteParameter(ctx, id, s.now()); err != nil {
		if isStorageUnavailable(err) {
			return ErrStorageUnavailable
		}
		return err
	}
	return nil
}

// DeleteParameters 批量软删除系统参数。
// ids 必须非空且全部为正数，避免误把空条件传给仓储层。
func (s *service) DeleteParameters(ctx context.Context, ids []int64) error {
	if s.repo == nil {
		return ErrStorageUnavailable
	}
	if len(ids) == 0 {
		return ErrInvalidInput
	}
	normalized := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return ErrInvalidInput
		}
		normalized = append(normalized, id)
	}
	if err := s.repo.DeleteParameters(ctx, normalized, s.now()); err != nil {
		if isStorageUnavailable(err) {
			return ErrStorageUnavailable
		}
		return err
	}
	return nil
}

// FindParameter 按 ID 读取系统参数。
func (s *service) FindParameter(ctx context.Context, id int64) (*model.Parameter, error) {
	if s.repo == nil {
		return nil, ErrStorageUnavailable
	}
	parameter, err := s.repo.FindParameterByID(ctx, id)
	if err != nil {
		return nil, mapParameterLookupError(err)
	}
	return parameter, nil
}

// FindParameterByKey 按 key 读取系统参数。
// key 会先去除首尾空白，空 key 会被视为无效输入。
func (s *service) FindParameterByKey(ctx context.Context, key string) (*model.Parameter, error) {
	if s.repo == nil {
		return nil, ErrStorageUnavailable
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, ErrInvalidInput
	}
	parameter, err := s.repo.FindParameterByKey(ctx, key)
	if err != nil {
		return nil, mapParameterLookupError(err)
	}
	return parameter, nil
}

// RecordOperation 写入一次操作审计记录。
// 请求体、响应体和错误信息会限制长度，避免异常大 payload 影响审计表性能。
func (s *service) RecordOperation(ctx context.Context, input OperationRecordInput) error {
	if s.repo == nil {
		return ErrStorageUnavailable
	}
	method := strings.ToUpper(strings.TrimSpace(input.Method))
	path := strings.TrimSpace(input.Path)
	if method == "" || path == "" {
		return ErrInvalidInput
	}
	now := s.now()
	record := &model.OperationRecord{
		ID:           s.ids.NextID(),
		Body:         trimOperationPayload(input.Body),
		CreatedAt:    now,
		ErrorMessage: trimOperationPayload(input.ErrorMessage),
		IPAddress:    strings.TrimSpace(input.IPAddress),
		LatencyMs:    input.LatencyMs,
		Method:       method,
		Path:         path,
		Response:     trimOperationPayload(input.Response),
		Status:       input.Status,
		TraceID:      strings.TrimSpace(input.TraceID),
		UserAgent:    trimOperationPayload(input.UserAgent),
		UserID:       input.UserID,
		Username:     strings.TrimSpace(input.Username),
	}
	if err := s.repo.CreateOperationRecord(ctx, record); err != nil {
		if isStorageUnavailable(err) {
			return ErrStorageUnavailable
		}
		return err
	}
	return nil
}

// ListOperationRecords 分页查询操作审计记录。
// StatusClass 支持按 4xx、5xx 或 error 聚合过滤；指定精确 Status 时会覆盖分类过滤。
func (s *service) ListOperationRecords(ctx context.Context, input OperationRecordFilter) (model.OperationRecordPage, error) {
	page := normalizePage(input.Page)
	pageSize := normalizePageSize(input.PageSize)
	result := model.OperationRecordPage{Page: page, PageSize: pageSize, StorageStatus: "unavailable"}
	if s.repo == nil {
		return result, nil
	}
	status := input.Status
	if status < 0 || status > 999 {
		return result, ErrInvalidInput
	}
	statusClass, err := normalizeOperationStatusClass(input.StatusClass)
	if err != nil {
		return result, err
	}
	if status > 0 {
		statusClass = ""
	}
	records, total, err := s.repo.ListOperationRecords(ctx, model.OperationRecordFilter{
		Method:      strings.ToUpper(strings.TrimSpace(input.Method)),
		Page:        page,
		PageSize:    pageSize,
		Path:        strings.TrimSpace(input.Path),
		Status:      status,
		StatusClass: statusClass,
	})
	if err != nil {
		if isStorageUnavailable(err) {
			return result, nil
		}
		return result, err
	}
	result.Items = records
	result.StorageStatus = "persisted"
	result.Total = total
	return result, nil
}

// normalizeOperationStatusClass 归一化操作记录状态分类过滤值。
func normalizeOperationStatusClass(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "all":
		return "", nil
	case "4xx", "client-error", "client_error":
		return "4xx", nil
	case "5xx", "server-error", "server_error":
		return "5xx", nil
	case "error", "errors":
		return "error", nil
	default:
		return "", ErrInvalidInput
	}
}

// DeleteOperationRecords 批量删除操作记录。
// ids 必须显式给出，防止调用方误触发无条件删除。
func (s *service) DeleteOperationRecords(ctx context.Context, ids []int64) error {
	if s.repo == nil {
		return ErrStorageUnavailable
	}
	if len(ids) == 0 {
		return ErrInvalidInput
	}
	normalized := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return ErrInvalidInput
		}
		normalized = append(normalized, id)
	}
	if err := s.repo.DeleteOperationRecords(ctx, normalized); err != nil {
		if isStorageUnavailable(err) {
			return ErrStorageUnavailable
		}
		return err
	}
	return nil
}

// RegisterAPIs 接收路由层发现的 API 清单并保存到服务内存态。
// 输入会被复制、规范化并排序，避免调用方后续修改切片影响系统模块展示结果。
func (s *service) RegisterAPIs(entries []model.APIEntry) {
	cloned := append([]model.APIEntry(nil), entries...)
	for i := range cloned {
		cloned[i].Access = normalizeAPIAccess(cloned[i])
		cloned[i].Scope = normalizePermissionScope(cloned[i].Scope)
		if cloned[i].Scope == "" {
			cloned[i].Scope = permissionScopeTenant
		}
	}
	sortAPIs(cloned)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.apis = cloned
}

// ListAPIs 返回按分组组织的 API 清单。
// 基础数据来自内存中的路由注册表，随后补充同步状态和权限注册状态；
// 该方法不写入仓储，适合用于管理端只读展示。
func (s *service) ListAPIs(ctx context.Context) ([]model.APIGroup, error) {
	s.mu.RLock()
	entries := append([]model.APIEntry(nil), s.apis...)
	s.mu.RUnlock()
	sortAPIs(entries)
	if err := s.applySyncMetadata(ctx, entries); err != nil {
		return nil, err
	}
	if err := s.applyPermissionMetadata(ctx, entries); err != nil {
		return nil, err
	}

	return groupAPIs(entries), nil
}

// SyncAPIs 将当前路由注册表同步到持久化 API 记录。
// 已存在的 method/path 会更新为 active，新增路由会创建记录，旧记录缺席时标记为 stale；
// 仓储不可用时返回内存结果，便于调用方区分“未持久化”和真正的同步失败。
func (s *service) SyncAPIs(ctx context.Context) (model.APISyncResult, error) {
	s.mu.RLock()
	entries := append([]model.APIEntry(nil), s.apis...)
	s.mu.RUnlock()
	sortAPIs(entries)

	now := s.now()
	result := model.APISyncResult{
		Groups:        groupAPIs(entries),
		StorageStatus: "memory",
		SyncedAt:      now,
		Total:         len(entries),
	}
	if s.repo == nil {
		return result, nil
	}

	existing, err := s.repo.ListAPIs(ctx)
	if err != nil {
		if isStorageUnavailable(err) {
			result.StorageStatus = "unavailable"
			return result, nil
		}
		return result, err
	}

	existingByKey := make(map[string]*model.APIRecord, len(existing))
	seen := make(map[string]struct{}, len(entries))
	for i := range existing {
		existingByKey[apiKey(existing[i].Method, existing[i].Path)] = &existing[i]
	}

	for _, entry := range entries {
		key := apiKey(entry.Method, entry.Path)
		seen[key] = struct{}{}
		record, ok := existingByKey[key]
		if !ok {
			record = &model.APIRecord{
				ID:        s.ids.NextID(),
				CreatedAt: now,
			}
			result.Created++
		} else {
			result.Updated++
		}
		applyEntryToRecord(record, entry, now)
		if ok {
			if err := s.repo.SaveAPI(ctx, record); err != nil {
				return result, err
			}
			continue
		}
		if err := s.repo.CreateAPI(ctx, record); err != nil {
			return result, err
		}
	}

	for _, record := range existing {
		if _, ok := seen[apiKey(record.Method, record.Path)]; ok || record.Status == model.APIStatusStale {
			continue
		}
		record.Status = model.APIStatusStale
		record.UpdatedAt = now
		if err := s.repo.SaveAPI(ctx, &record); err != nil {
			return result, err
		}
		result.Stale++
	}

	synced := make(map[string]model.APIRecord, len(entries))
	for _, entry := range entries {
		synced[apiKey(entry.Method, entry.Path)] = model.APIRecord{Status: model.APIStatusActive, SyncedAt: now}
	}
	annotated := append([]model.APIEntry(nil), entries...)
	applySyncMetadataFromRecords(annotated, synced)
	if err := s.applyPermissionMetadata(ctx, annotated); err != nil {
		return result, err
	}
	result.Groups = groupAPIs(annotated)
	result.Persisted = true
	result.StorageStatus = "persisted"
	return result, nil
}

// SyncPermissions 根据 API 声明的权限编码补齐权限表。
// 该方法只创建缺失权限，不覆盖已有权限名称或描述，避免破坏人工维护的权限元数据。
func (s *service) SyncPermissions(ctx context.Context) (model.PermissionSyncResult, error) {
	s.mu.RLock()
	entries := append([]model.APIEntry(nil), s.apis...)
	s.mu.RUnlock()

	now := s.now()
	specs := permissionSpecsFromAPIs(entries)
	result := model.PermissionSyncResult{
		Items:         make([]model.PermissionSyncItem, 0, len(specs)),
		StorageStatus: "unavailable",
		SyncedAt:      now,
		Total:         len(specs),
	}
	if s.permissionStore == nil {
		return result, nil
	}

	existing, err := s.permissionStore.ListPermissions(ctx)
	if err != nil {
		return result, err
	}
	existingByCode := make(map[string]model.PermissionEntry, len(existing))
	for _, permission := range existing {
		existingByCode[permissionKey(permission.ProductCode, permission.Scope, permission.Code)] = permission
	}

	for _, spec := range specs {
		item := model.PermissionSyncItem{
			Code:        spec.Code,
			Description: spec.Description,
			Name:        spec.Name,
			ProductCode: spec.ProductCode,
			Scope:       spec.Scope,
		}
		if _, ok := existingByCode[permissionKey(spec.ProductCode, spec.Scope, spec.Code)]; ok {
			item.Exists = true
			result.Skipped++
			result.Items = append(result.Items, item)
			continue
		}
		if err := s.permissionStore.CreatePermission(ctx, spec); err != nil {
			return result, err
		}
		item.Created = true
		result.Created++
		result.Items = append(result.Items, item)
	}
	result.Persisted = true
	result.StorageStatus = "persisted"
	return result, nil
}

func sortMenus(groups []model.MenuGroup) {
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].Order == groups[j].Order {
			return groups[i].Code < groups[j].Code
		}
		return groups[i].Order < groups[j].Order
	})
	for i := range groups {
		sort.SliceStable(groups[i].Items, func(a, b int) bool {
			if groups[i].Items[a].Order == groups[i].Items[b].Order {
				return groups[i].Items[a].Code < groups[i].Items[b].Code
			}
			return groups[i].Items[a].Order < groups[i].Items[b].Order
		})
	}
}

// sortAPIs 为 API 展示和同步结果提供稳定顺序。
// 先按分组和路径排序，再用 Order 与 Method 打破同一路径下的顺序差异。
func sortAPIs(entries []model.APIEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Group != entries[j].Group {
			return entries[i].Group < entries[j].Group
		}
		if entries[i].Path != entries[j].Path {
			return entries[i].Path < entries[j].Path
		}
		if entries[i].Order == entries[j].Order {
			return entries[i].Method < entries[j].Method
		}
		return entries[i].Order < entries[j].Order
	})
}

// groupAPIs 将扁平 API 清单转换为管理端需要的分组结构。
// 空分组会归入 other，且每个 entry 的访问级别会在分组时再次归一化。
func groupAPIs(entries []model.APIEntry) []model.APIGroup {
	byGroup := make(map[string][]model.APIEntry)
	for _, entry := range entries {
		group := normalizeGroup(entry.Group)
		if group == "" {
			group = "other"
		}
		entry.Group = group
		entry.Access = normalizeAPIAccess(entry)
		byGroup[group] = append(byGroup[group], entry)
	}

	groups := make([]model.APIGroup, 0, len(byGroup))
	for group, items := range byGroup {
		groups = append(groups, model.APIGroup{
			Code:  group,
			Label: apiGroupLabel(group),
			Count: len(items),
			Items: items,
		})
	}
	sort.SliceStable(groups, func(i, j int) bool {
		if apiGroupOrder(groups[i].Code) == apiGroupOrder(groups[j].Code) {
			return groups[i].Code < groups[j].Code
		}
		return apiGroupOrder(groups[i].Code) < apiGroupOrder(groups[j].Code)
	})
	return groups
}

// normalizeAPIAccess 统一 API 的访问级别。
// 未显式声明 access 但配置了权限编码时视为 permission，否则默认需要登录访问。
func normalizeAPIAccess(entry model.APIEntry) string {
	switch strings.ToLower(strings.TrimSpace(entry.Access)) {
	case model.APIAccessPublic:
		return model.APIAccessPublic
	case model.APIAccessAuthenticated:
		return model.APIAccessAuthenticated
	case model.APIAccessPermission:
		return model.APIAccessPermission
	}
	if normalizePermissionCode(entry.Permission) != "" {
		return model.APIAccessPermission
	}
	return model.APIAccessAuthenticated
}

// applySyncMetadata 用持久化 API 记录标注内存路由的同步状态。
// 仓储不可用时保持 entries 原样返回，避免只读列表因为状态补充失败而不可用。
func (s *service) applySyncMetadata(ctx context.Context, entries []model.APIEntry) error {
	if s.repo == nil {
		return nil
	}
	records, err := s.repo.ListAPIs(ctx)
	if err != nil {
		if isStorageUnavailable(err) {
			return nil
		}
		return err
	}
	byKey := make(map[string]model.APIRecord, len(records))
	for _, record := range records {
		byKey[apiKey(record.Method, record.Path)] = record
	}
	applySyncMetadataFromRecords(entries, byKey)
	return nil
}

// applyPermissionMetadata 标注 API 声明的权限是否已存在于权限表。
// 该状态只用于展示和引导同步，不会改变 API 或权限记录。
func (s *service) applyPermissionMetadata(ctx context.Context, entries []model.APIEntry) error {
	if s.permissionStore == nil {
		return nil
	}
	permissions, err := s.permissionStore.ListPermissions(ctx)
	if err != nil {
		return err
	}
	registered := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		if code := normalizePermissionCode(permission.Code); code != "" {
			registered[permissionKey(permission.ProductCode, permission.Scope, code)] = struct{}{}
		}
	}
	for i := range entries {
		code := normalizePermissionCode(entries[i].Permission)
		if code == "" {
			continue
		}
		_, entries[i].PermissionRegistered = registered[permissionKey(entries[i].ProductCode, entries[i].Scope, code)]
	}
	return nil
}

// applySyncMetadataFromRecords 根据 method/path 匹配 active API 记录。
// 只有 active 记录才会标记为已同步，stale 记录需要继续暴露为未同步状态。
func applySyncMetadataFromRecords(entries []model.APIEntry, records map[string]model.APIRecord) {
	for i := range entries {
		record, ok := records[apiKey(entries[i].Method, entries[i].Path)]
		if !ok || record.Status != model.APIStatusActive {
			continue
		}
		syncedAt := record.SyncedAt
		entries[i].Synced = true
		entries[i].SyncedAt = &syncedAt
		if strings.TrimSpace(entries[i].ProductCode) == "" {
			entries[i].ProductCode = record.ProductCode
		}
		if strings.TrimSpace(entries[i].Scope) == "" {
			entries[i].Scope = record.Scope
		}
	}
}

// applyEntryToRecord 将内存路由条目投影为持久化 API 记录。
// CreatedAt 只在新记录缺失时补齐，其他字段反映当前路由状态并刷新同步时间。
func applyEntryToRecord(record *model.APIRecord, entry model.APIEntry, now time.Time) {
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.Code = entry.Code
	record.Group = normalizeGroup(entry.Group)
	record.Method = strings.ToUpper(strings.TrimSpace(entry.Method))
	record.Path = entry.Path
	record.Description = entry.Description
	record.Permission = entry.Permission
	record.ProductCode = strings.ToLower(strings.TrimSpace(entry.ProductCode))
	record.Scope = normalizePermissionScope(entry.Scope)
	record.Status = model.APIStatusActive
	record.Source = "router"
	record.SyncedAt = now
	record.UpdatedAt = now
}

// permissionSpecsFromAPIs 从 API 清单派生需要补齐的权限定义。
// 权限编码会去重并按 code 排序，保证同步结果稳定；非法编码会被跳过。
func permissionSpecsFromAPIs(entries []model.APIEntry) []model.PermissionEntry {
	byCode := make(map[string]model.PermissionEntry)
	for _, entry := range entries {
		code := normalizePermissionCode(entry.Permission)
		if code == "" || !validPermissionCode(code) {
			continue
		}
		productCode := strings.ToLower(strings.TrimSpace(entry.ProductCode))
		scope := normalizePermissionScope(entry.Scope)
		if _, ok := byCode[permissionKey(productCode, scope, code)]; ok {
			continue
		}
		byCode[permissionKey(productCode, scope, code)] = model.PermissionEntry{
			Code:        code,
			ProductCode: productCode,
			Scope:       scope,
			Name:        permissionName(code),
			Description: permissionDescription(entry),
		}
	}
	specs := make([]model.PermissionEntry, 0, len(byCode))
	for _, spec := range byCode {
		specs = append(specs, spec)
	}
	sort.SliceStable(specs, func(i, j int) bool {
		left := permissionKey(specs[i].ProductCode, specs[i].Scope, specs[i].Code)
		right := permissionKey(specs[j].ProductCode, specs[j].Scope, specs[j].Code)
		return left < right
	})
	return specs
}

func normalizePermissionCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizePermissionScope(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case permissionScopePlatform, permissionScopeTenant, permissionScopeProduct:
		return value
	default:
		return ""
	}
}

func permissionKey(productCode string, scope string, code string) string {
	return strings.ToLower(strings.TrimSpace(productCode)) + "\x00" + normalizePermissionScope(scope) + "\x00" + normalizePermissionCode(code)
}

// validPermissionCode 要求权限编码至少包含 object:action 两段。
// 这里保持宽松校验，只拒绝缺少冒号或任一侧为空的编码。
func validPermissionCode(code string) bool {
	obj, act, ok := strings.Cut(code, ":")
	return ok && strings.TrimSpace(obj) != "" && strings.TrimSpace(act) != ""
}

// normalizeDictionaryCode 统一字典编码格式。
func normalizeDictionaryCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// validDictionaryCode 限制字典编码为可用于查询、导入导出和 URL 的安全字符集合。
func validDictionaryCode(code string) bool {
	if code == "" {
		return false
	}
	for _, char := range code {
		switch {
		case char >= 'a' && char <= 'z':
		case char >= '0' && char <= '9':
		case char == '_' || char == '-' || char == ':' || char == '.':
		default:
			return false
		}
	}
	return true
}

// normalizeDictionaryStatus 归一化字典和字典项状态。
// 空状态默认 active，便于创建和导入时减少必填字段。
func normalizeDictionaryStatus(value string) (string, error) {
	status := strings.ToLower(strings.TrimSpace(value))
	if status == "" {
		return model.DictionaryStatusActive, nil
	}
	switch status {
	case model.DictionaryStatusActive, model.DictionaryStatusDisabled:
		return status, nil
	default:
		return "", ErrInvalidInput
	}
}

// mapDictionaryLookupError 将字典仓储错误转换为服务层稳定错误。
func mapDictionaryLookupError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return ErrNotFound
	case isStorageUnavailable(err):
		return ErrStorageUnavailable
	default:
		return err
	}
}

// mapParameterLookupError 将参数仓储错误转换为服务层稳定错误。
func mapParameterLookupError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return ErrNotFound
	case isStorageUnavailable(err):
		return ErrStorageUnavailable
	default:
		return err
	}
}

// normalizePage 将非法页码回退到第一页。
func normalizePage(value int) int {
	if value < 1 {
		return 1
	}
	return value
}

// normalizePageSize 为列表接口设置统一分页上限。
// 最大值限制为 100，避免管理端误请求导致仓储和序列化压力过大。
func normalizePageSize(value int) int {
	if value < 1 {
		return 10
	}
	if value > 100 {
		return 100
	}
	return value
}

// trimOperationPayload 截断审计记录中的大文本字段。
// 保留截断标记便于排查时知道原始内容并未完整入库。
func trimOperationPayload(value string) string {
	value = strings.TrimSpace(value)
	const maxOperationPayloadBytes = 8192
	if len(value) <= maxOperationPayloadBytes {
		return value
	}
	return value[:maxOperationPayloadBytes] + "...(truncated)"
}

// permissionName 从权限编码生成默认展示名。
func permissionName(code string) string {
	obj, act, ok := strings.Cut(code, ":")
	if !ok {
		return code
	}
	return strings.ToUpper(obj[:1]) + obj[1:] + " " + act
}

// permissionDescription 为从 API 派生的权限生成默认描述。
func permissionDescription(entry model.APIEntry) string {
	if entry.Description != "" {
		return entry.Description
	}
	return entry.Method + " " + entry.Path
}

// apiKey 使用规范化 method 和 path 构造 API 唯一键。
func apiKey(method string, path string) string {
	return strings.ToUpper(strings.TrimSpace(method)) + " " + strings.TrimSpace(path)
}

// now 返回服务层统一使用的 UTC 时间。
// 测试可通过 Config.Now 注入固定时间，避免断言依赖真实时钟。
func (s *service) now() time.Time {
	if s.cfg.Now != nil {
		return s.cfg.Now().UTC()
	}
	return time.Now().UTC()
}

// normalizeGroup 统一菜单和 API 分组编码。
func normalizeGroup(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// apiGroupLabel 返回 API 分组的管理端展示名称。
func apiGroupLabel(code string) string {
	switch code {
	case "auth":
		return "璁よ瘉"
	case "orgs":
		return "缁勭粐/IAM"
	case "plugins":
		return "鎻掍欢"
	case "system":
		return "绯荤粺"
	default:
		return code
	}
}

// apiGroupOrder 定义 API 分组展示顺序。
func apiGroupOrder(code string) int {
	switch code {
	case "auth":
		return 10
	case "orgs":
		return 20
	case "system":
		return 30
	case "plugins":
		return 40
	default:
		return 100
	}
}

// cloneGroups 深拷贝菜单组切片。
func cloneGroups(src []model.MenuGroup) []model.MenuGroup {
	out := make([]model.MenuGroup, 0, len(src))
	for _, group := range src {
		out = append(out, cloneGroup(group))
	}
	return out
}

// cloneGroup 深拷贝单个菜单组中的菜单项切片。
func cloneGroup(src model.MenuGroup) model.MenuGroup {
	dst := src
	dst.Items = append([]model.MenuItem(nil), src.Items...)
	return dst
}

// cloneConfigSnapshot 深拷贝配置快照中的 section、group 和 item 切片。
func cloneConfigSnapshot(src model.ConfigSnapshot) model.ConfigSnapshot {
	dst := model.ConfigSnapshot{
		Sections: make([]model.ConfigSection, 0, len(src.Sections)),
	}
	for _, section := range src.Sections {
		cloned := section
		cloned.Items = append([]model.ConfigItem(nil), section.Items...)
		cloned.Groups = make([]model.ConfigGroup, 0, len(section.Groups))
		for _, group := range section.Groups {
			clonedGroup := group
			clonedGroup.Items = append([]model.ConfigItem(nil), group.Items...)
			cloned.Groups = append(cloned.Groups, clonedGroup)
		}
		dst.Sections = append(dst.Sections, cloned)
	}
	return dst
}

func stringConfigValue(snapshot model.ConfigSnapshot, key string) string {
	for _, section := range snapshot.Sections {
		for _, item := range section.Items {
			if item.Key == key {
				if value, ok := item.Value.(string); ok {
					return strings.TrimSpace(value)
				}
				return strings.TrimSpace(toString(item.Value))
			}
		}
		for _, group := range section.Groups {
			for _, item := range group.Items {
				if item.Key == key {
					if value, ok := item.Value.(string); ok {
						return strings.TrimSpace(value)
					}
					return strings.TrimSpace(toString(item.Value))
				}
			}
		}
	}
	return ""
}

func boolConfigValue(snapshot model.ConfigSnapshot, key string) bool {
	for _, section := range snapshot.Sections {
		for _, item := range section.Items {
			if item.Key == key {
				value, _ := item.Value.(bool)
				return value
			}
		}
		for _, group := range section.Groups {
			for _, item := range group.Items {
				if item.Key == key {
					value, _ := item.Value.(bool)
					return value
				}
			}
		}
	}
	return false
}

func stringSliceConfigValue(snapshot model.ConfigSnapshot, key string) []string {
	for _, section := range snapshot.Sections {
		if value, ok := stringSliceConfigValueFromItems(section.Items, key); ok {
			return value
		}
		for _, group := range section.Groups {
			if value, ok := stringSliceConfigValueFromItems(group.Items, key); ok {
				return value
			}
		}
	}
	return nil
}

func stringSliceConfigValueFromItems(items []model.ConfigItem, key string) ([]string, bool) {
	for _, item := range items {
		if item.Key != key {
			continue
		}
		switch value := item.Value.(type) {
		case []string:
			return append([]string(nil), value...), true
		case []any:
			out := make([]string, 0, len(value))
			for _, item := range value {
				text := strings.TrimSpace(toString(item))
				if text != "" {
					out = append(out, text)
				}
			}
			return out, true
		case string:
			parts := strings.Split(value, ",")
			out := make([]string, 0, len(parts))
			for _, part := range parts {
				text := strings.TrimSpace(part)
				if text != "" {
					out = append(out, text)
				}
			}
			return out, true
		default:
			return nil, true
		}
	}
	return nil, false
}

func toString(value any) string {
	switch value := value.(type) {
	case string:
		return value
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	case bool:
		return strconv.FormatBool(value)
	default:
		return ""
	}
}

// buildInfo 从 Go build metadata 中提取模块、版本和构建设置。
// debug.ReadBuildInfo 不可用时仍返回 Go 版本，保证服务器信息接口可降级。
func buildInfo() model.ServerBuildInfo {
	out := model.ServerBuildInfo{
		GoVersion: runtime.Version(),
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return out
	}
	out.GoVersion = info.GoVersion
	out.Module = info.Main.Path
	out.Path = info.Path
	out.Version = info.Main.Version
	out.Settings = make([]model.ServerBuildSetting, 0, len(info.Settings))
	for _, setting := range info.Settings {
		key := strings.TrimSpace(setting.Key)
		if key == "" {
			continue
		}
		out.Settings = append(out.Settings, model.ServerBuildSetting{
			Key:   key,
			Value: setting.Value,
		})
	}
	sort.SliceStable(out.Settings, func(i, j int) bool {
		return out.Settings[i].Key < out.Settings[j].Key
	})
	return out
}

// bytesToMB 将字节数按二进制 MB 向下取整。
func bytesToMB(value uint64) uint64 {
	const bytesPerMB = 1024 * 1024
	return value / bytesPerMB
}

// mapServerCPU 将采集器 CPU 指标转换为接口模型，并复制可变切片。
func mapServerCPU(src CPUInfo) model.ServerCPUInfo {
	return model.ServerCPUInfo{
		Cores:   src.Cores,
		Percent: append([]float64(nil), src.Percent...),
	}
}

// mapServerRAM 将采集器内存指标转换为接口模型。
func mapServerRAM(src RAMInfo) model.ServerRAMInfo {
	return model.ServerRAMInfo{
		TotalMB:     src.TotalMB,
		UsedMB:      src.UsedMB,
		UsedPercent: src.UsedPercent,
	}
}

// mapServerDisks 将采集器磁盘指标转换为接口模型。
func mapServerDisks(src []DiskInfo) []model.ServerDiskInfo {
	out := make([]model.ServerDiskInfo, 0, len(src))
	for _, item := range src {
		out = append(out, model.ServerDiskInfo{
			FSType:      item.FSType,
			MountPoint:  item.MountPoint,
			TotalGB:     item.TotalGB,
			TotalMB:     item.TotalMB,
			UsedGB:      item.UsedGB,
			UsedMB:      item.UsedMB,
			UsedPercent: item.UsedPercent,
		})
	}
	return out
}

func mapServerDiskIOSamples(src []DiskIOSample) []model.ServerDiskIOSample {
	out := make([]model.ServerDiskIOSample, 0, len(src))
	for _, item := range src {
		out = append(out, model.ServerDiskIOSample{
			Name:              item.Name,
			ReadMBPerSecond:   item.ReadMBPerSecond,
			WriteMBPerSecond:  item.WriteMBPerSecond,
			ReadOpsPerSecond:  item.ReadOpsPerSecond,
			WriteOpsPerSecond: item.WriteOpsPerSecond,
			IOLatencyMs:       item.IOLatencyMs,
		})
	}
	return out
}

// formatRuntimeDuration 以秒为粒度展示服务运行时长。
func formatRuntimeDuration(value time.Duration) string {
	if value <= 0 {
		return "0s"
	}
	return value.Truncate(time.Second).String()
}

// baseMenus 是系统模块内置菜单目录。
// 调用方必须通过 ListMenus 获取拷贝，不能直接修改该全局定义。
var baseMenus = []model.MenuGroup{
	{
		Code:     "workspace",
		LabelKey: "system.menus.groups.workspace.label",
		Order:    10,
		Items: []model.MenuItem{
			{Code: "dashboard", LabelKey: "system.menus.items.dashboard.label", Icon: "layout-dashboard", Path: "/", Mobile: true, Order: 10},
			{Code: "organizations", LabelKey: "system.menus.items.organizations.label", Icon: "building-2", Path: "/organizations", Permission: "org:read", Scope: permissionScopePlatform, Mobile: true, Order: 20},
			{Code: "users", LabelKey: "system.menus.items.users.label", Icon: "users", Path: "/users", Permission: "user:read", Scope: permissionScopeTenant, Mobile: true, Order: 30},
			{Code: "roles", LabelKey: "system.menus.items.roles.label", Icon: "shield-check", Path: "/roles", Permission: "role:read", Scope: permissionScopeTenant, Mobile: true, Order: 40},
		},
	},
	{
		Code:     "security",
		LabelKey: "system.menus.groups.security.label",
		Order:    20,
		Items: []model.MenuItem{
			{Code: "sessions", LabelKey: "system.menus.items.sessions.label", Icon: "monitor-check", Path: "/sessions", Permission: "session:read", Scope: permissionScopeTenant, Order: 10},
			{Code: "api-tokens", LabelKey: "system.menus.items.apiTokens.label", Icon: "key-round", Path: "/api-tokens", Permission: "api_token:read", Scope: permissionScopeTenant, Order: 20},
			{Code: "login-logs", LabelKey: "system.menus.items.loginLogs.label", Icon: "log-in", Path: "/login-logs", Permission: "audit:read", Scope: permissionScopeTenant, Order: 30},
			{Code: "audit-logs", LabelKey: "system.menus.items.auditLogs.label", Icon: "scroll-text", Path: "/audit-logs", Permission: "audit:read", Scope: permissionScopeTenant, Order: 40},
			{Code: "error-logs", LabelKey: "system.menus.items.errorLogs.label", Icon: "bug", Path: "/error-logs", Permission: "operation:read", Scope: permissionScopePlatform, Order: 50},
			{Code: "traffic-hijack", LabelKey: "system.menus.items.trafficHijack.label", Icon: "shield-alert", Path: "/traffic-hijack", Permission: "traffic_hijack:read", Scope: permissionScopePlatform, Order: 60},
			{Code: "security", LabelKey: "system.menus.items.security.label", Icon: "lock-keyhole", Path: "/security", Order: 70},
		},
	},
	{
		Code:     "system",
		LabelKey: "system.menus.groups.system.label",
		Order:    30,
		Items: []model.MenuItem{
			{Code: "menus", LabelKey: "system.menus.items.menus.label", Icon: "panel-left", Path: "/menus", Permission: "permission:read", Scope: permissionScopePlatform, Order: 10},
			{Code: "apis", LabelKey: "system.menus.items.apis.label", Icon: "code-2", Path: "/apis", Permission: "permission:read", Scope: permissionScopePlatform, Order: 20},
			{Code: "dictionaries", LabelKey: "system.menus.items.dictionaries.label", Icon: "book-open", Path: "/dictionaries", Permission: "dictionary:read", Scope: permissionScopePlatform, Order: 30},
			{Code: "operation-records", LabelKey: "system.menus.items.operationRecords.label", Icon: "history", Path: "/operation-records", Permission: "operation:read", Scope: permissionScopePlatform, Order: 40},
			{Code: "parameters", LabelKey: "system.menus.items.parameters.label", Icon: "compass", Path: "/parameters", Permission: "parameter:read", Scope: permissionScopePlatform, Order: 50},
			{Code: "versions", LabelKey: "system.menus.items.versions.label", Icon: "package-check", Path: "/versions", Permission: "version:read", Scope: permissionScopePlatform, Order: 60},
			{Code: "media-resumable", LabelKey: "system.menus.items.mediaResumable.label", Icon: "upload-cloud", Path: "/media/resumable", Permission: "media:upload", Scope: permissionScopePlatform, Order: 70},
			{Code: "media", LabelKey: "system.menus.items.media.label", Icon: "image-up", Path: "/media", Permission: "media:read", Scope: permissionScopePlatform, Order: 80},
			{Code: "system-config", LabelKey: "system.menus.items.system.label", Icon: "settings", Path: "/system", Permission: "config:read", Scope: permissionScopePlatform, Order: 90},
		},
	},
}
