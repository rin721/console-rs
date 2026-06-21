//go:build windows

package adapters

import (
	"os/exec"
	"syscall"
)

const detachedProcess = 0x00000008

func configureDetachedProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | detachedProcess,
	}
}
