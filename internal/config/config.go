package config

import (
	"fmt"
	"log"

	"flag"

	"github.com/caarlos0/env/v6"
)

// Config структура с флагами конфигурации
type Config struct {
	BaseURL  string `env:"BASE_URL" envDefault:"http://localhost:8080/"`
	Server   string `env:"SERVER_ADDRESS" envDefault:"localhost:8080"`
	File     string `env:"FILE_STORAGE_PATH"`
	Database string `env:"DATABASE_DSN"`
	HTTPS    string `env:"ENABLE_HTTPS"`
}

// Значения переменных конфигурации по умолчанию
var (
	localAddr = "localhost:8080"
	filename  = "link.log"
	baseURL   = "http://localhost:8080/"
	//database  = "postgres://habruser:habr@localhost:5432/habrdb"
	database  = "user=habruser password=habr host=localhost port=5432 dbname=habrdb sslmode=disable"
	BatchSize = 10
)

// GetConfig возвращает флаги конфигурации
func GetConfig() (Config, error) {
	var cfg Config
	var cfgFlag Config
	// Парсим переменные окружения
	log.Println("Parse")
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

	flag.StringVar(&cfgFlag.HTTPS, "s", "", "Enable HTTPS")
	flag.Parse()

	log.Printf("Флаги командной строки: %s\n", cfgFlag)

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

	if cfg.HTTPS == "" {
		cfg.HTTPS = cfgFlag.HTTPS
	}

	log.Printf("Переменные конфигурации: %s\n", &cfg)

	return cfg, err
}
