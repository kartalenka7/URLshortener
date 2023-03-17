package utils

import (
	"crypto/hmac"
	crypto "crypto/rand"
	"crypto/sha256"
	"math/rand"
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

func GenerateCookies() ([]byte, error) {
	// сгенерировать криптостойкий слайс случайных байты
	b := make([]byte, 512)
	_, err := crypto.Read(b)
	if err != nil {
		return nil, err
	}
	key := (b[:4])
	// подписываем алгоритмом HMAC, используя SHA256
	h := hmac.New(sha256.New, key)
	h.Write(b)
	dst := h.Sum(nil)
	return dst, nil
}
