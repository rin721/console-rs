package model

import "time"

type OperationRecordPage struct {
	Items         []OperationRecord `json:"items"`
	Page          int               `json:"page"`
	PageSize      int               `json:"pageSize"`
	StorageStatus string            `json:"storageStatus"`
	Total         int64             `json:"total"`
}

type OperationRecordFilter struct {
	Method      string
	Page        int
	PageSize    int
	Path        string
	Status      int
	StatusClass string
}

type OperationRecord struct {
	ID           int64     `gorm:"column:id;primaryKey" json:"id,string"`
	UserID       int64     `gorm:"column:user_id;not null;index" json:"userId,string"`
	Username     string    `gorm:"column:username;size:128;not null" json:"username"`
	IPAddress    string    `gorm:"column:ip_address;size:64;not null;index" json:"ipAddress"`
	Method       string    `gorm:"column:http_method;size:16;not null;index" json:"method"`
	Path         string    `gorm:"column:path;size:512;not null;index" json:"path"`
	Status       int       `gorm:"column:status;not null;index" json:"status"`
	LatencyMs    int64     `gorm:"column:latency_ms;not null" json:"latencyMs"`
	UserAgent    string    `gorm:"column:user_agent;type:text;not null" json:"userAgent"`
	ErrorMessage string    `gorm:"column:error_message;type:text;not null" json:"errorMessage"`
	Body         string    `gorm:"column:body;type:text;not null" json:"body"`
	Response     string    `gorm:"column:response;type:text;not null" json:"response"`
	TraceID      string    `gorm:"column:trace_id;size:128;not null;index" json:"traceId"`
	CreatedAt    time.Time `gorm:"column:created_at;not null;index" json:"createdAt"`
}

func (OperationRecord) TableName() string { return "system_operation_records" }
