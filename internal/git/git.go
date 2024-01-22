package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

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

func Init(dir string) (string, error) {
	err := runCmd("init", dir)
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, ".git"), nil
}

func Add(dir, pattern string) error {
	err := runCmd("-C", dir, "add", pattern)
	if err != nil {
		return err
	}

	return nil
}

func Commit(dir, msg string) error {
	err := runCmd("-C", dir, "commit", "-m", msg)
	if err != nil {
		return err
	}

	return nil
}

func AddAndCommit(dir, pattern, msg string) error {
	err := Add(dir, pattern)
	if err != nil {
		return err
	}

	err = Commit(dir, msg)
	if err != nil {
		return err
	}

	return nil
}

// exec a git command
func runCmd(args ...string) error {
	cmd := exec.Command("git", args...)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
