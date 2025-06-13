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
    if err == nil {
        // try to kill whole process group
        killErr := syscall.Kill(-pgid, syscall.SIGKILL)
        log.Err(killErr).Msg("killing process group")
        return killErr
    }

    // could not obtain pgid, log and fallback to direct kill
    log.Err(err).Msg("could not get process group id")
    killErr := cmd.Process.Kill()
    log.Err(killErr).Msg("killing process")
    return killErr
}
