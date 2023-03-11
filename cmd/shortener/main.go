package main

import (
	"log"
	"net/http"

	handlers "example.com/shortener/internal/app/handlers"
	storage "example.com/shortener/internal/app/storage"
	utils "example.com/shortener/internal/config/utils"
)

var (
	localAddr = "localhost:8080"
)

func main() {
	var cfg utils.Config
	var err error

	storage := storage.NewStorage()
	// получаем структуру с конфигурацией приложения
	cfg, err = utils.GetConfig()
	if err != nil {
		log.Fatal(err)
	}
	storage.SetConfig(cfg)
	router := handlers.NewRouter(storage)

	log.Println(cfg.Server)
	log.Fatal(http.ListenAndServe(cfg.Server, router))
}
