package service

import (
	"context"

	"example.com/shortener/internal/app/models"
	"example.com/shortener/internal/config"
)

// Storer - интерфейс взаимодействия с хранилищем
type Storer interface {
	AddLink(ctx context.Context, longURL string, user string) (string, error)
	GetLongURL(ctx context.Context, sToken string) (string, error)
	Ping(ctx context.Context) error
	GetAllURLS(cookie string, ctx context.Context) map[string]string
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
