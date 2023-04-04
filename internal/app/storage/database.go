package storage

import (
	"context"
	"database/sql"
	"errors"
	"log"
	urlNet "net/url"
	"time"

	"example.com/shortener/internal/config/utils"
	"github.com/lib/pq"
)

type DB struct {
	db         *sql.DB
	stmtInsert *sql.Stmt
	stmtSelect *sql.Stmt
}

const uniqViolation = pq.ErrorCode("23505")

var (
	createSQL = `CREATE TABLE IF NOT EXISTS urlsStore(
					"short_url" TEXT,
					"long_url" TEXT UNIQUE,
					"cookie" TEXT 
					);;`
	insertSQL      = `INSERT INTO urlsStore(short_url, long_url, cookie) VALUES ($1, $2, $3)`
	selectShortURL = `SELECT short_url FROM urlsStore WHERE long_url = $1`
)

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

	dbStruct := DB{
		db:         db,
		stmtInsert: stmtInsert,
		stmtSelect: stmtSelect,
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

func InsertLine(db DB, shortURL string, longURL string, cookie string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := db.stmtInsert.ExecContext(ctx, shortURL, longURL, cookie)
	if err != nil {
		log.Printf("database|Insert line|%s\n", err.Error())
		resSelect, errSelect := db.stmtSelect.QueryContext(ctx, longURL)
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

func ShortenBatch(batchReq []BatchReq, db DB, baseURL string, cookie string) ([]BatchResp, error) {

	// объявляем транзакцию
	tx, err := db.db.Begin()
	if err != nil {
		return nil, err
	}
	// если возникает ошибка, откатываем изменения
	defer tx.Rollback()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// не забываем освободить ресурс
	defer cancel()

	/* 	// готовим инструкцию
	   	stmt, err := tx.PrepareContext(ctx, "INSERT INTO urlsStore(short_url, long_url, cookie) VALUES ($1, $2, $3)")
	   	if err != nil {
	   		return nil, err
	   	}
	   	// не забываем закрыть инструкцию, когда она больше не нужна
	   	defer stmt.Close()

	   	// готовим инструкцию для выборки уже существующих сокращенных URL
	   	stmtSelect, errSelect := tx.PrepareContext(ctx, "SELECT short_url FROM urlsStore WHERE long_url = $1")
	   	if errSelect != nil {
	   		return nil, err
	   	}
	   	defer stmtSelect.Close() */

	response := make([]BatchResp, 0, 100)
	var link LinksData
	var errStmt error

	for _, batchValue := range batchReq {

		gToken := utils.RandStringBytes(10)
		log.Println(gToken)
		sToken := baseURL + gToken
		_, urlParseErr := urlNet.Parse(sToken)
		if urlParseErr != nil {
			sToken = baseURL + "/" + gToken
		}

		log.Printf("Записываем в бд %s, %s \n", sToken, batchValue.URL)
		if _, errStmt = db.stmtInsert.ExecContext(ctx, sToken, batchValue.URL, cookie); errStmt != nil {
			log.Printf("database|Insert line|%s\n", errStmt.Error())
			var pqErr *pq.Error
			if errors.As(errStmt, &pqErr) {
				if pqErr.Code == uniqViolation {
					// попытка сократить уже имеющийся в базе URL
					rows, err := db.stmtSelect.QueryContext(ctx, batchValue.URL)
					if err != nil {
						return nil, err
					}
					for rows.Next() {
						errSelect := rows.Scan(&link.ShortURL)
						if errSelect != nil {
							return nil, errSelect
						}
						log.Printf("Найденный в бд короткий URL %s\n", link.ShortURL)
						sToken = link.ShortURL
						break
					}

					if rows.Err() != nil {
						return nil, rows.Err()
					}
				} else {
					return nil, err
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
	return response, errStmt
}

func SelectLines(connString string, limit int) ([]LinksData, error) {
	db, err := sql.Open("postgres",
		connString)
	if err != nil {
		log.Printf("database|Select Lines|%s\n", err.Error())
		return nil, err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var link LinksData
	linksAll := make([]LinksData, 0, limit)

	rows, err := db.QueryContext(ctx, "SELECT short_url, long_url, cookie FROM urlsStore")
	if err != nil {
		return nil, err
	}

	// обязательно закрываем перед возвратом функции
	defer rows.Close()

	// пробегаем по всем записям
	for rows.Next() {
		err = rows.Scan(&link.ShortURL, &link.LongURL, &link.User)
		if err != nil {
			log.Printf("database|Select lines|%s\n", err.Error())
			return nil, err
		}

		linksAll = append(linksAll, link)
	}

	// проверяем на ошибки
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return linksAll, nil
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
