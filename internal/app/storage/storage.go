package storage

import (
	"errors"
	"fmt"

	"log"

	urlNet "net/url"

	"example.com/shortener/internal/config/utils"
)

// слой хранилища

type StorageLinks struct {
	linksMap map[string]string
}

type LinksFile struct {
	ShortURL string `json:"short"`
	LongURL  string `json:"long"`
}

var config utils.Config

func NewStorage() *StorageLinks {
	return &StorageLinks{
		linksMap: make(map[string]string),
	}
}

func (s StorageLinks) GetStorageLen() int {
	return len(s.linksMap)
}

func (s StorageLinks) SetConfig(cfg utils.Config) {
	config = cfg
}

func (s StorageLinks) AddLink(longURL string) (string, error) {
	var err error
	gToken := utils.RandStringBytes(10)
	log.Println(gToken)
	sToken := config.BaseURL + gToken
	_, urlParseErr := urlNet.Parse(sToken)
	if urlParseErr != nil {
		sToken = config.BaseURL + "/" + gToken
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
	if config.File == "" {
		return
	}
	producer, err := NewProducer(config.File)
	if err != nil {
		log.Fatal(err)
	}
	defer producer.Close()
	log.Println("Записываем в файл")
	log.Printf("Имя файла %s", config.File)

	for short, long := range s.linksMap {
		var links = LinksFile{
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

func (s StorageLinks) ReadFromFile() {
	if config.File == "" {
		return
	}
	//чтение из файла
	log.Println("Читаем из файла")
	log.Printf("Имя файла %s", config.File)
	consumer, err := NewConsumer(config.File)
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

	longToken := config.BaseURL + sToken
	_, urlParseErr := urlNet.Parse(longToken)
	if urlParseErr != nil {
		longToken = config.BaseURL + "/" + sToken
	}
	log.Printf("longToken %s", longToken)

	longURL, ok := s.linksMap[longToken]
	if !ok {
		return "", errors.New("link is not found")
	}
	return longURL, err
}
