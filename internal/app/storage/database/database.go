package database

import (
	"context"
	"database/sql"
	"errors"
	"log"
	urlNet "net/url"
	"time"

	"github.com/lib/pq"

	"example.com/shortener/internal/config"
	"example.com/shortener/internal/config/utils"
)

type DBStorage struct {
	config  config.Config
	db      DB
	context context.Context
}

type DB struct {
	db         *sql.DB
	stmtInsert *sql.Stmt
	stmtSelect *sql.Stmt
	stmtUser   *sql.Stmt
}

const UniqViolation = pq.ErrorCode("23505")

var (
	createSQL = `CREATE TABLE IF NOT EXISTS urlsStore(
					"short_url" TEXT,
					"long_url" TEXT UNIQUE,
					"cookie" TEXT 
					);`
	insertSQL      = `INSERT INTO urlsStore(short_url, long_url, cookie) VALUES ($1, $2, $3)`
	selectShortURL = `SELECT short_url FROM urlsStore WHERE long_url = $1`
	selectByUser   = `SELECT short_url, long_url FROM urlsStore WHERE cookie = $1`
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

func (s DBStorage) AddLink(longURL string, user string, ctx context.Context) (string, error) {

	sToken := utils.GenRandToken(s.config.BaseURL)
	log.Printf("Записываем в бд %s %s\n", sToken, longURL)
	// используем контекст запроса
	s.context = ctx
	shortURL, err := s.InsertLine(s.db, sToken, longURL, user)
	if err != nil {
		log.Println(err.Error())
		sToken = shortURL
	}
	return sToken, err
}

func (s DBStorage) GetLongURL(sToken string) (string, error) {

	longToken := s.config.BaseURL + sToken
	_, urlParseErr := urlNet.Parse(longToken)
	if urlParseErr != nil {
		longToken = s.config.BaseURL + "/" + sToken
	}
	log.Printf("longToken %s", longToken)

	longURL, err := SelectLink(s.db, longToken)
	if err != nil {
		log.Printf("storage|getLongURL|%s\n", err.Error())
		return "", errors.New("link is not found")
	}
	return longURL, nil
}

func (s DBStorage) GetStorageLen() int {
	panic("error")
}

func (s DBStorage) Ping(ctx context.Context) error {
	db, err := sql.Open("postgres", s.config.Database)
	if err != nil {
		log.Printf("database|Ping|%s\n", err.Error())
		return err
	}
	defer db.Close()

	return db.PingContext(ctx)
}

func (s DBStorage) GetAllURLS(cookie string, ctx context.Context) map[string]string {
	var link LinksData
	userLinks := make(map[string]string)

	rows, err := s.db.stmtUser.QueryContext(ctx, cookie)
	if err != nil {
		log.Printf("database|GetAllURLs|%s\n", err.Error())
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&link.ShortURL, &link.LongURL)
		if err != nil {
			log.Printf("database|GetAllURLs|%s\n", err.Error())
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
	db, err := sql.Open("postgres",
		connString)
	if err != nil {
		log.Printf("database|Init table|%s\n", err.Error())
		return DB{}, err
	}

	// конструируем контекст с 5-секундным тайм-аутом
	// после 5 секунд затянувшаяся операция с БД будет прервана
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// не забываем освободить ресурс
	defer cancel()

	if _, err = db.ExecContext(ctx, createSQL); err != nil {
		log.Printf("database|Ошибка при создании таблицы|%s\n", err.Error())
		return DB{}, err
	}

	stmtInsert, err := db.Prepare(insertSQL)
	if err != nil {
		log.Printf("database|Ошибка при подготовке Insert|%s\n", err.Error())
		return DB{}, err
	}

	stmtSelect, err := db.Prepare(selectShortURL)
	if err != nil {
		log.Printf("database|Ошибка при подготовке Select|%s\n", err.Error())
		return DB{}, err
	}

	stmtUser, err := db.Prepare(selectByUser)
	if err != nil {
		log.Printf("database|Ошибка при подготовке Select by User|%s\n", err.Error())
		return DB{}, err
	}

	dbStruct := DB{
		db:         db,
		stmtInsert: stmtInsert,
		stmtSelect: stmtSelect,
		stmtUser:   stmtUser,
	}
	return dbStruct, nil
}

func (db DB) Close() {
	defer func() {
		db.stmtSelect.Close()
		db.stmtInsert.Close()
		db.Close()
	}()
}

type LinksData struct {
	ShortURL string `json:"short"`
	LongURL  string `json:"long"`
	User     string `json:"user"`
}

func (s DBStorage) InsertLine(db DB, shortURL string, longURL string, cookie string) (string, error) {

	res, err := db.stmtInsert.ExecContext(s.context, shortURL, longURL, cookie)
	if err != nil {
		log.Printf("database|Insert line|%s\n", err.Error())
		resSelect, errSelect := db.stmtSelect.QueryContext(s.context, longURL)
		if errSelect != nil {
			return "", errSelect
		}
		defer resSelect.Close()

		var link LinksData
		for resSelect.Next() {
			errSelect := resSelect.Scan(&link.ShortURL)
			if errSelect != nil {
				return "", errSelect
			}
			log.Printf("Найденный короткий URL %s\n", link.ShortURL)

			var pqErr *pq.Error
			if errors.As(err, &pqErr) {
				log.Println(pqErr.Code)
				return link.ShortURL, err
			}
		}

		if resSelect.Err() != nil {
			return "", resSelect.Err()
		}

		return "", err
	}

	rows, err := res.RowsAffected()
	if err == nil {
		log.Printf("Вставлено строк %d\n", rows)
	} else {
		log.Println(err.Error())
	}
	return "", nil
}

type BatchReq struct {
	CorrID string `json:"correlation_id"`
	URL    string `json:"original_url"`
}

type BatchResp struct {
	CorrID   string `json:"correlation_id"`
	ShortURL string `json:"short_url"`
}

func (s DBStorage) ShortenBatch(batchReq []BatchReq, cookie string) ([]BatchResp, error) {

	// объявляем транзакцию
	tx, err := s.db.db.Begin()
	if err != nil {
		return nil, err
	}
	// если возникает ошибка, откатываем изменения
	defer tx.Rollback()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// не забываем освободить ресурс
	defer cancel()

	response := make([]BatchResp, 0, 100)
	var errStmt error

	for _, batchValue := range batchReq {

		sToken := utils.GenRandToken(s.config.BaseURL)

		log.Printf("Записываем в бд %s, %s \n", sToken, batchValue.URL)
		if _, errStmt = s.db.stmtInsert.ExecContext(ctx, sToken, batchValue.URL, cookie); errStmt != nil {
			log.Printf("database|Insert line|%s\n", errStmt.Error())
			var pqErr *pq.Error
			if errors.As(errStmt, &pqErr) {
				// отловили попытку сократить уже имеющийся в базе URL
				if pqErr.Code == UniqViolation {
					sToken, err = findErrorURL(s.db, ctx, batchValue.URL)
					if err != nil {
						return nil, err
					}
				} else {
					return nil, errStmt
				}
			}
		}

		// формируем структуру для ответа
		response = append(response, BatchResp{
			CorrID:   batchValue.CorrID,
			ShortURL: sToken,
		})
	}
	log.Printf("Структура ответа %s\n", response)
	// сохраняем изменения
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return response, nil
}

func findErrorURL(db DB, ctx context.Context, URL string) (string, error) {
	var link LinksData
	var sToken string

	rows, err := db.stmtSelect.QueryContext(ctx, URL)
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

func SelectLink(db DB, shortURL string) (string, error) {
	log.Println("Ищем длинный URL в бд")
	var longURL string
	err := db.db.QueryRow("SELECT long_url FROM urlsStore WHERE short_url = $1", shortURL).Scan(&longURL)
	if err != nil {
		return "", err
	}

	return longURL, nil
}

func (s DBStorage) Close() {
	s.db.Close()
}
