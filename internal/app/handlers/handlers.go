// Модуль handlers содержит инициализацию роутера,
// методы-обработчики запросов, а также методы для
// реализации middleware.
package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	"example.com/shortener/internal/app/models"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

// Параметры запроса
var (
	paramID         = "id"
	headerLocation  = "Location"
	contentTypeJSON = "application/json"
	encodGzip       = "gzip"
)

// DeleteURLs принимает строку с токенами и запускает горутину на удаление записей
func (s *Server) DeleteURLs(rw http.ResponseWriter, req *http.Request) {
	var sTokens []string
	s.log.Debug("delete URLs")
	// читаем строку в формате [ "a", "b", "c", "d", ...]
	b, err := io.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		s.log.Error(err.Error())
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

	// отправляем токены в канал
	go s.service.AddDeletedTokens(sTokens, cookieValue)

}

// ShortenURL - обработчик для запроса POST /
// возвращает сокращенный токен в теле ответа
func (s *Server) ShortenURL(rw http.ResponseWriter, req *http.Request) {
	var gToken string
	var errToken error
	s.log.Info("shorten URL")

	// Читаем строку URL из body
	b, err := io.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		s.log.Error(err.Error())
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	url := strings.Replace(string(b), "url=", "", 1)
	s.log.WithFields(logrus.Fields{"long url": url})

	var cookieValue string
	cookie, err := req.Cookie("User")
	if err == nil {
		cookieValue = cookie.Value
	}

	s.log.WithFields(logrus.Fields{"returned cookie": cookie})
	http.SetCookie(rw, cookie)

	// добавляем длинный url в хранилище, генерируем токен
	gToken, errToken = s.service.AddLink(req.Context(), "", url, cookieValue)

	if errToken != nil {
		if errors.Is(errToken, models.ErrorAlreadyExist) {
			// попытка сократить уже имеющийся в базе URL
			// возвращаем ответ с кодом 409
			rw.WriteHeader(http.StatusConflict)
			// пишем в тело ответа сокращенный URL
			s.log.WithFields(logrus.Fields{"short URL from db": gToken})
			fmt.Fprint(rw, gToken)
			return
		}
		http.Error(rw, errToken.Error(), http.StatusInternalServerError)
		return
	}

	// возвращаем ответ с кодом 201
	rw.WriteHeader(http.StatusCreated)

	// пишем в тело ответа сокращенный URL
	s.log.WithFields(logrus.Fields{"Short URL": gToken})
	fmt.Fprint(rw, gToken)
}

// shortenBatch сокращает URL, переданные в виде списка JSON объектов.
// возвращает результат с сокращенными токенами также в виде JSON объекта
func (s *Server) shortenBatch(rw http.ResponseWriter, req *http.Request) {

	s.log.Info("Shorten Batch")
	// чтение JSON объектов из body
	decoder := json.NewDecoder(req.Body)
	defer req.Body.Close()

	//десериализация в слайс
	buffer := make([]models.BatchReq, 0, 100)

	if err := decoder.Decode(&buffer); err != nil {
		s.log.Error(err.Error())
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	var cookieValue string
	cookie, err := req.Cookie("User")
	if err == nil {
		cookieValue = cookie.Value
	}

	rw.Header().Set("Content-Type", contentTypeJSON)
	s.log.WithFields(logrus.Fields{"cookie": cookie})
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
	s.log.WithFields(logrus.Fields{"response": response}).Info("Ответ, закодированный в JSON")
	fmt.Fprint(rw, buf)
}

// GetFullURL - обработчик запроса GET /{id}, где id - сокращенный токен
// возвращает исходный URL
func (s *Server) GetFullURL(rw http.ResponseWriter, req *http.Request) {
	s.log.Info("Get full url")

	//получаем сокращенный url из параметра
	shortURL := chi.URLParam(req, paramID)
	s.log.WithFields(logrus.Fields{"short url": shortURL})

	// получаем длинный url
	lToken := s.service.GetLongToken(shortURL)
	longURL, err := s.service.GetLongURL(req.Context(), lToken)
	if err != nil {
		s.log.Error(err.Error())
		if errors.Is(err, models.ErrLinkDeleted) {
			http.Error(rw, err.Error(), http.StatusGone)
			return
		}
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	// возвращаем длинный url в поле Location
	rw.Header().Set(headerLocation, longURL)
	s.log.WithFields(logrus.Fields{"header": rw.Header()}).Info("Заголовок возврата")

	// возвращаем ответ с кодом 307
	rw.WriteHeader(http.StatusTemporaryRedirect)

}

// Request - структура данных для запроса в формате JSON
type Request struct {
	LongURL string `json:"url"`
}

// Response - структура для ответа в формате JSON
type Response struct {
	ShortURL string `json:"result"`
}

// shortenJson принимает в теле запроса JSON объект с URL
// и возвращает JSON объект с сокращенным токеном
func (s *Server) shortenJSON(rw http.ResponseWriter, req *http.Request) {
	var gToken string
	var errToken error

	s.log.Info("POST JSON")
	// чтение JSON объекта из body
	decoder := json.NewDecoder(req.Body)
	defer req.Body.Close()
	//десериализация
	requestJSON := Request{}
	if err := decoder.Decode(&requestJSON); err != nil {
		s.log.Error(err.Error())
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

	s.log.WithFields(logrus.Fields{"cookie": cookie})
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
			s.log.Error(errToken.Error())
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
	s.log.WithFields(logrus.Fields{"short url": response.ShortURL})

	// пишем в тело ответа закодированный в JSON объект
	// который содержит сокращенный URL
	fmt.Fprint(rw, response.ToJSON())

}

// ToJSON записывает результат JSON-сериализации в хранилище байт
func (r *Response) ToJSON() *bytes.Buffer {
	buf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	encoder.Encode(r)
	return buf
}

// в CookiesURL записываем все URL, скоращенные пользователем
type CookiesURL struct {
	ShortURL string `json:"short_url"`
	OrigURL  string `json:"original_url"`
}

// getUserURLs возвращает все URL, сокращенным пользвателем
func (s *Server) GetUserURLs(rw http.ResponseWriter, req *http.Request) {
	s.log.Debug("Get all urls for user")
	user, err := req.Cookie("User")
	if err != nil {
		s.log.Error(err.Error())
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	s.log.WithFields(logrus.Fields{"cookie value": user.Value})
	links, err := s.service.GetAllURLS(req.Context(), user.Value)
	if err != nil {
		s.log.Error(err.Error())
	}

	if len(links) == 0 {
		s.log.Debug("Не нашли сокращенных пользователем URL")
		rw.WriteHeader(http.StatusNoContent)
		return
	}
	var urls []*CookiesURL
	// формируем json объект ответа
	for short, long := range links {
		urls = append(urls, &CookiesURL{
			ShortURL: short,
			OrigURL:  long})
		s.log.WithFields(logrus.Fields{"short url": short,
			"original url": long}).Info("Возвращаемые url для текущего пользователя ")
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

// PingConnection проверяет соединение с БД
func (s *Server) PingConnection(rw http.ResponseWriter, req *http.Request) {
	s.log.Info("Ping")
	if s.service.Ping(req.Context()) != nil {
		rw.WriteHeader(http.StatusInternalServerError)
	} else {
		rw.WriteHeader(http.StatusOK)
	}
}

// GetStats проверяет что ip клиента входит в доверенную подсеть
// и возвращает статистику в формате JSON
func (s *Server) GetStats(rw http.ResponseWriter, req *http.Request) {
	s.log.Info("Get stats")

	ipstr := req.Header.Get("X-Real-IP")

	ip := net.ParseIP(ipstr)
	if ip == nil {
		rw.WriteHeader(http.StatusBadRequest)
	}

	stats, err := s.service.CheckIPMask(req.Context(), ip)
	if err != nil {
		if errors.Is(err, models.ErrNotTrustedSubnet) {
			s.log.Info("IP не входит в доверенную подсеть")
			rw.WriteHeader(http.StatusForbidden)
		}
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	buf := bytes.NewBuffer([]byte{})
	if err := json.NewEncoder(buf).Encode(stats); err != nil {
		s.log.Error(err.Error())
		return
	}
	rw.WriteHeader(http.StatusOK)
	fmt.Fprint(rw, buf)

}
