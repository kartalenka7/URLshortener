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

/* type GzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w GzipWriter) Write(b []byte) (int, error) {
	// w.Writer будет отвечать за gzip-сжатие, поэтому пишем в него
	return w.Writer.Write(b)
} */

// middleware принимает параметром Handler и возвращает тоже Handler
/* func gzipHandle(next http.Handler) http.Handler {
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
} */

func gzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Заголовок до gzipHandle %s", r.Header)
		if !strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			// если gzip не поддерживается, передаём управление
			// дальше без изменений
			log.Println("no gzip")
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodPost {
			if r.Header.Get("Content-Type") != "application/json" {
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
