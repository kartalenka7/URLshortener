package database

import (
	"context"
	"errors"
	"log"
	"sync"

	"example.com/shortener/internal/app/models"
	"example.com/shortener/internal/config"
	"example.com/shortener/internal/config/utils"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type dbStorage struct {
	config  config.Config
	pgxConn *pgx.Conn
	mu      *sync.Mutex
}

var (
	createSQL = `CREATE TABLE IF NOT EXISTS urlsTable(
					short_url TEXT,
					long_url TEXT UNIQUE,
					cookie TEXT, 
					deleted BOOLEAN
					);`
	insertSQL      = `INSERT INTO urlsTable(short_url, long_url, cookie, deleted) VALUES ($1, $2, $3, false)`
	selectShortURL = `SELECT short_url FROM urlsTable WHERE long_url = $1`
	selectByUser   = `SELECT short_url, long_url FROM urlsTable WHERE cookie = $1`
	selectLongURL  = `SELECT long_url, deleted FROM urlsTable WHERE short_url = $1`
	deleteSQL      = `UPDATE urlsTable SET deleted = 'true' WHERE short_url = $1, cookie = $2`
)

func New(ctx context.Context, config config.Config) (*dbStorage, error) {
	pgxConn, err := InitTable(ctx, config.Database)
	if err != nil {
		log.Println("Не учитываем таблицу бд")
		return nil, err
	}
	storage := dbStorage{
		config:  config,
		pgxConn: pgxConn,
	}
	return &storage, nil
}

func (s *dbStorage) AddLink(ctx context.Context, sToken string, longURL string, user string) (string, error) {

	log.Printf("Записываем в бд %s %s\n", sToken, longURL)
	// используем контекст запроса
	shortURL, err := s.InsertLine(ctx, sToken, longURL, user)
	if err != nil {
		log.Println(err.Error())
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
	return s.pgxConn.Ping(ctx)
}

func (s *dbStorage) GetAllURLS(ctx context.Context, cookie string) (map[string]string, error) {
	var link models.LinksData
	userLinks := make(map[string]string)

	rows, err := s.pgxConn.Query(ctx, selectByUser, cookie)
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

func InitTable(ctx context.Context, connString string) (*pgx.Conn, error) {
	log.Println("Инициализация таблицы")

	// открываем соединение с бд
	pgxConn, err := pgx.Connect(ctx, connString)
	if err != nil {
		log.Printf("database|Init table|%v\n", err)
		return nil, err
	}

	if _, err = pgxConn.Exec(ctx, createSQL); err != nil {
		log.Printf("database|Ошибка при создании таблицы|%v\n", err)
		return nil, err
	}

	return pgxConn, nil
}

func (s *dbStorage) InsertLine(ctx context.Context, shortURL string, longURL string, cookie string) (string, error) {
	var pgxError *pgconn.PgError
	s.mu.Lock()         // берём мьютекс
	defer s.mu.Unlock() // отпускаем мьютекс

	res, err := s.pgxConn.Exec(ctx, insertSQL, shortURL, longURL, cookie)
	if err == nil {
		rows := res.RowsAffected()
		if rows > 0 {
			log.Printf("Вставлено строк %d\n", rows)
		}
		return "", nil
	}

	log.Printf("database|Insert line|%v\n", err)
	if !errors.As(err, &pgxError) {
		return "", err
	}
	resSelect, errSelect := s.pgxConn.Query(ctx, selectShortURL, longURL)
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
		log.Printf("Найденный короткий URL %s\n", link.ShortURL)

		log.Println(pgxError.Code)
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
	batch := &pgx.Batch{}
	for _, v := range sTokens {
		batch.Queue(deleteSQL, v.Token, v.User)
	}
	br := s.pgxConn.SendBatch(ctx, batch)
	comTag, err := br.Exec()
	if err != nil {
		log.Printf("database|Batch delete request error|%v\n", err)
	}
	log.Printf("Изменено строк %d\n", comTag.RowsAffected())
}

func (s dbStorage) ShortenBatch(ctx context.Context, batchReq []models.BatchReq, cookie string) ([]models.BatchResp, error) {

	response := make([]models.BatchResp, 0, 100)
	var errStmt error

	batch := &pgx.Batch{}

	for _, batchValue := range batchReq {

		// проверяем, что в базе еще нет такого url
		_, err := findErrorURL(ctx, s, batchValue.URL)
		if err != nil {
			log.Printf("database|Find error URL|%v\n", errStmt)
			return nil, err
		}

		sToken := utils.GenRandToken(s.config.BaseURL)
		log.Printf("Записываем в бд %s, %s \n", sToken, batchValue.URL)
		batch.Queue(insertSQL, sToken, batchValue.URL, cookie)

		// формируем структуру для ответа
		response = append(response, models.BatchResp{
			CorrID:   batchValue.CorrID,
			ShortURL: sToken,
		})
	}

	br := s.pgxConn.SendBatch(ctx, batch)
	_, err := br.Exec()
	if err != nil {
		log.Printf("database|Batch req error|%v\n", err)
	}

	log.Printf("Структура ответа %s\n", response)

	err = br.Close()
	if err != nil {
		return nil, err
	}
	return response, nil
}

func findErrorURL(ctx context.Context, db dbStorage, URL string) (string, error) {
	var link models.LinksData
	var sToken string

	rows, err := db.pgxConn.Query(ctx, selectShortURL, URL)
	if err != nil {
		return "", err
	}
	for rows.Next() {
		err := rows.Scan(&link.ShortURL)
		if err != nil {
			return "", err
		}
		log.Printf("Найденный в бд короткий URL %s\n", link.ShortURL)
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
	log.Println("Ищем длинный URL в бд")
	var longURL string
	var deleted bool
	err := s.pgxConn.QueryRow(ctx, selectLongURL, shortURL).Scan(&longURL, &deleted)
	if err != nil {
		log.Println(err.Error())
		return "", models.ErrLinkNotFound
	}
	if deleted == true {
		return "", models.ErrLinkDeleted
	}
	return longURL, nil
}

func (s *dbStorage) Close() error {
	return s.pgxConn.Close(context.Background())
}
