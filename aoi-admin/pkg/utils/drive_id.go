package utils

// 本文件属于通用工具层，提供无业务状态的地址校验、端口选择、设备标识或国际化代理能力。

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"os"
	"runtime"
	"strings"
)

// getMACAddresses 获取真实网卡 MAC 地址
// 过滤回环、虚拟、空 MAC
func getMACAddresses() []string {
	var macs []string

	interfaces, err := net.Interfaces()
	if err != nil {
		return macs
	}

	for _, iface := range interfaces {
		// 跳过关闭的、回环的
		if iface.Flags&net.FlagUp == 0 ||
			iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// 跳过虚拟网卡（经验规则）
		name := strings.ToLower(iface.Name)
		if strings.Contains(name, "virtual") ||
			strings.Contains(name, "vm") ||
			strings.Contains(name, "docker") {
			continue
		}

		mac := iface.HardwareAddr.String()
		if mac != "" {
			macs = append(macs, mac)
		}
	}

	return macs
}

// buildFingerprintSource 构建设备指纹源数据
func buildFingerprintSource(appSalt string) string {
	var parts []string

	// 操作系统信息
	parts = append(parts, runtime.GOOS)
	parts = append(parts, runtime.GOARCH)

	// 主机名
	hostname, err := os.Hostname()
	if err == nil {
		parts = append(parts, hostname)
	}

	// MAC 地址
	macs := getMACAddresses()
	parts = append(parts, macs...)

	// 应用盐值（强烈建议固定写死或来自服务端）
	parts = append(parts, appSalt)

	return strings.Join(parts, "|")
}

// GenerateDeviceID 生成设备码
//
// appSalt 参数：
// - 用于防止同一台机器在不同软件中生成相同设备码
// - 一旦发布不要轻易修改
func GenerateDeviceID(appSalt string) string {
	source := buildFingerprintSource(appSalt)

	hash := sha256.Sum256([]byte(source))

	// 返回 hex 编码，适合传输 / 存储
	return hex.EncodeToString(hash[:])
}
