package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB() (*sql.DB, error) {
	dbDir := "./db"

	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		err = os.Mkdir(dbDir, 0755)
		if err != nil {
			return nil, fmt.Errorf("ошибка при создании директории db: %w", err)
		}
	}

	dbFile := filepath.Join(dbDir, "scheduler.db")

	_, err := os.Stat(dbFile)
	var install bool
	if err != nil {
		install = true
	}

	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, fmt.Errorf("ошибка при подключении к базе данных: %w", err)
	}

	if install {
		createTableSQL := `
		CREATE TABLE IF NOT EXISTS scheduler (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT,
			title TEXT NOT NULL,
			comment TEXT,
			repeat TEXT  CHECK (LENGTH(repeat) <= 128)
		);
		CREATE INDEX IF NOT EXISTS idx_date ON scheduler(date);
		`
		_, err = db.Exec(createTableSQL)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("ошибка при создании таблицы: %w", err)
		}

		fmt.Println("База данных и таблица scheduler успешно созданы.")
	} else {
		fmt.Println("База данных уже существует.")
	}

	return db, nil
}
