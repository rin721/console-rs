package output

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rei0721/go-scaffold/internal/app/cliapp/services/managed"
)

func TestPrintServiceLogsReadsHistoryAndFollowDetachesOnContext(t *testing.T) {
	dir := t.TempDir()
	stdoutPath := filepath.Join(dir, "stdout.log")
	stderrPath := filepath.Join(dir, "stderr.log")
	if err := os.WriteFile(stdoutPath, []byte("out-1\nout-2\nout-3\n"), 0o644); err != nil {
		t.Fatalf("write stdout log: %v", err)
	}
	if err := os.WriteFile(stderrPath, []byte("err-1\nerr-2\n"), 0o644); err != nil {
		t.Fatalf("write stderr log: %v", err)
	}

	var out bytes.Buffer
	state := managed.ServiceState{StdoutLogPath: stdoutPath, StderrLogPath: stderrPath}
	if err := PrintServiceLogs(context.Background(), &out, state, 2, false); err != nil {
		t.Fatalf("PrintServiceLogs(history) error = %v", err)
	}
	text := out.String()
	for _, want := range []string{"out-2", "out-3", "err-1", "err-2"} {
		if !strings.Contains(text, want) {
			t.Fatalf("history output missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "out-1") {
		t.Fatalf("history output should have tailed stdout lines:\n%s", text)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := PrintServiceLogs(ctx, &out, state, 2, true); err != nil {
		t.Fatalf("PrintServiceLogs(follow) error = %v", err)
	}
}
