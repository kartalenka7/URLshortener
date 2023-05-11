package memory

import (
	"encoding/json"
	"os"
)

type producer struct {
	file    *os.File // файл для записи
	encoder *json.Encoder
}

func NewProducer(filename string) (*producer, error) {
	// открываем файл для записи в конец
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0664)
	if err != nil {
		return nil, err
	}

	return &producer{
		file:    file,
		encoder: json.NewEncoder(file),
	}, nil
}

func (p *producer) Close() error {
	// закрываем файл
	return p.file.Close()
}

func (p *producer) WriteLinks(links *LinksData) error {
	return p.encoder.Encode(&links)
}
