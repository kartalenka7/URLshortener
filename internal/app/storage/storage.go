package storage

import (
	"example.com/shortener/internal/app/service"
	"example.com/shortener/internal/app/storage/database"
	"example.com/shortener/internal/app/storage/memory"
	"example.com/shortener/internal/config"
)

func New(cfg config.Config) service.Storer {
	var storer service.Storer
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
