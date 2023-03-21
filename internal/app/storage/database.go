package storage

import (
	"context"
	"database/sql"
	"log"
	urlNet "net/url"
	"time"

	"example.com/shortener/internal/config"
	"example.com/shortener/internal/config/utils"
	_ "github.com/lib/pq"
)

func InitTable(connString string) error {
	log.Println("Инициализация таблицы")
	// открываем соединение с бд
	db, err := sql.Open("postgres",
		connString)
	if err != nil {
		log.Printf("database|Init table|%s\n", err.Error())
		return err
	}
	defer db.Close()

	log.Println("Создаем контекст")
	// конструируем контекст с 5-секундным тайм-аутом
	// после 5 секунд затянувшаяся операция с БД будет прервана
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// не забываем освободить ресурс
	defer cancel()

	_, err = db.ExecContext(ctx,
		"CREATE TABLE IF NOT EXISTS urls("+
			`"short_url" TEXT,`+
			`"long_url" TEXT,`+
			`"cookie" TEXT`+
			`);`)
	if err != nil {
		log.Printf("database|Ошибка при создании таблицы|%s\n", err.Error())
		return err
	}
	return nil
}

func InsertLine(connString string, shortURL string, longURL string, cookie string) error {
	db, err := sql.Open("postgres",
		connString)
	if err != nil {
		log.Printf("database|Insert Lines|%s\n", err.Error())
		return err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = db.ExecContext(ctx, "INSERT INTO urls(short_url, long_url, cookie) VALUES ($1, $2, $3)", shortURL, longURL, cookie)

	if err != nil {
		log.Printf("database|Insert line|%s\n", err.Error())
		return err
	}
	return nil
}

func ShortenBatch(batchReq []BatchReq, config config.Config, cookie string) ([]BatchResp, error) {
	db, err := sql.Open("postgres", config.Database)
	if err != nil {
		log.Printf("database|Prepare transaction|%s\n", err.Error())
		return nil, err
	}
	// шаг 1 — объявляем транзакцию
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	// шаг 1.1 — если возникает ошибка, откатываем изменения
	defer tx.Rollback()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// не забываем освободить ресурс
	defer cancel()

	// шаг 2 — готовим инструкцию
	stmt, err := tx.PrepareContext(ctx, "INSERT INTO urls(short_url, long_url, cookie) VALUES ($1, $2, $3)")
	if err != nil {
		return nil, err
	}
	// шаг 2.1 — не забываем закрыть инструкцию, когда она больше не нужна
	defer stmt.Close()

	response := make([]BatchResp, 0, 100)
	for _, batchValue := range batchReq {

		gToken := utils.RandStringBytes(10)
		log.Println(gToken)
		sToken := config.BaseURL + gToken
		_, urlParseErr := urlNet.Parse(sToken)
		if urlParseErr != nil {
			sToken = config.BaseURL + "/" + gToken
		}

		if _, err = stmt.ExecContext(ctx, sToken, batchValue.URL, cookie); err != nil {
			return nil, err
		}

		// формируем структуру для ответа
		response = append(response, BatchResp{
			CorrId:   batchValue.CorrId,
			ShortURL: gToken,
		})
	}
	log.Printf("Структура ответа %s\n", response)
	// шаг 4 — сохраняем изменения
	tx.Commit()
	return response, nil
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

	rows, err := db.QueryContext(ctx, "SELECT short_url, long_url, cookie FROM urls")
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
