package daylog

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/araddon/dateparse"
	"github.com/notnmeyer/daylog-cli/internal/editor"
)

type DayLog struct {
	Path string // the path to the dir of the desired day
	File string // the path to the full file
}

func New(args []string) (*DayLog, error) {
	dir, err := resolveLogDir(args)
	if err != nil {
		return nil, err
	}

	return &DayLog{
		Path: dir,
		File: filepath.Join(dir, "log.md"),
	}, nil
}

// edit the log for the specified date
func (d *DayLog) Edit() error {
	if err := editor.Open(d.File); err != nil {
		return err
	}

	return nil
}

func (d *DayLog) Show() (string, error) {
	contents, err := editor.Read(d.Path)
	if err != nil {
		return "", err
	}

	return contents, nil
}

// returns the path to log dir for the requested date
func resolveLogDir(args []string) (string, error) {
	t, err := parseDateFromArgs(args)
	if err != nil {
		return "", err
	}

	year, month, day := t.Year(), int(t.Month()), t.Day()

	path, err := createDir(
		xdg.DataHome,
		filepath.Join(
			"daylog",
			strconv.Itoa(year),
			fmt.Sprintf("%02d", month),
			fmt.Sprintf("%02d", day),
		),
	)
	if err != nil {
		return "", err
	}

	return path, nil
}

// resolves root path, and creates directories specified in path
func createDir(rootPath, path string) (string, error) {
	absolutePath, err := filepath.Abs(rootPath)
	if err != nil {
		return "", err
	}

	completePath := filepath.Join(absolutePath, path)

	err = os.MkdirAll(completePath, os.ModePerm)
	if err != nil {
		return "", err
	}

	return completePath, nil
}

func parseDateFromArgs(args []string) (*time.Time, error) {
	if len(args) > 0 {
		dateString := strings.Join(args, " ")
		t, err := dateparse.ParseStrict(dateString)
		if err != nil {
			return nil, err
		}
		return &t, nil
	} else {
		t := time.Now()
		return &t, nil
	}
}
