package model

import "time"

const (
	VersionSourceExport = "export"
	VersionSourceImport = "import"
)

type VersionPage struct {
	Items         []Version `json:"items"`
	Page          int       `json:"page"`
	PageSize      int       `json:"pageSize"`
	StorageStatus string    `json:"storageStatus"`
	Total         int64     `json:"total"`
}

type VersionFilter struct {
	EndCreatedAt   *time.Time
	Page           int
	PageSize       int
	StartCreatedAt *time.Time
	VersionCode    string
	VersionName    string
}

type Version struct {
	ID                int64      `gorm:"column:id;primaryKey" json:"id,string"`
	VersionName       string     `gorm:"column:version_name;size:128;not null" json:"versionName"`
	VersionCode       string     `gorm:"column:version_code;size:128;not null" json:"versionCode"`
	Description       string     `gorm:"column:description;type:text;not null" json:"description"`
	VersionData       string     `gorm:"column:version_data;type:text;not null" json:"-"`
	MenuCount         int        `gorm:"column:menu_count;not null" json:"menuCount"`
	APICount          int        `gorm:"column:api_count;not null" json:"apiCount"`
	DictionaryCount   int        `gorm:"column:dictionary_count;not null" json:"dictionaryCount"`
	Source            string     `gorm:"column:source;size:32;not null" json:"source"`
	CreatedBy         int64      `gorm:"column:created_by;not null" json:"createdBy,string"`
	CreatedByUsername string     `gorm:"column:created_by_username;size:128;not null" json:"createdByUsername"`
	CreatedAt         time.Time  `gorm:"column:created_at;not null;index" json:"createdAt"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;not null" json:"updatedAt"`
	DeletedAt         *time.Time `gorm:"column:deleted_at" json:"-"`
}

func (Version) TableName() string { return "system_versions" }

type VersionPackageInfo struct {
	Code        string    `json:"code"`
	Description string    `json:"description"`
	ExportTime  time.Time `json:"exportTime"`
	Name        string    `json:"name"`
}

type VersionPackage struct {
	APIs         []APIEntry         `json:"apis"`
	Dictionaries []Dictionary       `json:"dictionaries"`
	Menus        []MenuGroup        `json:"menus"`
	Version      VersionPackageInfo `json:"version"`
}

type VersionDetail struct {
	Item    Version        `json:"item"`
	Package VersionPackage `json:"package"`
}

type VersionSourceCatalog struct {
	APICount        int          `json:"apiCount"`
	APIs            []APIGroup   `json:"apis"`
	Dictionaries    []Dictionary `json:"dictionaries"`
	DictionaryCount int          `json:"dictionaryCount"`
	MenuCount       int          `json:"menuCount"`
	Menus           []MenuGroup  `json:"menus"`
	StorageStatus   string       `json:"storageStatus"`
}

type VersionImportResult struct {
	APIsSkipped            int       `json:"apisSkipped"`
	DictionariesCreated    int       `json:"dictionariesCreated"`
	DictionariesSkipped    int       `json:"dictionariesSkipped"`
	DictionaryItemsCreated int       `json:"dictionaryItemsCreated"`
	ImportedAt             time.Time `json:"importedAt"`
	Item                   Version   `json:"item"`
	MenusSkipped           int       `json:"menusSkipped"`
	StorageStatus          string    `json:"storageStatus"`
}
