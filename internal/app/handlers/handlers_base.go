package handlers

import (
	"net/http/pprof"

	service "example.com/shortener/internal/app/service"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

// Server реализует все методы-обработчики.
// поле service служит для взаимодействия с модулем service
type Server struct {
	service service.Service
	log     *logrus.Logger
}

// NewRouter возвращает экземпляр роутера chi
// и определяет основные обработчики для приложения
func NewRouter(service *service.Service, log *logrus.Logger) chi.Router {
	log.Println("выбираем роутер")
	serv := &Server{
		service: *service,
		log:     log,
	}

	// определяем роутер chi
	r := chi.NewRouter()

	// создадим суброутер, который будет содержать две функции
	r.Route("/", func(r chi.Router) {
		// аутентификация пользователя
		r.Use(userAuth)
		// обработка сжатия gzip
		r.Use(gzipHandle)

		r.HandleFunc("/debug/pprof", pprof.Index)
		r.HandleFunc("/debug/pprof/profile", pprof.Profile)

		r.Post("/api/shorten/batch", serv.shortenBatch)
		//удаление URL пользователем
		r.Delete("/api/user/urls", serv.DeleteURLs)
		// сокращение URL в JSON формате
		r.Post("/api/shorten", serv.shortenJSON)
		// все URL пользователя, которые он сокращал
		r.Get("/api/user/urls", serv.GetUserURLs)
		// возвращает общее число сокращенных URL и пользователей
		r.Get("/api/internal/stats", serv.GetStats)
		// проверка соединения с бд
		r.Get("/ping", serv.PingConnection)
		// получение полного URL по сокращенному
		r.Get("/{id}", serv.GetFullURL)
		// сокращение URL
		r.Post("/", serv.ShortenURL)

	})
	return r
}
