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
	producer, err := NewProducer("links.log")
	if err != nil {
		log.Fatal(err)
	}
	defer producer.Close()

	var links = LinksFile{
		ShortURL: gToken,
		LongURL:  longURL,
	}
	if err := producer.WriteLinks(&links); err != nil {
		log.Fatal(err)
	}

	return gToken, err
}

func (s StorageLinks) GetLongURL(sToken string, file string) (string, error) {
	var err error

	if file != "" {
		longURL, ok := s.linksMap[sToken]
		if !ok {
			return "", errors.New("link is not found")
		}
	}
	//чтение из файла
	consumer, err := NewConsumer(file)
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
		if readlinks.ShortURL == sToken {
			fmt.Printf("Нашли в файле, %s\n", readlinks.LongURL)
			return readlinks.LongURL, err
		}
		fmt.Println(readlinks)
	}

	return longURL, err
}
