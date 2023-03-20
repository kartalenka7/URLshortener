package storage

import (
	"context"
	"database/sql"
	"log"
	"time"

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
