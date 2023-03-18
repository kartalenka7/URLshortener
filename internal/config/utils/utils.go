package utils

import (
	"crypto/hmac"
	crypto "crypto/rand"
	"crypto/sha256"
	"math/rand"
	"net/http"
	"time"
)

var (
	localAddr = "localhost:8080"
	filename  = "link.log"
	baseURL   = "http://localhost:8080/"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// Генерирование короткой ссылки
func RandStringBytes(n int) string {
	rand.Seed(time.Now().UnixNano())

	link := make([]byte, n)
	for i := range link {
		link[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(link)
}

func GenerateCookies(cookie *http.Cookie) error {
	// сгенерировать криптостойкий слайс случайных байт
	key := make([]byte, 8)
	_, err := crypto.Read(key)
	if err != nil {
		return err
	}
	// подписываем алгоритмом HMAC, используя SHA256
	h := hmac.New(sha256.New, key)
	h.Write([]byte(cookie.Value))
	sign := h.Sum(nil)
	cookie.Value = string(sign) + cookie.Value
	return nil
}
