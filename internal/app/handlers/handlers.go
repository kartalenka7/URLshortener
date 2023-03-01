package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	urlNet "net/url"
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
	GetStorageLen() int
}

type Config struct {
	BaseURL string `env:"BASE_URL" envDefault:"http://localhost:8080/"`
	Server  string `env:"SERVER_ADDRESS" envDefault:"localhost:8080"`
	File    string `env:"FILE_STORAGE_PATH"`
}

func (s *Server) shortenURL(rw http.ResponseWriter, req *http.Request) {
	var cfg Config
	// получаем переменные окружения
	err := env.Parse(&cfg)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Println("shorten URL")
	// Читаем строку URL из body
	b, err := io.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	url := strings.Replace(string(b), "url=", "", 1)
	log.Printf("long url %s\n", url)
	// добавляем длинный url в хранилище, генерируем токен
	gToken, err := s.storage.AddLink(url, cfg.File)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	// возвращаем ответ с кодом 201
	rw.WriteHeader(http.StatusCreated)
	// пишем в тело ответа сокращенный URL
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	sToken := cfg.BaseURL + gToken
	_, urlParseErr := urlNet.Parse(sToken)
	if urlParseErr != nil {
		sToken = cfg.BaseURL + "/" + gToken
		fmt.Fprint(rw, sToken)
		return
	}
	fmt.Fprint(rw, sToken)
}
func (s *Server) getFullURL(rw http.ResponseWriter, req *http.Request) {
	var cfg Config
	log.Println("Get full url")
	// получаем переменные окружения
	err := env.Parse(&cfg)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	//получаем сокращенный url из параметра
	shortURL := chi.URLParam(req, paramID)
	log.Printf("short url %s\n", shortURL)
	// получаем длинный url
	longURL, err := s.storage.GetLongURL(shortURL, cfg.File)
	if err != nil {
		fmt.Println(err.Error())
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	// возвращаем длинный url в поле Location
	rw.Header().Set(headerLocation, longURL)
	// возвращаем ответ с кодом 307
	rw.WriteHeader(http.StatusTemporaryRedirect)
}

type Request struct {
	LongURL string `json:"url"`
}

func (s *Server) shortenJSON(rw http.ResponseWriter, req *http.Request) {
	var requestJSON Request
	var cfg Config
	// Получаем переменные окружения
	err := env.Parse(&cfg)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	// чтение JSON объекта из body
	decoder := json.NewDecoder(req.Body)
	defer req.Body.Close()
	//десериализация
	if err := decoder.Decode(&requestJSON); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	// добавляем длинный url в хранилище, генерируем токен
	gToken, err := s.storage.AddLink(requestJSON.LongURL, cfg.File)
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
	_, urlParseErr := urlNet.Parse(response.ShortURL)
	if urlParseErr != nil {
		response.ShortURL = cfg.BaseURL + "/" + gToken
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
