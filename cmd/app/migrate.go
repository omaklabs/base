package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/pressly/goose/v3"

	"github.com/omaklabs/base/internal/config"
	"github.com/omaklabs/base/internal/db"
	"github.com/omaklabs/base/migrations"
)

func cmdMigrate(action string) {
	if action == "" {
		action = "up"
	}

	cfg := config.Load()
	dbConn, err := db.Connect(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer dbConn.Close()

	switch action {
	case "up":
		if err := migrateUp(dbConn); err != nil {
			log.Fatalf("migrate up: %v", err)
		}
		fmt.Println("migrations applied successfully")

	case "down":
		if err := migrateDown(dbConn); err != nil {
			log.Fatalf("migrate down: %v", err)
		}
		fmt.Println("rolled back last migration")

	case "status":
		if err := migrateStatus(dbConn); err != nil {
			log.Fatalf("migrate status: %v", err)
		}

	case "reset":
		if !cfg.IsDev() {
			fmt.Fprintln(os.Stderr, "migrate reset is only allowed in development (ENV=development)")
			os.Exit(1)
		}
		if err := migrateReset(dbConn); err != nil {
			log.Fatalf("migrate reset: %v", err)
		}
		fmt.Println("database reset and re-migrated successfully")

	default:
		fmt.Fprintf(os.Stderr, "unknown migrate action: %s\nValid actions: up, down, status, reset\n", action)
		os.Exit(1)
	}
}

func initGoose() error {
	goose.SetBaseFS(migrations.FS)
	return goose.SetDialect("sqlite3")
}

func migrateUp(dbConn *sql.DB) error {
	if err := initGoose(); err != nil {
		return err
	}
	return goose.Up(dbConn, ".")
}

func migrateDown(dbConn *sql.DB) error {
	if err := initGoose(); err != nil {
		return err
	}
	return goose.Down(dbConn, ".")
}

func migrateStatus(dbConn *sql.DB) error {
	if err := initGoose(); err != nil {
		return err
	}
	return goose.Status(dbConn, ".")
}

func migrateReset(dbConn *sql.DB) error {
	if err := initGoose(); err != nil {
		return err
	}
	if err := goose.Reset(dbConn, "."); err != nil {
		return err
	}
	return goose.Up(dbConn, ".")
}
