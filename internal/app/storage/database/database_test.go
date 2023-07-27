package database

import (
	"context"
	"testing"

	"example.com/shortener/internal/config"
	"example.com/shortener/internal/config/utils"
	"example.com/shortener/internal/logger"
)

func BenchmarkStorage(b *testing.B) {
	var err error
	var storage *dbStorage

	log := logger.InitLog()
	cfg, err := config.GetConfig()

	if err != nil {
		return
	}
	b.ResetTimer()
	b.Run("Create storage", func(b *testing.B) {
		storage, err = New(context.Background(), cfg, log)
	})

	if err != nil {
		return
	}

	b.Run("Add link", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			storage.AddLink(context.Background(), utils.RandStringBytes(10),
				utils.RandStringBytes(30), utils.RandStringBytes(20))
		}
	})

	b.Run("Get long URL", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			storage.GetLongURL(context.Background(), utils.RandStringBytes(10))
		}
	})

	b.Run("Find URL that already exists", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			storage.findErrorURL(context.Background(), utils.RandStringBytes(10))
		}
	})

}
