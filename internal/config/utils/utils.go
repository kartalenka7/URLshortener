package utils

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"flag"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	BaseURL string `env:"BASE_URL" envDefault:"http://localhost:8080/"`
	Server  string `env:"SERVER_ADDRESS" envDefault:"localhost:8080"`
	File    string `env:"FILE_STORAGE_PATH"`
}

var (
	localAddr = "localhost:8080"
	filename  = "links.log"
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

func VarParse() (Config, error) {
	var cfg Config
	var cfgFlag Config
	// Парсим переменные окружения
	fmt.Println("Parse")
	err := env.Parse(&cfg)
	if err != nil {
		fmt.Printf("ошибка %s", err.Error())
		return cfg, err
	}

	log.Println(cfgFlag)
	// флаг -a, отвечающий за адрес запуска HTTP-сервера
	flag.StringVar(&cfgFlag.Server, "a", localAddr, "HTTP server address")
	// флаг -f, отвечающий за путь до файла с сокращёнными URL
	flag.StringVar(&cfgFlag.File, "f", filename, "File name")
	// флаг -b отвечающий за базовый адрес результирующего сокращённого URL
	flag.StringVar(&cfgFlag.BaseURL, "b", baseURL, "Base URL")
	flag.Parse()

	log.Println(cfgFlag.BaseURL)
	log.Println(cfgFlag.Server)
	log.Println(cfgFlag.File)
	log.Printf("Переменные конфигурации: %s", &cfg)
	/*
		if cfgFlag.Server == localAddr && cfg.Server != "" {
			cfgFlag.Server = cfg.Server
		}

		if cfgFlag.File == filename && cfg.File != "" {
			cfgFlag.File = cfg.File
		}

		if cfgFlag.BaseURL == baseURL && cfg.BaseURL != "" {
			cfgFlag.BaseURL = cfg.BaseURL
		} */

	if cfg.Server == "" {
		cfg.Server = cfgFlag.Server
	}

	if cfg.File == "" {
		cfg.File = cfgFlag.File
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = cfgFlag.BaseURL
	}

	return cfg, err
}
