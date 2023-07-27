// модуль memory реализует хранение данных в файле
package memory

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"example.com/shortener/internal/app/models"
	"example.com/shortener/internal/config"
)

// LinksData структура записи в файл
type LinksData struct {
	ShortURL string `json:"short"`
	LongURL  string `json:"long"`
	User     string `json:"user"`
	deleted  bool
}

var mutex sync.Mutex

type MemoryStorage struct {
	linksMap   map[string]string
	cookiesMap map[string]string
	deletedMap map[string]bool
	config     config.Config
	mu         *sync.Mutex
}

func New(config config.Config) *MemoryStorage {
	memStore := &MemoryStorage{
		linksMap:   make(map[string]string),
		cookiesMap: map[string]string{},
		deletedMap: make(map[string]bool),
		config:     config,
		mu:         &mutex,
	}
	if config.File != "" {
		memStore.ReadFromFile()
	}
	return memStore
}

func (s MemoryStorage) AddLink(ctx context.Context, sToken string, longURL string, user string) (string, error) {
	var err error
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.linksMap[sToken]
	if ok {
		log.Println("link already exists")
		return "", errors.New("link already exists")
	}

	s.linksMap[sToken] = longURL
	s.cookiesMap[sToken] = user

	log.Printf("Мапа со ссылками: %s\n", s.linksMap)

	s.WriteInFile()
	return sToken, err
}

func (s MemoryStorage) GetLongURL(ctx context.Context, sToken string) (string, error) {
	var err error

	longURL, ok := s.linksMap[sToken]
	if !ok {
		return "", errors.New("link is not found")
	}
	if deleted := s.deletedMap[sToken]; deleted {
		return "", models.ErrLinkDeleted
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

func (s MemoryStorage) GetAllURLS(ctx context.Context, cookie string) (map[string]string, error) {
	userLinks := make(map[string]string)
	for short, user := range s.cookiesMap {
		if user != cookie {
			continue
		}
		userLinks[short] = s.linksMap[short]
	}
	return userLinks, nil
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
		_, ok := s.linksMap[readlinks.ShortURL]
		if ok {
			continue
		}
		s.linksMap[readlinks.ShortURL] = readlinks.LongURL
		s.cookiesMap[readlinks.ShortURL] = readlinks.User
		s.deletedMap[readlinks.ShortURL] = readlinks.deleted
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
			deleted:  s.deletedMap[short],
		}
		if err := producer.WriteLinks(&links); err != nil {
			log.Println(err.Error())
			log.Fatal(err)
		}
	}
}

func (s MemoryStorage) BatchDelete(ctx context.Context, sTokens []models.TokenUser) {
	log.Println("Batch delete для in-memory")
	s.ReadFromFile()
	for _, v := range sTokens {
		user, ok := s.cookiesMap[v.Token]
		if user != v.User || !ok {
			continue
		}
		s.deletedMap[v.Token] = true
	}
	s.WriteInFile()
}
