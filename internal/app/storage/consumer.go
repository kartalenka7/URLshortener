package storage

import (
	"encoding/json"
	"os"
)

type consumer struct {
	file    *os.File
	decoder *json.Decoder
}

func NewConsumer(filename string) (*consumer, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 07664)
	if err != nil {
		return nil, err
	}

	return &consumer{
		file:    file,
		decoder: json.NewDecoder(file),
	}, nil
}

func (c *consumer) ReadLinks() (*LinksFile, error) {
	links := &LinksFile{}
	if err := c.decoder.Decode(&links); err != nil {
		return nil, err
	}
	return links, nil
}
func (c *consumer) Close() error {
	return c.file.Close()
}
