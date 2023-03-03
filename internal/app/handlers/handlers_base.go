package handlers

import (
	"log"

	"example.com/shortener/cmd/utils"
	"example.com/shortener/internal/app/storage"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	storage storage.StorageLinks
	config  utils.Config
}

func NewRouter(s *storage.StorageLinks, cfg *utils.Config) chi.Router {
	serv := &Server{
		storage: *s,
		config:  *cfg,
	}
	log.Println("выбираем роутер")
	// определяем роутер chi
	r := chi.NewRouter()
	// создадим суброутер, который будет содержать две функции
	r.Route("/", func(r chi.Router) {
		r.Post("/api/shorten", serv.shortenJSON)
		r.Get("/{id}", serv.getFullURL)
		r.Post("/", serv.shortenURL)
	})
	return r
}
