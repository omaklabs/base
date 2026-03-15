package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/omakase-dev/go-boilerplate/internal/config"
	"github.com/omakase-dev/go-boilerplate/internal/db"
	"github.com/omakase-dev/go-boilerplate/internal/email"
	"github.com/omakase-dev/go-boilerplate/internal/jobs"
	"github.com/omakase-dev/go-boilerplate/internal/logger"
)

func cmdServe() {
	config.LoadDotEnv(".env")
	cfg := config.Load()
	appLogger := logger.New(os.Stdout)

	dbConn, err := db.Connect(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer dbConn.Close()

	if err := migrateUp(dbConn); err != nil {
		log.Fatalf("running migrations: %v", err)
	}

	queries := db.New(dbConn)
	queue := jobs.NewQueue(queries)
	emailStore := email.NewStore(queries)

	// Create mailer based on environment
	var mailer email.Mailer
	if cfg.IsDev() {
		mailer = email.NewDevMailer(emailStore, appLogger)
	} else {
		mailer = email.NewSMTPMailer(emailStore, email.SMTPConfig{
			Host:       cfg.Mail.Host,
			Port:       cfg.Mail.Port,
			Username:   cfg.Mail.Username,
			Password:   cfg.Mail.Password,
			Encryption: cfg.Mail.Encryption,
		}, appLogger)
	}
	_ = mailer // available for use by handlers/jobs

	// Start background job worker
	jobCtx, jobCancel := context.WithCancel(context.Background())
	defer jobCancel()
	go func() {
		if err := queue.Process(jobCtx); err != nil && jobCtx.Err() == nil {
			appLogger.Error("job worker stopped", "error", err.Error())
		}
	}()

	reloadFn := func() error {
		if err := config.LoadDotEnv(".env"); err != nil {
			return err
		}
		cfg = config.Load()
		appLogger.Info("config reloaded")
		return nil
	}

	router := buildRouter(cfg, dbConn, cfg.DatabasePath, queries, queue, emailStore, appLogger, reloadFn)

	// Graceful shutdown
	srv := &http.Server{Addr: cfg.Addr, Handler: router}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		appLogger.Info("server started", "addr", cfg.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("server error", "error", err.Error())
		}
	}()

	<-ctx.Done()
	appLogger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		appLogger.Error("shutdown error", "error", err.Error())
	}
}
