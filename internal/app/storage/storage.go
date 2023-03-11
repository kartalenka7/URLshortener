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

	sToken := config.BaseURL + gToken
	_, urlParseErr := urlNet.Parse(sToken)
	if urlParseErr != nil {
		sToken = config.BaseURL + "/" + gToken
		//fmt.Fprint(rw, sToken)
		log.Printf("Short URL %s", sToken)
	}

	if config.File == "" {
		_, ok := s.linksMap[gToken]
		if ok {
			return "", errors.New("link already exists")
		}
		s.linksMap[gToken] = longURL
		return sToken, err
	}

	// запись в файл
	producer, err := NewProducer(config.File)
	if err != nil {
		log.Fatal(err)
	}
	defer producer.Close()

	var links = LinksFile{
		ShortURL: gToken,
		LongURL:  longURL,
	}
	log.Printf("Записываем в файл %s", links)
	log.Printf("Имя файла %s", config.File)
	if err := producer.WriteLinks(&links); err != nil {
		log.Println(err.Error())
		log.Fatal(err)
	}
	return sToken, err
}

func (s StorageLinks) GetLongURL(sToken string) (string, error) {
	var err error

	if config.File == "" {
		longURL, ok := s.linksMap[sToken]
		if !ok {
			return "", errors.New("link is not found")
		}
		return longURL, err
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
		if readlinks.ShortURL == sToken {
			fmt.Printf("Нашли в файле, %s\n", readlinks.LongURL)
			return readlinks.LongURL, err
		}
	}

	return "", err
}
