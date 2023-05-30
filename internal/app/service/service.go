package service

import (
	"context"
	"log"
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

type Service struct {
	Config  config.Config
	storage Storer
}

func New(config config.Config, storage Storer) *Service {
	return &Service{
		Config:  config,
		storage: storage,
	}
}

func (s Service) AddDeletedTokens(sTokens []string, inputCh chan string) {
	for _, token := range sTokens {
		inputCh <- token
	}
	close(inputCh) // закрываем канал
}

func (s Service) RecieveTokensFromChannel(ctx context.Context, inputCh chan string, user string) {
	var deletedTokens []models.TokenUser
	log.Println("Считываем значения из канала")
	deletedTokens = make([]models.TokenUser, config.BatchSize)
	timer := time.NewTimer(10 * time.Second)
	// считываем значения из канала, пока он не будет закрыт
	for i := range inputCh {
		deletedTokens = append(deletedTokens, models.TokenUser{
			Token: i,
			User:  user,
		})
		select {
		case <-timer.C:
			s.storage.BatchDelete(ctx, deletedTokens)
			deletedTokens = deletedTokens[:0]
		default:
			if len(deletedTokens) == config.BatchSize {
				log.Println(deletedTokens)
				s.storage.BatchDelete(ctx, deletedTokens)
				deletedTokens = deletedTokens[:0]
			}
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
	return s.storage.Close()
}
