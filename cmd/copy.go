package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/spf13/cobra"
)

var copyCmd = &cobra.Command{
	Use:   "copy",
	Short: "Copy the specified log to the clipboard",
	Long:  "Copy the specified log to the clipboard",
	Run: runCommand(func(cmd *cobra.Command, dl *daylog.DayLog) error {
		logContents, err := dl.Show("text")
		if err != nil {
			return err
		}

		if err := copy([]byte(logContents)); err != nil {
			return err
		}

		fmt.Println("Copied to clipboard.")
		return nil
	}),
}

func init() {
	rootCmd.AddCommand(copyCmd)
}

func copy(content []byte) error {
	switch runtime.GOOS {
	case "darwin":
		return pbcopy(content)
	default:
		// only mac for now. help wanted for other platforms.
		return fmt.Errorf("copy not supported on %s", runtime.GOOS)
	}
}

func pbcopy(content []byte) error {
	cmd := exec.Command("pbcopy")

	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	_, err = in.Write(content)
	if err != nil {
		return err
	}

	err = in.Close()
	if err != nil {
		return err
	}

	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}
