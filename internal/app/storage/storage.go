package storage

import (
	"errors"
	"fmt"

	"log"

	utils "example.com/shortener/cmd/utils"
)

// слой хранилища

type StorageLinks struct {
	linksMap map[string]string
}

type LinksFile struct {
	ShortURL string `json:"short"`
	LongURL  string `json:"long"`
}

func NewStorage() *StorageLinks {
	return &StorageLinks{
		linksMap: make(map[string]string),
	}
}

func (s StorageLinks) GetStorageLen() int {
	return len(s.linksMap)
}

func (s StorageLinks) AddLink(longURL string, filename string) (string, error) {
	var err error
	gToken := utils.RandStringBytes(10)

	if filename == "" {
		_, ok := s.linksMap[gToken]
		if ok {
			return "", errors.New("link already exists")
		}
		s.linksMap[gToken] = longURL
		return gToken, err
	}

	// запись в файл
	producer, err := NewProducer(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer producer.Close()

	var links = LinksFile{
		ShortURL: gToken,
		LongURL:  longURL,
	}
	log.Printf("Записываем в файл %s", links)
	log.Printf("Имя файла %s", filename)
	if err := producer.WriteLinks(&links); err != nil {
		log.Println(err.Error())
		log.Fatal(err)
	}
	return gToken, err
}

func (s StorageLinks) GetLongURL(sToken string, filename string) (string, error) {
	var err error

	if filename == "" {
		longURL, ok := s.linksMap[sToken]
		if !ok {
			return "", errors.New("link is not found")
		}
		return longURL, err
	}
	//чтение из файла
	log.Println("Читаем из файла")
	log.Printf("Имя файла %s", filename)
	consumer, err := NewConsumer(filename)
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
