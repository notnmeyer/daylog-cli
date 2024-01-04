package editor

import (
	"fmt"
	"os"
	"os/exec"
)

func chooseEditor() string {
	if _, exists := os.LookupEnv("EDITOR"); exists {
		return os.Getenv("EDITOR")
	}

	if _, exists := os.LookupEnv("VISUAL"); exists {
		return os.Getenv("VISUAL")
	}

	return "nano"
}

func Open(logFile string) error {
	editor := chooseEditor()
	cmd := exec.Command(editor, logFile)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("Error starting %s: %s", editor, err)
	}

	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}

func Read(logFile string) (string, error) {
	contents, err := os.ReadFile(logFile)
	if err != nil {
		return "", nil
	}

	return string(contents), nil
}
