package model

type ConfigSnapshot struct {
	Sections []ConfigSection `json:"sections"`
}

type PublicSettings struct {
	Brand            PublicBrandSettings `json:"brand"`
	Auth             PublicAuthSettings  `json:"auth"`
	DefaultLocale    string              `json:"defaultLocale"`
	FallbackLocale   string              `json:"fallbackLocale"`
	SupportedLocales []string            `json:"supportedLocales"`
}

type PublicBrandSettings struct {
	ProductName string `json:"productName"`
	ProductCode string `json:"productCode"`
	VersionName string `json:"versionName"`
}

type PublicAuthSettings struct {
	CSRFEnabled        bool   `json:"csrfEnabled"`
	CSRFCookieName     string `json:"csrfCookieName"`
	CSRFHeaderName     string `json:"csrfHeaderName"`
	RegistrationMode   string `json:"registrationMode"`
	ProductHeader      string `json:"productHeader"`
	ClientTypeHeader   string `json:"clientTypeHeader"`
	DefaultProductCode string `json:"defaultProductCode"`
	DefaultClientType  string `json:"defaultClientType"`
}

const (
	ConfigValueTypeArray   = "array"
	ConfigValueTypeBoolean = "boolean"
	ConfigValueTypeNumber  = "number"
	ConfigValueTypeObject  = "object"
	ConfigValueTypeString  = "string"
	ConfigValueTypeUnknown = "unknown"
)

type ConfigSection struct {
	Code           string        `json:"code"`
	Description    string        `json:"description"`
	DescriptionKey string        `json:"descriptionKey,omitempty"`
	Groups         []ConfigGroup `json:"groups"`
	Icon           string        `json:"icon"`
	Items          []ConfigItem  `json:"items"`
	Label          string        `json:"label"`
	LabelKey       string        `json:"labelKey,omitempty"`
	Order          int           `json:"order"`
}

type ConfigGroup struct {
	Description    string               `json:"description"`
	DescriptionKey string               `json:"descriptionKey,omitempty"`
	Items          []ConfigItem         `json:"items"`
	Key            string               `json:"key"`
	Label          string               `json:"label"`
	LabelKey       string               `json:"labelKey,omitempty"`
	Risk           string               `json:"risk,omitempty"`
	Testable       bool                 `json:"testable"`
	VisibleWhen    *VisibilityCondition `json:"visibleWhen,omitempty"`
}

type ConfigItem struct {
	Description    string               `json:"description"`
	DescriptionKey string               `json:"descriptionKey,omitempty"`
	Editor         string               `json:"editor,omitempty"`
	Editable       bool                 `json:"editable"`
	GroupKey       string               `json:"groupKey,omitempty"`
	Key            string               `json:"key"`
	Label          string               `json:"label"`
	LabelKey       string               `json:"labelKey,omitempty"`
	Options        []ConfigOption       `json:"options,omitempty"`
	Risk           string               `json:"risk,omitempty"`
	Secret         bool                 `json:"secret"`
	Source         string               `json:"source"`
	Testable       bool                 `json:"testable"`
	Value          any                  `json:"value"`
	ValueType      string               `json:"valueType"`
	VisibleWhen    *VisibilityCondition `json:"visibleWhen,omitempty"`
}

type ConfigOption struct {
	Description    string `json:"description,omitempty"`
	DescriptionKey string `json:"descriptionKey,omitempty"`
	Label          string `json:"label"`
	LabelKey       string `json:"labelKey,omitempty"`
	Value          string `json:"value"`
}

type VisibilityCondition struct {
	Field string   `json:"field"`
	In    []string `json:"in"`
}
