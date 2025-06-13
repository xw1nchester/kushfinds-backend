package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	// "os"
	// "path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func main() {
	var migrationsPath, dsn string

	flag.StringVar(&migrationsPath, "migrations-path", "migrations", "path to migrations")
	flag.StringVar(&dsn, "dsn", "postgres://postgres:postgres@127.0.0.1:5432/kushfinds?sslmode=disable", "database dsn")
	flag.Parse()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		panic(err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		panic(err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres", driver)
	if err != nil {
		panic(err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("no migrations to apply")
			return
		}

		panic(err)
	}

	fmt.Println("all migrations have been successfully applied")
}
