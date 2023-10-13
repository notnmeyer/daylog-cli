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
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/notnmeyer/daylog-cli/internal/editor"
)

type DayLog struct {
	RootPath string // the root path of daylog's data dir, /blah/daylog
	Path     string // the path to the dir of the desired day, /blah/daylog/2023/01/01
	File     string // the path to the full file, /blah/daylog/2023/01/01/log.md
}

func New(args []string) (*DayLog, error) {
	dir, err := resolveLogDir(args)
	if err != nil {
		return nil, err
	}

	// split dir into a slice of the path components
	pathComponents := strings.Split(dir, "/")

	return &DayLog{
		// exclude year/month/day from rootPath
		RootPath: "/" + filepath.Join(pathComponents[:len(pathComponents)-3]...),
		Path:     dir,
		File:     filepath.Join(dir, "log.md"),
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

func (d *DayLog) Init() error {
	_, err := git.PlainInit(d.RootPath, false)
	if err != nil {
		return err
	}

	// TODO: create main, not master

	// make an initial commit
	r, err := git.PlainOpen(d.RootPath)
	if err != nil {
		return fmt.Errorf("couldn't open repo: %s\n", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("couldn't create worktree: %s\n", err)
	}

	// add today's files to the worktree
	w.AddGlob(d.Path)

	c, err := config.LoadConfig(config.GlobalScope)
	if err != nil {
		return err
	}

	// commit
	commitMsg := fmt.Sprintf("initial commit")
	commit, err := w.Commit(commitMsg, &git.CommitOptions{
		// TODO: load the details from the .gitconfig or env
		Author: &object.Signature{
			Name:  c.User.Name,
			Email: c.User.Email,
			When:  time.Now(),
		},
		AllowEmptyCommits: true,
	})
	if err != nil {
		return err
	}

	obj, err := r.CommitObject(commit)
	if err != nil {
		return fmt.Errorf("couldn't make commit: %s", err)
	}
	fmt.Println(obj)

	return nil
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
