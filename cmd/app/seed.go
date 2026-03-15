package main

import (
	"context"
	"fmt"
	"log"

	"github.com/omakase-dev/go-boilerplate/internal/config"
	"github.com/omakase-dev/go-boilerplate/internal/db"
)

// SeedFunc is a function that seeds data into the database.
// The agent adds seed functions here when building features.
type SeedFunc func(ctx context.Context, q *db.Queries) error

// seeds is the registry of seed functions.
// The agent appends to this slice when adding new domain seed data.
var seeds = []struct {
	Name string
	Fn   SeedFunc
}{
	// Example (agent adds entries like this):
	// {"users", seedUsers},
	// {"posts", seedPosts},
}

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

	if len(seeds) == 0 {
		fmt.Println("no seed functions registered")
		fmt.Println("add seed functions in cmd/server/seed.go")
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
