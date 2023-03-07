package handlers

import (
	"io"
	"log"
	"strings"

	"compress/gzip"
	"net/http"

	"example.com/shortener/cmd/utils"
	"example.com/shortener/internal/app/storage"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	storage storage.StorageLinks
	config  utils.Config
}

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	// w.Writer будет отвечать за gzip-сжатие, поэтому пишем в него
	return w.Writer.Write(b)
}

func gzipHandle(next http.Handler) http.Handler {
	log.Println("gzips")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Header.Get("Accept-Encoding"))
		// проверяем, что клиент поддерживает gzip-сжатие
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			// если gzip не поддерживается, передаём управление
			// дальше без изменений
			log.Println("not suited")
			next.ServeHTTP(w, r)
			return
		}

		// создаём gzip.Writer поверх текущего w
		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		log.Println(w.Header())
		// передаём обработчику страницы переменную типа gzipWriter для вывода данных
		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
	})
}

func NewRouter(s *storage.StorageLinks, cfg *utils.Config) chi.Router {
	serv := &Server{
		storage: *s,
		config:  *cfg,
	}
	log.Println("выбираем роутер")
	// определяем роутер chi
	r := chi.NewRouter()
	r.Use(gzipHandle)
	// создадим суброутер, который будет содержать две функции
	r.Route("/", func(r chi.Router) {
		r.Post("/api/shorten", serv.shortenJSON)
		r.Get("/{id}", serv.getFullURL)
		r.Post("/", serv.shortenURL)
	})
	return r
}
