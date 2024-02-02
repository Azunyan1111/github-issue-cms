package gh

import (
	"context"
	"errors"
	"github.com/google/go-github/v56/github"
	"go.uber.org/zap"
)

type CustomGitHubClient struct {
	Client *github.Client
	Logger *zap.SugaredLogger
}

func NewCustomGitHubClient(client *github.Client, logger *zap.SugaredLogger) *CustomGitHubClient {
	return &CustomGitHubClient{
		Client: client,
		Logger: logger,
	}
}

func (c *CustomGitHubClient) GetIssues(ctx context.Context, username, repository string) ([]*github.Issue, error) {
	if c.Client == nil {
		c.Logger.Error("client is nil")
		return nil, errors.New("client is nil")
	}

	// Get Issues
	if username == "" || repository == "" {
		c.Logger.Error("Please set username and repository in gic.config.yaml")
		return nil, errors.New("username and repository not set")
	}

	issuesAndPRs, _, err := c.Client.Issues.ListByRepo(
		ctx,
		username,
		repository,
		&github.IssueListByRepoOptions{
			State: "all",
		},
	)

	// Check Rate Limits
	var rateLimitError *github.RateLimitError
	if errors.As(err, &rateLimitError) {
		c.Logger.Error("hit rate limit")
		return nil, err
	}
	var abuseRateLimitError *github.AbuseRateLimitError
	if errors.As(err, &abuseRateLimitError) {
		c.Logger.Error("hit secondary rate limit")
		return nil, err
	}
	if err != nil {
		c.Logger.Error("failed to get issues")
		return nil, err
	}

	// Filter issues
	var issues []*github.Issue
	for _, item := range issuesAndPRs {
		// Skip if it is a pull request.
		if item.IsPullRequest() {
			continue
		}
		issues = append(issues, item)
	}

	return issues, nil
}
