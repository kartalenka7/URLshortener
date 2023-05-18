package service

import (
	"context"
	"log"

	urlNet "net/url"
	"path"

	"example.com/shortener/internal/app/models"
	"example.com/shortener/internal/config"
)

// Storer - интерфейс взаимодействия с хранилищем
type Storer interface {
	AddLink(ctx context.Context, longURL string, user string) (string, error)
	GetLongURL(ctx context.Context, sToken string) (string, error)
	Ping(ctx context.Context) error
	GetAllURLS(cookie string, ctx context.Context) (map[string]string, error)
	ShortenBatch(ctx context.Context, batchReq []models.BatchReq, cookie string) ([]models.BatchResp, error)
	Close() error

	GetStorageLen() int
}

type Service struct {
	Config  config.Config
	Storage Storer
}

func New(config config.Config, storage Storer) *Service {
	return &Service{Config: config, Storage: storage}
}

func (s Service) GetLongToken(sToken string) string {
	longToken := s.Config.BaseURL + sToken
	_, urlParseErr := urlNet.Parse(longToken)
	if urlParseErr != nil {
		longToken = path.Join(s.Config.BaseURL, sToken)
	}
	log.Printf("longToken %s", longToken)
	return longToken
}
