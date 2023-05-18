package utils

import (
	"crypto/hmac"
	crypto "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log"
	"math/rand"
	"net/http"
	urlNet "net/url"
	"time"
)

var (
	localAddr = "localhost:8080"
	filename  = "link.log"
	baseURL   = "http://localhost:8080/"
	secretkey = []byte("secret key")
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

func GenRandToken(baseURL string) string {
	gToken := RandStringBytes(10)
	log.Println(gToken)
	sToken := baseURL + gToken
	_, urlParseErr := urlNet.Parse(sToken)
	if urlParseErr != nil {
		sToken = baseURL + "/" + gToken
		log.Printf("Short URL %s", sToken)
	}
	return sToken
}

func generateUserToken(len int) (string, error) {
	// сгенерировать криптостойкий слайс случайных байт
	b := make([]byte, len)
	_, err := crypto.Read(b)
	if err != nil {
		return "", err
	}
	// кодируем массив, чтобы использовать его для куки
	UserToken := hex.EncodeToString(b)

	log.Printf("UserToken %s\n", UserToken)
	return UserToken, nil
}

func WriteCookies() (http.Cookie, error) {
	var err error
	cookie := http.Cookie{
		Name: "User",
	}
	cookie.Value, err = generateUserToken(16)
	if err != nil {
		return cookie, err
	}
	// подписываем алгоритмом HMAC, используя SHA256
	h := hmac.New(sha256.New, secretkey)
	h.Write([]byte(cookie.Value))
	log.Printf("cookie.Value %s\n", cookie.Value)
	sign := h.Sum(nil)

	log.Printf("sign %s\n", sign)
	cookie.Value = string(sign) + cookie.Value
	cookie.Value = base64.URLEncoding.EncodeToString([]byte(cookie.Value))
	return cookie, nil
}

func ReadCookies(cookie http.Cookie) error {
	signedValue, err := base64.URLEncoding.DecodeString(cookie.Value)
	if err != nil {
		log.Printf("utils|readCookies|%v\n", err)
		return err
	}

	signature := signedValue[:sha256.Size]
	value := signedValue[sha256.Size:]

	mac := hmac.New(sha256.New, secretkey)
	mac.Write([]byte(value))
	expectedSignature := mac.Sum(nil)

	if !hmac.Equal([]byte(signature), expectedSignature) {
		log.Printf("handlers_base|userAuth|%s\n", errors.New("подпись не совпадает"))
		return err
	}
	return nil

}
