package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/rei0721/go-scaffold/internal/modules/system/model"
)

type SeedResult struct {
	DictionariesCreated    int
	DictionaryItemsCreated int
	ParametersCreated      int
	SeededAt               time.Time
	StorageStatus          string
}

type dictionarySeed struct {
	Code        string
	Description string
	Items       []dictionaryItemSeed
	Name        string
}

type dictionaryItemSeed struct {
	Label string
	Sort  int
	Value string
}

type parameterSeed struct {
	Description string
	Key         string
	Name        string
	Value       string
}

var builtinDictionarySeeds = []dictionarySeed{
	{
		Code:        "system.status",
		Description: "系统记录通用启用/禁用状态。",
		Name:        "系统状态",
		Items: []dictionaryItemSeed{
			{Label: "启用", Sort: 10, Value: model.DictionaryStatusActive},
			{Label: "禁用", Sort: 20, Value: model.DictionaryStatusDisabled},
		},
	},
	{
		Code:        "http.method",
		Description: "系统 API 目录展示使用的 HTTP 方法。",
		Name:        "HTTP 方法",
		Items: []dictionaryItemSeed{
			{Label: "GET", Sort: 10, Value: "GET"},
			{Label: "POST", Sort: 20, Value: "POST"},
			{Label: "PATCH", Sort: 30, Value: "PATCH"},
			{Label: "PUT", Sort: 40, Value: "PUT"},
			{Label: "DELETE", Sort: 50, Value: "DELETE"},
		},
	},
	{
		Code:        "operation.result",
		Description: "后台操作记录使用的结果分类。",
		Name:        "操作结果",
		Items: []dictionaryItemSeed{
			{Label: "成功", Sort: 10, Value: "success"},
			{Label: "失败", Sort: 20, Value: "failed"},
		},
	},
}

var builtinParameterSeeds = []parameterSeed{
	{
		Description: "后台默认可见标题。",
		Key:         "admin.title",
		Name:        "后台标题",
		Value:       "system.brand.productName",
	},
	{
		Description: "Go 静态托管后台的默认路径。",
		Key:         "admin.home_path",
		Name:        "后台首页路径",
		Value:       "/admin",
	},
	{
		Description: "当前后台结构参考来源。",
		Key:         "system.reference",
		Name:        "系统参考",
		Value:       "admin-template-parity",
	},
}

func (s *service) SeedDefaults(ctx context.Context) (SeedResult, error) {
	result := SeedResult{
		SeededAt:      s.now(),
		StorageStatus: "unavailable",
	}
	if s.repo == nil {
		return result, nil
	}
	for _, spec := range builtinDictionarySeeds {
		dictionary, created, err := s.seedDictionary(ctx, spec)
		if err != nil {
			if isStorageUnavailable(err) {
				return result, nil
			}
			return result, err
		}
		if dictionary == nil {
			continue
		}
		if created {
			result.DictionariesCreated++
		}
		createdItems, err := s.seedDictionaryItems(ctx, dictionary.ID, spec.Items)
		if err != nil {
			if isStorageUnavailable(err) {
				return result, nil
			}
			return result, err
		}
		result.DictionaryItemsCreated += createdItems
	}
	createdParameters, err := s.seedParameters(ctx)
	if err != nil {
		if isStorageUnavailable(err) {
			return result, nil
		}
		return result, err
	}
	result.ParametersCreated = createdParameters
	result.StorageStatus = "persisted"
	return result, nil
}

func (s *service) seedDictionary(ctx context.Context, spec dictionarySeed) (*model.Dictionary, bool, error) {
	code := normalizeDictionaryCode(spec.Code)
	existing, err := s.repo.FindDictionaryByCode(ctx, code)
	if err == nil {
		return existing, false, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, false, err
	}
	now := s.now()
	dictionary := &model.Dictionary{
		ID:          s.ids.NextID(),
		Code:        code,
		CreatedAt:   now,
		Description: strings.TrimSpace(spec.Description),
		Name:        strings.TrimSpace(spec.Name),
		Status:      model.DictionaryStatusActive,
		UpdatedAt:   now,
	}
	if err := s.repo.CreateDictionary(ctx, dictionary); err != nil {
		if isStorageDuplicate(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return dictionary, true, nil
}

func (s *service) seedDictionaryItems(ctx context.Context, dictionaryID int64, specs []dictionaryItemSeed) (int, error) {
	existing, err := s.repo.ListDictionaryItems(ctx, dictionaryID)
	if err != nil {
		return 0, err
	}
	byValue := make(map[string]struct{}, len(existing))
	for _, item := range existing {
		byValue[item.Value] = struct{}{}
	}
	created := 0
	for _, spec := range specs {
		value := strings.TrimSpace(spec.Value)
		if value == "" {
			continue
		}
		if _, ok := byValue[value]; ok {
			continue
		}
		now := s.now()
		item := &model.DictionaryItem{
			ID:           s.ids.NextID(),
			CreatedAt:    now,
			DictionaryID: dictionaryID,
			Label:        strings.TrimSpace(spec.Label),
			Sort:         spec.Sort,
			Status:       model.DictionaryStatusActive,
			UpdatedAt:    now,
			Value:        value,
		}
		if item.Label == "" {
			item.Label = value
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

func (s *service) seedParameters(ctx context.Context) (int, error) {
	created := 0
	for _, spec := range builtinParameterSeeds {
		key := strings.TrimSpace(spec.Key)
		if key == "" {
			continue
		}
		if _, err := s.repo.FindParameterByKey(ctx, key); err == nil {
			continue
		} else if !errors.Is(err, ErrNotFound) {
			return created, err
		}
		now := s.now()
		parameter := &model.Parameter{
			ID:          s.ids.NextID(),
			CreatedAt:   now,
			Description: strings.TrimSpace(spec.Description),
			Key:         key,
			Name:        strings.TrimSpace(spec.Name),
			UpdatedAt:   now,
			Value:       strings.TrimSpace(spec.Value),
		}
		if parameter.Name == "" {
			parameter.Name = key
		}
		if parameter.Value == "" {
			continue
		}
		if err := s.repo.CreateParameter(ctx, parameter); err != nil {
			if isStorageDuplicate(err) {
				continue
			}
			return created, err
		}
		created++
	}
	return created, nil
}

func isStorageDuplicate(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "duplicate") ||
		strings.Contains(text, "duplicated") ||
		strings.Contains(text, "unique constraint") ||
		strings.Contains(text, "constraint failed") ||
		strings.Contains(text, "23505") ||
		strings.Contains(text, "1062")
}
