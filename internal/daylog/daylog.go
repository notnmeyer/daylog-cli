package daylog

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/glamour"
	"github.com/markusmobius/go-dateparser"
	"github.com/notnmeyer/daylog-cli/internal/editor"
)

type DayLog struct {
	Path string
}

func New(args []string) (*DayLog, error) {
	file, err := resolveLogPath(args)
	if err != nil {
		return nil, err
	}

	return &DayLog{
		Path: file,
	}, nil
}

// edit the log for the specified date
func (d *DayLog) Edit() error {
	if err := editor.Open(d.Path); err != nil {
		return err
	}

	return nil
}

func (d *DayLog) Show(renderMarkdown bool) (string, error) {
	contents, err := editor.Read(d.Path)
	if err != nil {
		return "", err
	}

	if renderMarkdown {
		renderer, _ := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
		)

		contents, err = renderer.Render(contents)
		if err != nil {
			return "", err
		}
	}

	return contents, nil
}

// returns the complete path to log file
func resolveLogPath(args []string) (string, error) {
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
