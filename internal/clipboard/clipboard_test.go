package clipboard

import (
	"runtime"
	"strings"
	"testing"
)

func TestCopy_UnsupportedOS(t *testing.T) {
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" || runtime.GOOS == "windows" {
		t.Skip("test only exercises the default/unhandled OS branch")
	}

	err := Copy([]byte("hello"))
	if err == nil {
		t.Fatalf("Copy() err = nil on %s, want non-nil", runtime.GOOS)
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("Copy() err = %q, want message about not being supported", err.Error())
	}
}
