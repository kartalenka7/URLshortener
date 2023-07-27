package storage

import (
	"context"
	"testing"
	"time"

	service "example.com/shortener/internal/app/service"
	memory "example.com/shortener/internal/app/storage/memory"
	"example.com/shortener/internal/config"
	"example.com/shortener/internal/config/utils"
	"example.com/shortener/internal/logger"
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
			//s := NewStorage(config.Config{File: tt.file})
			log := logger.InitLog()
			storer := memory.New(config.Config{File: tt.file})
			s := service.New(config.Config{File: tt.file}, storer, log)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			// Добавляем ссылку в хранилище
			sToken := utils.GenRandToken("http://localhost:8080/")
			gToken, err := s.AddLink(ctx, sToken, tt.longURL, "")
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
			got, err := s.GetLongURL(ctx, gToken)
			assert.Equal(t, got, tt.longURL)
			assert.NoError(t, err)
		})
	}
}
