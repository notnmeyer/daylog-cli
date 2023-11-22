package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/notnmeyer/daylog-cli/internal/daylog"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		dl, err := daylog.New(args, config.Project)
		if err != nil {
			log.Fatal(err)
		}

		err = filepath.Walk(dl.ProjectPath, visit(dl.ProjectPath))
		if err != nil {
			fmt.Printf("error walking the path %v: %v\n", dl.ProjectPath, err)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// listCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func visit(root string) filepath.WalkFunc {
	rootDepth := strings.Count(root, string(os.PathSeparator))

	return func(path string, info os.FileInfo, err error) error {
		// print the error, but we want to continue the traversal
		if err != nil {
			fmt.Println(err)
			return nil
		}

		depth := strings.Count(path, string(os.PathSeparator)) - rootDepth

		var item string
		if info.IsDir() {
			item = info.Name() + "/"
		} else {
			item = info.Name()
		}

		fmt.Println(strings.Repeat("  ", depth), item)
		return nil
	}
}
