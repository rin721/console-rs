package handlers

import "github.com/rei0721/go-scaffold/internal/app/cliapp/services/managed"

var newManagedManager = managed.NewManager

// SetManagedManagerFactoryForTest 临时替换托管服务管理器工厂，供包外测试注入假进程运行器。
func SetManagedManagerFactoryForTest(factory func() *managed.Manager) func() {
	previous := newManagedManager
	newManagedManager = factory
	return func() {
		newManagedManager = previous
	}
}
