package storage

import (
	"errors"
	"fmt"

	"log"

	urlNet "net/url"

	"example.com/shortener/internal/config"
	"example.com/shortener/internal/config/utils"
)

// слой хранилища

type StorageLinks struct {
	linksMap map[string]string
	config   config.Config
}

// Структура для записи в файл
type LinksData struct {
	ShortURL string `json:"short"`
	LongURL  string `json:"long"`
}

func NewStorage(cfg config.Config) *StorageLinks {
	links := &StorageLinks{linksMap: make(map[string]string)}
	links.config = cfg
	// открываем файл и читаем сохраненные ссылки
	if links.config.File != "" {
		ReadFromFile(links)
	}
	return links
}

func (s StorageLinks) GetStorageLen() int {
	return len(s.linksMap)
}

func (s StorageLinks) AddLink(longURL string) (string, error) {
	var err error
	gToken := utils.RandStringBytes(10)
	log.Println(gToken)
	sToken := s.config.BaseURL + gToken
	_, urlParseErr := urlNet.Parse(sToken)
	if urlParseErr != nil {
		sToken = s.config.BaseURL + "/" + gToken
		log.Printf("Short URL %s", sToken)
	}

	// in-memory
	_, ok := s.linksMap[sToken]
	if ok {
		return "", errors.New("link already exists")
	}
	s.linksMap[sToken] = longURL
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
		}
		if err := producer.WriteLinks(&links); err != nil {
			log.Println(err.Error())
			log.Fatal(err)
		}
		log.Printf("Записываем в файл %s", links)
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
		log.Println(readlinks)
		s.linksMap[readlinks.ShortURL] = readlinks.LongURL
	}

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
