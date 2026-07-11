package clipboard

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
)

func Copy(content []byte) error {
	switch runtime.GOOS {
	case "darwin":
		return pipeTo(content, "pbcopy")
	case "linux":
		return linuxCopy(content)
	case "windows":
		// bare Set-Clipboard doesn't read process stdin, and powershell
		// decodes it with the console code page; force UTF-8 and pipe $input
		return pipeTo(content, "powershell", "-command",
			"[Console]::InputEncoding=[Text.UTF8Encoding]::new(); $input | Set-Clipboard")
	default:
		return fmt.Errorf("copy not supported on %s", runtime.GOOS)
	}
}

func linuxCopy(content []byte) error {
	if _, err := exec.LookPath("wl-copy"); err == nil {
		return pipeTo(content, "wl-copy")
	}
	if _, err := exec.LookPath("xclip"); err == nil {
		return pipeTo(content, "xclip", "-selection", "clipboard")
	}
	if _, err := exec.LookPath("xsel"); err == nil {
		return pipeTo(content, "xsel", "-ib")
	}
	return fmt.Errorf("copy not supported on linux: install wl-clipboard, xclip, or xsel")
}

func pipeTo(content []byte, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = bytes.NewReader(content)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("copy via %s failed: %w", name, err)
	}
	return nil
}
