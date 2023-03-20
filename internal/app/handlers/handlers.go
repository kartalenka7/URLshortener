package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"context"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
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
	var gToken string
	var errToken error
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
	cookie, err := req.Cookie("User")
	if err != nil {
		log.Printf("handlers|shortenURL|%s\n", err.Error())
		gToken, errToken = s.storage.AddLink(url, "")
		if errToken != nil {
			log.Printf("handlers|AddLink|%s\n", errToken.Error())
			http.Error(rw, errToken.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		log.Println(cookie)
		gToken, errToken = s.storage.AddLink(url, cookie.Value)
		if errToken != nil {
			log.Printf("handlers|AddLink|%s\n", errToken.Error())
			http.Error(rw, errToken.Error(), http.StatusInternalServerError)
			return
		}
	}

	// записываем ссылки из мапы в файл
	s.storage.WriteInFile()

	fmt.Printf("Возвращены куки %s\n", cookie)
	//req.AddCookie(cookie)
	http.SetCookie(rw, cookie)

	// возвращаем ответ с кодом 201
	rw.WriteHeader(http.StatusCreated)

	// пишем в тело ответа сокращенный URL
	log.Printf("Short URL %s", gToken)
	fmt.Fprint(rw, gToken)
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
	var gToken string
	var errToken error

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
	cookie, err := req.Cookie("User")
	if err != nil {
		log.Printf("handlers|shortenJSON|%s\n", err.Error())
		gToken, errToken = s.storage.AddLink(requestJSON.LongURL, "")
		if errToken != nil {
			log.Printf("handlers|shortenJSON|%s\n", errToken.Error())
			http.Error(rw, errToken.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		gToken, errToken = s.storage.AddLink(requestJSON.LongURL, cookie.Value)
		if errToken != nil {
			log.Printf("handlers|shortenJSON|%s\n", errToken.Error())
			http.Error(rw, errToken.Error(), http.StatusInternalServerError)
			return
		}
	}

	// записываем ссылки из мапы в файл
	s.storage.WriteInFile()

	// формируем json объект ответа
	response := Response{
		ShortURL: gToken,
	}
	log.Printf("short url %s\n", response.ShortURL)

	fmt.Printf("Возвращены куки %s\n", cookie)
	//req.AddCookie(cookie)
	http.SetCookie(rw, cookie)

	rw.Header().Set("Content-Type", contentTypeJSON)
	// возвращаем ответ с кодом 201
	rw.WriteHeader(http.StatusCreated)

	// пишем в тело ответа закодированный в JSON объект
	// который содержит сокращенный URL
	fmt.Fprint(rw, response.ToJSON())

}

func (r *Response) ToJSON() *bytes.Buffer {
	// записываем результат JSON-сериализации в хранилище байт
	buf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	encoder.Encode(r)
	return buf
}

type CookiesURL struct {
	ShortURL string `json:"short_url"`
	OrigURL  string `json:"original_url"`
}

type cookies struct {
	URLs []*CookiesURL
}

func (s *Server) getUserURLs(rw http.ResponseWriter, req *http.Request) {
	log.Println("Get all urls for user")
	user, err := req.Cookie("User")
	if err != nil {
		log.Printf("handlers|getUserURLs|%s\n", err.Error())
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("куки value %s\n", user.Value)
	links := s.storage.GetAllURLS(user.Value)

	if len(links) == 0 {
		// не нашли сокращенных пользователем URL
		rw.WriteHeader(http.StatusNoContent)
		return
	}
	var urls []*CookiesURL
	// формируем json объект ответа
	for short, long := range links {
		urls = append(urls, &CookiesURL{
			ShortURL: short,
			OrigURL:  long})
		fmt.Printf("Возвращаемые url для текущего пользователя %s\n", &CookiesURL{
			ShortURL: short,
			OrigURL:  long})
	}

	rw.Header().Set("Content-Type", contentTypeJSON)
	log.Printf("Возвращаемый заголовок %s\n", rw.Header())
	// возвращаем ответ с кодом 201
	rw.WriteHeader(http.StatusCreated)

	// пишем в тело ответа закодированные JSON
	buf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	encoder.Encode(urls)
	fmt.Fprint(rw, buf)
}

func (s *Server) PostgresConnection(rw http.ResponseWriter, req *http.Request) {
	log.Println("Ping")
	connString := s.storage.GetConnSrtring()
	db, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		log.Printf("handlers|PostgresConnection|%s\n", err.Error())
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err = db.Ping(ctx); err != nil {
		log.Println(err.Error())
		rw.WriteHeader(http.StatusInternalServerError)
	}
	rw.WriteHeader(http.StatusOK)
}
