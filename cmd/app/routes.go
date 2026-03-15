package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/omaklabs/base/internal/config"
	"github.com/omaklabs/base/internal/db"
	"github.com/omaklabs/base/internal/email"
	"github.com/omaklabs/base/internal/jobs"
	"github.com/omaklabs/base/internal/logger"
	"github.com/omaklabs/base/internal/server"
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
