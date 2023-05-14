package service

import (
	"context"

	database "example.com/shortener/internal/app/storage/database"
	"example.com/shortener/internal/config"
)

// Storer - интерфейс взаимодействия с хранилищем
type Storer interface {
	AddLink(longURL string, user string, ctx context.Context) (string, error)
	GetLongURL(sToken string) (string, error)
	Ping(ctx context.Context) error
	GetAllURLS(cookie string, ctx context.Context) map[string]string
	ShortenBatch(ctx context.Context, batchReq []database.BatchReq, cookie string) ([]database.BatchResp, error)
	Close()

	GetStorageLen() int
}

type Service struct {
	Config  config.Config
	Storage Storer
}

func New(config config.Config, storage Storer) *Service {
	return &Service{Config: config, Storage: storage}
}
