package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"example.com/shortener/internal/app/models"
)

var (
	paramID         = "id"
	headerLocation  = "Location"
	contentTypeJSON = "application/json"
	encodGzip       = "gzip"
)

func (s *Server) deleteURLs(rw http.ResponseWriter, req *http.Request) {
	var sTokens []string
	log.Println("delete URLs")
	// читаем строку в формате [ "a", "b", "c", "d", ...]
	b, err := io.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		log.Printf("handlers|shortenURL|%v\n", err)
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	// конвертируем строку в массив с токенами
	sTokens = strings.Split(string(b), ",")

	// Запрос успешно принят 202 Accepted
	rw.WriteHeader(http.StatusAccepted)

	var cookieValue string
	cookie, err := req.Cookie("User")
	if err == nil {
		cookieValue = cookie.Value
	}

	workerChannel := make(chan string, len(sTokens))

	// в первой горутине отправляем токены в канал
	go s.service.AddDeletedTokens(sTokens, workerChannel)

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)

	s.service.Once.Do(func() {
		// во второй горутине получаем токены из канала и формируем  слайс для batch запроса
		go s.service.RecieveTokensFromChannel(ctx, workerChannel, cookieValue)
	})

	time.AfterFunc(6*time.Second, func() {
		log.Println("Запускаем cancel")
		cancel()
	})
}

func (s *Server) shortenURL(rw http.ResponseWriter, req *http.Request) {
	var gToken string
	var errToken error
	log.Println("shorten URL")

	// Читаем строку URL из body
	b, err := io.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		log.Printf("handlers|shortenURL|%v\n", err)
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	url := strings.Replace(string(b), "url=", "", 1)
	log.Printf("long url %s\n", url)

	var cookieValue string
	cookie, err := req.Cookie("User")
	if err == nil {
		cookieValue = cookie.Value
	}

	log.Printf("Возвращены куки %s\n", cookie)
	http.SetCookie(rw, cookie)

	// добавляем длинный url в хранилище, генерируем токен
	gToken, errToken = s.service.AddLink(req.Context(), "", url, cookieValue)

	if errToken != nil {
		if errors.Is(errToken, models.ErrorAlreadyExist) {
			// попытка сократить уже имеющийся в базе URL
			// возвращаем ответ с кодом 409
			rw.WriteHeader(http.StatusConflict)
			// пишем в тело ответа сокращенный URL
			log.Printf("Короткий URL из бд %s", gToken)
			fmt.Fprint(rw, gToken)
			return
		}
		http.Error(rw, errToken.Error(), http.StatusInternalServerError)
		return
	}

	// возвращаем ответ с кодом 201
	rw.WriteHeader(http.StatusCreated)

	// пишем в тело ответа сокращенный URL
	log.Printf("Short URL %s", gToken)
	fmt.Fprint(rw, gToken)
}

func (s *Server) shortenBatch(rw http.ResponseWriter, req *http.Request) {

	log.Println("Shorten Batch")
	// чтение JSON объектов из body
	decoder := json.NewDecoder(req.Body)
	defer req.Body.Close()

	//десериализация в слайс
	buffer := make([]models.BatchReq, 0, 100)

	if err := decoder.Decode(&buffer); err != nil {
		log.Printf("handlers|shortenBatch|%v\n", err)
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	var cookieValue string
	cookie, err := req.Cookie("User")
	if err == nil {
		cookieValue = cookie.Value
	}

	rw.Header().Set("Content-Type", contentTypeJSON)
	fmt.Printf("Возвращены куки %s\n", cookie)
	http.SetCookie(rw, cookie)

	response, err := s.service.ShortenBatch(req.Context(), buffer, cookieValue)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	} else {
		// возвращаем ответ с кодом 201
		rw.WriteHeader(http.StatusCreated)
	}

	// пишем в тело ответа закодированный в JSON объект
	// который содержит сокращенный URL
	// записываем результат JSON-сериализации в хранилище байт
	buf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	encoder.Encode(response)
	log.Printf("Ответ, закодированный в JSON, %s\n", response)
	fmt.Fprint(rw, buf)
}

func (s *Server) getFullURL(rw http.ResponseWriter, req *http.Request) {
	log.Println("Get full url")

	//получаем сокращенный url из параметра
	shortURL := chi.URLParam(req, paramID)
	log.Printf("short url %s\n", shortURL)

	// получаем длинный url
	lToken := s.service.GetLongToken(shortURL)
	longURL, err := s.service.GetLongURL(req.Context(), lToken)
	if err != nil {
		log.Printf("handlers|getFullURL %s |%v\n", lToken, err)
		if errors.Is(err, models.ErrLinkDeleted) {
			http.Error(rw, err.Error(), http.StatusGone)
			return
		}
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
		log.Printf("handlers|shortenJSON|%v\n", err)
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("request json %s\n", requestJSON)
	// добавляем длинный url в хранилище, генерируем токен
	var cookieValue string
	cookie, err := req.Cookie("User")
	if err == nil {
		cookieValue = cookie.Value
	}

	fmt.Printf("Возвращены куки %s\n", cookie)
	//req.AddCookie(cookie)
	http.SetCookie(rw, cookie)

	rw.Header().Set("Content-Type", contentTypeJSON)

	gToken, errToken = s.service.AddLink(req.Context(), "", requestJSON.LongURL, cookieValue)
	if errToken != nil {
		if errors.Is(errToken, models.ErrorAlreadyExist) {
			// попытка сократить уже имеющийся в базе URL
			// возвращаем ответ с кодом 409
			rw.WriteHeader(http.StatusConflict)
		} else {
			log.Printf("handlers|shortenJSON|%s\n", errToken.Error())
			http.Error(rw, errToken.Error(), http.StatusInternalServerError)
			return
		}

	} else {
		// возвращаем ответ с кодом 201
		rw.WriteHeader(http.StatusCreated)
	}

	// формируем json объект ответа
	response := Response{
		ShortURL: gToken,
	}
	log.Printf("short url %s\n", response.ShortURL)

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
		log.Printf("handlers|getUserURLs|%v\n", err)
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("куки value %s\n", user.Value)
	links, err := s.service.GetAllURLS(req.Context(), user.Value)
	if err != nil {
		log.Printf("database|GetAllURLs|%v\n", err)
	}

	if len(links) == 0 {
		log.Printf("Не нашли сокращенных пользователем URL")
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
	// возвращаем ответ с кодом 200
	rw.WriteHeader(http.StatusOK)

	// пишем в тело ответа закодированные JSON
	buf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	encoder.Encode(urls)
	fmt.Fprint(rw, buf)
}

func (s *Server) PingConnection(rw http.ResponseWriter, req *http.Request) {
	log.Println("Ping")
	if s.service.Ping(req.Context()) != nil {
		rw.WriteHeader(http.StatusInternalServerError)
	} else {
		rw.WriteHeader(http.StatusOK)
	}
}
