package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

var (
	paramID         = "id"
	headerLocation  = "Location"
	contentTypeJSON = "application/json"
	encodGzip       = "gzip"
)

type Repository interface {
	AddLink(gToken string, longURL string) error
	GetLongURL(sToken string) (string, error)
	GetStorageLen() int
}

// Структура для парсинга переменных окружения

func (s *Server) shortenURL(rw http.ResponseWriter, req *http.Request) {
	log.Println("shorten URL")

	// Читаем строку URL из body
	b, err := io.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		log.Printf("handlers|shortenURL|%s\n", err.Error())
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	url := strings.Replace(string(b), "url=", "", 1)
	log.Printf("long url %s\n", url)

	// добавляем длинный url в хранилище, генерируем токен
	gToken, err := s.storage.AddLink(url)
	if err != nil {
		log.Printf("handlers|AddLink|%s\n", err.Error())
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	// возвращаем ответ с кодом 201
	rw.WriteHeader(http.StatusCreated)
	// пишем в тело ответа сокращенный URL
	log.Printf("Short URL %s", gToken)

	fmt.Fprint(rw, gToken)
	// записываем ссылки из мапы в файл
	s.storage.WriteInFile()
}

func (s *Server) getFullURL(rw http.ResponseWriter, req *http.Request) {
	log.Println("Get full url")

	//получаем сокращенный url из параметра
	shortURL := chi.URLParam(req, paramID)
	log.Printf("short url %s\n", shortURL)
	// получаем длинный url
	longURL, err := s.storage.GetLongURL(shortURL)
	if err != nil {
		log.Printf("handlers|getFullURL|%s\n", err.Error())
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	// возвращаем длинный url в поле Location
	rw.Header().Set(headerLocation, longURL)
	log.Printf("Заголовок возврата %s \n", rw.Header())
	// возвращаем ответ с кодом 307
	rw.WriteHeader(http.StatusTemporaryRedirect)
}

type Request struct {
	LongURL string `json:"url"`
}

type Response struct {
	ShortURL string `json:"result"`
}

func (s *Server) shortenJSON(rw http.ResponseWriter, req *http.Request) {

	log.Println("POST JSON")
	// чтение JSON объекта из body
	decoder := json.NewDecoder(req.Body)
	defer req.Body.Close()
	//десериализация
	requestJSON := Request{}
	if err := decoder.Decode(&requestJSON); err != nil {
		log.Printf("handlers|shortenJSON|%s\n", err.Error())
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("request json %s\n", requestJSON)
	// добавляем длинный url в хранилище, генерируем токен
	gToken, err := s.storage.AddLink(requestJSON.LongURL)
	if err != nil {
		log.Printf("handlers|shortenJSON|%s\n", err.Error())
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	// формируем json объект ответа
	response := Response{
		ShortURL: gToken,
	}
	log.Printf("short url %s\n", response.ShortURL)

	rw.Header().Set("Content-Type", contentTypeJSON)
	// возвращаем ответ с кодом 201
	rw.WriteHeader(http.StatusCreated)
	// пишем в тело ответа закодированный в JSON объект
	// который содержит сокращенный URL
	fmt.Fprint(rw, response.ToJSON())
	// записываем ссылки из мапы в файл
	s.storage.WriteInFile()

}

func (r *Response) ToJSON() *bytes.Buffer {
	// записываем результат JSON-сериализации в хранилище байт
	buf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	encoder.Encode(r)
	return buf
}

func (s *Server) getUserURLs(rw http.ResponseWriter, req *http.Request) {
	// не нашли сокращенных пользователем URL
	rw.WriteHeader(http.StatusNoContent)
}
