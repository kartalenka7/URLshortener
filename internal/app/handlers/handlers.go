package handlers

import (
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
)

var (
	paramID        = "id"
	headerLocation = "Location"
)

type Repository interface {
	AddLink(gToken string, longURL string) error
	GetLongURL(sToken string) (string, error)
}

func (s *Server) shortenURL(rw http.ResponseWriter, req *http.Request) {
	// Читаем строку URL из body
	b, err := io.ReadAll(req.Body)
	defer req.Body.Close()
	url := string(b)
	// обрабатываем ошибку
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	// добавляем длинный url в хранилище, генерируем токен
	gToken, err := s.storage.AddLink(url)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	// возвращаем ответ с кодом 201
	rw.WriteHeader(http.StatusCreated)
	// пишем в тело ответа сокращенный URL
	sToken := fmt.Sprintf("http://localhost:8080/%s", gToken)
	fmt.Fprint(rw, sToken)
}
func (s *Server) getFullURL(rw http.ResponseWriter, req *http.Request) {
	shortURL := chi.URLParam(req, paramID)
	// получаем длинный url
	longURL, err := s.storage.GetLongURL(shortURL)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	rw.Header().Set(headerLocation, longURL)
	// возвращаем ответ с кодом 307
	rw.WriteHeader(http.StatusTemporaryRedirect)
}
