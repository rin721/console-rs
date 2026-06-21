package model

import "time"

const (
	DictionaryStatusActive   = "active"
	DictionaryStatusDisabled = "disabled"
)

type DictionaryCatalog struct {
	Items         []Dictionary `json:"items"`
	StorageStatus string       `json:"storageStatus"`
	Total         int          `json:"total"`
}

type Dictionary struct {
	ID          int64            `gorm:"column:id;primaryKey" json:"id,string"`
	Code        string           `gorm:"column:code;size:128;not null;uniqueIndex" json:"code"`
	Name        string           `gorm:"column:name;size:128;not null" json:"name"`
	Description string           `gorm:"column:description;type:text;not null" json:"description"`
	Status      string           `gorm:"column:status;size:32;not null" json:"status"`
	Items       []DictionaryItem `gorm:"-" json:"items"`
	CreatedAt   time.Time        `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt   time.Time        `gorm:"column:updated_at;not null" json:"updatedAt"`
	DeletedAt   *time.Time       `gorm:"column:deleted_at" json:"-"`
}

func (Dictionary) TableName() string { return "system_dictionaries" }

type DictionaryItem struct {
	ID           int64      `gorm:"column:id;primaryKey" json:"id,string"`
	DictionaryID int64      `gorm:"column:dictionary_id;not null;index" json:"dictionaryId,string"`
	Label        string     `gorm:"column:label;size:128;not null" json:"label"`
	Value        string     `gorm:"column:value;size:128;not null" json:"value"`
	Extra        string     `gorm:"column:extra;type:text;not null" json:"extra"`
	Status       string     `gorm:"column:status;size:32;not null" json:"status"`
	Sort         int        `gorm:"column:sort_order;not null" json:"sort"`
	CreatedAt    time.Time  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;not null" json:"updatedAt"`
	DeletedAt    *time.Time `gorm:"column:deleted_at" json:"-"`
}

func (DictionaryItem) TableName() string { return "system_dictionary_items" }
