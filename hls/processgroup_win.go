//go:build windows
// +build windows

package hls

import (
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

func ConfigureAsProcessGroup() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

func TaskkillWithChildrenWindows(cmd *exec.Cmd) error {
	// Function adopted from: https://stackoverflow.com/a/44551450/6278
	// Taskkill command documentation: https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/taskkill

	kill := exec.Command("TASKKILL", "/T", "/PID", strconv.Itoa(cmd.Process.Pid))
	kill.Stderr = os.Stderr
	kill.Stdout = os.Stdout
	return kill.Run()
}

func (m *ManagerCtx) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil && m.cmd.Process != nil {
		m.logger.Debug().Msg("performing stop")

		err := TaskkillWithChildrenWindows(m.cmd)
		if err == nil {
			m.logger.Debug().Msg("killing process group")
		} else {
			m.logger.Err(err).Msg("failed to kill process group")
		}
	}
}
