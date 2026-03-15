package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/omaklabs/base/internal/config"
	"github.com/omaklabs/base/internal/db"
	"github.com/omaklabs/base/internal/email"
	"github.com/omaklabs/base/internal/jobs"
	"github.com/omaklabs/base/internal/logger"
	"github.com/omaklabs/base/internal/server"
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

	// Start background job worker
	jobCtx, jobCancel := context.WithCancel(context.Background())
	defer jobCancel()
	go func() {
		if err := queue.Process(jobCtx); err != nil && jobCtx.Err() == nil {
			appLogger.Error("job worker stopped", "error", err.Error())
		}
	}()

	deps := &server.Deps{
		Queries: queries,
		Queue:   queue,
		Mailer:  mailer,
		Logger:  appLogger,
		IsDev:   cfg.IsDev(),
	}

	// Register jobs and schedules from all domain modules (defined in app.go)
	for _, m := range modules {
		for _, j := range m.Jobs {
			queue.Register(j.Type, j.Handler)
		}
		for _, s := range m.Schedules {
			sched := jobs.NewScheduler(queue)
			sched.Add(s)
			go sched.Start(jobCtx)
		}
	}

	reloadFn := func() error {
		if err := config.LoadDotEnv(".env"); err != nil {
			return err
		}
		cfg = config.Load()
		appLogger.Info("config reloaded")
		return nil
	}

	router := buildRouter(cfg, dbConn, cfg.DatabasePath, deps, emailStore, reloadFn)

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
