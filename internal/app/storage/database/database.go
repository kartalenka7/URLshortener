package database

import (
	"context"
	"database/sql"
	"errors"
	"log"
	urlNet "net/url"
	"time"

	//"github.com/lib/pq"

	"example.com/shortener/internal/config"
	"example.com/shortener/internal/config/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBStorage struct {
	config  config.Config
	db      DB
	context context.Context
}

type DB struct {
	db         *sql.DB
	pgxConn    *pgx.Conn
	stmtInsert *sql.Stmt
	stmtSelect *sql.Stmt
	stmtUser   *sql.Stmt
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

func (s DBStorage) AddLink(longURL string, user string, ctx context.Context) (string, error) {

	sToken := utils.GenRandToken(s.config.BaseURL)
	log.Printf("Записываем в бд %s %s\n", sToken, longURL)
	// используем контекст запроса
	s.context = ctx
	shortURL, err := s.InsertLine(sToken, longURL, user)
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

	s.context = ctx
	longURL, err := s.SelectLink(longToken)
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
	//db, err := sql.Open("postgres", s.config.Database)
	if err != nil {
		log.Printf("database|Ping|%v\n", err)
		return err
	}
	defer pgxConn.Close(ctx)

	return pgxConn.Ping(ctx)
}

func (s DBStorage) GetAllURLS(cookie string, ctx context.Context) map[string]string {
	var link LinksData
	userLinks := make(map[string]string)

	rows, err := s.db.pgxConn.Query(ctx, selectByUser, cookie)
	//rows, err := s.db.stmtUser.QueryContext(ctx, cookie)
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

	/* 	db, err := sql.Open("postgres",
	   		connString)
	   	if err != nil {
	   		log.Printf("database|Init table|%v\n", err)
	   		return DB{}, err
	   	} */

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

	/* 	if _, err = db.ExecContext(ctx, createSQL); err != nil {
		log.Printf("database|Ошибка при создании таблицы|%v\n", err)
		return DB{}, err
	} */

	/* 	stmtInsert, err := db.Prepare(insertSQL)
	   	if err != nil {
	   		log.Printf("database|Ошибка при подготовке Insert|%v\n", err)
	   		return DB{}, err
	   	}

	   	stmtSelect, err := db.Prepare(selectShortURL)
	   	if err != nil {
	   		log.Printf("database|Ошибка при подготовке Select|%v\n", err)
	   		return DB{}, err
	   	}

	   	stmtUser, err := db.Prepare(selectByUser)
	   	if err != nil {
	   		log.Printf("database|Ошибка при подготовке Select by User|%v\n", err)
	   		return DB{}, err
	   	} */

	dbStruct := DB{
		//db:         db,
		pgxConn: pgxConn,
		/* 		stmtInsert: stmtInsert,
		   		stmtSelect: stmtSelect,
		   		stmtUser:   stmtUser, */
	}
	return dbStruct, nil
}

func (db DB) Close() {
	defer func() {
		/* 		db.stmtSelect.Close()
		   		db.stmtInsert.Close() */
		db.pgxConn.Close(context.Background())
		db.db.Close()
	}()
}

type LinksData struct {
	ShortURL string `json:"short"`
	LongURL  string `json:"long"`
	User     string `json:"user"`
}

func (s DBStorage) InsertLine(shortURL string, longURL string, cookie string) (string, error) {

	res, err := s.db.pgxConn.Exec(s.context, insertSQL, shortURL, longURL, cookie)
	//res, err := s.db.stmtInsert.ExecContext(s.context, shortURL, longURL, cookie)
	if err != nil {
		log.Printf("database|Insert line|%v\n", err)
		resSelect, errSelect := s.db.pgxConn.Query(s.context, selectShortURL, longURL)
		//resSelect, errSelect := s.db.stmtSelect.QueryContext(s.context, longURL)
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

			//var pqErr *pq.Error
			var pgxError *pgconn.PgError
			/* 			if errors.As(err, &pqErr) {
				log.Println(pqErr.Code)
				return link.ShortURL, err
			} */
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

	//rows, err := res.RowsAffected()
	rows := res.RowsAffected()
	if rows > 0 {
		log.Printf("Вставлено строк %d\n", rows)
	}
	/* 	 else {
		log.Println(err.Error())
	} */
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

func (s DBStorage) ShortenBatch(ctx context.Context, batchReq []BatchReq, cookie string) ([]BatchResp, error) {

	response := make([]BatchResp, 0, 100)
	var errStmt error

	batch := &pgx.Batch{}

	for _, batchValue := range batchReq {

		// проверяем, что в базе еще нет такого url
		_, err := findErrorURL(s.db, ctx, batchValue.URL)
		if err != nil {
			log.Printf("database|Find error URL|%v\n", errStmt)
			return nil, err
		}

		sToken := utils.GenRandToken(s.config.BaseURL)
		log.Printf("Записываем в бд %s, %s \n", sToken, batchValue.URL)
		batch.Queue(insertSQL, sToken, batchValue.URL, cookie)

		// формируем структуру для ответа
		response = append(response, BatchResp{
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

func findErrorURL(db DB, ctx context.Context, URL string) (string, error) {
	var link LinksData
	var sToken string

	rows, err := db.pgxConn.Query(ctx, selectShortURL, URL)
	//rows, err := db.stmtSelect.QueryContext(ctx, URL)
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

func (s DBStorage) SelectLink(shortURL string) (string, error) {
	log.Println("Ищем длинный URL в бд")
	var longURL string
	err := s.db.pgxConn.QueryRow(s.context, selectLongURL, shortURL).Scan(&longURL)
	//err := db.db.QueryRow("SELECT long_url FROM urlsStore WHERE short_url = $1", shortURL).Scan(&longURL)
	if err != nil {
		return "", err
	}
	return longURL, nil
}

func (s DBStorage) Close() {
	s.db.Close()
}
