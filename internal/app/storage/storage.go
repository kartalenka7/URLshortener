package storage

import (
	"context"
	"time"

	"example.com/shortener/internal/app/service"
	"example.com/shortener/internal/app/storage/database"
	"example.com/shortener/internal/app/storage/memory"
	"example.com/shortener/internal/config"
	"github.com/sirupsen/logrus"
)

func New(cfg config.Config, log *logrus.Logger) service.Storer {
	var storer service.Storer
	var err error

	// конструируем контекст с 5-секундным тайм-аутом
	// после 5 секунд затянувшаяся операция с БД будет прервана
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// не забываем освободить ресурс
	defer cancel()
	// создаем объект хранилища
	if cfg.Database != "" {
		storer, err = database.New(ctx, cfg, log) // бд хранилище
		if err != nil {
			storer = memory.New(cfg)
		}
	} else {
		storer = memory.New(cfg) // in-memory хранилище
	}
	return storer
}
