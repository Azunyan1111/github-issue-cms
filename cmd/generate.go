package cmd

import (
	"context"
	"fmt"
	"github.com/Azunyan1111/github-issue-cms/internal/config"
	"github.com/Azunyan1111/github-issue-cms/internal/pkg/gh"
	"github.com/Azunyan1111/github-issue-cms/internal/service"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate articles from GitHub issues",
	Long: `Generate articles from GitHub issues.

This command will get issues from GitHub and create articles from them.
The articles will be saved in the "content" directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize
		logger := config.Logger
		config.SetupGitHubClient()
		githubClient := gh.NewCustomGitHubClient(config.GitHubClient, &logger)
		ctx := context.Background()
		svc := service.NewService(&logger, config.ImagesPath, config.GitHubToken)

		username := viper.GetString("github.username")
		repository := viper.GetString("github.repository")
		if username == "" || repository == "" {
			logger.Error("Please set username and repository in gic.config.yaml")
			return
		}
		url := "https://github.com/" + username + "/" + repository
		logger.Info("Target Repository: " + url)

		// Get issues
		logger.Info("Getting issues...")
		issues, err := githubClient.GetIssues(ctx, username, repository)
		if err != nil {
			logger.Error("Failed to get issues")
			return
		}
		if len(issues) == 0 {
			logger.Info("No issues found")
			return
		} else {
			logger.Infof("Found %d issues", len(issues))
		}

		// Create articles
		logger.Info("Creating articles...")
		for _, issue := range issues {
			article, err := svc.IssueToArticle(issue)
			if err != nil {
				logger.Errorf("Failed to create article: %s", err)
			}
			if article != nil {
				err := svc.ExportArticle(article, fmt.Sprintf("%d", issue.GetID()))
				if err != nil {
					logger.Errorf("Failed to export article: %s", err)
				}
			}
		}

	},
}

func init() {
	rootCmd.AddCommand(generateCmd)

	// GitHub Token
	generateCmd.Flags().StringVarP(&config.GitHubToken, "token", "t", "", "GitHub API Token")
	_ = generateCmd.MarkFlagRequired("token")
}
