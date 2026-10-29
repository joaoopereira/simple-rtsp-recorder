//go:build !windows

package recorder

import (
	"os/exec"
	"syscall"
)

// configureProcess puts ffmpeg in its own process group so Ctrl+C in the
// terminal reaches only us, and we stop ffmpeg cleanly.
func configureProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
