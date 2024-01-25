package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type CmdOutput struct {
	Stdout bytes.Buffer
	Stderr bytes.Buffer
}

// returns true if dit contains a git repo
func RepoExists(dir string) (bool, error) {
	gitDir := filepath.Join(dir, ".git")

	info, err := os.Stat(gitDir)
	switch {
	case os.IsNotExist(err):
		return false, nil
	case err != nil:
		return false, err
	case !info.IsDir():
		return false, fmt.Errorf("%s is not a directory\n", gitDir)
	}

	return true, nil
}

func Init(dir string) (*CmdOutput, error) {
	output, err := run(dir, "init")
	if err != nil {
		return output, err
	}

	return output, nil
}

func Add(dir, pattern string) (*CmdOutput, error) {
	output, err := run(dir, "add", pattern)
	if err != nil {
		return output, err
	}

	return output, nil
}

func Commit(dir, msg string) (*CmdOutput, error) {
	output, err := run(dir, "commit", "-m", msg)
	if err != nil {
		return output, err
	}

	return output, nil
}

func AddAndCommit(dir, pattern, msg string) (*CmdOutput, error) {
	output, err := Add(dir, pattern)
	if err != nil {
		return output, err
	}

	output, err = Commit(dir, msg)
	if err != nil {
		return output, err
	}

	return output, nil
}

// exec a git command
func run(path string, args ...string) (*CmdOutput, error) {
	args = append([]string{"-C", path}, args...)
	cmd := exec.Command("git", args...)

	var output CmdOutput
	cmd.Stdout = &output.Stdout
	cmd.Stderr = &output.Stderr

	if err := cmd.Run(); err != nil {
		return &output, err
	}

	return &output, nil
}
