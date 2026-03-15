package main

import (
	"fmt"
	"runtime"

	"github.com/pressly/goose/v3"

	"github.com/omaklabs/base/internal/config"
	"github.com/omaklabs/base/internal/db"
	"github.com/omaklabs/base/migrations"
)

func cmdVersion() {
	fmt.Printf("omakase-go  %s\n", version)
	fmt.Printf("go          %s\n", runtime.Version())
	fmt.Printf("os/arch     %s/%s\n", runtime.GOOS, runtime.GOARCH)

	// Try to show migration version
	cfg := config.Load()
	dbConn, err := db.Connect(cfg.DatabasePath)
	if err != nil {
		fmt.Printf("db          error: %v\n", err)
		return
	}
	defer dbConn.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		fmt.Printf("migrations  error: %v\n", err)
		return
	}

	ver, err := goose.GetDBVersion(dbConn)
	if err != nil {
		fmt.Printf("migrations  not initialized\n")
		return
	}
	fmt.Printf("migrations  version %d\n", ver)
	fmt.Printf("database    %s\n", cfg.DatabasePath)
}
