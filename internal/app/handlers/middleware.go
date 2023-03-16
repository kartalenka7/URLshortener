package handlers

import (
	"io"
	"log"
	"strings"

	"compress/gzip"
	"net/http"
)

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	// w.Writer будет отвечать за gzip-сжатие, поэтому пишем в него
	return w.Writer.Write(b)
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

		if r.Header.Get("Content-Encoding") == "gzip" {

			// Распаковать длинный url из body с помощью gzip
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				log.Printf("handlers_base|gzipHandle|%s\n", err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer gz.Close()
			// при чтении вернётся распакованный слайс байт
			b, err := io.ReadAll(gz)
			if err != nil {
				log.Printf("handlers_base|gzipHandle|%s\n", err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Printf("long url after gzip %s\n", string(b))
			// пишем в тело распакованный url и передаем дальше в хэндлеры
			r.Body = io.NopCloser(strings.NewReader(string(b)))

			/* 			//получаем сокращенный url из параметра
			   			shortURL := chi.URLParam(r, paramID)
			   			log.Printf("short url before gzip %s\n", shortURL)

			   			gz, err = gzip.NewReader(strings.NewReader(shortURL))
			   			if err != nil {
			   				log.Printf("handlers_base|gzipHandle|%s\n", err.Error())
			   				http.Error(w, err.Error(), http.StatusInternalServerError)
			   				return
			   			}
			   			defer gz.Close()
			   			short, err := io.ReadAll(gz)
			   			if err != nil {
			   				log.Printf("handlers_base|gzipHandle|%s\n", err.Error())
			   				http.Error(w, err.Error(), http.StatusInternalServerError)
			   				return
			   			}
			   			log.Printf("short url after gzip %s\n", string(short))
			   			// записываем распакованный короткий URL в качестве параметра "id"
			   			// и передаем дальше в хэндлеры
			   			chi.NewRouteContext().URLParams.Add(paramID, string(short)) */

		}

		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			// создаём gzip.Writer поверх текущего w
			gzWriter, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
			if err != nil {
				io.WriteString(w, err.Error())
				return
			}
			defer gzWriter.Close()
			next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gzWriter}, r)
		}

		// замыкание — используем ServeHTTP следующего хендлера
		next.ServeHTTP(w, r)
	})
}

func userAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// получаем куки
		_, err := r.Cookie("User")
		if err != nil {
			var Usercookie *http.Cookie
			// куки не найдены, выдать пользователю симметрично подписанную куку
			http.SetCookie(w, Usercookie)
		}
	})
}
