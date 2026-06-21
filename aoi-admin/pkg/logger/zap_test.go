package logger

// 本测试文件固定 zap 日志封装的重载与并发访问契约，防止注释补全和后续重构改变外部可观察行为。

import (
	"sync"
	"testing"
)

// TestReload_Success 测试正常重载流程
func TestReload_Success(t *testing.T) {
	// 创建初始 logger
	cfg := &Config{
		Level:  "info",
		Format: "console",
		Output: "stdout",
	}
	log, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// 记录一条日志确保 logger 工作
	log.Info("test message before reload")

	// 使用新配置重载
	newCfg := &Config{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	}
	if err := log.Reload(newCfg); err != nil {
		t.Fatalf("failed to reload logger: %v", err)
	}

	// 重载后记录日志确保新 logger 工作
	log.Debug("test debug message after reload")
	log.Info("test info message after reload")
}

// TestReload_LevelChange 测试日志级别变更
func TestReload_LevelChange(t *testing.T) {
	// 创建 debug 级别的 logger
	cfg := &Config{
		Level:  "debug",
		Format: "console",
		Output: "stdout",
	}
	log, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// debug 级别应该记录 debug 日志
	log.Debug("this should be logged")

	// 重载为 error 级别
	newCfg := &Config{
		Level:  "error",
		Format: "console",
		Output: "stdout",
	}
	if err := log.Reload(newCfg); err != nil {
		t.Fatalf("failed to reload logger: %v", err)
	}

	// error 级别不应该记录 debug/info/warn 日志
	// (这条日志不会实际输出,但不应该 panic)
	log.Debug("this should not be logged")
	log.Info("this should not be logged")
	log.Warn("this should not be logged")
	log.Error("this should be logged")
}

// TestReload_Concurrent 测试并发场景下的线程安全
func TestReload_Concurrent(t *testing.T) {
	cfg := &Config{
		Level:  "info",
		Format: "console",
		Output: "stdout",
	}
	log, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	var wg sync.WaitGroup
	const numGoroutines = 100
	const numLogsPerGoroutine = 100

	// 启动多个 goroutine 并发记录日志
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numLogsPerGoroutine; j++ {
				log.Info("concurrent log", "goroutine", id, "iteration", j)
			}
		}(i)
	}

	// 同时在主 goroutine 中执行多次 reload
	for i := 0; i < 10; i++ {
		newCfg := &Config{
			Level:  "debug",
			Format: "console",
			Output: "stdout",
		}
		if err := log.Reload(newCfg); err != nil {
			t.Errorf("failed to reload logger: %v", err)
		}
	}

	wg.Wait()
}

// TestReload_WithContext 测试带上下文的 logger 重载
func TestReload_WithContext(t *testing.T) {
	cfg := &Config{
		Level:  "info",
		Format: "console",
		Output: "stdout",
	}
	log, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// 创建带上下文的子 logger
	childLog := log.With("component", "test", "version", "1.0")
	childLog.Info("message from child logger")

	// 重载父 logger
	newCfg := &Config{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	}
	if err := log.Reload(newCfg); err != nil {
		t.Fatalf("failed to reload logger: %v", err)
	}

	// 子 logger 应该仍然可以工作(使用旧配置)
	// 这是预期行为,因为 With 返回的是新实例
	childLog.Info("message from child logger after parent reload")

	// 父 logger 使用新配置
	log.Debug("debug message from parent after reload")
}

// TestDefault 测试默认 logger
func TestDefault(t *testing.T) {
	log := Default()
	if log == nil {
		t.Fatal("Default() returned nil")
	}

	// 默认 logger 应该能正常记录日志
	log.Debug("debug from default logger")
	log.Info("info from default logger")
}
