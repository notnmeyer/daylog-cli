package cmd

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"

	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/spf13/cobra"
)

var copyCmd = &cobra.Command{
	Use:   "copy",
	Short: "Copy the specified log to the clipboard",
	Long:  "Copy the specified log to the clipboard",
	Run: func(cmd *cobra.Command, args []string) {
		projectPath, err := daylog.EnsureProjectPath(config.Project)
		if err != nil {
			log.Fatal(err)
		}

		dl, err := daylog.New(args, projectPath)
		if err != nil {
			log.Fatal(err)
		}

		logContents, err := dl.Show("text")
		if err != nil {
			log.Fatal(err)
		}

		if err := copy([]byte(logContents)); err != nil {
			log.Fatal(err)
		}

		fmt.Println("Copied to clipboard.")
	},
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
