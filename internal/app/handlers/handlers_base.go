package handlers

import (
	"log"

	"example.com/shortener/internal/app/storage"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	storage storage.StorageLinks
}

func NewRouter(s *storage.StorageLinks) chi.Router {
	log.Println("выбираем роутер")
	serv := &Server{
		storage: *s,
	}

	// определяем роутер chi
	r := chi.NewRouter()

	// создадим суброутер, который будет содержать две функции
	r.Route("/", func(r chi.Router) {
		// аутентификация пользователя
		r.Use(userAuth)
		// обработка сжатия gzip
		r.Use(gzipHandle)
		r.Post("/api/shorten/batch", serv.shortenBatch)
		// сокращение URL в JSON формате
		r.Post("/api/shorten", serv.shortenJSON)
		// все URL пользователя, которые он сокращал
		r.Get("/api/user/urls", serv.getUserURLs)
		// проверка соединения с бд
		r.Get("/ping", serv.PostgresConnection)
		// получение полного URL по скоращенному
		r.Get("/{id}", serv.getFullURL)
		// сокращение URL
		r.Post("/", serv.shortenURL)

	})
	return r
}
