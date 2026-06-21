package utils

import (
	"net"
	"net/netip"
	"strings"

	"github.com/rei0721/go-scaffold/pkg/web"
)

// ClientIPRealIP 按常见代理头优先级解析真实客户端 IP，最后回退到路由框架和 RemoteAddr。
// 返回值已经过规范化；无法识别有效 IP 时返回空字符串。
func ClientIPRealIP(c web.Context) string {
	for _, candidate := range []string{
		forwardedForHeader(c.GetHeader("Forwarded")),
		firstForwardedIP(c.GetHeader("X-Forwarded-For")),
		c.GetHeader("X-Real-IP"),
		c.GetHeader("CF-Connecting-IP"),
		c.GetHeader("True-Client-IP"),
		c.ClientIP(),
		remoteAddrIP(c),
	} {
		if ip := normalizeIP(candidate); ip != "" {
			return ip
		}
	}
	return ""
}

// firstForwardedIP 读取 X-Forwarded-For 的首个有效 IP，遵循代理链中最左侧为原始客户端的约定。
func firstForwardedIP(value string) string {
	for _, part := range strings.Split(value, ",") {
		if normalizeIP(part) != "" {
			return part
		}
	}
	return ""
}

// forwardedForHeader 解析 RFC 7239 Forwarded 头中的 for= 段，兼容多个代理元素。
func forwardedForHeader(value string) string {
	for _, element := range strings.Split(value, ",") {
		for _, part := range strings.Split(element, ";") {
			key, val, ok := strings.Cut(strings.TrimSpace(part), "=")
			if !ok || !strings.EqualFold(key, "for") {
				continue
			}
			if normalizeIP(val) != "" {
				return val
			}
		}
	}
	return ""
}

// remoteAddrIP 读取标准库 RemoteAddr，作为没有代理头时的最后兜底来源。
func remoteAddrIP(c web.Context) string {
	req := c.Request()
	if req == nil {
		return ""
	}
	return req.RemoteAddr
}

// normalizeIP 清理引号、unknown、host:port 和 IPv6 方括号，确保调用方只拿到标准 IP 字符串。
func normalizeIP(value string) string {
	value = strings.Trim(strings.TrimSpace(value), `"`)
	if value == "" || strings.EqualFold(value, "unknown") {
		return ""
	}
	if ip, err := netip.ParseAddr(value); err == nil {
		return ip.String()
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		if ip, err := netip.ParseAddr(strings.Trim(host, "[]")); err == nil {
			return ip.String()
		}
	}
	if ip, err := netip.ParseAddr(strings.Trim(value, "[]")); err == nil {
		return ip.String()
	}
	return ""
}
