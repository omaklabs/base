package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/omakase-dev/go-boilerplate/internal/config"
	"github.com/omakase-dev/go-boilerplate/internal/db"
	"github.com/omakase-dev/go-boilerplate/internal/email"
	"github.com/omakase-dev/go-boilerplate/internal/jobs"
	"github.com/omakase-dev/go-boilerplate/internal/logger"
	"github.com/omakase-dev/go-boilerplate/internal/server"
)

func cmdRoutes() {
	cfg := config.Load()

	// Build router with nil db — we only need the route tree
	deps := &server.Deps{
		Queries: &db.Queries{},
		Queue:   jobs.NewQueue(&db.Queries{}),
		Logger:  logger.New(nil),
	}
	router := buildRouter(cfg, nil, "", deps, email.NewStore(&db.Queries{}))

	fmt.Println("Registered routes:")
	fmt.Println()

	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		fmt.Printf("  %-7s %s\n", method, route)
		return nil
	}

	if err := chi.Walk(router, walkFunc); err != nil {
		fmt.Printf("error walking routes: %v\n", err)
	}
}
