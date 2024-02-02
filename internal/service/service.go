package service

import "go.uber.org/zap"

type Service struct {
	ArticleService
}

func NewService(
	logger *zap.SugaredLogger,
	imagePath string,
	githubToken string,
) Service {
	return Service{
		ArticleService: NewArticleService(logger, imagePath, githubToken),
	}
}
