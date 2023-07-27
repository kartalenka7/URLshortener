package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"example.com/shortener/internal/app/service"
	"example.com/shortener/internal/app/storage"
	"example.com/shortener/internal/config"
	"example.com/shortener/internal/config/utils"
	"example.com/shortener/internal/logger"
)

var endpoint = "http://localhost:8080/"

func ExampleServer_GetUserURLs() {
	// инициализируем необходимые сущности
	log := logger.InitLog()
	cfg, err := config.GetConfig()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	storage := storage.New(cfg, log)
	serv := &Server{
		service: *service.New(cfg, storage, log),
	}

	url := endpoint + "/api/user/urls"
	request := httptest.NewRequest(http.MethodGet, url, nil)
	// доставать URLы будем на основании значения куки
	cookie, _ := utils.WriteCookies()
	request.AddCookie(&cookie)

	// создаем новый Recorder
	w := httptest.NewRecorder()
	// выбираем хэндлер для теста
	handler := http.HandlerFunc(serv.GetUserURLs)
	// запускаем сервис
	handler.ServeHTTP(w, request)

	response := w.Result()
	fmt.Println(response.StatusCode)

	// Output:
	// 204
}

func ExampleServer_DeleteURLs() {
	// инициализируем необходимые сущности
	log := logger.InitLog()
	cfg, err := config.GetConfig()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	storage := storage.New(cfg, log)
	serv := &Server{
		service: *service.New(cfg, storage, log),
	}

	url := endpoint + "/api/user/urls"
	request := httptest.NewRequest(http.MethodDelete, url, nil)

	// создаем новый Recorder
	w := httptest.NewRecorder()
	// выбираем хэндлер для теста
	handler := http.HandlerFunc(serv.DeleteURLs)
	// запускаем сервис
	handler.ServeHTTP(w, request)

	response := w.Result()
	fmt.Println(response.StatusCode)

	// Output:
	// 202
}
