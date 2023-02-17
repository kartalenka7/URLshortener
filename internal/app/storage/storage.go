package storage

import "errors"

// слой хранилища

type StorageLinks struct {
	LinksMap map[string]string
}

func GetStorage(linksMap map[string]string) *StorageLinks {
	return &StorageLinks{
		LinksMap: linksMap,
	}
}

func (s StorageLinks) AddLink(gToken string, longURL string) error {
	var err error
	_, ok := s.LinksMap[gToken]
	if ok {
		err = errors.New("link already exists")
	} else {
		s.LinksMap[gToken] = longURL
	}
	return err
}

func (s StorageLinks) GetLongURL(sToken string) (string, error) {
	var err error
	longURL, ok := s.LinksMap[sToken]
	if !ok {
		err = errors.New("link is not found")
	}
	return longURL, err
}
