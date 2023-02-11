package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// Генерирование короткой ссылки
func randStringBytes(n int) string {
	link := make([]byte, n)
	for i := range link {
		link[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(link)
}

type SavedLinks struct {
	LinksMap map[string]string
}

func (s SavedLinks) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// проверяем, каким методом получили запрос
	switch r.Method {
	case "POST":
		//читаем строку URL из body
		b, err := io.ReadAll(r.Body)
		// обрабатываем ошибку
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		// получаем токен
		sToken := randStringBytes(10)
		url := string(b)
		// записываем связку короткий url - длинный url
		s.LinksMap[sToken] = url
		// возвращаем ответ с кодом 201
		w.WriteHeader(201)
		// пишем в тело ответа сокращенный URL
		fmt.Print(w, sToken)

	case "GET":
		shortURL := r.URL.String()
		shortToken := strings.Replace(shortURL, "/", "", -1)
		longURL := s.LinksMap[shortToken]
		w.Header().Set("Location", longURL)
		// возвращаем ответ с кодом 307
		w.WriteHeader(307)
	}
}

func main() {
	savedLinks := make(map[string]string)
	// маршрутизация запросов обработчику
	handler1 := SavedLinks{
		LinksMap: savedLinks,
	}
	server := &http.Server{
		Handler: handler1,
		Addr:    "localhost:8080",
	}
	// Запуск сервера
	log.Fatal(server.ListenAndServe())

}
