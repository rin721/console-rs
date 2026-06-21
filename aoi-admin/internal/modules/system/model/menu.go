package model

import "time"

type MenuGroup struct {
	Code           string     `json:"code"`
	Description    string     `json:"description,omitempty"`
	DescriptionKey string     `json:"descriptionKey,omitempty"`
	Label          string     `json:"label"`
	LabelKey       string     `json:"labelKey,omitempty"`
	Order          int        `json:"order"`
	Items          []MenuItem `json:"items"`
}

type MenuItem struct {
	Code           string `json:"code"`
	Description    string `json:"description,omitempty"`
	DescriptionKey string `json:"descriptionKey,omitempty"`
	Label          string `json:"label"`
	LabelKey       string `json:"labelKey,omitempty"`
	Icon           string `json:"icon"`
	Path           string `json:"path"`
	Permission     string `json:"permission,omitempty"`
	ProductCode    string `json:"productCode,omitempty"`
	Scope          string `json:"scope,omitempty"`
	Mobile         bool   `json:"mobile"`
	Order          int    `json:"order"`
}

type APIGroup struct {
	Code  string     `json:"code"`
	Label string     `json:"label"`
	Count int        `json:"count"`
	Items []APIEntry `json:"items"`
}

type APIEntry struct {
	Access               string     `json:"access"`
	Code                 string     `json:"code"`
	Group                string     `json:"group"`
	Method               string     `json:"method"`
	Path                 string     `json:"path"`
	Description          string     `json:"description"`
	Permission           string     `json:"permission,omitempty"`
	ProductCode          string     `json:"productCode,omitempty"`
	Scope                string     `json:"scope,omitempty"`
	PermissionRegistered bool       `json:"permissionRegistered"`
	Order                int        `json:"order"`
	Synced               bool       `json:"synced"`
	SyncedAt             *time.Time `json:"syncedAt,omitempty"`
}
