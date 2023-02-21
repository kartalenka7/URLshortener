package handlers

import (
	"example.com/shortener/internal/app/storage"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	storage storage.StorageLinks
}

func NewRouter(s *storage.StorageLinks) chi.Router {
	serv := &Server{
		storage: *s,
	}
	// определяем роутер chi
	r := chi.NewRouter()
	// создадим суброутер, который будет содержать две функции
	r.Route("/", func(r chi.Router) {
		r.Post("/", serv.shortenURL)
		r.Get("/{id}", serv.getFullURL)
	})
	return r
}
