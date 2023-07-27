package main

import (
	"net/http"

	"example.com/shortener/internal/app/handlers"
	"example.com/shortener/internal/app/service"
	"example.com/shortener/internal/app/storage"
	"example.com/shortener/internal/config"
	"example.com/shortener/internal/logger"
	"github.com/sirupsen/logrus"
)

var (
	localAddr = "localhost:8080"
)

func main() {
	var storer service.Storer
	var err error
	log := logger.InitLog()

	// получаем структуру с конфигурацией приложения
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}
	log.WithFields(logrus.Fields{"cfg": cfg}).Debug("Конфигурация приложения")

	// создаем объект хранилища
	storer = storage.New(cfg, log)
	service := service.New(cfg, storer, log)

	router := handlers.NewRouter(service, log)

	log.WithFields(logrus.Fields{"server": cfg.Server})
	log.Fatal(http.ListenAndServe(cfg.Server, router))

	err = service.Close()
	if err != nil {
		log.Fatal(err)
	}
}
