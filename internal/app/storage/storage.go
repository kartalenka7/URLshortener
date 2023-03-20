package storage

import (
	"errors"
	"fmt"

	"log"

	urlNet "net/url"

	"database/sql"

	"example.com/shortener/internal/config"
	"example.com/shortener/internal/config/utils"
)

// слой хранилища

type StorageLinks struct {
	linksMap   map[string]string
	cookiesMap map[string]string
	config     config.Config
	db         *sql.DB
}

// Структура для записи в файл
type LinksData struct {
	ShortURL string `json:"short"`
	LongURL  string `json:"long"`
	User     string `json:"user"`
}

func NewStorage(cfg config.Config) *StorageLinks {
	links := &StorageLinks{
		linksMap:   make(map[string]string),
		cookiesMap: map[string]string{}}
	links.config = cfg
	if links.config.Database != "" {
		db, errDB := InitTable(links.config.Database)
		if errDB == nil {
			links.db = db
			linksDB, err := SelectLines(links.config.Database, 100)
			if err != nil {
				log.Printf("database|Select lines|%s\n", err.Error())
				return nil
			}

			for _, link := range linksDB {
				links.linksMap[link.ShortURL] = links.linksMap[link.LongURL]
				links.cookiesMap[link.ShortURL] = links.cookiesMap[link.User]
			}
			return links
		} else {
			log.Printf("Не учитываем таблицу бд")
		}
	}
	// открываем файл и читаем сохраненные ссылки
	if links.config.File != "" {
		ReadFromFile(links)
	}
	return links
}

func (s StorageLinks) GetStorageLen() int {
	return len(s.linksMap)
}

func (s StorageLinks) AddLink(longURL string, user string) (string, error) {
	var err error
	gToken := utils.RandStringBytes(10)
	log.Println(gToken)
	sToken := s.config.BaseURL + gToken
	_, urlParseErr := urlNet.Parse(sToken)
	if urlParseErr != nil {
		sToken = s.config.BaseURL + "/" + gToken
		log.Printf("Short URL %s", sToken)
	}

	if s.config.Database != "" {
		InsertLine(s.config.Database, sToken, longURL, user)
	}

	// in-memory
	_, ok := s.linksMap[sToken]
	if ok {
		return "", errors.New("link already exists")
	}
	s.linksMap[sToken] = longURL
	s.cookiesMap[sToken] = user
	return sToken, err
}

func (s StorageLinks) WriteInFile() {
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
		if err := producer.WriteLinks(&links); err != nil {
			log.Println(err.Error())
			log.Fatal(err)
		}
	}
}

func ReadFromFile(s *StorageLinks) {

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

func (s StorageLinks) GetAllURLS(cookie string) map[string]string {
	userLinks := make(map[string]string)
	for short, user := range s.cookiesMap {
		if user != cookie {
			continue
		}
		userLinks[short] = s.linksMap[short]
	}
	return userLinks
}

func (s StorageLinks) GetLongURL(sToken string) (string, error) {
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

func (s StorageLinks) GetConnSrtring() string {
	return s.config.Database
}
