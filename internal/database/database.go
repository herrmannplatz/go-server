package database

import (
	"database/sql"
	"embed"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

func ConnectToDB(dbURL string, embedMigrations embed.FS) (*sql.DB, error) {
	goose.SetBaseFS(embedMigrations)

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return nil, err
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return nil, err
	}

	return db, nil
}
