package main

import (
	"database/sql"
	"flag"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func main() {
	var database string
	defaultDb := "postgres://user:password@localhost:5432/postgres?sslmode=disable"

	var migrations string
	defaultMigrations := "file://migrations"

	flag.StringVar(&database, "db", defaultDb, "database uri for migration")
	flag.StringVar(&migrations, "migrations", defaultMigrations, "database migration folder. Must be like url 'file://foldername' ")

	flag.Parse()

	db, err := sql.Open("postgres", database)

	if err != nil {
		log.Panic(err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})

	if err != nil {
		log.Panic(err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		migrations,
		"postgres",
		driver,
	)

	if err != nil {
		log.Panic(err)
	}

	err = m.Up()

	if err != nil {
		log.Panic(err)
	}
}
