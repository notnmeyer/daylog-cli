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
		dl, err := daylog.New(args, config.Project)
		if err != nil {
			log.Fatal(err)
		}

		logContents, err := dl.Show("text")
		if err != nil {
			log.Fatalf("%s", err.Error())
		}

		// only going to worry about mac for now. this is hard to test on headless linux. help wanted.
		if runtime.GOOS != "darwin" {
			msg := fmt.Sprintf("copy not supported on %s\n", runtime.GOOS)
			log.Fatal(msg)
		}

		err = copy([]byte(logContents))
		if err != nil {
			log.Fatalf("Failed to copy to clipboard: %v", err)
		}

		fmt.Println("Copied to clipboard.")
	},
}

func init() {
	rootCmd.AddCommand(copyCmd)
}

func copy(content []byte) error {
	var err error = nil

	switch runtime.GOOS {
	case "darwin":
		err = pbcopy([]byte(content))
	}

	return err
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
