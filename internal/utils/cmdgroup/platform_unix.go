//go:build !windows
// +build !windows

package cmdgroup

import (
	"os/exec"
	"syscall"

	"github.com/rs/zerolog/log"
)

func platformConfigure(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func platformKill(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		// could not obtain pgid, log and fallback to direct kill
		log.Err(err).Msg("could not get process group id")
		return cmd.Process.Kill()
	}

	return syscall.Kill(-pgid, syscall.SIGKILL)
}
