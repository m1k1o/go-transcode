//go:build !windows
// +build !windows

package hls

import "syscall"

func ConfigureAsProcessGroup() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

func (m *ManagerCtx) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil && m.cmd.Process != nil {
		m.logger.Debug().Msg("performing stop")

		pgid, err := syscall.Getpgid(m.cmd.Process.Pid)
		if err == nil {
			err := syscall.Kill(-pgid, syscall.SIGKILL)
			m.logger.Err(err).Msg("killing process group")
		} else {
			m.logger.Err(err).Msg("could not get process group id")
			err := m.cmd.Process.Kill()
			m.logger.Err(err).Msg("killing process")
		}
	}
}
