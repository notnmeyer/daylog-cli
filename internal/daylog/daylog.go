package daylog

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/notnmeyer/daylog-cli/internal/editor"
	"github.com/notnmeyer/daylog-cli/internal/file"
	"github.com/notnmeyer/daylog-cli/internal/output-formatter"
)

type DayLog struct {
	// the complete path to the log file
	Path string

	// the path to the project directory
	ProjectPath string

	// the date of the log
	Date *time.Time
}

func projectPathAt(base, project string) (string, error) {
	return filepath.Abs(filepath.Join(base, "daylog", project))
}

func ensureProjectPathAt(base, project string) (string, error) {
	p, err := projectPathAt(base, project)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(p, 0755); err != nil {
		return "", err
	}
	return p, nil
}

func ProjectPath(project string) (string, error) {
	return projectPathAt(xdg.DataHome, project)
}

func EnsureProjectPath(project string) (string, error) {
	return ensureProjectPathAt(xdg.DataHome, project)
}

// new performs no filesystem i/o; date subdirs are created lazily by the methods that need them
func New(t time.Time, projectPath string) (*DayLog, error) {
	year, month, day := t.Year(), int(t.Month()), t.Day()
	logFile := filepath.Join(
		projectPath,
		strconv.Itoa(year),
		fmt.Sprintf("%02d", month),
		fmt.Sprintf("%02d", day),
		"log.md",
	)

	return &DayLog{
		Path:        logFile,
		ProjectPath: projectPath,
		Date:        &t,
	}, nil
}

// append content to the log for the specified date
func (d *DayLog) Append(content string) error {
	if err := createIfMissing(d); err != nil {
		return err
	}

	existing, err := os.ReadFile(d.Path)
	if err != nil {
		return err
	}

	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		existing = append(existing, '\n')
	}

	content = strings.TrimRight(content, "\n") + "\n"

	if err := os.WriteFile(d.Path, append(existing, []byte(content)...), 0644); err != nil {
		return err
	}

	return nil
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

	contents, err = outputformatter.Format(format, contents)
	if err != nil {
		return "", err
	}

	return contents, nil
}

// usePrevious mutates d.Path to point at the most recent log before now.
func (d *DayLog) UsePrevious(now time.Time) error {
	prev, err := file.PreviousLog(d.ProjectPath, file.LogProvider{}, now)
	if err != nil {
		return err
	}
	d.Path = filepath.Join(d.ProjectPath, prev, "log.md")
	return nil
}

func createIfMissing(d *DayLog) error {
	_, err := os.Stat(d.Path)
	if err == nil {
		return nil
	}

	if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(d.Path), 0755); err != nil {
		return err
	}

	file, err := os.Create(d.Path)
	if err != nil {
		return err
	}
	defer file.Close()

	year, month, day := d.Date.Year(), int(d.Date.Month()), d.Date.Day()
	header := fmt.Sprintf("# %d/%02d/%02d\n\n", year, month, day)
	if _, err := file.WriteString(header); err != nil {
		return err
	}

	return nil
}
