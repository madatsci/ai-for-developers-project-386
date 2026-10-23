package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/madatsci/ai-for-developers-project-386/backend/internal/httpapi"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	err = pool.Ping(pingCtx)
	cancel()
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	addr := listenAddress()
	handler := httpapi.NewHandler(
		pool.Ping,
		httpapi.HandlerServices{
			PublicCallTypes:   httpapi.NewPublicCallTypesService(pool),
			PublicCalls:       httpapi.NewPublicCallsService(pool),
			AdminCallTypes:    httpapi.NewAdminCallTypesService(pool),
			AdminCalls:        httpapi.NewAdminCallsService(pool),
			AdminSchedules:    httpapi.NewAdminSchedulesService(pool),
			AdminWorkingHours: httpapi.NewAdminWorkingHoursService(pool),
		},
	)
	applicationHandler := handler
	if staticDir := os.Getenv("STATIC_DIR"); staticDir != "" {
		applicationHandler, err = httpapi.NewApplicationHandler(handler, staticDir)
		if err != nil {
			return fmt.Errorf("configure frontend: %w", err)
		}
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           applicationHandler,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("starting HTTP server", "address", addr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shut down HTTP server: %w", err)
	}
	if err := <-serverErr; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	return nil
}

func listenAddress() string {
	if port := os.Getenv("PORT"); port != "" {
		return "0.0.0.0:" + port
	}
	if addr := os.Getenv("HTTP_ADDR"); addr != "" {
		return addr
	}
	return "127.0.0.1:8080"
}
