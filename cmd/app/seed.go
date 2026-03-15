package main

import (
	"context"
	"fmt"
	"log"

	"github.com/omakase-dev/go-boilerplate/internal/config"
	"github.com/omakase-dev/go-boilerplate/internal/db"
)

func cmdSeed() {
	cfg := config.Load()
	dbConn, err := db.Connect(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer dbConn.Close()

	// Run migrations first
	if err := migrateUp(dbConn); err != nil {
		log.Fatalf("running migrations: %v", err)
	}

	queries := db.New(dbConn)
	ctx := context.Background()

	// Collect seeds from all domain modules (defined in app.go)
	var seeds []struct {
		Name string
		Fn   func(ctx context.Context, q *db.Queries) error
	}
	for _, m := range modules {
		for _, s := range m.Seeds {
			seeds = append(seeds, struct {
				Name string
				Fn   func(ctx context.Context, q *db.Queries) error
			}{Name: s.Name, Fn: s.Fn})
		}
	}

	if len(seeds) == 0 {
		fmt.Println("no seed functions registered")
		fmt.Println("add Seeds to your domain Module in app.go")
		return
	}

	for _, s := range seeds {
		fmt.Printf("seeding %s... ", s.Name)
		if err := s.Fn(ctx, queries); err != nil {
			log.Fatalf("failed: %v", err)
		}
		fmt.Println("done")
	}

	fmt.Printf("\n%d seed(s) applied successfully\n", len(seeds))
}
