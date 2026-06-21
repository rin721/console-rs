// 本文件实现媒体文件分片续传流程，包含会话复用、分片校验、最终合并和临时文件清理。
package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	urlpath "path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rei0721/go-scaffold/internal/modules/system/model"
	appconstants "github.com/rei0721/go-scaffold/types/constants"
)

const (
	// 分片大小限制用于平衡上传恢复粒度和服务端内存占用。
	defaultMediaChunkSize = int64(1024 * 1024)
	maxMediaChunkSize     = int64(5 * 1024 * 1024)
	minMediaChunkSize     = int64(64 * 1024)
	mediaUploadTTL        = 24 * time.Hour
)

// CheckMediaResumableUploadInput 描述客户端开始或恢复分片上传前提交的文件信息。
type CheckMediaResumableUploadInput struct {
	CategoryID         int64
	ChunkSize          int64
	ChunkTotal         int
	FileHash           string
	Filename           string
	SizeBytes          int64
	UploadedBy         int64
	UploadedByUsername string
}

// UploadMediaChunkInput 描述单个分片上传请求。
//
// ChunkHash 是当前分片内容的 SHA-256，用于在落盘前发现传输损坏或客户端索引错配。
type UploadMediaChunkInput struct {
	ChunkHash          string
	ChunkIndex         int
	ChunkTotal         int
	FileHash           string
	Filename           string
	Reader             io.Reader
	SessionID          int64
	Size               int64
	UploadedBy         int64
	UploadedByUsername string
}

// CompleteMediaResumableUploadInput 描述合并分片并生成媒体资产的请求。
type CompleteMediaResumableUploadInput struct {
	FileHash   string
	SessionID  int64
	UploadedBy int64
}

// AbortMediaResumableUploadInput 描述主动终止分片上传会话的请求。
type AbortMediaResumableUploadInput struct {
	FileHash   string
	SessionID  int64
	UploadedBy int64
}

// CheckMediaResumableUpload 创建或复用一个分片上传会话。
//
// input 中 FileHash/Filename/CategoryID/UploadedBy 共同决定可复用的会话；返回值包含已上传分片和缺失分片。
// 副作用是可能创建新的上传会话，或把过期的既有会话标记为 expired。
func (s *service) CheckMediaResumableUpload(ctx context.Context, input CheckMediaResumableUploadInput) (model.MediaResumableCheckResult, error) {
	result := s.emptyResumableCheckResult()
	if s.repo == nil || s.objectStore == nil {
		return result, ErrStorageUnavailable
	}
	fileHash := normalizeMediaHash(input.FileHash)
	fileName := cleanMediaFilename(input.Filename)
	chunkSize := normalizeMediaChunkSize(input.ChunkSize)
	if fileHash == "" || fileName == "" || input.SizeBytes <= 0 || input.SizeBytes > s.cfg.MediaMaxBytes || input.CategoryID < 0 {
		return result, ErrInvalidInput
	}
	chunkTotal := input.ChunkTotal
	expectedTotal := expectedMediaChunkTotal(input.SizeBytes, chunkSize)
	if chunkTotal == 0 {
		chunkTotal = expectedTotal
	}
	if chunkTotal <= 0 || chunkTotal != expectedTotal {
		return result, ErrInvalidInput
	}
	if input.CategoryID > 0 {
		// 提前校验分类存在，避免后续上传完所有分片后才发现资产无法归类。
		if _, err := s.repo.FindMediaCategoryByID(ctx, input.CategoryID); err != nil {
			return result, mapMediaLookupError(err)
		}
	}

	session, err := s.repo.FindMediaUploadSessionByHash(ctx, fileHash, fileName, input.CategoryID, input.UploadedBy)
	if err != nil && !errors.Is(err, ErrNotFound) {
		if isStorageUnavailable(err) {
			return result, ErrStorageUnavailable
		}
		return result, err
	}
	if session != nil {
		fresh, asset, err := s.normalizeExistingMediaUploadSession(ctx, session)
		if err != nil {
			return result, err
		}
		if fresh {
			// 已完成或仍有效的会话直接返回当前状态，支持客户端无损恢复。
			return s.resumableCheckResult(ctx, session, asset)
		}
	}

	now := s.now()
	session = &model.MediaUploadSession{
		ID:                 s.ids.NextID(),
		CategoryID:         input.CategoryID,
		ChunkSize:          chunkSize,
		ChunkTotal:         chunkTotal,
		CreatedAt:          now,
		DisplayName:        displayNameFromFilename(fileName),
		ExpiresAt:          now.Add(mediaUploadTTL),
		Extension:          strings.TrimPrefix(safeMediaExtension(fileName, ""), "."),
		FileHash:           fileHash,
		FileName:           fileName,
		SizeBytes:          input.SizeBytes,
		Status:             model.MediaUploadStatusActive,
		UpdatedAt:          now,
		UploadedBy:         input.UploadedBy,
		UploadedByUsername: cleanMediaName(input.UploadedByUsername, 128),
	}
	if err := s.repo.CreateMediaUploadSession(ctx, session); err != nil {
		return result, mapMediaStorageError(err)
	}
	return s.resumableCheckResult(ctx, session, nil)
}

// UploadMediaChunk 校验并保存单个上传分片。
//
// input.Reader 会被限制为 session.ChunkSize+1 读取，防止客户端声明小尺寸但发送超大内容。
// 成功后会更新分片记录和会话更新时间，并返回最新缺失分片列表。
func (s *service) UploadMediaChunk(ctx context.Context, input UploadMediaChunkInput) (model.MediaResumableChunkResult, error) {
	result := model.MediaResumableChunkResult{StorageStatus: "unavailable"}
	if s.repo == nil || s.objectStore == nil {
		return result, ErrStorageUnavailable
	}
	session, err := s.requireMediaUploadSession(ctx, input.SessionID, input.FileHash, input.UploadedBy)
	if err != nil {
		return result, err
	}
	if input.ChunkTotal > 0 && input.ChunkTotal != session.ChunkTotal {
		return result, ErrInvalidInput
	}
	if input.ChunkIndex < 0 || input.ChunkIndex >= session.ChunkTotal || input.Reader == nil {
		return result, ErrInvalidInput
	}
	if input.Size <= 0 || input.Size > session.ChunkSize || (input.ChunkIndex < session.ChunkTotal-1 && input.Size != session.ChunkSize) {
		return result, ErrInvalidInput
	}
	// LimitReader 让服务端以真实读取结果验证 Size，避免信任客户端传入的长度字段。
	data, err := io.ReadAll(io.LimitReader(input.Reader, session.ChunkSize+1))
	if err != nil {
		return result, err
	}
	if len(data) == 0 || int64(len(data)) > session.ChunkSize || int64(len(data)) != input.Size {
		return result, ErrInvalidInput
	}
	chunkHash := normalizeMediaHash(input.ChunkHash)
	if chunkHash == "" || sha256Hex(data) != chunkHash {
		return result, ErrInvalidInput
	}

	now := s.now()
	key := mediaChunkStorageKey(s.cfg.MediaPrefix, session.ID, input.ChunkIndex)
	// 分片先写入临时 chunks 目录，只有 Complete 阶段校验全部分片后才生成正式媒体资产。
	if err := s.objectStore.MkdirAll(mediaChunkStorageDir(s.cfg.MediaPrefix, session.ID), 0755); err != nil {
		return result, err
	}
	if err := s.objectStore.WriteFile(key, data, 0644); err != nil {
		return result, err
	}
	chunk, err := s.repo.FindMediaUploadChunk(ctx, session.ID, input.ChunkIndex)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return result, mapMediaStorageError(err)
	}
	if chunk == nil {
		chunk = &model.MediaUploadChunk{
			ID:         s.ids.NextID(),
			CreatedAt:  now,
			ChunkIndex: input.ChunkIndex,
			SessionID:  session.ID,
		}
	}
	chunk.ChunkHash = chunkHash
	chunk.SizeBytes = int64(len(data))
	chunk.StorageKey = key
	chunk.UpdatedAt = now
	// 允许同一分片重复上传：已有记录走 Save，新分片走 Create，从而支持客户端重试覆盖损坏分片。
	if chunk.CreatedAt.IsZero() {
		chunk.CreatedAt = now
	}
	if chunk.ID == 0 {
		chunk.ID = s.ids.NextID()
	}
	if err == nil {
		err = s.repo.SaveMediaUploadChunk(ctx, chunk)
	} else {
		err = s.repo.CreateMediaUploadChunk(ctx, chunk)
	}
	if err != nil {
		return result, mapMediaStorageError(err)
	}
	session.UpdatedAt = now
	if err := s.repo.SaveMediaUploadSession(ctx, session); err != nil {
		return result, mapMediaStorageError(err)
	}
	chunks, err := s.repo.ListMediaUploadChunks(ctx, session.ID)
	if err != nil {
		return result, mapMediaStorageError(err)
	}
	uploaded := mediaUploadedChunkIndexes(chunks)
	return model.MediaResumableChunkResult{
		ChunkIndex:     input.ChunkIndex,
		MissingChunks:  mediaMissingChunks(session.ChunkTotal, uploaded),
		Progress:       mediaUploadProgress(session.ChunkTotal, len(uploaded)),
		SessionID:      session.ID,
		Status:         session.Status,
		StorageStatus:  "persisted",
		UploadedChunks: uploaded,
	}, nil
}

// CompleteMediaResumableUpload 合并全部分片并创建最终媒体资产。
//
// 该方法会逐个校验分片索引、大小和 SHA-256，最后再校验整文件 hash；成功后清理临时分片。
// 如果会话已经完成，会尝试返回既有资产结果，让客户端重复提交 complete 时获得幂等体验。
func (s *service) CompleteMediaResumableUpload(ctx context.Context, input CompleteMediaResumableUploadInput) (model.MediaResumableCompleteResult, error) {
	result := model.MediaResumableCompleteResult{StorageStatus: "unavailable"}
	if s.repo == nil || s.objectStore == nil {
		return result, ErrStorageUnavailable
	}
	session, err := s.requireMediaUploadSession(ctx, input.SessionID, input.FileHash, input.UploadedBy)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			completed, completedErr := s.completedMediaUploadSession(ctx, input.SessionID, input.FileHash, input.UploadedBy)
			if completedErr == nil {
				return completed, nil
			}
		}
		return result, err
	}
	chunks, err := s.repo.ListMediaUploadChunks(ctx, session.ID)
	if err != nil {
		return result, mapMediaStorageError(err)
	}
	if len(chunks) != session.ChunkTotal {
		return result, ErrInvalidInput
	}
	sort.Slice(chunks, func(i, j int) bool { return chunks[i].ChunkIndex < chunks[j].ChunkIndex })
	var merged bytes.Buffer
	for expected, chunk := range chunks {
		// 必须严格按连续索引合并，防止重复分片或缺失分片拼出错误文件。
		if chunk.ChunkIndex != expected {
			return result, ErrInvalidInput
		}
		data, err := s.objectStore.ReadFile(chunk.StorageKey)
		if err != nil {
			return result, err
		}
		if int64(len(data)) != chunk.SizeBytes || sha256Hex(data) != normalizeMediaHash(chunk.ChunkHash) {
			return result, ErrInvalidInput
		}
		if int64(merged.Len()+len(data)) > s.cfg.MediaMaxBytes {
			return result, ErrInvalidInput
		}
		_, _ = merged.Write(data)
	}
	data := merged.Bytes()
	if int64(len(data)) != session.SizeBytes || sha256Hex(data) != session.FileHash {
		return result, ErrInvalidInput
	}
	mimeType := http.DetectContentType(data)
	if detected, err := s.objectStore.DetectMIMEFromBytes(data); err == nil && strings.TrimSpace(detected) != "" {
		mimeType = detected
	}
	now := s.now()
	id := s.ids.NextID()
	ext := safeMediaExtension(session.FileName, mimeType)
	dir := urlpath.Join(normalizeMediaPrefix(s.cfg.MediaPrefix), now.Format("2006"), now.Format("01"))
	key := urlpath.Join(dir, strconv.FormatInt(id, 10)+ext)
	if err := s.objectStore.MkdirAll(dir, 0755); err != nil {
		return result, err
	}
	if err := s.objectStore.WriteFile(key, data, 0644); err != nil {
		return result, err
	}
	asset := &model.MediaAsset{
		ID:                 id,
		CategoryID:         session.CategoryID,
		CreatedAt:          now,
		DisplayName:        session.DisplayName,
		Extension:          strings.TrimPrefix(ext, "."),
		External:           false,
		MIMEType:           mimeType,
		OriginalName:       session.FileName,
		SizeBytes:          int64(len(data)),
		Source:             model.MediaSourceResumable,
		StorageKey:         key,
		UpdatedAt:          now,
		UploadedBy:         session.UploadedBy,
		UploadedByUsername: session.UploadedByUsername,
		URL:                appconstants.MediaAssetDownloadPath(id),
	}
	if err := s.repo.CreateMediaAsset(ctx, asset); err != nil {
		// 数据库写入失败时回滚刚写入的正式对象，避免留下无法被索引的孤立文件。
		_ = s.objectStore.Remove(key)
		return result, mapMediaStorageError(err)
	}
	session.FinalAssetID = asset.ID
	session.MIMEType = mimeType
	session.Extension = asset.Extension
	session.Status = model.MediaUploadStatusCompleted
	session.CompletedAt = &now
	session.UpdatedAt = now
	if err := s.repo.SaveMediaUploadSession(ctx, session); err != nil {
		return result, mapMediaStorageError(err)
	}
	s.cleanupMediaUploadChunks(ctx, session.ID)
	return model.MediaResumableCompleteResult{
		Asset:         *asset,
		SessionID:     session.ID,
		StorageStatus: "persisted",
	}, nil
}

// AbortMediaResumableUpload 标记上传会话为 aborted 并清理临时分片。
//
// 只有会话创建者可以终止会话；已完成会话不能 abort，避免误删已生成资产对应的上下文。
func (s *service) AbortMediaResumableUpload(ctx context.Context, input AbortMediaResumableUploadInput) (model.MediaResumableAbortResult, error) {
	result := model.MediaResumableAbortResult{StorageStatus: "unavailable"}
	if s.repo == nil || s.objectStore == nil {
		return result, ErrStorageUnavailable
	}
	session, err := s.repo.FindMediaUploadSessionByID(ctx, input.SessionID)
	if err != nil {
		return result, mapMediaLookupError(err)
	}
	if session.UploadedBy != input.UploadedBy || (normalizeMediaHash(input.FileHash) != "" && normalizeMediaHash(input.FileHash) != session.FileHash) {
		return result, ErrInvalidInput
	}
	if session.Status == model.MediaUploadStatusCompleted {
		return result, ErrInvalidInput
	}
	session.Status = model.MediaUploadStatusAborted
	session.UpdatedAt = s.now()
	if err := s.repo.SaveMediaUploadSession(ctx, session); err != nil {
		return result, mapMediaStorageError(err)
	}
	s.cleanupMediaUploadChunks(ctx, session.ID)
	return model.MediaResumableAbortResult{
		SessionID:     session.ID,
		Status:        session.Status,
		StorageStatus: "persisted",
	}, nil
}

// emptyResumableCheckResult 返回对象存储不可用时的统一响应结构。
func (s *service) emptyResumableCheckResult() model.MediaResumableCheckResult {
	return model.MediaResumableCheckResult{
		ChunkSize:         defaultMediaChunkSize,
		ObjectStorage:     s.mediaObjectStorageStatus(),
		StorageStatus:     "unavailable",
		UploadMaxBytes:    s.cfg.MediaMaxBytes,
		UploadMaxMB:       s.cfg.MediaMaxBytes / 1024 / 1024,
		UploadUnavailable: s.objectStore == nil,
	}
}

// normalizeExistingMediaUploadSession 判断既有会话是否仍可复用。
//
// completed 会话会尝试加载最终资产；active 但过期的会话会被标记为 expired 并要求客户端重新建会话。
func (s *service) normalizeExistingMediaUploadSession(ctx context.Context, session *model.MediaUploadSession) (bool, *model.MediaAsset, error) {
	now := s.now()
	switch session.Status {
	case model.MediaUploadStatusCompleted:
		if session.FinalAssetID <= 0 {
			return false, nil, nil
		}
		asset, err := s.repo.FindMediaAssetByID(ctx, session.FinalAssetID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return false, nil, nil
			}
			return false, nil, mapMediaLookupError(err)
		}
		return true, asset, nil
	case model.MediaUploadStatusActive:
		if now.After(session.ExpiresAt) {
			session.Status = model.MediaUploadStatusExpired
			session.UpdatedAt = now
			if err := s.repo.SaveMediaUploadSession(ctx, session); err != nil {
				return false, nil, mapMediaStorageError(err)
			}
			return false, nil, nil
		}
		return true, nil, nil
	default:
		return false, nil, nil
	}
}

// resumableCheckResult 根据会话和分片记录生成客户端恢复上传所需的状态。
func (s *service) resumableCheckResult(ctx context.Context, session *model.MediaUploadSession, asset *model.MediaAsset) (model.MediaResumableCheckResult, error) {
	chunks, err := s.repo.ListMediaUploadChunks(ctx, session.ID)
	if err != nil {
		return model.MediaResumableCheckResult{}, mapMediaStorageError(err)
	}
	uploaded := mediaUploadedChunkIndexes(chunks)
	if session.Status == model.MediaUploadStatusCompleted {
		uploaded = make([]int, 0, session.ChunkTotal)
		for i := 0; i < session.ChunkTotal; i++ {
			uploaded = append(uploaded, i)
		}
	}
	return model.MediaResumableCheckResult{
		Asset:             asset,
		ChunkSize:         session.ChunkSize,
		MissingChunks:     mediaMissingChunks(session.ChunkTotal, uploaded),
		ObjectStorage:     s.mediaObjectStorageStatus(),
		Progress:          mediaUploadProgress(session.ChunkTotal, len(uploaded)),
		Session:           *session,
		StorageStatus:     "persisted",
		UploadMaxBytes:    s.cfg.MediaMaxBytes,
		UploadMaxMB:       s.cfg.MediaMaxBytes / 1024 / 1024,
		UploadedChunks:    uploaded,
		UploadUnavailable: s.objectStore == nil,
	}, nil
}

// requireMediaUploadSession 加载并校验可继续上传的会话。
//
// sessionID、fileHash 和 uploadedBy 必须同时匹配，避免用户借用他人的 session_id 写入分片。
func (s *service) requireMediaUploadSession(ctx context.Context, sessionID int64, fileHash string, uploadedBy int64) (*model.MediaUploadSession, error) {
	if sessionID <= 0 {
		return nil, ErrInvalidInput
	}
	session, err := s.repo.FindMediaUploadSessionByID(ctx, sessionID)
	if err != nil {
		return nil, mapMediaLookupError(err)
	}
	hash := normalizeMediaHash(fileHash)
	if session.UploadedBy != uploadedBy || hash == "" || session.FileHash != hash {
		return nil, ErrInvalidInput
	}
	if session.Status != model.MediaUploadStatusActive || s.now().After(session.ExpiresAt) {
		if session.Status == model.MediaUploadStatusActive {
			// 被动发现过期时同步落库，后续 check 可直接创建新会话。
			session.Status = model.MediaUploadStatusExpired
			session.UpdatedAt = s.now()
			_ = s.repo.SaveMediaUploadSession(ctx, session)
		}
		return nil, ErrInvalidInput
	}
	return session, nil
}

// completedMediaUploadSession 返回已完成会话的最终资产结果，用于 complete 重试的幂等兜底。
func (s *service) completedMediaUploadSession(ctx context.Context, sessionID int64, fileHash string, uploadedBy int64) (model.MediaResumableCompleteResult, error) {
	session, err := s.repo.FindMediaUploadSessionByID(ctx, sessionID)
	if err != nil {
		return model.MediaResumableCompleteResult{}, mapMediaLookupError(err)
	}
	if session.UploadedBy != uploadedBy || session.FileHash != normalizeMediaHash(fileHash) || session.Status != model.MediaUploadStatusCompleted || session.FinalAssetID <= 0 {
		return model.MediaResumableCompleteResult{}, ErrInvalidInput
	}
	asset, err := s.repo.FindMediaAssetByID(ctx, session.FinalAssetID)
	if err != nil {
		return model.MediaResumableCompleteResult{}, mapMediaLookupError(err)
	}
	return model.MediaResumableCompleteResult{Asset: *asset, SessionID: session.ID, StorageStatus: "persisted"}, nil
}

// cleanupMediaUploadChunks 尽力清理会话临时分片文件和数据库记录。
//
// 清理失败不向上传主流程返回错误，避免资产已创建后因为临时垃圾清理失败导致客户端看到失败。
func (s *service) cleanupMediaUploadChunks(ctx context.Context, sessionID int64) {
	if s.objectStore != nil {
		_ = s.objectStore.RemoveAll(mediaChunkStorageDir(s.cfg.MediaPrefix, sessionID))
	}
	if s.repo != nil {
		_ = s.repo.DeleteMediaUploadChunks(ctx, sessionID)
	}
}

// normalizeMediaChunkSize 规范化客户端请求的分片大小。
//
// 返回 -1 表示分片大小越界，调用方会把它作为无效输入处理。
func normalizeMediaChunkSize(value int64) int64 {
	if value <= 0 {
		return defaultMediaChunkSize
	}
	if value < minMediaChunkSize || value > maxMediaChunkSize {
		return -1
	}
	return value
}

// expectedMediaChunkTotal 根据文件大小和分片大小计算分片总数。
//
// 返回 0 表示输入不合法或分片数过大，避免客户端用极端小分片制造过多记录。
func expectedMediaChunkTotal(sizeBytes int64, chunkSize int64) int {
	if sizeBytes <= 0 || chunkSize <= 0 {
		return 0
	}
	total := sizeBytes / chunkSize
	if sizeBytes%chunkSize != 0 {
		total++
	}
	if total > 10000 {
		return 0
	}
	return int(total)
}

// normalizeMediaHash 校验并规范化十六进制 hash 字符串。
func normalizeMediaHash(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) < 32 || len(value) > 128 {
		return ""
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'f') || (char >= '0' && char <= '9') {
			continue
		}
		return ""
	}
	return value
}

// sha256Hex 计算数据的 SHA-256 十六进制摘要。
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// mediaChunkStorageDir 返回某个上传会话的临时分片目录。
func mediaChunkStorageDir(prefix string, sessionID int64) string {
	return urlpath.Join(normalizeMediaPrefix(prefix), "chunks", strconv.FormatInt(sessionID, 10))
}

// mediaChunkStorageKey 返回单个分片的临时对象 key。
func mediaChunkStorageKey(prefix string, sessionID int64, index int) string {
	return urlpath.Join(mediaChunkStorageDir(prefix, sessionID), fmt.Sprintf("%06d.part", index))
}

// mediaUploadedChunkIndexes 返回已上传分片索引的去重有序列表。
func mediaUploadedChunkIndexes(chunks []model.MediaUploadChunk) []int {
	indexes := make([]int, 0, len(chunks))
	seen := make(map[int]struct{}, len(chunks))
	for _, chunk := range chunks {
		if chunk.ChunkIndex < 0 {
			continue
		}
		if _, ok := seen[chunk.ChunkIndex]; ok {
			continue
		}
		seen[chunk.ChunkIndex] = struct{}{}
		indexes = append(indexes, chunk.ChunkIndex)
	}
	sort.Ints(indexes)
	return indexes
}

// mediaMissingChunks 根据分片总数和已上传索引计算缺失列表。
func mediaMissingChunks(total int, uploaded []int) []int {
	if total <= 0 {
		return nil
	}
	seen := make(map[int]struct{}, len(uploaded))
	for _, index := range uploaded {
		seen[index] = struct{}{}
	}
	missing := make([]int, 0, total-len(seen))
	for i := 0; i < total; i++ {
		if _, ok := seen[i]; !ok {
			missing = append(missing, i)
		}
	}
	return missing
}

// mediaUploadProgress 计算上传进度百分比。
func mediaUploadProgress(total int, uploaded int) int {
	if total <= 0 || uploaded <= 0 {
		return 0
	}
	if uploaded >= total {
		return 100
	}
	return int(float64(uploaded) / float64(total) * 100)
}
