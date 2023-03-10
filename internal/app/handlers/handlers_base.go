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

type GzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w GzipWriter) Write(b []byte) (int, error) {
	// w.Writer будет отвечать за gzip-сжатие, поэтому пишем в него
	return w.Writer.Write(b)
}

// middleware принимает параметром Handler и возвращает тоже Handler
func gzipHandle(next http.Handler) http.Handler {
	log.Println("gzips")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// проверяем, что клиент поддерживает gzip-сжатие
		log.Printf("Заголовок до gzipHandle %s", r.Header)
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
		log.Printf("Заголовок после GzipHandler, %s", w.Header())
		// замыкание — используем ServeHTTP следующего хендлера
		// передаём обработчику страницы переменную типа gzipWriter для вывода данных
		next.ServeHTTP(GzipWriter{ResponseWriter: w, Writer: gz}, r)
	})
}

func ReaderHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		longURLGzip := r.Header.Get("Location")
		log.Printf("Gzip 2,%s \n", longURLGzip)
		/* 		if longURLGzip == "" {
		   			return
		   		}
		   		if r.Header.Get(`Content-Encoding`) == `gzip` {
		   			gz, err := gzip.NewReader(strings.NewReader(longURLGzip))
		   			if err != nil {
		   				http.Error(w, err.Error(), http.StatusInternalServerError)
		   				return
		   			}
		   			defer gz.Close()

		   			location, err := io.ReadAll(gz)
		   			if err != nil {
		   				http.Error(w, err.Error(), http.StatusInternalServerError)
		   				return
		   			}
		   			longURLGzip = string(location)
		   		}

		   		w.Header().Set(headerLocation, longURLGzip)
		   		// возвращаем ответ с кодом 307
		   		w.WriteHeader(http.StatusTemporaryRedirect)

		   		log.Printf("Итоговый длинный URL %s\n", longURLGzip) */
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
	// создадим суброутер, который будет содержать две функции
	r.Route("/", func(r chi.Router) {
		r.Use(gzipHandle)
		r.Post("/api/shorten", serv.shortenJSON)
		r.Get("/{id}", serv.getFullURL)
		r.Post("/", serv.shortenURL)
	})
	return r
}
