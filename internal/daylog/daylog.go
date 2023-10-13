package daylog

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/markusmobius/go-dateparser"
	"github.com/notnmeyer/daylog-cli/internal/editor"
	gitconfig "github.com/tcnksm/go-gitconfig"
)

type DayLog struct {
	// the path to the data dir:
	//   /blah/daylog
	RootPath string
	// the path to the dir of the desired day:
	//   /blah/daylog/2023/01/01
	Path string
	// full path to the log file:
	//   /blah/daylog/2023/01/01/log.md
	File string
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

	name, err := gitconfig.Username()
	email, err := gitconfig.Email()
	if err != nil {
		return fmt.Errorf("couldn't read from .gitconfig: %s", err)
	}

	// commit
	commitMsg := fmt.Sprintf("initial commit")
	commit, err := w.Commit(commitMsg, &git.CommitOptions{
		// TODO: load the details from the .gitconfig or env
		Author: &object.Signature{
			Name:  name,
			Email: email,
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
