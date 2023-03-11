package storage

import (
	"testing"

	"example.com/shortener/internal/config/utils"
	"github.com/stretchr/testify/assert"
)

func TestStorage(t *testing.T) {
	type want struct {
		tokenLen int
	}
	tests := []struct {
		name    string
		longURL string
		file    string
		want    want
	}{
		{
			name:    "Simple add and get",
			longURL: "https://www.youtube.com/",
			file:    "",
			want: want{
				tokenLen: 10,
			},
		},
		{
			name:    "Add and get from file links.log",
			longURL: "https://www.youtube.com/",
			file:    "links.log",
			want: want{
				tokenLen: 10,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStorage()
			s.SetConfig(utils.Config{File: tt.file})
			// Добавляем ссылку в хранилище
			gToken, err := s.AddLink(tt.longURL)
			if err != nil {
				t.Errorf("StorageLinks.GetLongURL() error = %v", err)
				return
			}
			// Проверяем, что сгенерированный токен не пустой и длина 10
			assert.NotEmpty(t, gToken)
			tokenLen := len(gToken)
			assert.Equal(t, tokenLen, 10)

			// Проверяем, что добавлена одна запись (для варианта с сохранением в память)
			if tt.file == "" {
				assert.Equal(t, s.GetStorageLen(), 1)
			}

			// Получаем ссылку
			got, err := s.GetLongURL(gToken)
			assert.Equal(t, got, tt.longURL)
			assert.NoError(t, err)
		})
	}
}
