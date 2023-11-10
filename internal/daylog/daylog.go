package daylog

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/markusmobius/go-dateparser"
	"github.com/notnmeyer/daylog-cli/internal/editor"
	"github.com/notnmeyer/daylog-cli/internal/output-formatter"
)

type DayLog struct {
	// the complete path to the log file
	Path string

	// the date of the log
	Date *time.Time
}

func New(args []string) (*DayLog, error) {
	t, err := parseDateFromArgs(args)
	if err != nil {
		return nil, err
	}

	year, month, day := t.Year(), int(t.Month()), t.Day()
	file, err := resolveLogPath(year, month, day)
	if err != nil {
		return nil, err
	}

	return &DayLog{
		Path: file,
		Date: t,
	}, nil
}

// edit the log for the specified date
func (d *DayLog) Edit() error {
	if err := createIfMissing(d); err != nil {
		return err
	}

	if err := editor.Open(d.Path); err != nil {
		return err
	}

	return nil
}

func (d *DayLog) Show(format string) (string, error) {
	contents, err := editor.Read(d.Path)
	if err != nil {
		return "", err
	}

	contents, err = outputFormatter.Format(format, contents)
	if err != nil {
		return "", err
	}

	return contents, nil
}

// returns the complete path to log file
func resolveLogPath(year, month, day int) (string, error) {
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

	return filepath.Join(path, "log.md"), nil
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
		d, err := dateparser.Parse(nil, dateString)
		if err != nil {
			return nil, err
		}
		return &d.Time, nil
	} else {
		t := time.Now()
		return &t, nil
	}
}

func createIfMissing(d *DayLog) error {
	_, err := os.Stat(d.Path)
	if err == nil {
		return nil
	}

	var file *os.File
	if os.IsNotExist(err) {
		file, err = os.Create(d.Path)
		if err != nil {
			return err
		}
		defer file.Close()
	}

	year, month, day := d.Date.Year(), int(d.Date.Month()), d.Date.Day()
	header := fmt.Sprintf("# %d/%d/%d", year, month, day)
	_, err = file.WriteString(header)
	if err != nil {
		return err
	}

	return nil
}
