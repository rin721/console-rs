// Package service 承载系统模块的应用层规则。
package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/rei0721/go-scaffold/internal/modules/system/model"
)

// VersionFilter 描述版本记录列表的查询条件。
// Page 和 PageSize 由服务层统一归一化；时间范围用于限制创建时间，
// VersionCode 和 VersionName 会在访问仓储前去除首尾空白。
type VersionFilter struct {
	EndCreatedAt   *time.Time
	Page           int
	PageSize       int
	StartCreatedAt *time.Time
	VersionCode    string
	VersionName    string
}

// ExportVersionInput 描述一次版本导出的选择范围和审计信息。
// MenuCodes、APICodes、DictionaryCodes 至少需要命中一个资源；CreatedBy
// 与 CreatorUsername 会写入版本记录，属于该操作的持久化副作用。
type ExportVersionInput struct {
	APICodes        []string
	CreatedBy       int64
	CreatorUsername string
	Description     string
	DictionaryCodes []string
	MenuCodes       []string
	VersionCode     string
	VersionName     string
}

// ImportVersionInput 描述一次版本导入请求。
// VersionData 是导出的 JSON 包文本；CreatedBy 与 CreatorUsername 用于记录导入审计。
type ImportVersionInput struct {
	CreatedBy       int64
	CreatorUsername string
	VersionData     string
}

// ListVersionSources 汇总当前可导出的菜单、API 和字典资源。
// 返回值包含资源明细、计数以及字典仓储状态；该方法不产生持久化副作用。
func (s *service) ListVersionSources(ctx context.Context) (model.VersionSourceCatalog, error) {
	menus, err := s.ListMenus(ctx)
	if err != nil {
		return model.VersionSourceCatalog{}, err
	}
	apis, err := s.ListAPIs(ctx)
	if err != nil {
		return model.VersionSourceCatalog{}, err
	}
	catalog, err := s.ListDictionaries(ctx)
	if err != nil {
		return model.VersionSourceCatalog{}, err
	}
	return model.VersionSourceCatalog{
		APICount:        countAPIs(apis),
		APIs:            apis,
		Dictionaries:    catalog.Items,
		DictionaryCount: len(catalog.Items),
		MenuCount:       countMenus(menus),
		Menus:           menus,
		StorageStatus:   catalog.StorageStatus,
	}, nil
}

// ListVersions 按过滤条件分页读取版本记录。
// 当仓储不可用时返回带 unavailable 状态的空页，便于管理端展示降级状态；
// 起止时间必须形成有效半开区间，否则返回 ErrInvalidInput。
func (s *service) ListVersions(ctx context.Context, input VersionFilter) (model.VersionPage, error) {
	page := normalizePage(input.Page)
	pageSize := normalizePageSize(input.PageSize)
	result := model.VersionPage{Page: page, PageSize: pageSize, StorageStatus: "unavailable"}
	if s.repo == nil {
		return result, nil
	}
	if input.StartCreatedAt != nil && input.EndCreatedAt != nil && !input.StartCreatedAt.Before(*input.EndCreatedAt) {
		return result, ErrInvalidInput
	}
	versions, total, err := s.repo.ListVersions(ctx, model.VersionFilter{
		EndCreatedAt:   input.EndCreatedAt,
		Page:           page,
		PageSize:       pageSize,
		StartCreatedAt: input.StartCreatedAt,
		VersionCode:    strings.TrimSpace(input.VersionCode),
		VersionName:    strings.TrimSpace(input.VersionName),
	})
	if err != nil {
		if isStorageUnavailable(err) {
			return result, nil
		}
		return result, err
	}
	result.Items = versions
	result.StorageStatus = "persisted"
	result.Total = total
	return result, nil
}

// FindVersion 读取单个版本记录并解析其版本包。
// id 必须为正数；返回值同时包含持久化元数据和 JSON 包内容。
func (s *service) FindVersion(ctx context.Context, id int64) (*model.VersionDetail, error) {
	version, err := s.findVersionRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	pkg, err := decodeVersionPackage(version.VersionData)
	if err != nil {
		return nil, err
	}
	return &model.VersionDetail{Item: *version, Package: pkg}, nil
}

// GetVersionPackage 只返回版本记录中的包内容。
// 该方法复用记录查找和 JSON 校验逻辑，避免调用方绕过版本包完整性检查。
func (s *service) GetVersionPackage(ctx context.Context, id int64) (model.VersionPackage, error) {
	version, err := s.findVersionRecord(ctx, id)
	if err != nil {
		return model.VersionPackage{}, err
	}
	return decodeVersionPackage(version.VersionData)
}

// ExportVersion 根据输入选择器生成版本包并写入版本记录。
// 返回值包含新建记录和完整包内容；副作用是创建一条可审计的版本快照。
// JSON 使用缩进格式保存，方便排查和人工比对导出内容。
func (s *service) ExportVersion(ctx context.Context, input ExportVersionInput) (*model.VersionDetail, error) {
	if s.repo == nil {
		return nil, ErrStorageUnavailable
	}
	name := strings.TrimSpace(input.VersionName)
	code := normalizeVersionCode(input.VersionCode)
	if name == "" || !validVersionCode(code) {
		return nil, ErrInvalidInput
	}
	pkg, err := s.buildVersionPackage(ctx, ExportVersionInput{
		APICodes:        input.APICodes,
		CreatedBy:       input.CreatedBy,
		CreatorUsername: strings.TrimSpace(input.CreatorUsername),
		Description:     strings.TrimSpace(input.Description),
		DictionaryCodes: input.DictionaryCodes,
		MenuCodes:       input.MenuCodes,
		VersionCode:     code,
		VersionName:     name,
	})
	if err != nil {
		return nil, err
	}
	raw, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return nil, err
	}
	now := s.now()
	version := &model.Version{
		ID:                s.ids.NextID(),
		APICount:          len(pkg.APIs),
		CreatedAt:         now,
		CreatedBy:         input.CreatedBy,
		CreatedByUsername: strings.TrimSpace(input.CreatorUsername),
		Description:       strings.TrimSpace(input.Description),
		DictionaryCount:   len(pkg.Dictionaries),
		MenuCount:         countMenus(pkg.Menus),
		Source:            model.VersionSourceExport,
		UpdatedAt:         now,
		VersionCode:       code,
		VersionData:       string(raw),
		VersionName:       name,
	}
	if err := s.repo.CreateVersion(ctx, version); err != nil {
		if isStorageUnavailable(err) {
			return nil, ErrStorageUnavailable
		}
		return nil, err
	}
	return &model.VersionDetail{Item: *version, Package: pkg}, nil
}

// ImportVersion 校验版本包并导入其中可持久化的资源。
// 当前只会把字典和字典项落库，菜单与 API 作为快照保存在版本记录中并计入 skipped；
// 返回值记录创建、跳过数量以及最终版本记录，便于调用方展示导入结果。
func (s *service) ImportVersion(ctx context.Context, input ImportVersionInput) (model.VersionImportResult, error) {
	result := model.VersionImportResult{
		ImportedAt:    s.now(),
		StorageStatus: "unavailable",
		APIsSkipped:   0,
		MenusSkipped:  0,
	}
	if s.repo == nil {
		return result, ErrStorageUnavailable
	}
	pkg, err := decodeVersionPackage(input.VersionData)
	if err != nil {
		return result, err
	}
	result.MenusSkipped = countMenus(pkg.Menus)
	result.APIsSkipped = len(pkg.APIs)
	createdDictionaries, skippedDictionaries, createdItems, err := s.importVersionDictionaries(ctx, pkg.Dictionaries)
	if err != nil {
		if isStorageUnavailable(err) {
			return result, ErrStorageUnavailable
		}
		return result, err
	}
	result.DictionariesCreated = createdDictionaries
	result.DictionariesSkipped = skippedDictionaries
	result.DictionaryItemsCreated = createdItems

	raw, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return result, err
	}
	now := s.now()
	version := &model.Version{
		ID:                s.ids.NextID(),
		APICount:          len(pkg.APIs),
		CreatedAt:         now,
		CreatedBy:         input.CreatedBy,
		CreatedByUsername: strings.TrimSpace(input.CreatorUsername),
		Description:       strings.TrimSpace(pkg.Version.Description),
		DictionaryCount:   len(pkg.Dictionaries),
		MenuCount:         countMenus(pkg.Menus),
		Source:            model.VersionSourceImport,
		UpdatedAt:         now,
		VersionCode:       normalizeVersionCode(pkg.Version.Code),
		VersionData:       string(raw),
		VersionName:       strings.TrimSpace(pkg.Version.Name),
	}
	if err := s.repo.CreateVersion(ctx, version); err != nil {
		if isStorageUnavailable(err) {
			return result, ErrStorageUnavailable
		}
		return result, err
	}
	result.Item = *version
	result.StorageStatus = "persisted"
	return result, nil
}

// DeleteVersion 软删除单个版本记录。
// 删除前会先确认记录存在，以便把不存在和仓储不可用映射成稳定的服务层错误。
func (s *service) DeleteVersion(ctx context.Context, id int64) error {
	if s.repo == nil {
		return ErrStorageUnavailable
	}
	if _, err := s.repo.FindVersionByID(ctx, id); err != nil {
		return mapVersionLookupError(err)
	}
	if err := s.repo.DeleteVersion(ctx, id, s.now()); err != nil {
		if isStorageUnavailable(err) {
			return ErrStorageUnavailable
		}
		return err
	}
	return nil
}

// DeleteVersions 批量软删除版本记录。
// ids 必须非空且全部为正数；该方法只负责版本记录本身，不回滚已导入的业务资源。
func (s *service) DeleteVersions(ctx context.Context, ids []int64) error {
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
	if err := s.repo.DeleteVersions(ctx, normalized, s.now()); err != nil {
		if isStorageUnavailable(err) {
			return ErrStorageUnavailable
		}
		return err
	}
	return nil
}

// buildVersionPackage 从当前系统资源中抽取指定菜单、API 和字典，组装导出包。
// 选择器为空或最终没有命中任何资源都会被视为无效输入；字典导出依赖持久化仓储。
func (s *service) buildVersionPackage(ctx context.Context, input ExportVersionInput) (model.VersionPackage, error) {
	menuCodes := normalizedStringSet(input.MenuCodes)
	apiCodes := normalizedStringSet(input.APICodes)
	dictionaryCodes := normalizedStringSet(input.DictionaryCodes)
	if len(menuCodes) == 0 && len(apiCodes) == 0 && len(dictionaryCodes) == 0 {
		return model.VersionPackage{}, ErrInvalidInput
	}
	menus, err := s.ListMenus(ctx)
	if err != nil {
		return model.VersionPackage{}, err
	}
	apis, err := s.ListAPIs(ctx)
	if err != nil {
		return model.VersionPackage{}, err
	}
	dictionaryCatalog, err := s.ListDictionaries(ctx)
	if err != nil {
		return model.VersionPackage{}, err
	}
	if len(dictionaryCodes) > 0 && dictionaryCatalog.StorageStatus != "persisted" {
		return model.VersionPackage{}, ErrStorageUnavailable
	}
	pkg := model.VersionPackage{
		APIs:         selectedAPIs(apis, apiCodes),
		Dictionaries: selectedDictionaries(dictionaryCatalog.Items, dictionaryCodes),
		Menus:        selectedMenus(menus, menuCodes),
		Version: model.VersionPackageInfo{
			Code:        normalizeVersionCode(input.VersionCode),
			Description: strings.TrimSpace(input.Description),
			ExportTime:  s.now(),
			Name:        strings.TrimSpace(input.VersionName),
		},
	}
	if countMenus(pkg.Menus) == 0 && len(pkg.APIs) == 0 && len(pkg.Dictionaries) == 0 {
		return model.VersionPackage{}, ErrInvalidInput
	}
	return pkg, nil
}

// importVersionDictionaries 以幂等方式导入版本包中的字典。
// 已存在的字典不会被覆盖，但会继续尝试补齐缺失的字典项；返回值分别是
// 新建字典数、跳过字典数和新建字典项数。
func (s *service) importVersionDictionaries(ctx context.Context, dictionaries []model.Dictionary) (int, int, int, error) {
	createdDictionaries := 0
	skippedDictionaries := 0
	createdItems := 0
	for _, src := range dictionaries {
		code := normalizeDictionaryCode(src.Code)
		name := strings.TrimSpace(src.Name)
		if !validDictionaryCode(code) || name == "" {
			return createdDictionaries, skippedDictionaries, createdItems, ErrInvalidInput
		}
		status, err := normalizeDictionaryStatus(src.Status)
		if err != nil {
			status = model.DictionaryStatusActive
		}
		dictionary, err := s.repo.FindDictionaryByCode(ctx, code)
		if err == nil {
			skippedDictionaries++
		} else if errors.Is(err, ErrNotFound) {
			now := s.now()
			dictionary = &model.Dictionary{
				ID:          s.ids.NextID(),
				Code:        code,
				CreatedAt:   now,
				Description: strings.TrimSpace(src.Description),
				Name:        name,
				Status:      status,
				UpdatedAt:   now,
			}
			if err := s.repo.CreateDictionary(ctx, dictionary); err != nil {
				if isStorageDuplicate(err) {
					skippedDictionaries++
				} else {
					return createdDictionaries, skippedDictionaries, createdItems, err
				}
			} else {
				createdDictionaries++
			}
		} else {
			return createdDictionaries, skippedDictionaries, createdItems, err
		}
		if dictionary == nil {
			continue
		}
		items, err := s.importVersionDictionaryItems(ctx, dictionary.ID, src.Items)
		if err != nil {
			return createdDictionaries, skippedDictionaries, createdItems, err
		}
		createdItems += items
	}
	return createdDictionaries, skippedDictionaries, createdItems, nil
}

// importVersionDictionaryItems 把版本包中的字典项导入到指定字典。
// 去重以 Value 为准，保证重复导入同一个版本包时不会制造重复枚举值。
func (s *service) importVersionDictionaryItems(ctx context.Context, dictionaryID int64, srcItems []model.DictionaryItem) (int, error) {
	existing, err := s.repo.ListDictionaryItems(ctx, dictionaryID)
	if err != nil {
		return 0, err
	}
	byValue := make(map[string]struct{}, len(existing))
	for _, item := range existing {
		byValue[item.Value] = struct{}{}
	}
	created := 0
	for _, src := range srcItems {
		value := strings.TrimSpace(src.Value)
		if value == "" {
			return created, ErrInvalidInput
		}
		if _, ok := byValue[value]; ok {
			continue
		}
		label := strings.TrimSpace(src.Label)
		if label == "" {
			label = value
		}
		status, err := normalizeDictionaryStatus(src.Status)
		if err != nil {
			status = model.DictionaryStatusActive
		}
		now := s.now()
		item := &model.DictionaryItem{
			ID:           s.ids.NextID(),
			CreatedAt:    now,
			DictionaryID: dictionaryID,
			Extra:        strings.TrimSpace(src.Extra),
			Label:        label,
			Sort:         src.Sort,
			Status:       status,
			UpdatedAt:    now,
			Value:        value,
		}
		if err := s.repo.CreateDictionaryItem(ctx, item); err != nil {
			if isStorageDuplicate(err) {
				continue
			}
			return created, err
		}
		byValue[value] = struct{}{}
		created++
	}
	return created, nil
}

// findVersionRecord 封装版本记录查找的输入校验和错误归一化。
// 这样详情、下载和删除入口可以保持相同的错误语义。
func (s *service) findVersionRecord(ctx context.Context, id int64) (*model.Version, error) {
	if s.repo == nil {
		return nil, ErrStorageUnavailable
	}
	if id <= 0 {
		return nil, ErrInvalidInput
	}
	version, err := s.repo.FindVersionByID(ctx, id)
	if err != nil {
		return nil, mapVersionLookupError(err)
	}
	return version, nil
}

// decodeVersionPackage 解析并校验持久化的版本包 JSON。
// 除 JSON 格式外，还会校验版本元数据和至少一个资源集合，避免导入空快照。
func decodeVersionPackage(raw string) (model.VersionPackage, error) {
	var pkg model.VersionPackage
	if strings.TrimSpace(raw) == "" {
		return pkg, ErrInvalidInput
	}
	if err := json.Unmarshal([]byte(raw), &pkg); err != nil {
		return pkg, ErrInvalidInput
	}
	pkg.Version.Name = strings.TrimSpace(pkg.Version.Name)
	pkg.Version.Code = normalizeVersionCode(pkg.Version.Code)
	pkg.Version.Description = strings.TrimSpace(pkg.Version.Description)
	if pkg.Version.Name == "" || !validVersionCode(pkg.Version.Code) {
		return pkg, ErrInvalidInput
	}
	if pkg.Version.ExportTime.IsZero() {
		pkg.Version.ExportTime = time.Now().UTC()
	}
	if countMenus(pkg.Menus) == 0 && len(pkg.APIs) == 0 && len(pkg.Dictionaries) == 0 {
		return pkg, ErrInvalidInput
	}
	return pkg, nil
}

// selectedMenus 根据菜单组或菜单项选择器保留导出范围。
// 选择器既支持 group:item 形式，也兼容只传组编码或菜单项编码的调用方式。
func selectedMenus(groups []model.MenuGroup, selected map[string]struct{}) []model.MenuGroup {
	if len(selected) == 0 {
		return nil
	}
	out := make([]model.MenuGroup, 0, len(groups))
	for _, group := range groups {
		items := make([]model.MenuItem, 0, len(group.Items))
		_, includeGroup := selected[normalizeSelector(group.Code)]
		for _, item := range group.Items {
			_, includeFull := selected[normalizeSelector(group.Code+":"+item.Code)]
			_, includeItem := selected[normalizeSelector(item.Code)]
			if includeGroup || includeFull || includeItem {
				items = append(items, item)
			}
		}
		if len(items) == 0 {
			continue
		}
		group.Items = items
		out = append(out, group)
	}
	return out
}

// selectedAPIs 根据 API code 或 method:path 选择器抽取导出项。
// 支持路径维度选择是为了让路由注册表缺少显式 code 时仍可稳定导出。
func selectedAPIs(groups []model.APIGroup, selected map[string]struct{}) []model.APIEntry {
	if len(selected) == 0 {
		return nil
	}
	out := make([]model.APIEntry, 0)
	for _, group := range groups {
		for _, item := range group.Items {
			if _, ok := selected[normalizeSelector(item.Code)]; ok {
				out = append(out, item)
				continue
			}
			if _, ok := selected[normalizeSelector(apiKey(item.Method, item.Path))]; ok {
				out = append(out, item)
				continue
			}
		}
	}
	return out
}

// selectedDictionaries 根据字典 code 选择导出项。
// 字典项已经挂在 Dictionary.Items 上，调用方不需要再单独选择子项。
func selectedDictionaries(dictionaries []model.Dictionary, selected map[string]struct{}) []model.Dictionary {
	if len(selected) == 0 {
		return nil
	}
	out := make([]model.Dictionary, 0, len(dictionaries))
	for _, dictionary := range dictionaries {
		if _, ok := selected[normalizeSelector(dictionary.Code)]; ok {
			out = append(out, dictionary)
		}
	}
	return out
}

// countMenus 统计菜单组内的实际菜单项数量。
func countMenus(groups []model.MenuGroup) int {
	total := 0
	for _, group := range groups {
		total += len(group.Items)
	}
	return total
}

// countAPIs 统计 API 分组内的实际接口数量。
func countAPIs(groups []model.APIGroup) int {
	total := 0
	for _, group := range groups {
		total += len(group.Items)
	}
	return total
}

// normalizedStringSet 将用户选择器归一化为集合。
// 空值会被丢弃，重复选择器自然合并，便于后续 O(1) 命中判断。
func normalizedStringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := normalizeSelector(value)
		if normalized == "" {
			continue
		}
		set[normalized] = struct{}{}
	}
	return set
}

func normalizeSelector(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeVersionCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// validVersionCode 限制版本编码为可跨系统传输的安全字符集合。
// 这里刻意不接受空白和其他符号，避免编码进入 URL、文件名或权限表达式时产生歧义。
func validVersionCode(code string) bool {
	if code == "" {
		return false
	}
	for _, char := range code {
		switch {
		case char >= 'a' && char <= 'z':
		case char >= '0' && char <= '9':
		case char == '_' || char == '-' || char == '.' || char == ':':
		default:
			return false
		}
	}
	return true
}

// mapVersionLookupError 将仓储层查找错误转换为稳定的服务层错误。
func mapVersionLookupError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return ErrNotFound
	case isStorageUnavailable(err):
		return ErrStorageUnavailable
	default:
		return err
	}
}
