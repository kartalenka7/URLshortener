package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"flag"

	"github.com/caarlos0/env/v6"
)

// Config структура с флагами конфигурации
type Config struct {
	BaseURL    string `env:"BASE_URL" envDefault:"http://localhost:8080/" json:"base_url"`
	Server     string `env:"SERVER_ADDRESS" envDefault:"localhost:8080" json:"server_address"`
	File       string `env:"FILE_STORAGE_PATH" json:"file_storage_path"`
	Database   string `env:"DATABASE_DSN" json:"database_dsn"`
	HTTPS      bool   `env:"ENABLE_HTTPS" json:"enable_https"`
	ConfigFile string `env:"CONFIG"`
	Subnet     string `env:"TRUSTED_SUBNET" json:"trusted_subnet"`
}

// Значения переменных конфигурации по умолчанию
var (
	localAddr = "localhost:8080"
	filename  = "link.log"
	baseURL   = "http://localhost:8080/"
	//database  = "postgres://habruser:habr@localhost:5432/habrdb"
	database   = "user=habruser password=habr host=localhost port=5432 dbname=habrdb sslmode=disable"
	BatchSize  = 10
	configFile = "config.json"
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

	flag.BoolVar(&cfgFlag.HTTPS, "s", false, "Enable HTTPS")

	flag.StringVar(&cfgFlag.ConfigFile, "c", configFile, "Way to config file")
	flag.Parse()

	log.Printf("Флаги командной строки: %v\n", cfgFlag)

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

	if cfg.HTTPS == false {
		cfg.HTTPS = cfgFlag.HTTPS
	}

	if cfg.ConfigFile == "" {
		cfg.ConfigFile = cfgFlag.ConfigFile
	}

	ConfigFile, errReadFile := ReadConfigFile(cfg.ConfigFile)
	if errReadFile != nil {
		log.Println(errReadFile.Error())
	}

	if cfg.Server == "" {
		cfg.Server = ConfigFile.Server
	}

	if cfg.File == "" {
		cfg.File = ConfigFile.File
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = ConfigFile.BaseURL
	}

	if cfg.Database == "" {
		cfg.Database = ConfigFile.Database
	}

	if cfg.HTTPS == false {
		cfg.HTTPS = ConfigFile.HTTPS
	}

	if cfg.Subnet == "" {
		cfg.Subnet = ConfigFile.Subnet
	}

	log.Printf("Переменные конфигурации: %v\n", &cfg)

	return cfg, err
}

// ReadConfigFile читает конфигурационный файл в формате json
func ReadConfigFile(filename string) (*Config, error) {
	config := &Config{}

	file, err := os.OpenFile(filename, os.O_RDONLY, 0664)
	if err != nil {
		return config, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	if err := decoder.Decode(&config); err != nil {
		return config, err
	}

	return config, err
}
