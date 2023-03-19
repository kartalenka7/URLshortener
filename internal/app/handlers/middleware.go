package handlers

import (
	"fmt"
	"io"
	"log"
	"strings"

	"compress/gzip"
	"net/http"

	"example.com/shortener/internal/config/utils"
)

func gzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				next.ServeHTTP(w, r)
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
			// пишем в тело распакованный url и передаем дальше в хэндлеры
			r.Body = io.NopCloser(strings.NewReader(string(b)))

		}
		// замыкание — используем ServeHTTP следующего хендлера
		next.ServeHTTP(w, r)
	})
}

func userAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// получаем куки
		cookie, err := r.Cookie("User")
		if err != nil {
			fmt.Println("Не нашли куки User")
			Usercookie := http.Cookie{}
			Usercookie, err := utils.WriteCookies()
			if err != nil {
				log.Printf("handlers_base|userAuth|%s\n", err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// куки не найдены, выдать пользователю симметрично подписанную куку
			log.Printf("куки %s\n", &Usercookie)
			r.AddCookie(&Usercookie)
			next.ServeHTTP(w, r)
			return
		}
		log.Printf("Нашли куки %s\n", cookie)
		err = utils.ReadCookies(*cookie)
		if err != nil {
			log.Printf("handlers_base|userAuth|%s\n", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// замыкание — используем ServeHTTP следующего хендлера
		next.ServeHTTP(w, r)
	})
}
