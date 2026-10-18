package migrations

import (
	"database/sql"
	"embed"
	"fmt"
)

//go:embed *.sql
var files embed.FS

func Run(db *sql.DB) error {
	// Bootstrap the tracking table before checking which files have run.
	bootstrap, err := files.ReadFile("000_migrations.sql")
	if err != nil {
		return err
	}
	if _, err := db.Exec(string(bootstrap)); err != nil {
		return err
	}
	entries, err := files.ReadDir(".") // Sorted by filename.
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := apply(db, entry.Name()); err != nil {
			return fmt.Errorf("migration %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func apply(db *sql.DB, name string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var applied bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM _migrations WHERE name = ?)", name).Scan(&applied)
	if err != nil {
		return err
	}
	if applied {
		return nil
	}
	statement, err := files.ReadFile(name)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(string(statement)); err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT INTO _migrations (name) VALUES (?)", name); err != nil {
		return err
	}
	return tx.Commit()
}
