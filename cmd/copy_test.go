package cmd

import (
	"runtime"
	"strings"
	"testing"
)

func TestCopy_UnsupportedOS(t *testing.T) {
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" || runtime.GOOS == "windows" {
		t.Skip("test only exercises the default/unhandled OS branch")
	}

	err := copy([]byte("hello"))
	if err == nil {
		t.Fatalf("copy() err = nil on %s, want non-nil", runtime.GOOS)
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("copy() err = %q, want message about not being supported", err.Error())
	}
}
