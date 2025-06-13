//go:build windows
// +build windows

package cmdgroup

import (
    "os"
    "os/exec"
    "strconv"
    "syscall"
    "github.com/rs/zerolog/log"
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
    err := kill.Run()
    if err == nil {
        log.Debug().Msg("killing process group")
    } else {
        log.Err(err).Msg("failed to kill process group")
    }
    return err
}
