package utils

import (
	"math/rand"
	"time"
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
