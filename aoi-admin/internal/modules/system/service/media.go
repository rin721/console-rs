package service

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	urlpath "path"
	"strconv"
	"strings"

	"github.com/rei0721/go-scaffold/internal/modules/system/model"
	appconstants "github.com/rei0721/go-scaffold/types/constants"
)

type MediaObjectStorage interface {
	ReadFile(string) ([]byte, error)
	WriteFile(string, []byte, os.FileMode) error
	Remove(string) error
	RemoveAll(string) error
	MkdirAll(string, os.FileMode) error
	DetectMIMEFromBytes([]byte) (string, error)
}

type MediaAssetFilter struct {
	CategoryID int64
	Keyword    string
	Page       int
	PageSize   int
}

type UpsertMediaCategoryInput struct {
	ID       int64
	ParentID int64
	Name     string
	Sort     int
}

type UploadMediaAssetInput struct {
	CategoryID         int64
	Filename           string
	Reader             io.Reader
	Size               int64
	UploadedBy         int64
	UploadedByUsername string
}

type MediaURLImportItem struct {
	Name string
	URL  string
}

type ImportMediaURLsInput struct {
	CategoryID         int64
	Items              []MediaURLImportItem
	UploadedBy         int64
	UploadedByUsername string
}

type UpdateMediaAssetInput struct {
	DisplayName string
}

type MediaDownload struct {
	ContentType string
	Data        []byte
	Filename    string
}

func (s *service) ListMediaCategories(ctx context.Context) (model.MediaCategoryCatalog, error) {
	catalog := model.MediaCategoryCatalog{StorageStatus: "unavailable"}
	if s.repo == nil {
		return catalog, nil
	}
	categories, err := s.repo.ListMediaCategories(ctx)
	if err != nil {
		if isStorageUnavailable(err) {
			return catalog, nil
		}
		return catalog, err
	}
	catalog.Items = buildMediaCategoryTree(categories)
	catalog.StorageStatus = "persisted"
	catalog.Total = len(categories)
	return catalog, nil
}

func (s *service) UpsertMediaCategory(ctx context.Context, input UpsertMediaCategoryInput) (*model.MediaCategory, error) {
	if s.repo == nil {
		return nil, ErrStorageUnavailable
	}
	name := cleanMediaName(input.Name, 128)
	if name == "" || input.ParentID < 0 || (input.ID > 0 && input.ID == input.ParentID) {
		return nil, ErrInvalidInput
	}
	if input.ParentID > 0 {
		if _, err := s.repo.FindMediaCategoryByID(ctx, input.ParentID); err != nil {
			return nil, mapMediaLookupError(err)
		}
	}
	if exists, err := s.mediaCategoryNameExists(ctx, input.ID, input.ParentID, name); err != nil {
		return nil, err
	} else if exists {
		return nil, ErrDuplicate
	}
	if input.ID > 0 {
		category, err := s.repo.FindMediaCategoryByID(ctx, input.ID)
		if err != nil {
			return nil, mapMediaLookupError(err)
		}
		if createsMediaCategoryCycle(ctx, s.repo, category.ID, input.ParentID) {
			return nil, ErrInvalidInput
		}
		category.Name = name
		category.ParentID = input.ParentID
		category.Sort = input.Sort
		if err := s.repo.SaveMediaCategory(ctx, category); err != nil {
			return nil, mapMediaStorageError(err)
		}
		return category, nil
	}
	now := s.now()
	category := &model.MediaCategory{
		ID:        s.ids.NextID(),
		CreatedAt: now,
		Name:      name,
		ParentID:  input.ParentID,
		Sort:      input.Sort,
		UpdatedAt: now,
	}
	if err := s.repo.CreateMediaCategory(ctx, category); err != nil {
		if isStorageDuplicate(err) {
			return nil, ErrDuplicate
		}
		return nil, mapMediaStorageError(err)
	}
	return category, nil
}

func (s *service) DeleteMediaCategory(ctx context.Context, id int64) error {
	if s.repo == nil {
		return ErrStorageUnavailable
	}
	if id <= 0 {
		return ErrInvalidInput
	}
	if _, err := s.repo.FindMediaCategoryByID(ctx, id); err != nil {
		return mapMediaLookupError(err)
	}
	categories, err := s.repo.ListMediaCategories(ctx)
	if err != nil {
		return mapMediaStorageError(err)
	}
	for _, category := range categories {
		if category.ParentID == id {
			return ErrInvalidInput
		}
	}
	_, total, err := s.repo.ListMediaAssets(ctx, model.MediaAssetFilter{CategoryID: id, Page: 1, PageSize: 1})
	if err != nil {
		return mapMediaStorageError(err)
	}
	if total > 0 {
		return ErrInvalidInput
	}
	if err := s.repo.DeleteMediaCategory(ctx, id, s.now()); err != nil {
		return mapMediaStorageError(err)
	}
	return nil
}

func (s *service) ListMediaAssets(ctx context.Context, input MediaAssetFilter) (model.MediaAssetPage, error) {
	page := normalizePage(input.Page)
	pageSize := normalizePageSize(input.PageSize)
	result := model.MediaAssetPage{
		ObjectStorage:     s.mediaObjectStorageStatus(),
		Page:              page,
		PageSize:          pageSize,
		StorageStatus:     "unavailable",
		UploadMaxBytes:    s.cfg.MediaMaxBytes,
		UploadMaxMB:       s.cfg.MediaMaxBytes / 1024 / 1024,
		UploadUnavailable: s.objectStore == nil,
	}
	if s.repo == nil {
		return result, nil
	}
	if input.CategoryID < 0 {
		return result, ErrInvalidInput
	}
	assets, total, err := s.repo.ListMediaAssets(ctx, model.MediaAssetFilter{
		CategoryID: input.CategoryID,
		Keyword:    strings.TrimSpace(input.Keyword),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		if isStorageUnavailable(err) {
			return result, nil
		}
		return result, err
	}
	result.Items = assets
	result.StorageStatus = "persisted"
	result.Total = total
	return result, nil
}

func (s *service) UploadMediaAsset(ctx context.Context, input UploadMediaAssetInput) (*model.MediaAsset, error) {
	if s.repo == nil || s.objectStore == nil {
		return nil, ErrStorageUnavailable
	}
	if input.Reader == nil || input.CategoryID < 0 {
		return nil, ErrInvalidInput
	}
	if input.CategoryID > 0 {
		if _, err := s.repo.FindMediaCategoryByID(ctx, input.CategoryID); err != nil {
			return nil, mapMediaLookupError(err)
		}
	}
	originalName := cleanMediaFilename(input.Filename)
	if originalName == "" {
		return nil, ErrInvalidInput
	}
	if input.Size > s.cfg.MediaMaxBytes {
		return nil, ErrInvalidInput
	}
	data, err := io.ReadAll(io.LimitReader(input.Reader, s.cfg.MediaMaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || int64(len(data)) > s.cfg.MediaMaxBytes {
		return nil, ErrInvalidInput
	}
	mimeType := http.DetectContentType(data)
	if detected, err := s.objectStore.DetectMIMEFromBytes(data); err == nil && strings.TrimSpace(detected) != "" {
		mimeType = detected
	}
	id := s.ids.NextID()
	now := s.now()
	ext := safeMediaExtension(originalName, mimeType)
	dir := urlpath.Join(normalizeMediaPrefix(s.cfg.MediaPrefix), now.Format("2006"), now.Format("01"))
	key := urlpath.Join(dir, strconv.FormatInt(id, 10)+ext)
	if err := s.objectStore.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	if err := s.objectStore.WriteFile(key, data, 0644); err != nil {
		return nil, err
	}
	asset := &model.MediaAsset{
		ID:                 id,
		CategoryID:         input.CategoryID,
		CreatedAt:          now,
		DisplayName:        displayNameFromFilename(originalName),
		Extension:          strings.TrimPrefix(ext, "."),
		External:           false,
		MIMEType:           mimeType,
		OriginalName:       originalName,
		SizeBytes:          int64(len(data)),
		Source:             model.MediaSourceUpload,
		StorageKey:         key,
		UpdatedAt:          now,
		UploadedBy:         input.UploadedBy,
		UploadedByUsername: cleanMediaName(input.UploadedByUsername, 128),
		URL:                appconstants.MediaAssetDownloadPath(id),
	}
	if err := s.repo.CreateMediaAsset(ctx, asset); err != nil {
		_ = s.objectStore.Remove(key)
		if isStorageUnavailable(err) {
			return nil, ErrStorageUnavailable
		}
		return nil, err
	}
	return asset, nil
}

func (s *service) ImportMediaURLs(ctx context.Context, input ImportMediaURLsInput) (model.MediaURLImportResult, error) {
	result := model.MediaURLImportResult{StorageStatus: "unavailable"}
	if s.repo == nil {
		return result, ErrStorageUnavailable
	}
	if input.CategoryID < 0 || len(input.Items) == 0 || len(input.Items) > 100 {
		return result, ErrInvalidInput
	}
	if input.CategoryID > 0 {
		if _, err := s.repo.FindMediaCategoryByID(ctx, input.CategoryID); err != nil {
			return result, mapMediaLookupError(err)
		}
	}
	now := s.now()
	result.Items = make([]model.MediaAsset, 0, len(input.Items))
	for _, item := range input.Items {
		cleanURL, parsed, err := normalizeMediaExternalURL(item.URL)
		if err != nil {
			return result, err
		}
		name := mediaURLDisplayName(item.Name, parsed)
		if name == "" {
			return result, ErrInvalidInput
		}
		id := s.ids.NextID()
		ext := strings.TrimPrefix(safeMediaExtension(name, ""), ".")
		mimeType := mediaTypeFromExtension(ext)
		asset := model.MediaAsset{
			ID:                 id,
			CategoryID:         input.CategoryID,
			CreatedAt:          now,
			DisplayName:        displayNameFromFilename(name),
			Extension:          ext,
			External:           true,
			MIMEType:           mimeType,
			OriginalName:       name,
			SizeBytes:          0,
			Source:             model.MediaSourceURL,
			StorageKey:         "external:" + strconv.FormatInt(id, 10),
			UpdatedAt:          now,
			UploadedBy:         input.UploadedBy,
			UploadedByUsername: cleanMediaName(input.UploadedByUsername, 128),
			URL:                cleanURL,
		}
		if err := s.repo.CreateMediaAsset(ctx, &asset); err != nil {
			if isStorageUnavailable(err) {
				return result, ErrStorageUnavailable
			}
			return result, err
		}
		result.Items = append(result.Items, asset)
	}
	result.Imported = len(result.Items)
	result.StorageStatus = "persisted"
	return result, nil
}

func (s *service) UpdateMediaAsset(ctx context.Context, id int64, input UpdateMediaAssetInput) (*model.MediaAsset, error) {
	if s.repo == nil {
		return nil, ErrStorageUnavailable
	}
	name := cleanMediaName(input.DisplayName, 255)
	if id <= 0 || name == "" {
		return nil, ErrInvalidInput
	}
	asset, err := s.repo.FindMediaAssetByID(ctx, id)
	if err != nil {
		return nil, mapMediaLookupError(err)
	}
	asset.DisplayName = name
	if err := s.repo.SaveMediaAsset(ctx, asset); err != nil {
		return nil, mapMediaStorageError(err)
	}
	return asset, nil
}

func (s *service) DownloadMediaAsset(ctx context.Context, id int64) (MediaDownload, error) {
	if s.repo == nil {
		return MediaDownload{}, ErrStorageUnavailable
	}
	if id <= 0 {
		return MediaDownload{}, ErrInvalidInput
	}
	asset, err := s.repo.FindMediaAssetByID(ctx, id)
	if err != nil {
		return MediaDownload{}, mapMediaLookupError(err)
	}
	if asset.External {
		return MediaDownload{}, ErrExternalMedia
	}
	if s.objectStore == nil {
		return MediaDownload{}, ErrStorageUnavailable
	}
	data, err := s.objectStore.ReadFile(asset.StorageKey)
	if err != nil {
		if os.IsNotExist(err) {
			return MediaDownload{}, ErrNotFound
		}
		return MediaDownload{}, err
	}
	filename := cleanMediaFilename(asset.DisplayName)
	if filename == "" {
		filename = asset.OriginalName
	}
	if urlpath.Ext(filename) == "" && asset.Extension != "" {
		filename += "." + strings.TrimPrefix(asset.Extension, ".")
	}
	return MediaDownload{
		ContentType: mediaContentType(asset.MIMEType),
		Data:        data,
		Filename:    filename,
	}, nil
}

func (s *service) DeleteMediaAsset(ctx context.Context, id int64) error {
	if s.repo == nil {
		return ErrStorageUnavailable
	}
	if id <= 0 {
		return ErrInvalidInput
	}
	asset, err := s.repo.FindMediaAssetByID(ctx, id)
	if err != nil {
		return mapMediaLookupError(err)
	}
	if !asset.External {
		if s.objectStore == nil {
			return ErrStorageUnavailable
		}
		if err := s.objectStore.Remove(asset.StorageKey); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err := s.repo.DeleteMediaAsset(ctx, id, s.now()); err != nil {
		return mapMediaStorageError(err)
	}
	return nil
}

func (s *service) mediaCategoryNameExists(ctx context.Context, currentID int64, parentID int64, name string) (bool, error) {
	categories, err := s.repo.ListMediaCategories(ctx)
	if err != nil {
		return false, mapMediaStorageError(err)
	}
	for _, category := range categories {
		if category.ID == currentID || category.ParentID != parentID {
			continue
		}
		if strings.EqualFold(category.Name, name) {
			return true, nil
		}
	}
	return false, nil
}

func (s *service) mediaObjectStorageStatus() string {
	if s.objectStore == nil {
		return "unavailable"
	}
	return "enabled"
}

func createsMediaCategoryCycle(ctx context.Context, repo Repository, currentID int64, parentID int64) bool {
	for parentID > 0 {
		if parentID == currentID {
			return true
		}
		parent, err := repo.FindMediaCategoryByID(ctx, parentID)
		if err != nil {
			return false
		}
		parentID = parent.ParentID
	}
	return false
}

func buildMediaCategoryTree(categories []model.MediaCategory) []model.MediaCategory {
	known := make(map[int64]struct{}, len(categories))
	for _, category := range categories {
		known[category.ID] = struct{}{}
	}
	byParent := make(map[int64][]model.MediaCategory, len(categories))
	for _, category := range categories {
		category.Children = nil
		parentID := category.ParentID
		if parentID != 0 {
			if _, ok := known[parentID]; !ok {
				parentID = 0
			}
		}
		byParent[parentID] = append(byParent[parentID], category)
	}
	var attach func(int64) []model.MediaCategory
	attach = func(parentID int64) []model.MediaCategory {
		items := append([]model.MediaCategory(nil), byParent[parentID]...)
		for i := range items {
			items[i].Children = attach(items[i].ID)
		}
		return items
	}
	return attach(0)
}

func normalizeMediaPrefix(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	value = strings.Trim(value, "/")
	if value == "" || value == "." {
		return "media"
	}
	cleaned := strings.Trim(urlpath.Clean("/"+value), "/")
	if cleaned == "" || strings.HasPrefix(cleaned, "..") {
		return "media"
	}
	return cleaned
}

func cleanMediaFilename(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	value = urlpath.Base(value)
	value = cleanMediaName(value, 255)
	if value == "." || value == "/" {
		return ""
	}
	return value
}

func cleanMediaName(value string, limit int) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\x00", ""))
	value = strings.Map(func(r rune) rune {
		if r < 32 || r == '/' || r == '\\' {
			return -1
		}
		return r
	}, value)
	runes := []rune(value)
	if limit > 0 && len(runes) > limit {
		runes = runes[:limit]
	}
	return strings.TrimSpace(string(runes))
}

func displayNameFromFilename(value string) string {
	value = cleanMediaFilename(value)
	ext := urlpath.Ext(value)
	name := strings.TrimSuffix(value, ext)
	if strings.TrimSpace(name) == "" {
		return value
	}
	return cleanMediaName(name, 255)
}

func safeMediaExtension(filename string, mimeType string) string {
	ext := strings.ToLower(urlpath.Ext(cleanMediaFilename(filename)))
	if validMediaExtension(ext) {
		return ext
	}
	if strings.HasPrefix(mimeType, "image/jpeg") {
		return ".jpg"
	}
	if strings.HasPrefix(mimeType, "image/png") {
		return ".png"
	}
	if strings.HasPrefix(mimeType, "image/gif") {
		return ".gif"
	}
	if strings.HasPrefix(mimeType, "image/webp") {
		return ".webp"
	}
	if strings.HasPrefix(mimeType, "application/pdf") {
		return ".pdf"
	}
	return ".bin"
}

func validMediaExtension(ext string) bool {
	if len(ext) < 2 || len(ext) > 16 || ext[0] != '.' {
		return false
	}
	for _, char := range ext[1:] {
		switch {
		case char >= 'a' && char <= 'z':
		case char >= '0' && char <= '9':
		default:
			return false
		}
	}
	return true
}

func normalizeMediaExternalURL(raw string) (string, *url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil, ErrInvalidInput
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed == nil || parsed.Host == "" {
		return "", nil, ErrInvalidInput
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return "", nil, ErrInvalidInput
	}
	return parsed.String(), parsed, nil
}

func mediaURLDisplayName(name string, parsed *url.URL) string {
	name = cleanMediaFilename(name)
	if name != "" {
		return name
	}
	if parsed == nil {
		return ""
	}
	name = cleanMediaFilename(parsed.Path)
	if name != "" {
		return name
	}
	return cleanMediaName(parsed.Host, 255)
}

func mediaTypeFromExtension(ext string) string {
	if ext == "" {
		return "application/octet-stream"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	if typ := mime.TypeByExtension(ext); typ != "" {
		return typ
	}
	return "application/octet-stream"
}

func mediaContentType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "application/octet-stream"
	}
	return value
}

func mapMediaLookupError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return ErrNotFound
	case isStorageUnavailable(err):
		return ErrStorageUnavailable
	default:
		return err
	}
}

func mapMediaStorageError(err error) error {
	if isStorageUnavailable(err) {
		return ErrStorageUnavailable
	}
	return err
}
