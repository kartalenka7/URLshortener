package service

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	urlNet "net/url"

	"example.com/shortener/internal/app/models"
	"example.com/shortener/internal/config"
	"example.com/shortener/internal/config/utils"
)

// Storer - интерфейс взаимодействия с хранилищем
type Storer interface {
	AddLink(ctx context.Context, sToken string, longURL string, user string) (string, error)
	GetLongURL(ctx context.Context, sToken string) (string, error)
	Ping(ctx context.Context) error
	GetAllURLS(ctx context.Context, cookie string) (map[string]string, error)
	ShortenBatch(ctx context.Context, batchReq []models.BatchReq, cookie string) ([]models.BatchResp, error)
	BatchDelete(ctx context.Context, sTokens []models.TokenUser)
	Close() error
	GetStorageLen() int
}

var once sync.Once

type Service struct {
	Config  config.Config
	storage Storer
	Once    *sync.Once
	OutCh   chan string
	user    string
}

func New(cfg config.Config, storage Storer) *Service {
	service := &Service{
		Config:  cfg,
		storage: storage,
		Once:    &once,
		OutCh:   make(chan string, config.BatchSize),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	go service.RecieveTokensFromChannel(ctx)
	time.AfterFunc(60*time.Second, func() {
		log.Println("Запускаем cancel")
		cancel()
	})

	return service
}

func (s Service) AddDeletedTokens(sTokens []string, user string) {
	s.user = user
	replacer := strings.NewReplacer(`"`, ``, `[`, ``, `]`, ``)
	for _, token := range sTokens {
		token = replacer.Replace(token)
		sToken := s.GetLongToken(token)
		log.Printf("Добавляем значение в канал %s", sToken)
		s.OutCh <- sToken
	}

}

var deletedTokens = make([]models.TokenUser, 0, config.BatchSize*2)

func (s Service) RecieveTokensFromChannel(ctx context.Context) {
	log.Println("Запустили канал")
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	// считываем значения из канала, пока он не будет закрыт
	for {
		select {
		case x := <-s.OutCh:
			deletedTokens = append(deletedTokens, models.TokenUser{
				Token: x,
				User:  s.user,
			})
			log.Printf("Приняли токенов из канала: %d\n", len(deletedTokens))
			if len(deletedTokens) >= config.BatchSize {
				log.Println(deletedTokens)
				s.storage.BatchDelete(ctx, deletedTokens)
				deletedTokens = deletedTokens[:0]
			}
		case <-ticker.C:
			log.Println("Запуск по таймеру")
			s.storage.BatchDelete(ctx, deletedTokens)
			deletedTokens = deletedTokens[:0]

		case <-ctx.Done():
			log.Println("Отменился контекст")
			return
		}
	}
}

func (s Service) GetLongToken(sToken string) string {
	longToken := s.Config.BaseURL + sToken
	_, urlParseErr := urlNet.Parse(longToken)
	if urlParseErr != nil {
		longToken = s.Config.BaseURL + "/" + sToken
	}
	log.Printf("longToken %s", longToken)
	return longToken
}

func (s Service) AddLink(ctx context.Context, sToken string, longURL string, user string) (string, error) {
	token := utils.GenRandToken(s.Config.BaseURL)
	return s.storage.AddLink(ctx, token, longURL, user)
}

func (s Service) GetLongURL(ctx context.Context, sToken string) (string, error) {
	return s.storage.GetLongURL(ctx, sToken)
}

func (s Service) GetAllURLS(ctx context.Context, cookie string) (map[string]string, error) {
	return s.storage.GetAllURLS(ctx, cookie)
}

func (s Service) ShortenBatch(ctx context.Context, batchReq []models.BatchReq, cookie string) ([]models.BatchResp, error) {
	return s.storage.ShortenBatch(ctx, batchReq, cookie)
}

func (s Service) Ping(ctx context.Context) error {
	return s.storage.Ping(ctx)
}

func (s Service) GetStorageLen() int {
	return s.storage.GetStorageLen()
}

func (s Service) Close() error {
	close(s.OutCh)
	return s.storage.Close()
}
