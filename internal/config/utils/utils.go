package utils

import (
	"crypto/hmac"
	crypto "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"log"
	"math/big"
	"math/rand"
	"net"
	"net/http"
	urlNet "net/url"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	localAddr = "localhost:8080"
	filename  = "link.log"
	baseURL   = "http://localhost:8080/"
	secretkey = []byte("secret key")
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// RandStringBytes генерирует короткий токен
func RandStringBytes(n int) string {
	rand.Seed(time.Now().UnixNano())

	link := make([]byte, n)
	for i := range link {
		link[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(link)
}

// GenRandToken возваращает сокращенный URL
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

// generateUserToken генерирует криптостойкий слайс случайных байт
func generateUserToken(len int) (string, error) {
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

// WriteCookies формирует cookie для пользователя
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

// ReadCookies проверяет hmac подпись в cookie
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

// GenerateTSL генерирует сертификат x.509 и RSA приватный ключ
func GenerateCertTSL(log *logrus.Logger) error {
	// создаем шаблон сертификата
	cert := &x509.Certificate{
		// указываем уникальный номер сертификата
		SerialNumber: big.NewInt(1568),
		// Заполняем базовую информацию о владельце сертификата
		Subject: pkix.Name{
			Organization: []string{"Yandex.Praktikum"},
			Country:      []string{"RU"},
		},
		// разрешаем использование сертификата для 127.0.0.1 и ::1
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		// сертификат верен, начиная со времени создания
		NotBefore: time.Now(),
		// время жизни сертификата — 10 лет
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		// устанавливаем использование ключа для цифровой подписи,
		// а также клиентской и серверной авторизации
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}

	// создаём новый приватный RSA-ключ длиной 4096 бит
	privateKey, err := rsa.GenerateKey(crypto.Reader, 4096)
	if err != nil {
		log.Fatal(err.Error())
	}

	// создаём сертификат x.509
	certBytes, err := x509.CreateCertificate(crypto.Reader, cert, cert,
		&privateKey.PublicKey, privateKey)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	// кодируем ключ и сертификат в формат REM, который используется
	// для хранения и обмена криптографическими ключами

	certOut, err := os.Create("cert.pem")
	if err != nil {
		log.Error(err.Error())
		return err
	}
	pem.Encode(
		certOut, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certBytes,
		},
	)
	certOut.Close()

	keyOut, err := os.OpenFile("key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	pem.Encode(
		keyOut, &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		},
	)
	keyOut.Close()

	return nil
}
