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

type dbStorage struct {
	config  config.Config
	pgxPool *pgxpool.Pool
	log     *logrus.Logger
}

// проверка на имплементацию интерфейса
var (
	_ service.Storer = (*dbStorage)(nil)
)

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
	pgOnce         sync.Once
	storage        dbStorage
)

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

func (s *dbStorage) AddLink(
	ctx context.Context,
	sToken string,
	longURL string,
	user string) (string, error) {

	s.log.WithFields(logrus.Fields{"sToken": sToken,
		"longURL": longURL,
		"user":    user}).Debug("Записываем в бд")

	// используем контекст запроса
	shortURL, err := s.InsertLine(ctx, sToken, longURL, user)
	if err != nil {
		s.log.Error(err.Error())
		sToken = shortURL
	}
	return sToken, err
}

func (s *dbStorage) GetLongURL(ctx context.Context, sToken string) (string, error) {

	longURL, err := s.SelectLink(ctx, sToken)
	if err != nil {
		return "", err
	}
	return longURL, nil
}

func (s dbStorage) GetStorageLen() int {
	return 0
}

func (s *dbStorage) Ping(ctx context.Context) error {
	return s.pgxPool.Ping(ctx)
}

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

func InitTable(ctx context.Context, connString string, log *logrus.Logger) (*pgxpool.Pool, error) {
	log.Debug("Инициализация таблицы")

	// открываем соединение с бд
	pgxPool, err := pgxpool.Connect(ctx, connString)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	if _, err = pgxPool.Exec(ctx, createSQL); err != nil {
		log.Error(err.Error())
		return nil, err
	}

	return pgxPool, nil
}

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
			s.log.WithFields(logrus.Fields{"rows": rows}).Debug("Вставлено строк")
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
		s.log.WithFields(logrus.Fields{"shortURL": link.ShortURL}).Debug("Найден короткий URL")

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

	s.log.WithFields(logrus.Fields{"changed rows": comTag.RowsAffected()}).Debug("После удаления Изменено строк")
}

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
			"URL": batchValue.URL}).Debug("Записываем в бд")
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

	s.log.WithFields(logrus.Fields{"response": response}).Debug("Структура ответа")

	return response, nil
}

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
		s.log.WithFields(logrus.Fields{"Short URL": link.ShortURL}).Debug("Найденный в бд короткий URL")
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

func (s *dbStorage) SelectLink(ctx context.Context, shortURL string) (string, error) {
	s.log.Debug("Ищем длинный URL в бд")
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

func (s *dbStorage) Close() error {
	s.pgxPool.Close()
	return nil
}
