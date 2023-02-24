package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"bytes"

	"github.com/go-chi/chi/v5"
)

var (
	paramID         = "id"
	headerLocation  = "Location"
	contentTypeJSON = "application/json"
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
	fmt.Println(shortURL)
	fmt.Println(s.storage)
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

type Request struct {
	LongURL string `json:"url"`
}

func (s *Server) shortenJSON(rw http.ResponseWriter, req *http.Request) {
	var requestJSON Request

	// чтение JSON объекта из body
	decoder := json.NewDecoder(req.Body)
	defer req.Body.Close()
	//десериализация
	if err := decoder.Decode(&requestJSON); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	// добавляем длинный url в хранилище, генерируем токен
	gToken, err := s.storage.AddLink(requestJSON.LongURL)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	// формируем json объект ответа
	response := struct {
		ShortURL string `json:"result"`
	}{
		ShortURL: fmt.Sprintf("http://localhost:8080/%s", gToken),
	}
	buf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	encoder.Encode(response)

	rw.Header().Set("Content-Type", contentTypeJSON)
	// возвращаем ответ с кодом 201
	rw.WriteHeader(http.StatusCreated)
	// пишем в тело ответа сокращенный URL
	fmt.Fprint(rw, buf)
}
