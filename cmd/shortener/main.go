package main

import (
	"log"
	"net/http"

	handlers "example.com/shortener/internal/app/handlers"
	storage "example.com/shortener/internal/app/storage"
	config "example.com/shortener/internal/config"
)

var (
	localAddr = "localhost:8080"
)

func main() {

	// получаем структуру с конфигурацией приложения
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}
	storage := storage.NewStorage(cfg)
	router := handlers.NewRouter(storage)

	log.Println(cfg.Server)
	log.Fatal(http.ListenAndServe(cfg.Server, router))

	storage.Close()
}
