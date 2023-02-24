package main

import (
	"log"
	"net/http"

	handlers "example.com/shortener/internal/app/handlers"
	storage "example.com/shortener/internal/app/storage"
)

var (
	localAddr = "localhost:8080"
)

func main() {
	storage := storage.NewStorage()
	router := handlers.NewRouter(storage)
	log.Fatal(http.ListenAndServe(localAddr, router))
}
