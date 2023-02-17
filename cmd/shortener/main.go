package main

import (
	"log"
	"net/http"

	utils "example.com/shortener/cmd/utils"
	handlers "example.com/shortener/internal/app/handlers"
	storage "example.com/shortener/internal/app/storage"
)

var (
	localAddr = "localhost:8080"
)

func main() {
	storage := storage.GetStorage(make(map[string]string))
	router := handlers.NewRouter(*storage, utils.RandStringBytes(10))
	log.Fatal(http.ListenAndServe(localAddr, router))
}
