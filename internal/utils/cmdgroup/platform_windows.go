//go:build windows
// +build windows

package cmdgroup

import (
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

func platformConfigure(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

func platformKill(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	// TASKKILL /T /PID <pid>
	kill := exec.Command("TASKKILL", "/T", "/PID", strconv.Itoa(cmd.Process.Pid))
	kill.Stdout = os.Stdout
	kill.Stderr = os.Stderr
	return kill.Run()
}
