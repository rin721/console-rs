package output

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rei0721/go-scaffold/internal/app/cliapp/services/managed"
)

// PrintServiceLogs 输出 stdout/stderr 历史日志，并可持续跟随新增内容。
func PrintServiceLogs(ctx context.Context, w io.Writer, state managed.ServiceState, lines int, follow bool) error {
	if lines <= 0 {
		lines = 100
	}
	printLogHistory(w, "stdout", state.StdoutLogPath, lines)
	printLogHistory(w, "stderr", state.StderrLogPath, lines)
	if !follow {
		return nil
	}
	fmt.Fprintln(w, "\n--- following logs; press Ctrl+C to detach ---")
	offsets := map[string]int64{
		state.StdoutLogPath: fileSize(state.StdoutLogPath),
		state.StderrLogPath: fileSize(state.StderrLogPath),
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			for label, path := range map[string]string{"stdout": state.StdoutLogPath, "stderr": state.StderrLogPath} {
				nextOffset, err := printNewLogContent(w, label, path, offsets[path])
				if err != nil {
					continue
				}
				offsets[path] = nextOffset
			}
		}
	}
}

func printLogHistory(w io.Writer, label string, path string, lines int) {
	fmt.Fprintf(w, "\n--- %s: %s ---\n", label, path)
	items, err := tailLines(path, lines)
	if err != nil {
		fmt.Fprintf(w, "%v\n", err)
		return
	}
	for _, item := range items {
		fmt.Fprintln(w, item)
	}
}

func tailLines(path string, limit int) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > limit {
			copy(lines, lines[1:])
			lines = lines[:limit]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func printNewLogContent(w io.Writer, label string, path string, offset int64) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return offset, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return offset, err
	}
	if info.Size() < offset {
		offset = 0
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return offset, err
	}
	raw, err := io.ReadAll(file)
	if err != nil {
		return offset, err
	}
	if len(raw) == 0 {
		return info.Size(), nil
	}
	text := strings.TrimRight(string(raw), "\n")
	for _, line := range strings.Split(text, "\n") {
		fmt.Fprintf(w, "[%s] %s\n", label, line)
	}
	return info.Size(), nil
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
