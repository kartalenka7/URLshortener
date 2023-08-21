// модуль memory реализует хранение данных в файле
package memory

import (
	"context"
	"errors"
	"sync"

	"example.com/shortener/internal/app/models"
	"example.com/shortener/internal/config"
	"github.com/sirupsen/logrus"
)

// LinksData структура записи в файл
type LinksData struct {
	ShortURL string `json:"short"`
	LongURL  string `json:"long"`
	User     string `json:"user"`
	deleted  bool
}

var mutex sync.Mutex

// MemoryStorage реализует методы для взаимодействия с хранилищем из файла
type MemoryStorage struct {
	linksMap   map[string]string
	cookiesMap map[string]string
	deletedMap map[string]bool
	config     config.Config
	mu         *sync.Mutex
	log        *logrus.Logger
}

// New - конструктор для MemoryStorage
func New(config config.Config, log *logrus.Logger) *MemoryStorage {
	memStore := &MemoryStorage{
		linksMap:   make(map[string]string),
		cookiesMap: map[string]string{},
		deletedMap: make(map[string]bool),
		config:     config,
		mu:         &mutex,
		log:        log,
	}
	if config.File != "" {
		memStore.ReadFromFile()
	}
	return memStore
}

// AddLink записывает связку исходный URL- сокращенный токен в файл
func (s MemoryStorage) AddLink(ctx context.Context, sToken string, longURL string, user string) (string, error) {
	var err error
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.linksMap[sToken]
	if ok {
		s.log.Info("link already exists")
		return "", errors.New("link already exists")
	}

	s.linksMap[sToken] = longURL
	s.cookiesMap[sToken] = user

	s.log.WithFields(logrus.Fields{"Мапа со ссылками": s.linksMap})

	s.WriteInFile()
	return sToken, err
}

// GetLongURL возвращает исходный URL из файла
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

// метод заглушка
func (s MemoryStorage) GetStorageLen() int {
	return len(s.linksMap)
}

// метод заглушка
func (s MemoryStorage) Ping(ctx context.Context) error {
	return errors.New("база данных не активна")
}

// метод заглушка
func (s MemoryStorage) ShortenBatch(ctx context.Context, batchReq []models.BatchReq, cookie string) ([]models.BatchResp, error) {
	return nil, errors.New("база данных не активна")
}

// метод заглушка
func (s MemoryStorage) Close() error {
	return errors.New("база данных не активна")
}

// GetAllURLs возвращает все URL сокращенные пользователем из файла
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

// ReadFromFile читает данные из файла в мапу
func (s MemoryStorage) ReadFromFile() {
	s.log.Debug("Читаем из файла")
	s.log.WithFields(logrus.Fields{"Имя файла": s.config.File})
	s.mu.Lock()
	defer s.mu.Unlock()
	consumer, err := NewConsumer(s.config.File)
	if err != nil {
		s.log.Fatal(err.Error())
	}
	defer consumer.Close()

	for {
		readlinks, err := consumer.ReadLinks()
		if err != nil {
			s.log.Debug(err.Error())
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

// WriteInFile реализует запись в файл
func (s MemoryStorage) WriteInFile() {
	if s.config.File == "" {
		return
	}
	producer, err := NewProducer(s.config.File)
	if err != nil {
		s.log.Fatal(err.Error())
	}
	defer producer.Close()
	s.log.Info("Записываем в файл")
	s.log.WithFields(logrus.Fields{"Имя файла %s": s.config.File})

	for short, long := range s.linksMap {
		var links = LinksData{
			ShortURL: short,
			LongURL:  long,
			User:     s.cookiesMap[short],
			deleted:  s.deletedMap[short],
		}
		if err := producer.WriteLinks(&links); err != nil {
			s.log.Fatal(err.Error())
		}
	}
}

// BatchDelete ставит метки удаления на строки из мапы и записывает в файл
func (s MemoryStorage) BatchDelete(ctx context.Context, sTokens []models.TokenUser) {
	s.log.Info("Batch delete для in-memory")
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
