package memory

import (
	"encoding/json"
	"os"
)

type consumer struct {
	file    *os.File
	decoder *json.Decoder
}

func NewConsumer(filename string) (*consumer, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0664)
	if err != nil {
		return nil, err
	}

	return &consumer{
		file:    file,
		decoder: json.NewDecoder(file),
	}, nil
}

func (c *consumer) ReadLinks() (*LinksData, error) {
	links := &LinksData{}
	if err := c.decoder.Decode(&links); err != nil {
		return nil, err
	}
	return links, nil
}
func (c *consumer) Close() error {
	return c.file.Close()
}
