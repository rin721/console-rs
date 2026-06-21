package managed

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const managedExecutableBaseName = "go-scaffold-managed"

func (m *Manager) executable() string {
	if strings.TrimSpace(m.Executable) != "" {
		return m.Executable
	}
	executable, _ := os.Executable()
	return executable
}

// managedExecutable 返回适合后台托管的可执行文件路径。
//
// `go run` 生成的临时 main(.exe) 会在父进程退出后消失，因此需要复制到 runtime/bin 后再启动后台进程。
func (m *Manager) managedExecutable(runtimeDir string) (string, error) {
	executable := strings.TrimSpace(m.executable())
	if executable == "" {
		return "", errors.New("executable is required")
	}
	executable = filepath.Clean(executable)
	if !isGoRunTemporaryExecutable(executable) {
		return executable, nil
	}
	target := filepath.Join(runtimeDir, "bin", managedExecutableFileName(executable))
	if err := copyExecutable(executable, target); err != nil {
		return "", fmt.Errorf("prepare managed executable: copy %s to %s: %w; stop the existing managed service or build a stable binary with go build before running it in the background", executable, target, err)
	}
	return target, nil
}

// managedExecutableFileName 根据平台保留可执行文件扩展名。
func managedExecutableFileName(source string) string {
	if strings.EqualFold(filepath.Ext(source), ".exe") {
		return managedExecutableBaseName + ".exe"
	}
	return managedExecutableBaseName
}

// isGoRunTemporaryExecutable 识别 Go 工具链为 `go run` 生成的临时 main 可执行文件。
//
// 这里只通过路径形态判断，避免对普通用户构建的二进制做不必要复制。
func isGoRunTemporaryExecutable(path string) bool {
	path = filepath.Clean(path)
	base := strings.ToLower(filepath.Base(path))
	if base != "main" && base != "main.exe" {
		return false
	}
	if strings.ToLower(filepath.Base(filepath.Dir(path))) != "exe" {
		return false
	}
	dir := filepath.Dir(filepath.Dir(path))
	for {
		name := strings.ToLower(filepath.Base(dir))
		if strings.HasPrefix(name, "go-build") {
			return true
		}
		next := filepath.Dir(dir)
		if next == dir {
			return false
		}
		dir = next
	}
}

// copyExecutable 将当前可执行文件复制到稳定位置并保持可执行权限。
//
// 复制过程同样采用临时文件加 rename，降低后台启动时读到不完整二进制的风险。
func copyExecutable(source string, target string) error {
	sourceAbs, sourceErr := filepath.Abs(source)
	targetAbs, targetErr := filepath.Abs(target)
	if sourceErr == nil && targetErr == nil && sourceAbs == targetAbs {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp := target + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	_ = os.Remove(target)
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
