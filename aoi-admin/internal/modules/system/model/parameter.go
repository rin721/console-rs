package model

import "time"

type ParameterPage struct {
	Items         []Parameter `json:"items"`
	Page          int         `json:"page"`
	PageSize      int         `json:"pageSize"`
	StorageStatus string      `json:"storageStatus"`
	Total         int64       `json:"total"`
}

type ParameterFilter struct {
	EndCreatedAt   *time.Time
	Key            string
	Name           string
	Page           int
	PageSize       int
	StartCreatedAt *time.Time
}

type Parameter struct {
	ID          int64      `gorm:"column:id;primaryKey" json:"id,string"`
	Name        string     `gorm:"column:name;size:128;not null" json:"name"`
	Key         string     `gorm:"column:param_key;size:128;not null;uniqueIndex" json:"key"`
	Value       string     `gorm:"column:param_value;type:text;not null" json:"value"`
	Description string     `gorm:"column:description;type:text;not null" json:"description"`
	CreatedAt   time.Time  `gorm:"column:created_at;not null;index" json:"createdAt"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;not null" json:"updatedAt"`
	DeletedAt   *time.Time `gorm:"column:deleted_at" json:"-"`
}

func (Parameter) TableName() string { return "system_parameters" }
