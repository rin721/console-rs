package model

import "time"

const (
	MediaSourceResumable = "resumable"
	MediaSourceUpload    = "upload"
	MediaSourceURL       = "url"
)

const (
	MediaUploadStatusAborted   = "aborted"
	MediaUploadStatusActive    = "active"
	MediaUploadStatusCompleted = "completed"
	MediaUploadStatusExpired   = "expired"
)

type MediaCategory struct {
	ID        int64           `gorm:"column:id;primaryKey" json:"id,string"`
	ParentID  int64           `gorm:"column:parent_id;not null;index" json:"parentId,string"`
	Name      string          `gorm:"column:name;size:128;not null" json:"name"`
	Sort      int             `gorm:"column:sort_order;not null" json:"sort"`
	CreatedAt time.Time       `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt time.Time       `gorm:"column:updated_at;not null" json:"updatedAt"`
	DeletedAt *time.Time      `gorm:"column:deleted_at" json:"-"`
	Children  []MediaCategory `gorm:"-" json:"children,omitempty"`
}

func (MediaCategory) TableName() string { return "system_media_categories" }

type MediaCategoryCatalog struct {
	Items         []MediaCategory `json:"items"`
	StorageStatus string          `json:"storageStatus"`
	Total         int             `json:"total"`
}

type MediaAsset struct {
	ID                 int64      `gorm:"column:id;primaryKey" json:"id,string"`
	CategoryID         int64      `gorm:"column:category_id;not null;index" json:"categoryId,string"`
	DisplayName        string     `gorm:"column:display_name;size:255;not null" json:"displayName"`
	OriginalName       string     `gorm:"column:original_name;size:255;not null" json:"originalName"`
	StorageKey         string     `gorm:"column:storage_key;size:512;not null" json:"storageKey"`
	URL                string     `gorm:"column:url;type:text;not null" json:"url"`
	MIMEType           string     `gorm:"column:mime_type;size:128;not null" json:"mimeType"`
	Extension          string     `gorm:"column:extension;size:32;not null" json:"extension"`
	SizeBytes          int64      `gorm:"column:size_bytes;not null" json:"sizeBytes"`
	Source             string     `gorm:"column:source;size:32;not null" json:"source"`
	External           bool       `gorm:"column:external;not null" json:"external"`
	UploadedBy         int64      `gorm:"column:uploaded_by;not null" json:"uploadedBy,string"`
	UploadedByUsername string     `gorm:"column:uploaded_by_username;size:128;not null" json:"uploadedByUsername"`
	CreatedAt          time.Time  `gorm:"column:created_at;not null;index" json:"createdAt"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;not null" json:"updatedAt"`
	DeletedAt          *time.Time `gorm:"column:deleted_at" json:"-"`
}

func (MediaAsset) TableName() string { return "system_media_assets" }

type MediaAssetFilter struct {
	CategoryID int64
	Keyword    string
	Page       int
	PageSize   int
}

type MediaAssetPage struct {
	Items             []MediaAsset `json:"items"`
	ObjectStorage     string       `json:"objectStorage"`
	Page              int          `json:"page"`
	PageSize          int          `json:"pageSize"`
	StorageStatus     string       `json:"storageStatus"`
	Total             int64        `json:"total"`
	UploadMaxBytes    int64        `json:"uploadMaxBytes"`
	UploadMaxMB       int64        `json:"uploadMaxMb"`
	UploadUnavailable bool         `json:"uploadUnavailable"`
}

type MediaURLImportResult struct {
	Items         []MediaAsset `json:"items"`
	Imported      int          `json:"imported"`
	StorageStatus string       `json:"storageStatus"`
}

type MediaUploadSession struct {
	ID                 int64      `gorm:"column:id;primaryKey" json:"id,string"`
	CategoryID         int64      `gorm:"column:category_id;not null;index" json:"categoryId,string"`
	FileHash           string     `gorm:"column:file_hash;size:128;not null;index" json:"fileHash"`
	FileName           string     `gorm:"column:file_name;size:255;not null" json:"fileName"`
	DisplayName        string     `gorm:"column:display_name;size:255;not null" json:"displayName"`
	MIMEType           string     `gorm:"column:mime_type;size:128;not null" json:"mimeType"`
	Extension          string     `gorm:"column:extension;size:32;not null" json:"extension"`
	SizeBytes          int64      `gorm:"column:size_bytes;not null" json:"sizeBytes"`
	ChunkSize          int64      `gorm:"column:chunk_size;not null" json:"chunkSize"`
	ChunkTotal         int        `gorm:"column:chunk_total;not null" json:"chunkTotal"`
	Status             string     `gorm:"column:status;size:32;not null;index" json:"status"`
	FinalAssetID       int64      `gorm:"column:final_asset_id;not null" json:"finalAssetId,string"`
	UploadedBy         int64      `gorm:"column:uploaded_by;not null;index" json:"uploadedBy,string"`
	UploadedByUsername string     `gorm:"column:uploaded_by_username;size:128;not null" json:"uploadedByUsername"`
	ExpiresAt          time.Time  `gorm:"column:expires_at;not null;index" json:"expiresAt"`
	CompletedAt        *time.Time `gorm:"column:completed_at" json:"completedAt,omitempty"`
	CreatedAt          time.Time  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;not null" json:"updatedAt"`
	DeletedAt          *time.Time `gorm:"column:deleted_at" json:"-"`
}

func (MediaUploadSession) TableName() string { return "system_media_upload_sessions" }

type MediaUploadChunk struct {
	ID         int64     `gorm:"column:id;primaryKey" json:"id,string"`
	SessionID  int64     `gorm:"column:session_id;not null;index" json:"sessionId,string"`
	ChunkIndex int       `gorm:"column:chunk_index;not null" json:"chunkIndex"`
	ChunkHash  string    `gorm:"column:chunk_hash;size:128;not null" json:"chunkHash"`
	StorageKey string    `gorm:"column:storage_key;size:512;not null" json:"storageKey"`
	SizeBytes  int64     `gorm:"column:size_bytes;not null" json:"sizeBytes"`
	CreatedAt  time.Time `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt  time.Time `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (MediaUploadChunk) TableName() string { return "system_media_upload_chunks" }

type MediaResumableCheckResult struct {
	Asset             *MediaAsset        `json:"asset,omitempty"`
	ChunkSize         int64              `json:"chunkSize"`
	MissingChunks     []int              `json:"missingChunks"`
	ObjectStorage     string             `json:"objectStorage"`
	Progress          int                `json:"progress"`
	Session           MediaUploadSession `json:"session"`
	StorageStatus     string             `json:"storageStatus"`
	UploadMaxBytes    int64              `json:"uploadMaxBytes"`
	UploadMaxMB       int64              `json:"uploadMaxMb"`
	UploadedChunks    []int              `json:"uploadedChunks"`
	UploadUnavailable bool               `json:"uploadUnavailable"`
}

type MediaResumableChunkResult struct {
	ChunkIndex     int    `json:"chunkIndex"`
	MissingChunks  []int  `json:"missingChunks"`
	Progress       int    `json:"progress"`
	SessionID      int64  `json:"sessionId,string"`
	Status         string `json:"status"`
	StorageStatus  string `json:"storageStatus"`
	UploadedChunks []int  `json:"uploadedChunks"`
}

type MediaResumableCompleteResult struct {
	Asset         MediaAsset `json:"asset"`
	SessionID     int64      `json:"sessionId,string"`
	StorageStatus string     `json:"storageStatus"`
}

type MediaResumableAbortResult struct {
	SessionID     int64  `json:"sessionId,string"`
	Status        string `json:"status"`
	StorageStatus string `json:"storageStatus"`
}
