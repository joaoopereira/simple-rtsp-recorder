package recorder

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// configureProcess hides ffmpeg's console window and keeps Ctrl+C in the
// parent console from killing it before we can stop it cleanly.
func configureProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW | windows.CREATE_NEW_PROCESS_GROUP,
	}
}
