package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rei0721/go-scaffold/internal/modules/system/model"
	systemservice "github.com/rei0721/go-scaffold/internal/modules/system/service"
	database "github.com/rei0721/go-scaffold/internal/ports"
)

type Repository = systemservice.Repository

type repository struct {
	db database.Executor
}

func New(db database.Executor) Repository {
	return &repository{db: storageAwareExecutor{inner: db}}
}

type storageAwareExecutor struct {
	inner database.Executor
}

func (e storageAwareExecutor) Create(ctx context.Context, value any) error {
	return mapStorageError(e.inner.Create(ctx, value))
}

func (e storageAwareExecutor) Save(ctx context.Context, value any) error {
	return mapStorageError(e.inner.Save(ctx, value))
}

func (e storageAwareExecutor) First(ctx context.Context, dest any, opts ...database.QueryOption) error {
	return mapStorageError(e.inner.First(ctx, dest, opts...))
}

func (e storageAwareExecutor) Find(ctx context.Context, dest any, opts ...database.QueryOption) error {
	return mapStorageError(e.inner.Find(ctx, dest, opts...))
}

func (e storageAwareExecutor) Update(ctx context.Context, model any, values map[string]any, opts ...database.QueryOption) (database.Result, error) {
	result, err := e.inner.Update(ctx, model, values, opts...)
	return result, mapStorageError(err)
}

func (e storageAwareExecutor) Delete(ctx context.Context, model any, opts ...database.QueryOption) (database.Result, error) {
	result, err := e.inner.Delete(ctx, model, opts...)
	return result, mapStorageError(err)
}

func (e storageAwareExecutor) Exec(ctx context.Context, sql string, args ...any) (database.Result, error) {
	result, err := e.inner.Exec(ctx, sql, args...)
	return result, mapStorageError(err)
}

func (e storageAwareExecutor) Raw(ctx context.Context, dest any, sql string, args ...any) (database.Result, error) {
	result, err := e.inner.Raw(ctx, dest, sql, args...)
	return result, mapStorageError(err)
}

func (e storageAwareExecutor) Count(ctx context.Context, model any, opts ...database.QueryOption) (int64, error) {
	count, err := e.inner.Count(ctx, model, opts...)
	return count, mapStorageError(err)
}

func (e storageAwareExecutor) HasTable(ctx context.Context, model any) (bool, error) {
	ok, err := e.inner.HasTable(ctx, model)
	return ok, mapStorageError(err)
}

func (r *repository) CreateAPI(ctx context.Context, api *model.APIRecord) error {
	return r.db.Create(ctx, api)
}

func (r *repository) CreateDictionary(ctx context.Context, dictionary *model.Dictionary) error {
	return r.db.Create(ctx, dictionary)
}

func (r *repository) CreateDictionaryItem(ctx context.Context, item *model.DictionaryItem) error {
	return r.db.Create(ctx, item)
}

func (r *repository) CreateMediaAsset(ctx context.Context, asset *model.MediaAsset) error {
	return r.db.Create(ctx, asset)
}

func (r *repository) CreateMediaCategory(ctx context.Context, category *model.MediaCategory) error {
	return r.db.Create(ctx, category)
}

func (r *repository) CreateMediaUploadChunk(ctx context.Context, chunk *model.MediaUploadChunk) error {
	return r.db.Create(ctx, chunk)
}

func (r *repository) CreateMediaUploadSession(ctx context.Context, session *model.MediaUploadSession) error {
	return r.db.Create(ctx, session)
}

func (r *repository) CreateOperationRecord(ctx context.Context, record *model.OperationRecord) error {
	return r.db.Create(ctx, record)
}

func (r *repository) CreateParameter(ctx context.Context, parameter *model.Parameter) error {
	return r.db.Create(ctx, parameter)
}

func (r *repository) CreateTrafficHijackEvent(ctx context.Context, event *model.TrafficHijackEvent) error {
	return r.db.Create(ctx, event)
}

func (r *repository) CreateTrafficProbeResult(ctx context.Context, result *model.TrafficProbeResult) error {
	return r.db.Create(ctx, result)
}

func (r *repository) CreateTrafficProbeTarget(ctx context.Context, target *model.TrafficProbeTarget) error {
	return r.db.Create(ctx, target)
}

func (r *repository) CreateVersion(ctx context.Context, version *model.Version) error {
	return r.db.Create(ctx, version)
}

func (r *repository) DeleteDictionary(ctx context.Context, id int64, deletedAt time.Time) error {
	_, err := r.db.Update(ctx, &model.Dictionary{}, map[string]any{
		"deleted_at": deletedAt,
		"updated_at": deletedAt,
	}, database.Where("id = ?", id), alive())
	if err != nil {
		return err
	}
	_, err = r.db.Update(ctx, &model.DictionaryItem{}, map[string]any{
		"deleted_at": deletedAt,
		"updated_at": deletedAt,
	}, database.Where("dictionary_id = ?", id), alive())
	return err
}

func (r *repository) DeleteDictionaryItem(ctx context.Context, id int64, deletedAt time.Time) error {
	_, err := r.db.Update(ctx, &model.DictionaryItem{}, map[string]any{
		"deleted_at": deletedAt,
		"updated_at": deletedAt,
	}, database.Where("id = ?", id), alive())
	return err
}

func (r *repository) DeleteMediaAsset(ctx context.Context, id int64, deletedAt time.Time) error {
	_, err := r.db.Update(ctx, &model.MediaAsset{}, map[string]any{
		"deleted_at": deletedAt,
		"updated_at": deletedAt,
	}, database.Where("id = ?", id), alive())
	return err
}

func (r *repository) DeleteMediaCategory(ctx context.Context, id int64, deletedAt time.Time) error {
	_, err := r.db.Update(ctx, &model.MediaCategory{}, map[string]any{
		"deleted_at": deletedAt,
		"updated_at": deletedAt,
	}, database.Where("id = ?", id), alive())
	return err
}

func (r *repository) DeleteMediaUploadChunks(ctx context.Context, sessionID int64) error {
	_, err := r.db.Delete(ctx, &model.MediaUploadChunk{}, database.Where("session_id = ?", sessionID))
	return err
}

func (r *repository) DeleteOperationRecords(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.db.Delete(ctx, &model.OperationRecord{}, database.Where("id IN ?", ids))
	return err
}

func (r *repository) DeleteParameter(ctx context.Context, id int64, deletedAt time.Time) error {
	_, err := r.db.Update(ctx, &model.Parameter{}, map[string]any{
		"deleted_at": deletedAt,
		"updated_at": deletedAt,
	}, database.Where("id = ?", id), alive())
	return err
}

func (r *repository) DeleteParameters(ctx context.Context, ids []int64, deletedAt time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.db.Update(ctx, &model.Parameter{}, map[string]any{
		"deleted_at": deletedAt,
		"updated_at": deletedAt,
	}, database.Where("id IN ?", ids), alive())
	return err
}

func (r *repository) DeleteOldTrafficProbeResults(ctx context.Context, targetID int64, keep int) error {
	if targetID <= 0 || keep <= 0 {
		return nil
	}
	var boundary []model.TrafficProbeResult
	if err := r.db.Find(ctx, &boundary,
		database.Where("target_id = ?", targetID),
		database.Order("created_at DESC, id DESC"),
		database.Limit(1),
		database.Offset(keep-1),
	); err != nil {
		return err
	}
	if len(boundary) == 0 {
		return nil
	}
	item := boundary[0]
	_, err := r.db.Delete(ctx, &model.TrafficProbeResult{},
		database.Where("target_id = ?", targetID),
		database.Where("(created_at < ? OR (created_at = ? AND id < ?))", item.CreatedAt, item.CreatedAt, item.ID),
	)
	return err
}

func (r *repository) DeleteTrafficProbeTarget(ctx context.Context, id int64, deletedAt time.Time) error {
	_, err := r.db.Update(ctx, &model.TrafficProbeTarget{}, map[string]any{
		"deleted_at": deletedAt,
		"updated_at": deletedAt,
		"enabled":    false,
	}, database.Where("id = ?", id), alive())
	return err
}

func (r *repository) DeleteVersion(ctx context.Context, id int64, deletedAt time.Time) error {
	_, err := r.db.Update(ctx, &model.Version{}, map[string]any{
		"deleted_at": deletedAt,
		"updated_at": deletedAt,
	}, database.Where("id = ?", id), alive())
	return err
}

func (r *repository) DeleteVersions(ctx context.Context, ids []int64, deletedAt time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.db.Update(ctx, &model.Version{}, map[string]any{
		"deleted_at": deletedAt,
		"updated_at": deletedAt,
	}, database.Where("id IN ?", ids), alive())
	return err
}

func (r *repository) FindAPI(ctx context.Context, method string, path string) (*model.APIRecord, error) {
	var api model.APIRecord
	err := r.db.First(ctx, &api,
		database.Where("http_method = ?", method),
		database.Where("path = ?", path),
	)
	if err != nil {
		return nil, err
	}
	return &api, nil
}

func (r *repository) FindDictionaryByCode(ctx context.Context, code string) (*model.Dictionary, error) {
	var dictionary model.Dictionary
	if err := r.db.First(ctx, &dictionary, database.Where("code = ?", code), alive()); err != nil {
		return nil, err
	}
	return &dictionary, nil
}

func (r *repository) FindDictionaryByID(ctx context.Context, id int64) (*model.Dictionary, error) {
	var dictionary model.Dictionary
	if err := r.db.First(ctx, &dictionary, database.Where("id = ?", id), alive()); err != nil {
		return nil, err
	}
	return &dictionary, nil
}

func (r *repository) FindDictionaryItemByID(ctx context.Context, id int64) (*model.DictionaryItem, error) {
	var item model.DictionaryItem
	if err := r.db.First(ctx, &item, database.Where("id = ?", id), alive()); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *repository) FindMediaAssetByID(ctx context.Context, id int64) (*model.MediaAsset, error) {
	var asset model.MediaAsset
	if err := r.db.First(ctx, &asset, database.Where("id = ?", id), alive()); err != nil {
		return nil, err
	}
	return &asset, nil
}

func (r *repository) FindMediaCategoryByID(ctx context.Context, id int64) (*model.MediaCategory, error) {
	var category model.MediaCategory
	if err := r.db.First(ctx, &category, database.Where("id = ?", id), alive()); err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *repository) FindMediaUploadChunk(ctx context.Context, sessionID int64, chunkIndex int) (*model.MediaUploadChunk, error) {
	var chunk model.MediaUploadChunk
	if err := r.db.First(ctx, &chunk,
		database.Where("session_id = ?", sessionID),
		database.Where("chunk_index = ?", chunkIndex),
	); err != nil {
		return nil, err
	}
	return &chunk, nil
}

func (r *repository) FindMediaUploadSessionByHash(ctx context.Context, fileHash string, fileName string, categoryID int64, uploadedBy int64) (*model.MediaUploadSession, error) {
	var session model.MediaUploadSession
	if err := r.db.First(ctx, &session,
		database.Where("file_hash = ?", fileHash),
		database.Where("file_name = ?", fileName),
		database.Where("category_id = ?", categoryID),
		database.Where("uploaded_by = ?", uploadedBy),
		alive(),
		database.Order("created_at DESC, id DESC"),
	); err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *repository) FindMediaUploadSessionByID(ctx context.Context, id int64) (*model.MediaUploadSession, error) {
	var session model.MediaUploadSession
	if err := r.db.First(ctx, &session, database.Where("id = ?", id), alive()); err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *repository) FindParameterByID(ctx context.Context, id int64) (*model.Parameter, error) {
	var parameter model.Parameter
	if err := r.db.First(ctx, &parameter, database.Where("id = ?", id), alive()); err != nil {
		return nil, err
	}
	return &parameter, nil
}

func (r *repository) FindParameterByKey(ctx context.Context, key string) (*model.Parameter, error) {
	var parameter model.Parameter
	if err := r.db.First(ctx, &parameter, database.Where("param_key = ?", key), alive()); err != nil {
		return nil, err
	}
	return &parameter, nil
}

func (r *repository) FindTrafficHijackEvent(ctx context.Context, id int64) (*model.TrafficHijackEvent, error) {
	var event model.TrafficHijackEvent
	if err := r.db.First(ctx, &event, database.Where("id = ?", id)); err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *repository) FindTrafficProbeTargetByID(ctx context.Context, id int64) (*model.TrafficProbeTarget, error) {
	var target model.TrafficProbeTarget
	if err := r.db.First(ctx, &target, database.Where("id = ?", id), alive()); err != nil {
		return nil, err
	}
	return &target, nil
}

func (r *repository) FindOpenTrafficHijackEvent(ctx context.Context, targetID int64, reason string, evidenceHash string) (*model.TrafficHijackEvent, error) {
	var event model.TrafficHijackEvent
	if err := r.db.First(ctx, &event,
		database.Where("target_id = ?", targetID),
		database.Where("reason = ?", reason),
		database.Where("evidence_hash = ?", evidenceHash),
		database.Where("state = ?", model.TrafficHijackEventStateOpen),
	); err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *repository) FindVersionByID(ctx context.Context, id int64) (*model.Version, error) {
	var version model.Version
	if err := r.db.First(ctx, &version, database.Where("id = ?", id), alive()); err != nil {
		return nil, err
	}
	return &version, nil
}

func (r *repository) ListAPIs(ctx context.Context) ([]model.APIRecord, error) {
	var apis []model.APIRecord
	err := r.db.Find(ctx, &apis, database.Order("api_group ASC, path ASC, http_method ASC"))
	return apis, err
}

func (r *repository) ListDictionaries(ctx context.Context) ([]model.Dictionary, error) {
	var dictionaries []model.Dictionary
	err := r.db.Find(ctx, &dictionaries, alive(), database.Order("code ASC"))
	return dictionaries, err
}

func (r *repository) ListDictionaryItems(ctx context.Context, dictionaryID int64) ([]model.DictionaryItem, error) {
	var items []model.DictionaryItem
	err := r.db.Find(ctx, &items,
		database.Where("dictionary_id = ?", dictionaryID),
		alive(),
		database.Order("sort_order ASC, value ASC"),
	)
	return items, err
}

func (r *repository) ListMediaCategories(ctx context.Context) ([]model.MediaCategory, error) {
	var categories []model.MediaCategory
	err := r.db.Find(ctx, &categories, alive(), database.Order("sort_order ASC, name ASC, id ASC"))
	return categories, err
}

func (r *repository) ListMediaAssets(ctx context.Context, filter model.MediaAssetFilter) ([]model.MediaAsset, int64, error) {
	opts := mediaAssetOptions(filter)
	var total int64
	var err error
	total, err = r.db.Count(ctx, &model.MediaAsset{}, opts...)
	if err != nil {
		return nil, 0, err
	}
	page := filter.Page
	pageSize := filter.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	opts = append(opts,
		database.Order("created_at DESC, id DESC"),
		database.Limit(pageSize),
		database.Offset((page-1)*pageSize),
	)
	var assets []model.MediaAsset
	err = r.db.Find(ctx, &assets, opts...)
	return assets, total, err
}

func (r *repository) ListMediaUploadChunks(ctx context.Context, sessionID int64) ([]model.MediaUploadChunk, error) {
	var chunks []model.MediaUploadChunk
	err := r.db.Find(ctx, &chunks,
		database.Where("session_id = ?", sessionID),
		database.Order("chunk_index ASC"),
	)
	return chunks, err
}

func (r *repository) ListOperationRecords(ctx context.Context, filter model.OperationRecordFilter) ([]model.OperationRecord, int64, error) {
	opts := operationRecordOptions(filter)
	var total int64
	var err error
	total, err = r.db.Count(ctx, &model.OperationRecord{}, opts...)
	if err != nil {
		return nil, 0, err
	}
	page := filter.Page
	pageSize := filter.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	opts = append(opts,
		database.Order("created_at DESC, id DESC"),
		database.Limit(pageSize),
		database.Offset((page-1)*pageSize),
	)
	var records []model.OperationRecord
	err = r.db.Find(ctx, &records, opts...)
	return records, total, err
}

func (r *repository) ListParameters(ctx context.Context, filter model.ParameterFilter) ([]model.Parameter, int64, error) {
	opts := parameterOptions(filter)
	var total int64
	var err error
	total, err = r.db.Count(ctx, &model.Parameter{}, opts...)
	if err != nil {
		return nil, 0, err
	}
	page := filter.Page
	pageSize := filter.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	opts = append(opts,
		database.Order("created_at DESC, id DESC"),
		database.Limit(pageSize),
		database.Offset((page-1)*pageSize),
	)
	var parameters []model.Parameter
	err = r.db.Find(ctx, &parameters, opts...)
	return parameters, total, err
}

func (r *repository) ListTrafficHijackEvents(ctx context.Context, filter model.TrafficHijackEventFilter) ([]model.TrafficHijackEvent, int64, error) {
	opts := trafficHijackEventOptions(filter)
	total, err := r.db.Count(ctx, &model.TrafficHijackEvent{}, opts...)
	if err != nil {
		return nil, 0, err
	}
	page := filter.Page
	pageSize := filter.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	opts = append(opts,
		database.Order("last_seen_at DESC, id DESC"),
		database.Limit(pageSize),
		database.Offset((page-1)*pageSize),
	)
	var events []model.TrafficHijackEvent
	err = r.db.Find(ctx, &events, opts...)
	return events, total, err
}

func (r *repository) ListTrafficProbeResults(ctx context.Context, filter model.TrafficProbeResultFilter) ([]model.TrafficProbeResult, error) {
	opts := trafficProbeResultOptions(filter)
	limit := filter.Limit
	if limit < 1 {
		limit = 50
	}
	if limit > 201 {
		limit = 201
	}
	opts = append(opts,
		database.Order("created_at DESC, id DESC"),
		database.Limit(limit),
	)
	var results []model.TrafficProbeResult
	err := r.db.Find(ctx, &results, opts...)
	return results, err
}

func (r *repository) ListTrafficProbeTargets(ctx context.Context) ([]model.TrafficProbeTarget, error) {
	var targets []model.TrafficProbeTarget
	err := r.db.Find(ctx, &targets, alive(), database.Order("created_at DESC, id DESC"))
	return targets, err
}

func (r *repository) ListVersions(ctx context.Context, filter model.VersionFilter) ([]model.Version, int64, error) {
	opts := versionOptions(filter)
	var total int64
	var err error
	total, err = r.db.Count(ctx, &model.Version{}, opts...)
	if err != nil {
		return nil, 0, err
	}
	page := filter.Page
	pageSize := filter.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	opts = append(opts,
		database.Order("created_at DESC, id DESC"),
		database.Limit(pageSize),
		database.Offset((page-1)*pageSize),
	)
	var versions []model.Version
	err = r.db.Find(ctx, &versions, opts...)
	return versions, total, err
}

func (r *repository) SaveAPI(ctx context.Context, api *model.APIRecord) error {
	api.UpdatedAt = time.Now().UTC()
	return r.db.Save(ctx, api)
}

func (r *repository) SaveDictionary(ctx context.Context, dictionary *model.Dictionary) error {
	dictionary.UpdatedAt = time.Now().UTC()
	return r.db.Save(ctx, dictionary)
}

func (r *repository) SaveDictionaryItem(ctx context.Context, item *model.DictionaryItem) error {
	item.UpdatedAt = time.Now().UTC()
	return r.db.Save(ctx, item)
}

func (r *repository) SaveMediaAsset(ctx context.Context, asset *model.MediaAsset) error {
	asset.UpdatedAt = time.Now().UTC()
	return r.db.Save(ctx, asset)
}

func (r *repository) SaveMediaCategory(ctx context.Context, category *model.MediaCategory) error {
	category.UpdatedAt = time.Now().UTC()
	return r.db.Save(ctx, category)
}

func (r *repository) SaveMediaUploadChunk(ctx context.Context, chunk *model.MediaUploadChunk) error {
	chunk.UpdatedAt = time.Now().UTC()
	return r.db.Save(ctx, chunk)
}

func (r *repository) SaveMediaUploadSession(ctx context.Context, session *model.MediaUploadSession) error {
	session.UpdatedAt = time.Now().UTC()
	return r.db.Save(ctx, session)
}

func (r *repository) SaveParameter(ctx context.Context, parameter *model.Parameter) error {
	parameter.UpdatedAt = time.Now().UTC()
	return r.db.Save(ctx, parameter)
}

func (r *repository) SaveTrafficHijackEvent(ctx context.Context, event *model.TrafficHijackEvent) error {
	event.UpdatedAt = time.Now().UTC()
	return r.db.Save(ctx, event)
}

func (r *repository) SaveTrafficProbeTarget(ctx context.Context, target *model.TrafficProbeTarget) error {
	target.UpdatedAt = time.Now().UTC()
	return r.db.Save(ctx, target)
}

func alive() database.QueryOption {
	return database.Where("deleted_at IS NULL")
}

func operationRecordOptions(filter model.OperationRecordFilter) []database.QueryOption {
	opts := make([]database.QueryOption, 0, 3)
	method := strings.ToUpper(strings.TrimSpace(filter.Method))
	if method != "" {
		opts = append(opts, database.Where("http_method = ?", method))
	}
	path := strings.TrimSpace(filter.Path)
	if path != "" {
		opts = append(opts, database.Where("path LIKE ?", "%"+path+"%"))
	}
	if filter.Status > 0 {
		opts = append(opts, database.Where("status = ?", filter.Status))
		return opts
	}
	switch strings.ToLower(strings.TrimSpace(filter.StatusClass)) {
	case "4xx":
		opts = append(opts, database.Where("status >= ? AND status < ?", 400, 500))
	case "5xx":
		opts = append(opts, database.Where("status >= ? AND status < ?", 500, 600))
	case "error":
		opts = append(opts, database.Where("status >= ?", 400))
	}
	return opts
}

func parameterOptions(filter model.ParameterFilter) []database.QueryOption {
	opts := []database.QueryOption{alive()}
	name := strings.TrimSpace(filter.Name)
	if name != "" {
		opts = append(opts, database.Where("name LIKE ?", "%"+name+"%"))
	}
	key := strings.TrimSpace(filter.Key)
	if key != "" {
		opts = append(opts, database.Where("param_key LIKE ?", "%"+key+"%"))
	}
	if filter.StartCreatedAt != nil {
		opts = append(opts, database.Where("created_at >= ?", *filter.StartCreatedAt))
	}
	if filter.EndCreatedAt != nil {
		opts = append(opts, database.Where("created_at < ?", *filter.EndCreatedAt))
	}
	return opts
}

func versionOptions(filter model.VersionFilter) []database.QueryOption {
	opts := []database.QueryOption{alive()}
	name := strings.TrimSpace(filter.VersionName)
	if name != "" {
		opts = append(opts, database.Where("version_name LIKE ?", "%"+name+"%"))
	}
	code := strings.TrimSpace(filter.VersionCode)
	if code != "" {
		opts = append(opts, database.Where("version_code LIKE ?", "%"+code+"%"))
	}
	if filter.StartCreatedAt != nil {
		opts = append(opts, database.Where("created_at >= ?", *filter.StartCreatedAt))
	}
	if filter.EndCreatedAt != nil {
		opts = append(opts, database.Where("created_at < ?", *filter.EndCreatedAt))
	}
	return opts
}

func trafficProbeResultOptions(filter model.TrafficProbeResultFilter) []database.QueryOption {
	opts := make([]database.QueryOption, 0, 2)
	if filter.TargetID > 0 {
		opts = append(opts, database.Where("target_id = ?", filter.TargetID))
	}
	if filter.Cursor > 0 {
		opts = append(opts, database.Where("id < ?", filter.Cursor))
	}
	return opts
}

func trafficHijackEventOptions(filter model.TrafficHijackEventFilter) []database.QueryOption {
	opts := make([]database.QueryOption, 0, 3)
	if filter.TargetID > 0 {
		opts = append(opts, database.Where("target_id = ?", filter.TargetID))
	}
	if severity := strings.TrimSpace(filter.Severity); severity != "" {
		opts = append(opts, database.Where("severity = ?", strings.ToLower(severity)))
	}
	if state := strings.TrimSpace(filter.State); state != "" {
		opts = append(opts, database.Where("state = ?", strings.ToLower(state)))
	}
	return opts
}

func mediaAssetOptions(filter model.MediaAssetFilter) []database.QueryOption {
	opts := []database.QueryOption{alive()}
	if filter.CategoryID > 0 {
		opts = append(opts, database.Where("category_id = ?", filter.CategoryID))
	}
	keyword := strings.TrimSpace(filter.Keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		opts = append(opts, database.Where("(display_name LIKE ? OR original_name LIKE ? OR url LIKE ?)", like, like, like))
	}
	return opts
}

func mapStorageError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, database.ErrNotFound) {
		return systemservice.ErrNotFound
	}
	if errors.Is(err, systemservice.ErrStorageUnavailable) {
		return err
	}
	if isStorageUnavailable(err) {
		return fmt.Errorf("%w: %v", systemservice.ErrStorageUnavailable, err)
	}
	return err
}

func isStorageUnavailable(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "no such table") ||
		strings.Contains(text, "doesn't exist") ||
		strings.Contains(text, "undefined_table") ||
		strings.Contains(text, "unknown table")
}
