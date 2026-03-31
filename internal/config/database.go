package config

import (
	"database/sql"

	_ "github.com/lib/pq"
)

func ConnectDB() (*sql.DB, error) {
	connStr := "host=localhost port=5432 user=postgres password=123 dbname=access_control sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Проверяем подключение
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}
