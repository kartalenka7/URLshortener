package memory

import (
	"context"
	"errors"
	"fmt"
	"log"
	urlNet "net/url"

	"example.com/shortener/internal/app/models"
	"example.com/shortener/internal/config"
	"example.com/shortener/internal/config/utils"
)

// Структура для записи в файл
type LinksData struct {
	ShortURL string `json:"short"`
	LongURL  string `json:"long"`
	User     string `json:"user"`
}

type MemoryStorage struct {
	linksMap   map[string]string
	cookiesMap map[string]string
	config     config.Config
}

func New(config config.Config) *MemoryStorage {
	memStore := &MemoryStorage{
		linksMap:   make(map[string]string),
		cookiesMap: map[string]string{},
		config:     config,
	}
	if config.File != "" {
		memStore.ReadFromFile()
	}
	return memStore
}

func (s MemoryStorage) AddLink(ctx context.Context, longURL string, user string) (string, error) {
	var err error

	sToken := utils.GenRandToken(s.config.BaseURL)

	_, ok := s.linksMap[sToken]
	if ok {
		log.Println("link already exists")
		return "", errors.New("link already exists")
	}

	s.linksMap[sToken] = longURL
	s.cookiesMap[sToken] = user

	s.WriteInFile()
	return sToken, err
}

func (s MemoryStorage) GetLongURL(ctx context.Context, sToken string) (string, error) {
	var err error

	longToken := s.config.BaseURL + sToken
	_, urlParseErr := urlNet.Parse(longToken)
	if urlParseErr != nil {
		longToken = s.config.BaseURL + "/" + sToken
	}
	log.Printf("longToken %s", longToken)

	longURL, ok := s.linksMap[longToken]
	if !ok {
		return "", errors.New("link is not found")
	}
	return longURL, err
}

func (s MemoryStorage) GetStorageLen() int {
	return len(s.linksMap)
}

func (s MemoryStorage) Ping(ctx context.Context) error {
	return errors.New("база данных не активна")
}

func (s MemoryStorage) ShortenBatch(ctx context.Context, batchReq []models.BatchReq, cookie string) ([]models.BatchResp, error) {
	return nil, errors.New("база данных не активна")
}

func (s MemoryStorage) Close() error {
	return errors.New("база данных не активна")
}

func (s MemoryStorage) GetAllURLS(cookie string, ctx context.Context) map[string]string {
	userLinks := make(map[string]string)
	for short, user := range s.cookiesMap {
		if user != cookie {
			continue
		}
		userLinks[short] = s.linksMap[short]
	}
	return userLinks
}

func (s MemoryStorage) ReadFromFile() {

	//чтение из файла
	log.Println("Читаем из файла")
	log.Printf("Имя файла %s", s.config.File)
	consumer, err := NewConsumer(s.config.File)
	if err != nil {
		log.Fatal(err)
	}
	defer consumer.Close()

	for {
		readlinks, err := consumer.ReadLinks()
		if err != nil {
			fmt.Println(err.Error())
			break
		}
		s.linksMap[readlinks.ShortURL] = readlinks.LongURL
		s.cookiesMap[readlinks.ShortURL] = readlinks.User
	}

}

func (s MemoryStorage) WriteInFile() {
	if s.config.File == "" {
		return
	}
	producer, err := NewProducer(s.config.File)
	if err != nil {
		log.Fatal(err)
	}
	defer producer.Close()
	log.Println("Записываем в файл")
	log.Printf("Имя файла %s", s.config.File)

	for short, long := range s.linksMap {
		var links = LinksData{
			ShortURL: short,
			LongURL:  long,
			User:     s.cookiesMap[short],
		}
		log.Println(links)
		if err := producer.WriteLinks(&links); err != nil {
			log.Println(err.Error())
			log.Fatal(err)
		}
	}
}
