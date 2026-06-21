package constants

// 本文件定义跨前后端文档、后端路由目录和服务端生成链接共享的 HTTP API 路径契约。

import (
	urlpath "path"
	"strconv"
	"strings"
)

const (
	// HTTPHealthPath 定义轻量存活探针的稳定路径。
	HTTPHealthPath = "/health"
	// HTTPReadyPath 定义就绪探针的稳定路径。
	HTTPReadyPath = "/ready"

	// APIPathRoot 定义 HTTP API 命名空间根路径，用于避免 WebUI 挂载覆盖 API。
	APIPathRoot = "/api"
	// APIBasePath 定义当前公开 HTTP API 版本前缀；这是外部契约，不是运行时配置。
	APIBasePath = APIPathRoot + "/v1"
	// APIBasePrefix 定义带尾斜杠的 API 前缀，用于判定具体业务接口路径。
	APIBasePrefix = APIBasePath + "/"

	// SystemMediaAssetsAPIPath 定义媒体资源 API 集合路径，服务端生成下载 URL 时复用。
	SystemMediaAssetsAPIPath = APIBasePath + "/system/media/assets"
)

// APIPath 以当前 API 版本前缀拼接业务路径，避免各层重复手写 `/api/v1`。
func APIPath(parts ...string) string {
	if len(parts) == 0 {
		return APIBasePath
	}
	segments := make([]string, 0, len(parts)+1)
	segments = append(segments, APIBasePath)
	for _, part := range parts {
		part = strings.Trim(strings.TrimSpace(part), "/")
		if part == "" {
			continue
		}
		segments = append(segments, part)
	}
	return urlpath.Join(segments...)
}

// IsAPIPath 判断路径是否位于当前业务 API 前缀下；裸 `/api/v1` 不代表业务接口。
func IsAPIPath(path string) bool {
	return strings.HasPrefix(path, APIBasePrefix)
}

// TrimAPIPathPrefix 去掉当前业务 API 前缀，用于从路由路径推导 API 分组。
func TrimAPIPathPrefix(path string) string {
	return strings.TrimPrefix(path, APIBasePrefix)
}

// MediaAssetDownloadPath 生成本地媒体资源下载接口路径，保持资产记录中的 URL 与路由契约一致。
func MediaAssetDownloadPath(assetID int64) string {
	return APIPath("system", "media", "assets", strconv.FormatInt(assetID, 10), "download")
}
