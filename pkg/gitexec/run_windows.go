//go:build windows

package gitexec

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

//nolint:unused
func withSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.DETACHED_PROCESS,
	}
}
