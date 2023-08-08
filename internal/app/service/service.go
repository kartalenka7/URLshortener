// Модуль service служит прослойкой между модулем с обработчиками handlers
// и модулем, реализующим хранение данных - storages
package service

import (
	"context"
	"strings"
	"sync"
	"time"

	urlNet "net/url"

	"example.com/shortener/internal/app/models"
	"example.com/shortener/internal/config"
	"example.com/shortener/internal/config/utils"
	"github.com/sirupsen/logrus"
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

// Service - реализует методы, которые подготавливают и обрабатывают
// данные из модуля handlers для последующей передачи в хранилище
type Service struct {
	Config  config.Config
	storage Storer
	Once    *sync.Once
	OutCh   chan string
	userCh  chan string
	log     *logrus.Logger
}

// New - конструктор для пакета service
func New(cfg config.Config, storage Storer, log *logrus.Logger) *Service {
	service := &Service{
		Config:  cfg,
		storage: storage,
		OutCh:   make(chan string, config.BatchSize),
		userCh:  make(chan string),
		log:     log,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	go service.RecieveTokensFromChannel(ctx)
	time.AfterFunc(60*time.Second, func() {
		log.Debug("Запускаем cancel")
		cancel()
	})

	return service
}

// AddDeletedTokens складывает токены, переданные пользователем для удаления в канал
// service.outCh
func (s Service) AddDeletedTokens(sTokens []string, user string) {
	replacer := strings.NewReplacer(`"`, ``, `[`, ``, `]`, ``)

	s.log.WithFields(logrus.Fields{"cookies": user}).Debug("Куки в AddDeletedTokens")
	s.userCh <- user
	for _, token := range sTokens {
		token = replacer.Replace(token)
		sToken := s.GetLongToken(token)
		s.log.WithFields(logrus.Fields{"sToken": sToken}).Debug("Добавляем значение в канал")
		s.OutCh <- sToken
	}

}

var deletedTokens = make([]models.TokenUser, 0, config.BatchSize*2)

// RecieveTokensFromChannel получает куки пользователя
// из канала service.userCh, токены для удаления из канала service.outCh
// и запускает удаление с помощью batch запроса
func (s Service) RecieveTokensFromChannel(ctx context.Context) {
	var user string
	s.log.Debug("Запустили канал")
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	// считываем значения из канала, пока он не будет закрыт
	for {
		select {
		case u := <-s.userCh:
			user = u
		case x := <-s.OutCh:
			s.log.WithFields(logrus.Fields{"cookie": user}).Debug("Куки в RecieveTokensFromChannel")
			deletedTokens = append(deletedTokens, models.TokenUser{
				Token: x,
				User:  user,
			})
			s.log.WithFields(logrus.Fields{"tokens count": len(deletedTokens)}).
				Debug("Приняли токенов из канала")
			if len(deletedTokens) >= config.BatchSize {
				s.log.WithFields(logrus.Fields{"deleted tokens": deletedTokens})
				s.storage.BatchDelete(ctx, deletedTokens)
				deletedTokens = deletedTokens[:0]
			}
		case <-ticker.C:
			s.storage.BatchDelete(ctx, deletedTokens)
			deletedTokens = deletedTokens[:0]

		case <-ctx.Done():
			s.log.Debug("Отменился контекст")
			return
		}
	}
}

// GetLongToken склеивает BaseURL с токеном
func (s Service) GetLongToken(sToken string) string {
	longToken := s.Config.BaseURL + sToken
	_, urlParseErr := urlNet.Parse(longToken)
	if urlParseErr != nil {
		longToken = s.Config.BaseURL + "/" + sToken
	}
	s.log.WithFields(logrus.Fields{"longToken": longToken})
	return longToken
}

// AddLink сохраняет сокращенный URL в хранилище
func (s Service) AddLink(ctx context.Context, sToken string, longURL string, user string) (string, error) {
	token := utils.GenRandToken(s.Config.BaseURL)
	return s.storage.AddLink(ctx, token, longURL, user)
}

// GetLongURL возвращает исходный URL из хранилища
func (s Service) GetLongURL(ctx context.Context, sToken string) (string, error) {
	return s.storage.GetLongURL(ctx, sToken)
}

// GetAllURLs возвращает все URL пользователя
func (s Service) GetAllURLS(ctx context.Context, cookie string) (map[string]string, error) {
	return s.storage.GetAllURLS(ctx, cookie)
}

// ShortenBatch обрабатывает URL, переданные в виде JSON объектов
func (s Service) ShortenBatch(ctx context.Context, batchReq []models.BatchReq, cookie string) ([]models.BatchResp, error) {
	return s.storage.ShortenBatch(ctx, batchReq, cookie)
}

// Ping проверяет соединение с БД
func (s Service) Ping(ctx context.Context) error {
	return s.storage.Ping(ctx)
}

// Проверяем, что в мапе с URL есть записи
func (s Service) GetStorageLen() int {
	return s.storage.GetStorageLen()
}

// Close - закрывает каналы
func (s Service) Close() error {
	close(s.OutCh)
	close(s.userCh)
	return s.storage.Close()
}
