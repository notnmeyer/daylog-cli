package editor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRead(t *testing.T) {
	t.Run("returns contents on existing file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "log.md")
		want := "# 2025/12/02\n\nhello\n"
		if err := os.WriteFile(path, []byte(want), 0644); err != nil {
			t.Fatalf("setup: %v", err)
		}

		got, err := Read(path)
		if err != nil {
			t.Fatalf("Read() unexpected error: %v", err)
		}
		if got != want {
			t.Errorf("Read() = %q, want %q", got, want)
		}
	})

	t.Run("returns empty string with no error for empty file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "log.md")
		if err := os.WriteFile(path, []byte(""), 0644); err != nil {
			t.Fatalf("setup: %v", err)
		}

		got, err := Read(path)
		if err != nil {
			t.Fatalf("Read() unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("Read() = %q, want empty string", got)
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "does-not-exist.md")

		got, err := Read(missing)
		if err == nil {
			t.Errorf("Read() err = nil, want non-nil for missing file; got %q", got)
		}
		if got != "" {
			t.Errorf("Read() = %q, want empty string on error", got)
		}
	})

	t.Run("returns error for unreadable file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "log.md")
		if err := os.WriteFile(path, []byte("x"), 0000); err != nil {
			t.Fatalf("setup: %v", err)
		}
		// on some platforms (notably Windows) mode 0000 is still readable
		// by the owning user. skip the readability assertion in that case
		if _, err := os.ReadFile(path); err == nil {
			t.Skip("platform does not enforce 0000 for owner; cannot test unreadable-file path")
		}

		got, err := Read(path)
		if err == nil {
			t.Errorf("Read() err = nil, want non-nil for unreadable file; got %q", got)
		}
		if got != "" {
			t.Errorf("Read() = %q, want empty string on error", got)
		}
	})
}

// withPath returns a t.Cleanup that replaces $PATH with dir prepended to the
// existing PATH, restoring the original on test exit
func withPath(t *testing.T, dir string) {
	t.Helper()
	orig := os.Getenv("PATH")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+orig)
}

// withFakeNano creates an executable named "nano" inside dir. returns dir
func withFakeNano(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "nano")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("creating fake nano: %v", err)
	}
	return dir
}

// unsetenv removes key from the environment for the duration of the test,
// restoring the original value (or absence) on cleanup. t.Setenv cannot be
// used to truly unset a variable
func unsetenv(t *testing.T, key string) {
	t.Helper()
	orig, hadOrig := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unsetenv %s: %v", key, err)
	}
	t.Cleanup(func() {
		if hadOrig {
			os.Setenv(key, orig)
		} else {
			os.Unsetenv(key)
		}
	})
}

func TestChooseEditor(t *testing.T) {
	t.Run("EDITOR wins when set", func(t *testing.T) {
		t.Setenv("EDITOR", "emacs")
		unsetenv(t, "VISUAL")
		withPath(t, withFakeNano(t))

		got, err := chooseEditor()
		if err != nil {
			t.Fatalf("chooseEditor() err = %v, want nil", err)
		}
		if got != "emacs" {
			t.Errorf("chooseEditor() = %q, want %q", got, "emacs")
		}
	})

	t.Run("VISUAL used when EDITOR unset", func(t *testing.T) {
		unsetenv(t, "EDITOR")
		t.Setenv("VISUAL", "vim")
		withPath(t, withFakeNano(t))

		got, err := chooseEditor()
		if err != nil {
			t.Fatalf("chooseEditor() err = %v, want nil", err)
		}
		if got != "vim" {
			t.Errorf("chooseEditor() = %q, want %q", got, "vim")
		}
	})

	t.Run("falls back to nano when found on PATH", func(t *testing.T) {
		unsetenv(t, "EDITOR")
		unsetenv(t, "VISUAL")
		withPath(t, withFakeNano(t))

		got, err := chooseEditor()
		if err != nil {
			t.Fatalf("chooseEditor() err = %v, want nil", err)
		}
		if got != "nano" {
			t.Errorf("chooseEditor() = %q, want %q", got, "nano")
		}
	})

	t.Run("returns error when nothing is available", func(t *testing.T) {
		unsetenv(t, "EDITOR")
		unsetenv(t, "VISUAL")
		// empty PATH means LookPath cannot resolve "nano"
		t.Setenv("PATH", "")

		got, err := chooseEditor()
		if err == nil {
			t.Errorf("chooseEditor() err = nil, want non-nil; got %q", got)
		}
		if got != "" {
			t.Errorf("chooseEditor() = %q, want empty string on error", got)
		}
		if err != nil && !strings.Contains(err.Error(), "$EDITOR") {
			t.Errorf("error %q should mention $EDITOR as a remediation hint", err.Error())
		}
	})
}
