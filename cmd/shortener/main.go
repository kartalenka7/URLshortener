package main

import (
	"log"
	"net/http"

	handlers "example.com/shortener/internal/app/handlers"
	database "example.com/shortener/internal/app/storage/database"
	memory "example.com/shortener/internal/app/storage/memory"
	service "example.com/shortener/internal/app/storage/service"
	config "example.com/shortener/internal/config"
)

var (
	localAddr = "localhost:8080"
)

func main() {
	var storer service.Storer
	var err error
	// получаем структуру с конфигурацией приложения
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	// создаем объект хранилища
	//storage := storage.NewStorage(cfg)
	if cfg.Database != "" {
		storer, err = database.New(cfg) // бд хранилище
		if err != nil {
			storer = memory.New(cfg)
		}
	} else {
		storer = memory.New(cfg) // in-memory хранилище
	}
	service := service.New(cfg, storer)

	router := handlers.NewRouter(service)

	log.Println(cfg.Server)
	log.Fatal(http.ListenAndServe(cfg.Server, router))

	service.Storage.Close()
}
