package storage

import (
	"context"

	"example.com/shortener/internal/app/models"
	"example.com/shortener/internal/app/storage/database"
	"example.com/shortener/internal/app/storage/memory"
	"example.com/shortener/internal/config"
)

type Storer interface {
	AddLink(ctx context.Context, longURL string, user string) (string, error)
	GetLongURL(ctx context.Context, sToken string) (string, error)
	Ping(ctx context.Context) error
	GetAllURLS(cookie string, ctx context.Context) (map[string]string, error)
	ShortenBatch(ctx context.Context, batchReq []models.BatchReq, cookie string) ([]models.BatchResp, error)
	Close() error

	GetStorageLen() int
}

func New(cfg config.Config) Storer {
	var storer Storer
	var err error
	// создаем объект хранилища
	if cfg.Database != "" {
		storer, err = database.New(cfg) // бд хранилище
		if err != nil {
			storer = memory.New(cfg)
		}
	} else {
		storer = memory.New(cfg) // in-memory хранилище
	}
	return storer
}
