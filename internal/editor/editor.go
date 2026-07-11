package editor

import (
	"fmt"
	"os"
	"os/exec"
)

func chooseEditor() (string, error) {
	if v := os.Getenv("VISUAL"); v != "" {
		return v, nil
	}

	if v := os.Getenv("EDITOR"); v != "" {
		return v, nil
	}

	if _, err := exec.LookPath("nano"); err == nil {
		return "nano", nil
	}

	return "", fmt.Errorf("couldn't find an editor. try setting $EDITOR or $VISUAL to your preferred editor.")
}

// Command returns an unstarted editor command with no stdio wired,
// so callers like tea.ExecProcess can attach their own
func Command(logFile string) (*exec.Cmd, error) {
	editor, err := chooseEditor()
	if err != nil {
		return nil, err
	}

	return exec.Command(editor, logFile), nil
}

func Open(logFile string) error {
	cmd, err := Command(logFile)
	if err != nil {
		return err
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("Error starting %s: %s", cmd.Args[0], err)
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
		return "", err
	}

	return string(contents), nil
}
