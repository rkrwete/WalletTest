package main

import (
	"os"
	"fmt"
	"time"
	"errors"
	"context"
	"syscall"
	"log/slog"
	"net/http"
	"os/signal"

	"wallet_test/internal/config"
	"wallet_test/internal/handler"
	"wallet_test/internal/migrate"
	"wallet_test/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := migrate.Up(cfg.DB.DSN(), cfg.App.MigrationsPath, log); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.DB.DSN())
	if err != nil {
		return fmt.Errorf("parse db config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return fmt.Errorf("create pool: %w", err)
	}
	defer pool.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}
	log.Info("connected to postgres")

	repo := repository.NewPostgres(pool)
	h := handler.New(repo, log)

	srv := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      h.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("http server started", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
		case err := <-serverErr:
			return fmt.Errorf("http server: %w", err)
		case <-ctx.Done():
			log.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	log.Info("stopped")
	return nil
}