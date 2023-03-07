package handlers

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	urlNet "net/url"
	"strings"

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

// Структура для парсинга переменных окружения

func (s *Server) shortenURL(rw http.ResponseWriter, req *http.Request) {
	var url string
	var err error
	log.Println("shorten URL")
	if !strings.Contains(req.Header.Get("Content-Encoding"), "gzip") {
		// Читаем строку URL из body
		b, err := io.ReadAll(req.Body)
		defer req.Body.Close()
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		url = strings.Replace(string(b), "url=", "", 1)
		log.Printf("long url %s\n", url)

	} else {
		gz, err := gzip.NewReader(req.Body)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		// не забывайте потом закрыть *gzip.Reader
		defer gz.Close()

		// при чтении вернётся распакованный слайс байт
		b, err := io.ReadAll(gz)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		url = strings.Replace(string(b), "url=", "", 1)
		log.Printf("long url gzip %s\n", url)
	}
	// добавляем длинный url в хранилище, генерируем токен
	gToken, err := s.storage.AddLink(url, s.config.File)
	if err != nil {
		log.Println(err.Error())
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	// возвращаем ответ с кодом 201
	rw.WriteHeader(http.StatusCreated)
	// пишем в тело ответа сокращенный URL
	sToken := s.config.BaseURL + gToken
	_, urlParseErr := urlNet.Parse(sToken)
	if urlParseErr != nil {
		sToken = s.config.BaseURL + "/" + gToken
		fmt.Fprint(rw, sToken)
		log.Printf("Short URL %s", sToken)
		return
	}
	log.Printf("Short URL %s", sToken)

	fmt.Fprint(rw, sToken)
}
func (s *Server) getFullURL(rw http.ResponseWriter, req *http.Request) {
	var err error
	log.Println("Get full url")

	//получаем сокращенный url из параметра
	shortURL := chi.URLParam(req, paramID)
	log.Printf("short url %s\n", shortURL)
	// получаем длинный url
	longURL, err := s.storage.GetLongURL(shortURL, s.config.File)
	if err != nil {
		fmt.Println(err.Error())
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	// возвращаем длинный url в поле Location

	log.Printf("Заголовок %s\n", req.Header)
	rw.Header().Set(headerLocation, longURL)
	log.Printf("Заголовок возврата %s \n", rw.Header())
	// возвращаем ответ с кодом 307
	rw.WriteHeader(http.StatusTemporaryRedirect)
}

type Request struct {
	LongURL string `json:"url"`
}

func (s *Server) shortenJSON(rw http.ResponseWriter, req *http.Request) {
	var requestJSON Request

	log.Println("POST JSON")
	// чтение JSON объекта из body
	decoder := json.NewDecoder(req.Body)
	defer req.Body.Close()
	//десериализация
	if err := decoder.Decode(&requestJSON); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	// добавляем длинный url в хранилище, генерируем токен
	gToken, err := s.storage.AddLink(requestJSON.LongURL, s.config.File)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	// формируем json объект ответа
	response := struct {
		ShortURL string `json:"result"`
	}{
		ShortURL: s.config.BaseURL + gToken,
	}
	_, urlParseErr := urlNet.Parse(response.ShortURL)
	if urlParseErr != nil {
		response.ShortURL = s.config.BaseURL + "/" + gToken
	}
	log.Printf("short url %s\n", response.ShortURL)

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
