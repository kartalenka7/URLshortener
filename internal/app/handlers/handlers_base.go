package handlers

import (
	"io"
	"log"
	"strings"

	"compress/gzip"
	"net/http"

	"example.com/shortener/internal/app/storage"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	storage storage.StorageLinks
}

func gzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Заголовок до gzipHandle %s", r.Header)
		if !strings.Contains(r.Header.Get("Content-Encoding"), encodGzip) {
			// если gzip не поддерживается, передаём управление
			// дальше без изменений
			log.Println("no gzip")
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodPost {
			if r.Header.Get("Content-Type") != contentTypeJSON {
				// Распаковать длинный url из body с помощью gzip
				gz, err := gzip.NewReader(r.Body)
				if err != nil {
					log.Println(err.Error())
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				defer gz.Close()
				// при чтении вернётся распакованный слайс байт
				b, err := io.ReadAll(gz)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				log.Printf("long url after gzip %s\n", string(b))
				// пишем в тело распакованный url и передаем дальше в хэндлеры
				r.Body = io.NopCloser(strings.NewReader(string(b)))
			}

		} else if r.Method == http.MethodGet {
			//получаем сокращенный url из параметра
			shortURL := chi.URLParam(r, paramID)
			log.Printf("short url before gzip %s\n", shortURL)

			gz, err := gzip.NewReader(strings.NewReader(shortURL))
			if err != nil {
				log.Println(err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer gz.Close()
			short, err := io.ReadAll(gz)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Printf("short url after gzip %s\n", string(short))
			// записываем распакованный короткий URL в качестве параметра "id"
			// и передаем дальше в хэндлеры
			chi.NewRouteContext().URLParams.Add(paramID, string(short))
		}

		// замыкание — используем ServeHTTP следующего хендлера
		next.ServeHTTP(w, r)
	})
}

func NewRouter(s *storage.StorageLinks) chi.Router {
	serv := &Server{
		storage: *s,
	}
	// открываем файл и читаем сохраненные ссылки
	s.ReadFromFile()

	log.Println("выбираем роутер")
	// определяем роутер chi
	r := chi.NewRouter()

	// создадим суброутер, который будет содержать две функции
	r.Route("/", func(r chi.Router) {
		r.Use(gzipHandle)
		r.Post("/api/shorten", serv.shortenJSON)
		r.Get("/{id}", serv.getFullURL)
		r.Post("/", serv.shortenURL)
	})
	return r
}
