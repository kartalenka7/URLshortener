package handlers

import (
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"strings"

	"example.com/shortener/internal/config/utils"
)

func gzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Content-Encoding"), encodGzip) {
			// если gzip не поддерживается, передаём управление
			// дальше без изменений
			log.Println("no gzip")
			log.Printf("Заголовок %s\n", r.Header)
			next.ServeHTTP(w, r)
			return
		}

		if r.Header.Get("Content-Encoding") == "gzip" {
			// Распаковать длинный url из body с помощью gzip
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				log.Printf("handlers_base|gzipHandle|%v\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				next.ServeHTTP(w, r)
				return
			}
			defer gz.Close()
			// при чтении вернётся распакованный слайс байт
			b, err := io.ReadAll(gz)
			if err != nil {
				log.Printf("handlers_base|gzipHandle|%v\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// пишем в тело распакованный url и передаем дальше в хэндлеры
			r.Body = io.NopCloser(strings.NewReader(string(b)))

		}
		// замыкание — используем ServeHTTP следующего хендлера
		next.ServeHTTP(w, r)
	})
}

func AddCookie(r *http.Request) error {
	log.Println("Не нашли куки User")
	usercookie, err := utils.WriteCookies()
	if err != nil {
		log.Printf("handlers_base|userAuth|%v\n", err)
		return err
	}
	// выдать пользователю симметрично подписанную куку
	log.Printf("куки %s\n", &usercookie)
	r.AddCookie(&usercookie)
	return nil
}

func userAuth(next http.Handler) http.Handler {
	log.Println("middleware")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// получаем куки
		cookie, err := r.Cookie("User")
		if err != nil {
			if err = AddCookie(r); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		log.Printf("Нашли куки %s\n", cookie)
		err = utils.ReadCookies(*cookie)
		if err != nil {
			log.Printf("handlers_base|userAuth|%v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		r.AddCookie(cookie)

		// замыкание — используем ServeHTTP следующего хендлера
		next.ServeHTTP(w, r)
	})
}
