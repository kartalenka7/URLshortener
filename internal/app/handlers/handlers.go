package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/caarlos0/env/v6"
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

type config struct {
	BaseURL string `env:"BASE_URL" envDefault:"http://localhost:8080/"`
}

func (s *Server) shortenURL(rw http.ResponseWriter, req *http.Request) {
	var cfg config
	fmt.Println("shortenURL")
	// Читаем строку URL из body
	b, err := io.ReadAll(req.Body)
	defer req.Body.Close()
	url := strings.Replace(string(b), "url=", "", 1)
	fmt.Println(url)
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
	err = env.Parse(&cfg)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	sToken := cfg.BaseURL + gToken
	log.Println(sToken)
	fmt.Fprint(rw, sToken)
}
func (s *Server) getFullURL(rw http.ResponseWriter, req *http.Request) {
	log.Println("Get full url")
	shortURL := chi.URLParam(req, paramID)
	log.Printf("short url %s", shortURL)
	// получаем длинный url
	longURL, err := s.storage.GetLongURL(shortURL)
	log.Println(longURL)
	if err != nil {
		fmt.Println(err.Error())
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
	var cfg config

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
	err = env.Parse(&cfg)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	// формируем json объект ответа
	response := struct {
		ShortURL string `json:"result"`
	}{
		ShortURL: cfg.BaseURL + gToken,
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
