package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/spf13/cobra"
)

var copyCmd = &cobra.Command{
	Use:   "copy",
	Short: "Copy the specified log to the clipboard",
	Long:  "Copy the specified log to the clipboard",
	Example: `
		daylog copy
		daylog copy -- yesterday
		daylog copy --prev
		daylog copy -p work -- 2023/01/07
    `,

	Run: runCommand(func(cmd *cobra.Command, dl *daylog.DayLog) error {
		logContents, err := dl.Show("text")
		if err != nil {
			return err
		}

		if err := copy([]byte(logContents)); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Copied to clipboard.")
		return nil
	}),
}

func init() {
	rootCmd.AddCommand(copyCmd)
}

func copy(content []byte) error {
	switch runtime.GOOS {
	case "darwin":
		return pipeTo(content, "pbcopy")
	case "linux":
		return linuxCopy(content)
	case "windows":
		return pipeTo(content, "powershell", "-command", "Set-Clipboard")
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
	return cmd.Run()
}
