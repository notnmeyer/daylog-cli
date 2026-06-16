package editor

import (
	"os"
	"path/filepath"
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
		// On some platforms (notably Windows) mode 0000 is still readable
		// by the owning user. Skip the readability assertion in that case.
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
