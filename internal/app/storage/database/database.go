// модуль database реализует хранение данных в БД
package database

import (
	"context"
	"errors"
	"sync"

	"example.com/shortener/internal/app/models"
	"example.com/shortener/internal/app/service"
	"example.com/shortener/internal/config"
	"example.com/shortener/internal/config/utils"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sirupsen/logrus"
)

// dbStorage - структура, которая реализует все методы взаимодействия с бд
type dbStorage struct {
	config  config.Config
	pgxPool *pgxpool.Pool
	log     *logrus.Logger
}

// проверка на имплементацию интерфейса
var (
	_ service.Storer = (*dbStorage)(nil)
)

// запросы в бд
var (
	createSQL = `CREATE TABLE IF NOT EXISTS urlsDBTable(
					short_url TEXT,
					long_url TEXT UNIQUE,
					cookie TEXT, 
					deleted BOOLEAN
					);`
	insertSQL      = `INSERT INTO urlsDBTable(short_url, long_url, cookie, deleted) VALUES ($1, $2, $3, false)`
	selectShortURL = `SELECT short_url FROM urlsDBTable WHERE long_url = $1`
	selectByUser   = `SELECT short_url, long_url FROM urlsDBTable WHERE cookie = $1`
	selectLongURL  = `SELECT long_url, deleted FROM urlsDBTable WHERE short_url = $1`
	deleteSQL      = `UPDATE urlsDBTable SET deleted = 'true' WHERE short_url = $1 AND cookie = $2`
	tokensCount    = `SELECT COUNT(*) FROM urlsDBTable`
	differentUsers = `SELECT DISTINCT cookie FROM urlsDBTable`
	pgOnce         sync.Once
	storage        dbStorage
)

// New - конструктор для структуры dbStorage
func New(ctx context.Context, config config.Config, log *logrus.Logger) (*dbStorage, error) {
	var err error
	var pgxPool *pgxpool.Pool

	pgxPool, err = InitTable(ctx, config.Database, log)
	if err != nil {
		log.Debug("Не учитываем таблицу бд")
		return nil, err
	}
	storage = dbStorage{
		config:  config,
		pgxPool: pgxPool,
		log:     log,
	}

	return &storage, nil
}

// AddLink записывает исходный URL и сокращенный токен в бд
func (s *dbStorage) AddLink(
	ctx context.Context,
	sToken string,
	longURL string,
	user string) (string, error) {

	s.log.WithFields(logrus.Fields{"sToken": sToken,
		"longURL": longURL,
		"user":    user}).Info("Записываем в бд")

	// используем контекст запроса
	shortURL, err := s.InsertLine(ctx, sToken, longURL, user)
	if err != nil {
		s.log.Error(err.Error())
		sToken = shortURL
	}
	return sToken, err
}

// GetLongURL выбирает из бд исходный URL
func (s *dbStorage) GetLongURL(ctx context.Context, sToken string) (string, error) {

	longURL, err := s.SelectLink(ctx, sToken)
	if err != nil {
		return "", err
	}
	return longURL, nil
}

// метод заглушка
func (s dbStorage) GetStorageLen() int {
	return 0
}

// Ping возвращает ошибку, если не удалось установить связь с бд
func (s *dbStorage) Ping(ctx context.Context) error {
	return s.pgxPool.Ping(ctx)
}

// GetAllURLs выбирает все сокращенные токены и исходные URL конкретного пользователя
func (s *dbStorage) GetAllURLS(ctx context.Context, cookie string) (map[string]string, error) {
	var link models.LinksData
	userLinks := make(map[string]string)

	rows, err := s.pgxPool.Query(ctx, selectByUser, cookie)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&link.ShortURL, &link.LongURL)
		if err != nil {
			return nil, err
		}
		userLinks[link.ShortURL] = link.LongURL
	}
	if rows.Err() != nil {
		return nil, err
	}
	return userLinks, nil
}

// InitTable инициализирует пул соединений pgxpool и создает таблицы, если их еше нет в бд
func InitTable(ctx context.Context, connString string, log *logrus.Logger) (*pgxpool.Pool, error) {
	log.Debug("Инициализация таблицы")

	// открываем соединение с бд
	pgxPool, err := pgxpool.Connect(ctx, connString)
	if err != nil {
		log.Debug(err.Error())
		return nil, err
	}

	if _, err = pgxPool.Exec(ctx, createSQL); err != nil {
		log.Debug(err.Error())
		return nil, err
	}

	return pgxPool, nil
}

// InsertLine добавляет строку с сокращенным токеном в бд,
// если для исходного URL еще не была добавлена строка
func (s *dbStorage) InsertLine(
	ctx context.Context,
	shortURL string,
	longURL string,
	cookie string) (string, error) {

	var pgxError *pgconn.PgError
	pgxConn, err := s.pgxPool.Acquire(ctx)
	if err != nil {
		return "", err
	}
	defer pgxConn.Release()

	res, err := pgxConn.Exec(ctx, insertSQL, shortURL, longURL, cookie)
	if err == nil {
		rows := res.RowsAffected()
		if rows > 0 {
			s.log.WithFields(logrus.Fields{"rows": rows}).Info("Вставлено строк")
		}
		return "", nil
	}

	s.log.Error(err.Error())
	if !errors.As(err, &pgxError) {
		return "", err
	}
	resSelect, errSelect := pgxConn.Query(ctx, selectShortURL, longURL)
	if errSelect != nil {
		return "", errSelect
	}
	defer resSelect.Close()

	var link models.LinksData
	for resSelect.Next() {
		errSelect := resSelect.Scan(&link.ShortURL)
		if errSelect != nil {
			return "", errSelect
		}
		s.log.WithFields(logrus.Fields{"shortURL": link.ShortURL}).Info("Найден короткий URL")

		s.log.WithFields(logrus.Fields{"pgx error code": pgxError.Code})
		if pgxError.Code == pgerrcode.UniqueViolation {
			return link.ShortURL, models.ErrorAlreadyExist
		} else {
			return link.ShortURL, pgxError
		}

	}

	if resSelect.Err() != nil {
		return "", resSelect.Err()
	}

	return "", err
}

// BatchDelete удаляет строки из бд с помощью Batch запроса
func (s dbStorage) BatchDelete(ctx context.Context, sTokens []models.TokenUser) {

	if len(sTokens) == 0 {
		return
	}

	batch := &pgx.Batch{}
	for _, v := range sTokens {
		batch.Queue(deleteSQL, v.Token, v.User)
	}
	br := s.pgxPool.SendBatch(ctx, batch)
	defer br.Close()
	comTag, err := br.Exec()
	if err != nil {
		s.log.Error(err.Error())
	}

	s.log.WithFields(logrus.Fields{"changed rows": comTag.RowsAffected()}).Info("После удаления Изменено строк")
}

// ShortenBatch записывает новые токены в бд с помощью Batch запроса
func (s dbStorage) ShortenBatch(ctx context.Context, batchReq []models.BatchReq, cookie string) ([]models.BatchResp, error) {
	response := make([]models.BatchResp, 0, 100)

	batch := &pgx.Batch{}

	for _, batchValue := range batchReq {

		// проверяем, что в базе еще нет такого url
		_, err := s.findErrorURL(ctx, batchValue.URL)
		if err != nil {
			s.log.Error(err.Error())
			return nil, err
		}

		sToken := utils.GenRandToken(s.config.BaseURL)
		s.log.WithFields(logrus.Fields{"sToken": sToken,
			"URL": batchValue.URL}).Info("Записываем в бд")
		batch.Queue(insertSQL, sToken, batchValue.URL, cookie)

		// формируем структуру для ответа
		response = append(response, models.BatchResp{
			CorrID:   batchValue.CorrID,
			ShortURL: sToken,
		})
	}

	br := s.pgxPool.SendBatch(ctx, batch)
	defer br.Close()
	_, err := br.Exec()
	if err != nil {
		s.log.Error(err.Error())
	}

	s.log.WithFields(logrus.Fields{"response": response}).Info("Структура ответа")

	return response, nil
}

// findErrorURL проверяет, существует ли для URL сокращенный токен в бд
func (s *dbStorage) findErrorURL(ctx context.Context, URL string) (string, error) {
	var link models.LinksData
	var sToken string

	rows, err := s.pgxPool.Query(ctx, selectShortURL, URL)
	if err != nil {
		return "", err
	}
	for rows.Next() {
		err := rows.Scan(&link.ShortURL)
		if err != nil {
			return "", err
		}
		s.log.WithFields(logrus.Fields{"Short URL": link.ShortURL}).Info("Найденный в бд короткий URL")
		sToken = link.ShortURL
		if sToken != "" {
			break
		}
	}
	if rows.Err() != nil {
		return "", rows.Err()
	}
	return sToken, nil
}

// SelectLink выбирает исходный URL из бд
func (s *dbStorage) SelectLink(ctx context.Context, shortURL string) (string, error) {
	s.log.Info("Ищем длинный URL в бд")
	var longURL string
	var deleted bool

	err := s.pgxPool.QueryRow(ctx, selectLongURL, shortURL).Scan(&longURL, &deleted)
	if err != nil {
		s.log.Error(err.Error())
		return "", models.ErrLinkNotFound
	}
	if deleted {
		return "", models.ErrLinkDeleted
	}
	return longURL, nil
}

// Close закрывает пул соединений с бд
func (s *dbStorage) Close() error {
	s.pgxPool.Close()
	return nil
}

// GetStats возвращает данные по общему числу
// пользователей и сокращенных URL из бд
func (s *dbStorage) GetStats(ctx context.Context) models.Stats {
	var stats models.Stats
	row := s.pgxPool.QueryRow(ctx, tokensCount)

	if err := row.Scan(&stats.URLs); err != nil {
		s.log.Error(err.Error())
		return models.Stats{}
	}

	rowsUsers, err := s.pgxPool.Query(ctx, differentUsers)
	if err != nil {
		s.log.Error(err.Error())
		return models.Stats{}
	}

	var users int
	for rowsUsers.Next() {
		users += 1
	}
	stats.Users = users
	return stats
}
