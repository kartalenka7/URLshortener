package main

import (
	"log"
	"net/http"

	"example.com/shortener/internal/app/handlers"
	"example.com/shortener/internal/app/service"
	"example.com/shortener/internal/app/storage"
	"example.com/shortener/internal/config"
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
	storer = storage.New(cfg)
	service := service.New(cfg, storer)

	router := handlers.NewRouter(service)

	log.Println(cfg.Server)
	log.Fatal(http.ListenAndServe(cfg.Server, router))

	err = service.Close()
	if err != nil {
		log.Printf("Ошибка при завершении работы : %v\n", err)
	}
}
