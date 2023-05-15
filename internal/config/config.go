package config

import (
	"fmt"
	"log"

	"flag"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	BaseURL  string `env:"BASE_URL" envDefault:"http://localhost:8080/"`
	Server   string `env:"SERVER_ADDRESS" envDefault:"localhost:8080"`
	File     string `env:"FILE_STORAGE_PATH"`
	Database string `env:"DATABASE_DSN"`
}

var (
	localAddr = "localhost:8080"
	filename  = "link.log"
	baseURL   = "http://localhost:8080/"
	//database  = "postgres://habruser:habr@localhost:5432/habrdb"
	database = "user=habruser password=habr host=localhost port=5432 database=habrdb sslmode=disable"
)

func GetConfig() (Config, error) {
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

	flag.StringVar(&cfgFlag.Database, "d", database, "Database connections")
	flag.Parse()

	log.Printf("Флаги командной строки: %s\n", cfgFlag)
	log.Printf("Переменные конфигурации: %s\n", &cfg)

	if cfg.Server == "" || cfg.Server == localAddr {
		cfg.Server = cfgFlag.Server
	}

	if cfg.File == "" || cfg.File == filename {
		cfg.File = cfgFlag.File
	}

	if cfg.BaseURL == "" || cfg.BaseURL == baseURL {
		cfg.BaseURL = cfgFlag.BaseURL
	}

	if cfg.Database == "" || cfg.Database == database {
		cfg.Database = cfgFlag.Database
	}
	return cfg, err
}
