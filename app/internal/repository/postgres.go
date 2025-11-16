package repository

import (
	_ "embed"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

//go:embed migrations/001_init.sql
var mig1 string

//go:embed seeds/seed.sql
var seedSQL string

func OpenPostgres(dsn string) (*sqlx.DB, error) {
	return sqlx.Open("pgx", dsn)
}

func Migrate(db *sqlx.DB) error {
	_, err := db.Exec(mig1)
	return err
}

func Seed(db *sqlx.DB) error {
	_, err := db.Exec(seedSQL)
	return err
}

func Tx(db *sqlx.DB, fn func(*sqlx.Tx) error) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
