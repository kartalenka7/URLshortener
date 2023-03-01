package main

import (
	"log"
	"net/http"

	handlers "example.com/shortener/internal/app/handlers"
	storage "example.com/shortener/internal/app/storage"
	"github.com/caarlos0/env/v6"
)

var (
	localAddr = "localhost:8080"
)

/* type Config struct {
	Server string `env:"SERVER_ADDRESS" envDefault:"localhost:8080"`
} */

func main() {
	var cfg handlers.Config
	storage := storage.NewStorage()
	router := handlers.NewRouter(storage)
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(cfg.Server)
	log.Fatal(http.ListenAndServe(cfg.Server, router))
}
