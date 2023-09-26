package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/spf13/cobra"
)

var logDir string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "daylog",
	Short: "A tool for keeping track of what you did today",
	Long:  `DayLog: Fighter of the Night Log!`,

	Run: func(cmd *cobra.Command, args []string) {
		t, err := parseDateFromArgs(args)
		if err != nil {
			fmt.Errorf("Couldn't parse date: %s", err.Error())
		}

		logFile, err := setup(*t)
		if err != nil {
			log.Fatal(err)
		}

		execEditor(logFile)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVarP(&logDir, "dir", "d", "~/.config/daylog", "--dir log/directory")
	logDir = expandTilde(logDir)

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().StringVarP(&logDir, "dir", "d", "./", "--dir log/directory")
}

func chooseEditor() string {
	if _, ok := os.LookupEnv("EDITOR"); ok {
		return os.Getenv("EDITOR")
	}

	return "vim"
}

func execEditor(logFile string) {
	editor := chooseEditor()
	cmd := exec.Command(editor, logFile)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		fmt.Printf("Error starting %s: %s", editor, err)
		return
	}

	err = cmd.Wait()
	if err != nil {
		fmt.Println("Error:", err)
	}
}

func splitDate(t time.Time) (year int, month int, day int) {
	year = t.Year()
	month = int(t.Month())
	day = t.Day()
	return
}

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

func setup(t time.Time) (string, error) {
	year, month, day := splitDate(t)
	resolvedPath, err := createDir(
		logDir,
		filepath.Join(strconv.Itoa(year), itos(month), itos(day)),
	)
	if err != nil {
		return "", err
	}

	logFile := filepath.Join(resolvedPath, "log.md")
	return logFile, nil
}

// formats an integer as a 2 digit string: 1 -> 01
func itos(i int) string {
	return fmt.Sprintf("%02d", i)
}

func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		dirname, _ := os.UserHomeDir()
		path = filepath.Join(dirname, path[2:])
	}
	return path
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
