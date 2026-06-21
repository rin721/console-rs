package output

import (
	"fmt"
	"io"

	"github.com/rei0721/go-scaffold/internal/app/cliapp/localization"
	"github.com/rei0721/go-scaffold/internal/app/cliapp/services/managed"
)

// PrintServiceState 输出托管服务状态。
func PrintServiceState(w io.Writer, state managed.ServiceState, localizers ...*localization.Localizer) {
	localizer := firstLocalizer(localizers...)
	fmt.Fprintf(w, "%s: %s\n", localizer.T("cli.service.state.service"), state.Service)
	fmt.Fprintf(w, "%s: %s\n", localizer.T("cli.service.state.status"), state.Status)
	if state.PID > 0 {
		fmt.Fprintf(w, "PID: %d\n", state.PID)
	}
	if state.ListenAddr != "" {
		fmt.Fprintf(w, "%s: %s\n", localizer.T("cli.service.state.listen"), state.ListenAddr)
	}
	if state.ExecutablePath != "" {
		fmt.Fprintf(w, "%s: %s\n", localizer.T("cli.service.state.executable"), state.ExecutablePath)
	}
	if state.ConfigPath != "" {
		fmt.Fprintf(w, "%s: %s\n", localizer.T("cli.service.state.config"), state.ConfigPath)
	}
	if state.StdoutLogPath != "" {
		fmt.Fprintf(w, "stdout: %s\n", state.StdoutLogPath)
	}
	if state.StderrLogPath != "" {
		fmt.Fprintf(w, "stderr: %s\n", state.StderrLogPath)
	}
	if state.LastError != "" {
		fmt.Fprintf(w, "%s: %s\n", localizer.T("cli.service.state.error"), state.LastError)
	}
}

func firstLocalizer(localizers ...*localization.Localizer) *localization.Localizer {
	if len(localizers) > 0 && localizers[0] != nil {
		return localizers[0]
	}
	return localization.ForArgs(nil)
}
