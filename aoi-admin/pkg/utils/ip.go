package utils

// 本文件属于通用工具层，提供无业务状态的地址校验、端口选择、设备标识或国际化代理能力。

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// IsValidHTTPListenAddr 判断 addr 是否是一个可以被 http.Server 真实绑定的地址
//
// 核心原则：
// 只要 net.Listen("tcp", addr) 能成功，就认为是合法
//
// 它可以精准拦截:
//   - 127.0.1
//   - 127.0.0
//   - abc
//   - 非本机IP
//   - 错误IPv6
func IsValidHTTPListenAddr(addr string) error {
	if strings.TrimSpace(addr) == "" {
		return errors.New("empty listen addr")
	}

	// 直接尝试 bind
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	_ = l.Close()
	return nil
}

// IsValidListenAddr 校验 addr 是否是一个合法且可被 http.Server 绑定的监听地址
//
// 允许:
//
//	:8080
//	0.0.0.0:8080
//	127.0.0.1:8080
//	localhost:8080
//	[::]:8080
//	本机网卡IP:端口
//
// 禁止:
//
//	公网IP
//	非本机IP
//	非法host
//	非法端口
func IsValidListenAddr(addr string) error {
	if addr == "" {
		return errors.New("empty listen addr")
	}

	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		// 允许 ":8080"
		if strings.HasPrefix(addr, ":") {
			portStr = addr[1:]
			host = ""
		} else {
			return fmt.Errorf("invalid addr format: %w", err)
		}
	}

	// 校验端口
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return errors.New("invalid port")
	}

	// 空 host 表示监听所有网卡
	if host == "" {
		return nil
	}

	// localhost 视为本机地址。
	if strings.EqualFold(host, "localhost") {
		return nil
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return errors.New("invalid ip or hostname")
	}

	// 0.0.0.0 / :: 表示监听所有网卡。
	if ip.IsUnspecified() {
		return nil
	}

	// loopback 地址表示本机回环。
	if ip.IsLoopback() {
		return nil
	}

	// 必须是本机真实IP
	if !isLocalIP(ip) {
		return fmt.Errorf("ip %s is not bound to this machine", ip.String())
	}

	return nil
}

// 判断 IP 是否属于本机网卡
func isLocalIP(ip net.IP) bool {
	ifaces, err := net.Interfaces()
	if err != nil {
		return false
	}

	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			var localIP net.IP

			switch v := addr.(type) {
			case *net.IPNet:
				localIP = v.IP
			case *net.IPAddr:
				localIP = v.IP
			}

			if localIP != nil && localIP.Equal(ip) {
				return true
			}
		}
	}
	return false
}
