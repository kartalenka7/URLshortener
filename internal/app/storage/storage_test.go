package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStorage(t *testing.T) {
	type want struct {
		tokenLen int
	}
	tests := []struct {
		name    string
		longURL string
		want    want
	}{
		{
			name:    "Simple add and get",
			longURL: "https://www.youtube.com/",
			want: want{
				tokenLen: 10,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStorage()
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

			// Проверяем, что добавлена одна запись
			assert.Equal(t, s.GetStorageLen(), 1)

			// Получаем ссылку
			got, err := s.GetLongURL(gToken)
			assert.Equal(t, got, tt.longURL)
			assert.NoError(t, err)
		})
	}
}
