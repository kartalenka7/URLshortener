package utils

import (
	"math/rand"
	"time"
)

var (
	localAddr = "localhost:8080"
	filename  = "link.log"
	baseURL   = "http://localhost:8080/"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Генерирование короткой ссылки
func RandStringBytes(n int) string {
	link := make([]byte, n)
	for i := range link {
		link[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(link)
}
