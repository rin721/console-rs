package config

import (
	"errors"
	"strings"
)

type BrandConfig struct {
	ProductName string `mapstructure:"productName" envname:"BRAND_PRODUCT_NAME" json:"productName" yaml:"productName" toml:"productName"`
	ProductCode string `mapstructure:"productCode" envname:"BRAND_PRODUCT_CODE" json:"productCode" yaml:"productCode" toml:"productCode"`
	VersionName string `mapstructure:"versionName" envname:"BRAND_VERSION_NAME" json:"versionName" yaml:"versionName" toml:"versionName"`
}

func (c *BrandConfig) ValidateName() string {
	return AppBrandName
}

func (c *BrandConfig) ValidateRequired() bool {
	return true
}

func (c *BrandConfig) Validate() error {
	if strings.TrimSpace(c.ProductName) == "" {
		return errors.New("productName is required")
	}
	if strings.TrimSpace(c.ProductCode) == "" {
		return errors.New("productCode is required")
	}
	if strings.TrimSpace(c.VersionName) == "" {
		return errors.New("versionName is required")
	}
	return nil
}

func (c BrandConfig) TemplateData() map[string]any {
	return map[string]any{
		"ProductName": c.ProductName,
		"ProductCode": c.ProductCode,
		"VersionName": c.VersionName,
	}
}
