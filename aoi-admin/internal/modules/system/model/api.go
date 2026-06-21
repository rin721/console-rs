package model

import "time"

const (
	APIStatusActive = "active"
	APIStatusStale  = "stale"

	APIAccessAuthenticated = "authenticated"
	APIAccessPermission    = "permission"
	APIAccessPublic        = "public"
)

type APIRecord struct {
	ID          int64     `gorm:"column:id;primaryKey" json:"id"`
	Code        string    `gorm:"column:code;size:256;not null" json:"code"`
	Group       string    `gorm:"column:api_group;size:64;not null" json:"group"`
	Method      string    `gorm:"column:http_method;size:16;not null;uniqueIndex:idx_system_api_method_path" json:"method"`
	Path        string    `gorm:"column:path;size:512;not null;uniqueIndex:idx_system_api_method_path" json:"path"`
	Description string    `gorm:"column:description;type:text;not null" json:"description"`
	Permission  string    `gorm:"column:permission;size:128;not null" json:"permission"`
	ProductCode string    `gorm:"column:product_code;size:64;not null;index" json:"productCode"`
	Scope       string    `gorm:"column:scope;size:32;not null;index" json:"scope"`
	Status      string    `gorm:"column:status;size:32;not null" json:"status"`
	Source      string    `gorm:"column:source;size:64;not null" json:"source"`
	SyncedAt    time.Time `gorm:"column:synced_at;not null" json:"syncedAt"`
	CreatedAt   time.Time `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (APIRecord) TableName() string { return "system_apis" }

type APISyncResult struct {
	Created       int        `json:"created"`
	Groups        []APIGroup `json:"groups"`
	Persisted     bool       `json:"persisted"`
	Stale         int        `json:"stale"`
	StorageStatus string     `json:"storageStatus"`
	SyncedAt      time.Time  `json:"syncedAt"`
	Total         int        `json:"total"`
	Updated       int        `json:"updated"`
}
