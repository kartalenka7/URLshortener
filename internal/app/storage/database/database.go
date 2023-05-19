package database

import (
	"context"
	"errors"
	"log"
	"time"

	"example.com/shortener/internal/config"
	"example.com/shortener/internal/config/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"example.com/shortener/internal/app/models"
)

type dbStorage struct {
	config  config.Config
	pgxConn *pgx.Conn
}

const UniqViolation = "23505"

var (
	createSQL = `CREATE TABLE IF NOT EXISTS urlsBase(
					short_url TEXT,
					long_url TEXT UNIQUE,
					cookie TEXT 
					);`
	insertSQL      = `INSERT INTO urlsBase(short_url, long_url, cookie) VALUES ($1, $2, $3)`
	selectShortURL = `SELECT short_url FROM urlsBase WHERE long_url = $1`
	selectByUser   = `SELECT short_url, long_url FROM urlsBase WHERE cookie = $1`
	selectLongURL  = `SELECT long_url FROM urlsBase WHERE short_url = $1`
)

func New(config config.Config) (*dbStorage, error) {
	pgxConn, err := InitTable(config.Database)
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

func (s dbStorage) AddLink(ctx context.Context, sToken, longURL string, user string) (string, error) {

	log.Printf("Записываем в бд %s %s\n", sToken, longURL)
	// используем контекст запроса
	shortURL, err := s.InsertLine(ctx, sToken, longURL, user)
	if err != nil {
		log.Println(err.Error())
		sToken = shortURL
	}
	return sToken, err
}

func (s dbStorage) GetLongURL(ctx context.Context, sToken string) (string, error) {

	longURL, err := s.SelectLink(ctx, sToken)
	if err != nil {
		log.Printf("storage|getLongURL|%v\n", err)
		return "", errors.New("link is not found")
	}
	return longURL, nil
}

func (s dbStorage) GetStorageLen() int {
	panic("error")
}

func (s dbStorage) Ping(ctx context.Context) error {
	pgxConn, err := pgx.Connect(ctx, s.config.Database)
	if err != nil {
		log.Printf("database|Ping|%v\n", err)
		return err
	}
	defer pgxConn.Close(ctx)

	return pgxConn.Ping(ctx)
}

func (s dbStorage) GetAllURLS(ctx context.Context, cookie string) (map[string]string, error) {
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

func InitTable(connString string) (*pgx.Conn, error) {
	log.Println("Инициализация таблицы")

	// открываем соединение с бд

	// конструируем контекст с 5-секундным тайм-аутом
	// после 5 секунд затянувшаяся операция с БД будет прервана
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// не забываем освободить ресурс
	defer cancel()

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

func (s dbStorage) InsertLine(ctx context.Context, shortURL string, longURL string, cookie string) (string, error) {

	res, err := s.pgxConn.Exec(ctx, insertSQL, shortURL, longURL, cookie)
	if err != nil {
		log.Printf("database|Insert line|%v\n", err)
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

			var pgxError *pgconn.PgError
			if errors.As(err, &pgxError) {
				log.Println(pgxError.Code)
				return link.ShortURL, err
			}
		}

		if resSelect.Err() != nil {
			return "", resSelect.Err()
		}

		return "", err
	}

	rows := res.RowsAffected()
	if rows > 0 {
		log.Printf("Вставлено строк %d\n", rows)
	}
	return "", nil
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

func (s dbStorage) SelectLink(ctx context.Context, shortURL string) (string, error) {
	log.Println("Ищем длинный URL в бд")
	var longURL string
	err := s.pgxConn.QueryRow(ctx, selectLongURL, shortURL).Scan(&longURL)
	if err != nil {
		return "", err
	}
	return longURL, nil
}

func (s dbStorage) Close() error {
	return s.pgxConn.Close(context.Background())
}
