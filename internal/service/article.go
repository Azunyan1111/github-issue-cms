package service

import (
	"bufio"
	"fmt"
	"github.com/Azunyan1111/github-issue-cms/internal/model"
	"github.com/google/go-github/v56/github"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ArticleService struct {
	Logger      *zap.SugaredLogger
	ImagePath   string
	GitHubToken string
}

func NewArticleService(
	logger *zap.SugaredLogger,
	imagePath string,
	githubToken string,
) ArticleService {
	return ArticleService{
		Logger:      logger,
		ImagePath:   imagePath,
		GitHubToken: githubToken,
	}
}

func (as ArticleService) IssueToArticle(issue *github.Issue) (*model.Article, error) {
	//Skip if it is a pull request.
	if issue.IsPullRequest() {
		return nil, errors.New("is pull request")
	}

	// Get ID
	id := fmt.Sprintf("%d", *issue.ID)
	as.Logger.Debugf("Processing gh ID: %s Title: %v", id, issue.GetTitle())

	// Get content
	content := issue.GetBody()
	content = strings.Replace(content, "\r", "", -1)

	// Get front matter
	frontMatter := func() []string {
		re := regexp.MustCompile("(?s)^\\s*```\\r?\\n(.*?)\\r?\\n```")
		match := re.FindStringSubmatch(content)
		if len(match) > 0 {
			return match
		}

		return nil
	}()

	// Remove front matter from content
	if frontMatter != nil {
		content = strings.Replace(content, frontMatter[0], "", 1)
	}

	// Check front matter exists
	if len(frontMatter) == 0 {
		as.Logger.Error("front matter not found or front matter missing in first line")
		as.Logger.Errorf("gh URL: %s", issue.GetHTMLURL())
		return nil, errors.New("front matter not found")
	}

	// customAuthor returns the author name from the front matter.
	customAuthor := func() string {
		for _, s := range frontMatter {
			re := regexp.MustCompile(`author: ['"](.*)['"]`)
			match := re.FindStringSubmatch(s)
			if len(match) > 0 {
				return match[1]
			}
		}
		// Default to the gh author if not found in the front matter.
		return issue.GetUser().GetLogin()
	}

	// Remove empty lines at the beginning
	content = strings.TrimLeft(content, "\n")

	// Insert empty line at the end if not exists
	if !strings.HasSuffix(content, "\n") {
		content = content + "\n"
	}

	// Replace image URL to local path
	re := regexp.MustCompile(`!\[.*]\((.*)\)`)
	match := re.FindAllStringSubmatch(content, -1)
	for i, m := range match {
		url := m[1]
		before := m[0]
		replaced := "![" + url + "](/images/" + id + "/" + fmt.Sprintf("%d", i) + ".png)"

		// Skip if already replaced
		if strings.Contains(content, replaced) {
			continue
		}

		// Download image
		contentType, err := as.downloadImage(url, id, fmt.Sprint(i))
		if err != nil {
			as.Logger.Error("Failed to download image: " + url)
		}
		replaced = fmt.Sprintf("![/images/%s/%d%s](/images/%s/%d%s)", id, i, contentType, id, i, contentType)

		// Replace url to local path
		content = strings.Replace(content, before, replaced, -1)
	}

	re = regexp.MustCompile(`<img width="\d+" alt="(\w+)" src="(\S+)">`)
	match = re.FindAllStringSubmatch(content, -1)
	for i, m := range match {
		alt := m[1]
		url := m[2]
		before := m[0]
		replaced := "![" + alt + "](images/" + id + "/" + fmt.Sprintf("%d", i) + ".png)"

		contentType, err := as.downloadImage(url, id, fmt.Sprint(i))
		if err != nil {
			as.Logger.Error("Failed to download image: " + url)
		}
		replaced = fmt.Sprintf("![/images/%s/%d%s](/images/%s/%d%s)", id, i, contentType, id, i, contentType)

		content = strings.Replace(content, before, replaced, -1)
	}

	// Get tags
	tags := make([]string, 0, len(issue.Labels))
	for _, label := range issue.Labels {
		tags = append(tags, label.GetName())
	}

	// Create article
	return &model.Article{
		Author:           customAuthor(),
		Title:            issue.GetTitle(),
		Date:             issue.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
		Category:         issue.GetMilestone().GetTitle(),
		Draft:            issue.GetState() != "closed",
		Content:          content,
		Tags:             tags,
		ExtraFrontMatter: frontMatter[1],
	}, nil
}

// DownloadImage downloads an image from the URL and save it to the local file system.
func (as ArticleService) downloadImage(url string, articleID string, filename string) (string, error) {
	if strings.Contains(url, "facebook.com") {
		return "", nil
	}
	as.Logger.Info("Downloading image: " + url)

	// Download image
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to create request")
	}
	req.Header.Set("Authorization", "token "+as.GitHubToken)
	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to get response")
	}
	defer resp.Body.Close()

	// Check Redirect
	if resp.StatusCode == 301 || resp.StatusCode == 302 {
		as.Logger.Info("Redirected to: " + resp.Header.Get("Location"))
		return as.downloadImage(resp.Header.Get("Location"), articleID, filename)
	}

	// Check response
	var extension string
	contentType := resp.Header.Get("Content-Type")
	switch contentType {
	case "image/png":
		extension = ".png"
	case "image/jpeg":
		extension = ".jpg"
	case "image/gif":
		extension = ".gif"
	default:
		return "", errors.New("unsupported content type")
	}
	if resp.StatusCode != 200 {
		as.Logger.Error(fmt.Sprintf("Response: %d %s", resp.StatusCode, contentType))
		return "", errors.New("response error")
	}
	as.Logger.Info(fmt.Sprintf("Response: %d %s", resp.StatusCode, contentType))

	// Expect like this: ./static/images/{articleID}/{filename}.png
	imagesPath := as.ImagePath
	base := filepath.Join(imagesPath, articleID)
	dest := filepath.Join(base, fmt.Sprintf("%v", filename)+extension)

	// Create directory
	if _, err := os.Stat(base); os.IsNotExist(err) {
		as.Logger.Info("Creating directory: " + base)
		err := os.MkdirAll(base, 0777)
		if err != nil {
			return "", errors.Wrap(err, "failed to create directory")
		}
	}

	// check exist file
	if _, err := os.Stat(dest); err == nil {
		as.Logger.Info("Image already exists: " + dest)
		return extension, nil
	}

	// Prepare a new file
	file, err := os.Create(dest)
	if err != nil {
		return "", errors.Wrap(err, "failed to create file")
	}
	defer file.Close()

	// Write the body to file
	written, err := io.Copy(file, resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to write body")
	}
	as.Logger.Info("Download complete. image: " + dest + " (" + fmt.Sprintf("%d", written) + " bytes)")
	return extension, nil
}

func (as ArticleService) ExportArticle(article *model.Article, id string) error {
	// Create directory
	path := filepath.Join("content", "posts")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0777); err != nil {
			return errors.Wrap(err, "failed to create directory")
		}
	}

	// Create file
	path = filepath.Join("content", "posts", id+".md")
	file, err := os.Create(path)
	if err != nil {
		return errors.Wrap(err, "failed to create file")
	}
	defer file.Close()
	as.Logger.Info("Export article: " + path)

	// Build string
	var tags string
	for _, t := range article.Tags {
		tags += " - '" + t + "'\n"
	}
	if tags == "" {
		tags = " -\n"
	}
	content := strings.TrimSpace(fmt.Sprintf(`
---
title: '%s'
author: '%s'
date: '%s'
draft: '%t'
categories:
 - '%s'
tags:
%s
%s
---
	
%s`,
		article.Title,
		article.Author,
		article.Date,
		article.Draft,
		article.Category,
		tags,
		article.ExtraFrontMatter,
		article.Content))

	// Write file
	w := bufio.NewWriter(file)
	_, _ = w.WriteString(content)
	_ = w.Flush()
	return nil
}
