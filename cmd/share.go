package cmd

import (
	"context"
	"log"
	"os"

	"github.com/shurcooL/githubv4"
	"github.com/spf13/cobra"

	"github.com/notnmeyer/daylog-cli/internal/github"
)

// shareCmd represents the share command
var shareCmd = &cobra.Command{
	Use:   "share",
	Short: "post a log to GH discussions",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		ghToken := os.Getenv("GITHUB_TOKEN")
		client := github.GraphQLClient(ghToken)

		// create the query
		var mutation struct {
			CreateDiscussion struct {
				Discussion struct {
					ID  githubv4.ID
					URL string
				}
			} `graphql:"createDiscussion(input:$input)"`
		}

		// build the query input
		// TODO: dont hardcode repo
		repoID, err := github.RepoID("notnmeyer", "daylog-cli")
		if err != nil {
			log.Fatalf("failed to get GH repo ID: %v\n", err)
		}

		// TODO: ditto
		catID, err := github.DiscussionCategoryID("notnmeyer", "daylog-cli")
		if err != nil {
			log.Fatalf("couldn't get discussion category ID: %v\n", err)
		}

		input := githubv4.CreateDiscussionInput{
			RepositoryID: repoID,
			CategoryID:   catID,
			Title:        "Testing 1 2 3",
			Body:         "It works!",
		}

		err = client.Mutate(context.Background(), &mutation, input, nil)
		if err != nil {
			log.Fatalf("Failed to create discussion: %v", err)
		}

		log.Printf("Discussion created: %s (ID: %s)\n", mutation.CreateDiscussion.Discussion.URL, mutation.CreateDiscussion.Discussion.ID)
	},
}

func init() {
	rootCmd.AddCommand(shareCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// shareCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// shareCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
