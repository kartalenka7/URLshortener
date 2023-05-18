package database

import (
	"context"
	"errors"
	"log"
	urlNet "net/url"
	"time"

	"example.com/shortener/internal/config"
	"example.com/shortener/internal/config/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"example.com/shortener/internal/app/models"
)

type DBStorage struct {
	config config.Config
	db     DB
}

type DB struct {
	pgxConn *pgx.Conn
}

const UniqViolation = "23505"

var (
	createSQL = `CREATE TABLE IF NOT EXISTS storage(
					short_url TEXT,
					long_url TEXT UNIQUE,
					cookie TEXT 
					);`
	insertSQL      = `INSERT INTO storage(short_url, long_url, cookie) VALUES ($1, $2, $3)`
	selectShortURL = `SELECT short_url FROM storage WHERE long_url = $1`
	selectByUser   = `SELECT short_url, long_url FROM storage WHERE cookie = $1`
	selectLongURL  = `SELECT long_url FROM storage WHERE short_url = $1`
)

func New(config config.Config) (*DBStorage, error) {
	db, err := InitTable(config.Database)
	if err != nil {
		log.Println("Не учитываем таблицу бд")
		return nil, err
	}
	storage := DBStorage{
		config: config,
		db:     db,
	}
	return &storage, nil
}

func (s DBStorage) AddLink(ctx context.Context, longURL string, user string) (string, error) {

	sToken := utils.GenRandToken(s.config.BaseURL)
	log.Printf("Записываем в бд %s %s\n", sToken, longURL)
	// используем контекст запроса
	shortURL, err := s.InsertLine(ctx, sToken, longURL, user)
	if err != nil {
		log.Println(err.Error())
		sToken = shortURL
	}
	return sToken, err
}

func (s DBStorage) GetLongURL(ctx context.Context, sToken string) (string, error) {

	longToken := s.config.BaseURL + sToken
	_, urlParseErr := urlNet.Parse(longToken)
	if urlParseErr != nil {
		longToken = s.config.BaseURL + "/" + sToken
	}
	log.Printf("longToken %s", longToken)

	longURL, err := s.SelectLink(ctx, longToken)
	if err != nil {
		log.Printf("storage|getLongURL|%v\n", err)
		return "", errors.New("link is not found")
	}
	return longURL, nil
}

func (s DBStorage) GetStorageLen() int {
	panic("error")
}

func (s DBStorage) Ping(ctx context.Context) error {
	pgxConn, err := pgx.Connect(ctx, s.config.Database)
	if err != nil {
		log.Printf("database|Ping|%v\n", err)
		return err
	}
	defer pgxConn.Close(ctx)

	return pgxConn.Ping(ctx)
}

func (s DBStorage) GetAllURLS(cookie string, ctx context.Context) map[string]string {
	var link models.LinksData
	userLinks := make(map[string]string)

	rows, err := s.db.pgxConn.Query(ctx, selectByUser, cookie)
	if err != nil {
		log.Printf("database|GetAllURLs|%v\n", err)
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&link.ShortURL, &link.LongURL)
		if err != nil {
			log.Printf("database|GetAllURLs|%v\n", err)
			return nil
		}
		userLinks[link.ShortURL] = link.LongURL
	}
	if rows.Err() != nil {
		log.Printf("database|GetAllURLs|%s\n", rows.Err())
		return nil
	}
	return userLinks
}

func InitTable(connString string) (DB, error) {
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
		return DB{}, err
	}

	if _, err = pgxConn.Exec(ctx, createSQL); err != nil {
		log.Printf("database|Ошибка при создании таблицы|%v\n", err)
		return DB{}, err
	}

	dbStruct := DB{
		pgxConn: pgxConn,
	}
	return dbStruct, nil
}

func (db DB) Close() {
	defer func() {
		db.pgxConn.Close(context.Background())
	}()
}

func (s DBStorage) InsertLine(ctx context.Context, shortURL string, longURL string, cookie string) (string, error) {

	res, err := s.db.pgxConn.Exec(ctx, insertSQL, shortURL, longURL, cookie)
	if err != nil {
		log.Printf("database|Insert line|%v\n", err)
		resSelect, errSelect := s.db.pgxConn.Query(ctx, selectShortURL, longURL)
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

func (s DBStorage) ShortenBatch(ctx context.Context, batchReq []models.BatchReq, cookie string) ([]models.BatchResp, error) {

	response := make([]models.BatchResp, 0, 100)
	var errStmt error

	batch := &pgx.Batch{}

	for _, batchValue := range batchReq {

		// проверяем, что в базе еще нет такого url
		_, err := findErrorURL(ctx, s.db, batchValue.URL)
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

	br := s.db.pgxConn.SendBatch(ctx, batch)
	for range batchReq {
		_, err := br.Exec()
		if err != nil {
			log.Printf("database|Batch req error|%v\n", err)
		}
	}

	log.Printf("Структура ответа %s\n", response)

	err := br.Close()
	if err != nil {
		return nil, err
	}
	return response, nil
}

func findErrorURL(ctx context.Context, db DB, URL string) (string, error) {
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

func (s DBStorage) SelectLink(ctx context.Context, shortURL string) (string, error) {
	log.Println("Ищем длинный URL в бд")
	var longURL string
	err := s.db.pgxConn.QueryRow(ctx, selectLongURL, shortURL).Scan(&longURL)
	if err != nil {
		return "", err
	}
	return longURL, nil
}

func (s DBStorage) Close() {
	s.db.Close()
}
