package main

import (
	"os/exec"
	"runtime"
)

// tarCmd returns an *exec.Cmd that extracts tarPath into destDir.
// On all supported platforms we rely on the system tar (GNU tar on Linux,
// BSD tar on macOS, and tar bundled with Git for Windows / WSL).
func tarCmd(tarPath, destDir string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		// Windows 10 1803+ ships tar.exe in System32
		return exec.Command("tar", "-xzf", tarPath, "-C", destDir)
	}
	return exec.Command("tar", "-xzf", tarPath, "-C", destDir)
}
