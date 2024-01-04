package github

import (
	"context"
	"fmt"
	"os"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

func GraphQLClient(s string) *githubv4.Client {
	client := oauth2.NewClient(
		context.Background(),
		oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: s},
		),
	)
	return githubv4.NewClient(client)
}

func RepoID(owner, repo string) (githubv4.ID, error) {
	ghToken := os.Getenv("GITHUB_TOKEN")
	client := GraphQLClient(ghToken)

	var query struct {
		Repository struct {
			ID githubv4.ID
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	variables := map[string]interface{}{
		"owner": githubv4.String(owner),
		"name":  githubv4.String(repo),
	}

	err := client.Query(context.Background(), &query, variables)
	if err != nil {
		return "", err
	}

	return query.Repository.ID, nil
}

func DiscussionCategoryID(owner, repo string) (string, error) {
	ghToken := os.Getenv("GITHUB_TOKEN")
	client := GraphQLClient(ghToken)

	var query struct {
		Repository struct {
			DiscussionCategories struct {
				Nodes []struct {
					ID   githubv4.ID
					Name githubv4.String
				}
			} `graphql:"discussionCategories(first: 10)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	variables := map[string]interface{}{
		"owner": githubv4.String(owner),
		"name":  githubv4.String(repo),
	}

	err := client.Query(context.Background(), &query, variables)
	if err != nil {
		return "", err
	}

	for _, category := range query.Repository.DiscussionCategories.Nodes {
		if category.Name == "General" {
			return category.ID.(string), nil
		}
	}

	return "", fmt.Errorf("couldn't find category id\n")
}
