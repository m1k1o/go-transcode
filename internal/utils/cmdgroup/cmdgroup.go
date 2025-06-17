package cmdgroup

import "os/exec"

// Configure applies platform-specific settings so the command starts in its own
// process-group / job-object. Call this before cmd.Start().
func Configure(cmd *exec.Cmd) {
	platformConfigure(cmd)
}

// Kill terminates the command together with all of its children.
// It is safe to call this even if the command has already exited.
func Kill(cmd *exec.Cmd) error {
	return platformKill(cmd)
}
