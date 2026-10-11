// Package database owns the SQLite connection and the schema migrations that
// run when the application starts.
package database

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"
)

// Embedding the migrations keeps the binary independent of the working
// directory it is started from.
//
//go:embed migrations/*.sql
var migrationFiles embed.FS

// Init opens the SQLite database at path, creating the file if needed, and
// applies every pending migration. The caller owns the returned handle and
// must Close it.
func Init(path string) (*sql.DB, error) {
	db, err := open(path)
	if err != nil {
		return nil, err
	}
	if err := migrateUp(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// dsn builds the connection string. SQLite enforces foreign keys per
// connection and leaves them off by default, so every pooled connection must
// switch them on or ON DELETE CASCADE is silently ignored.
func dsn(path string) string {
	return path + "?_foreign_keys=on"
}

// open returns a connection pool for the file at path. sql.Open is lazy, so
// the ping surfaces problems such as an unwritable path at startup rather
// than on the first request.
func open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn(path))
	if err != nil {
		return nil, fmt.Errorf("open database %q: %w", path, err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to database %q: %w", path, err)
	}
	return db, nil
}

// newMigrator wires the embedded migration files to db. It reuses the
// caller's handle so the settings from dsn apply to migrations as well.
func newMigrator(db *sql.DB) (*migrate.Migrate, error) {
	src, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("load migrations: %w", err)
	}
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return nil, fmt.Errorf("prepare migration driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "sqlite3", driver)
	if err != nil {
		return nil, fmt.Errorf("create migrator: %w", err)
	}
	return m, nil
}

// migrateUp applies every pending migration. A database that is already up
// to date counts as success, so restarting the application is a no-op.
//
// The migrator is deliberately never closed: the sqlite3 driver's Close
// closes the *sql.DB it was given, which here is the caller's live handle.
func migrateUp(db *sql.DB) error {
	m, err := newMigrator(db)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
