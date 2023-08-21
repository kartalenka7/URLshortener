package memory

import (
	"encoding/json"
	"os"
)

type producer struct {
	file    *os.File
	encoder *json.Encoder
}

// NewProducer открывает файл для записи
func NewProducer(filename string) (*producer, error) {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0664)
	if err != nil {
		return nil, err
	}

	return &producer{
		file:    file,
		encoder: json.NewEncoder(file),
	}, nil
}

// Close закрывает файл
func (p *producer) Close() error {
	return p.file.Close()
}

// WriteLinks декодирует данные структуры LinksData в json объект и записывает в файл
func (p *producer) WriteLinks(links *LinksData) error {
	return p.encoder.Encode(&links)
}
