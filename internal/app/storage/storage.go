package storage

import (
	"errors"
	"fmt"

	"log"

	urlNet "net/url"

	//"database/sql"

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
	var err error
	links := &StorageLinks{
		linksMap:   make(map[string]string),
		cookiesMap: map[string]string{}}
	links.config = cfg
	if links.config.Database != "" {
		links.db, err = InitTable(links.config.Database)
		if err != nil {
			log.Println("Не учитываем таблицу бд")
			links.config.Database = ""
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

func (s StorageLinks) Close() {
	if s.config.Database != "" {
		s.db.Close()
	}
}

func (s StorageLinks) AddLink(longURL string, user string) (string, error) {
	var err error
	var shortURL string
	gToken := utils.RandStringBytes(10)
	log.Println(gToken)
	sToken := s.config.BaseURL + gToken
	_, urlParseErr := urlNet.Parse(sToken)
	if urlParseErr != nil {
		sToken = s.config.BaseURL + "/" + gToken
		log.Printf("Short URL %s", sToken)
	}

	_, ok := s.linksMap[sToken]
	if ok {
		log.Println("link already exists")
		return "", errors.New("link already exists")
	}

	log.Printf("Database conn %s\n", s.config.Database)
	if s.config.Database != "" {
		log.Printf("Записываем в бд %s %s\n", sToken, longURL)
		shortURL, err = InsertLine(s.db, sToken, longURL, user)
		if err != nil {
			log.Println(err.Error())
			//return shortURL, err
			sToken = shortURL
		}
	}
	s.linksMap[sToken] = longURL
	//log.Printf("мапа со ссылками %s\n", s.linksMap)
	s.cookiesMap[sToken] = user
	//log.Printf("Мапа с куки %s\n", s.cookiesMap)

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
		log.Println(links)
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
		longURL, err = SelectLink(s.db, longToken)
		if err != nil {
			log.Printf("storage|getLongURL|%s\n", err.Error())
			return "", errors.New("link is not found")
		}
	}
	return longURL, err
}

func (s StorageLinks) GetConnSrtring() string {
	return s.config.Database
}

type BatchReq struct {
	CorrID string `json:"correlation_id"`
	URL    string `json:"original_url"`
}

type BatchResp struct {
	CorrID   string `json:"correlation_id"`
	ShortURL string `json:"short_url"`
}

func (s StorageLinks) ShortenBatchTr(batchReq []BatchReq, cookie string) ([]BatchResp, error) {
	return ShortenBatch(batchReq, s.db, s.config.BaseURL, cookie)
}
