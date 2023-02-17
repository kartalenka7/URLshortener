package handlers

import (
	"fmt"
	"io"
	"net/http"

	storage "example.com/shortener/internal/app/storage"
	"github.com/go-chi/chi/v5"
)

var (
	paramId        = "id"
	headerLocation = "Location"
)

type Repository interface {
	AddLink(gToken string, longURL string) error
	GetLongURL(sToken string) (string, error)
}

func NewRouter(s storage.StorageLinks, gToken string) chi.Router {
	// определяем роутер chi
	r := chi.NewRouter()
	// создадим суброутер, который будет содержать две функции
	r.Route("/", func(r chi.Router) {
		r.Post("/", func(rw http.ResponseWriter, req *http.Request) {
			// Читаем строку URL из body
			b, err := io.ReadAll(req.Body)
			defer req.Body.Close()
			url := string(b)
			// обрабатываем ошибку
			if err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}
			// записываем связку короткий url - длинный url
			err = s.AddLink(gToken, url)
			if err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}
			// возвращаем ответ с кодом 201
			rw.WriteHeader(http.StatusCreated)
			// пишем в тело ответа сокращенный URL
			sToken := fmt.Sprintf("http://localhost:8080/%s", gToken)
			fmt.Fprint(rw, sToken)
		})
		r.Get("/{id}", func(rw http.ResponseWriter, req *http.Request) {
			shortURL := chi.URLParam(req, paramId)
			longURL, err := s.GetLongURL(shortURL)
			if err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}
			rw.Header().Set(headerLocation, longURL)
			// возвращаем ответ с кодом 307
			rw.WriteHeader(http.StatusTemporaryRedirect)

		})
	})

	return r
}
