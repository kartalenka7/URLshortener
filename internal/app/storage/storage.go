package storage

import (
	"errors"

	utils "example.com/shortener/cmd/utils"
)

// слой хранилища

type StorageLinks struct {
	linksMap map[string]string
}

func NewStorage() *StorageLinks {
	return &StorageLinks{
		linksMap: make(map[string]string),
	}
}

func (s StorageLinks) GetStorageLen() int {
	return len(s.linksMap)
}

func (s StorageLinks) AddLink(longURL string) (string, error) {
	var err error
	gToken := utils.RandStringBytes(10)
	_, ok := s.linksMap[gToken]
	if ok {
		return "", errors.New("link already exists")
	}

	s.linksMap[gToken] = longURL
	return gToken, err
}

func (s StorageLinks) GetLongURL(sToken string) (string, error) {
	var err error
	longURL, ok := s.linksMap[sToken]
	if !ok {
		return "", errors.New("link is not found")
	}
	return longURL, err
}
