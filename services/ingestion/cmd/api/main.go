package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/MK4070/aura/services/ingestion/internal/adapters/publisher"
	"github.com/MK4070/aura/services/ingestion/internal/adapters/storage"
	"github.com/MK4070/aura/services/ingestion/internal/ingest/upload"
	"github.com/MK4070/aura/services/ingestion/internal/platform/config"
	"github.com/MK4070/aura/services/ingestion/internal/platform/logger"
	"github.com/MK4070/aura/services/ingestion/internal/platform/server"
	"github.com/MK4070/aura/services/ingestion/internal/transport/rest"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "application failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	log := logger.NewLogger(cfg.Environment, cfg.LogLevel)
	slog.SetDefault(log)

	store, err := storage.NewPostgresStore(cfg.TableName, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to initialize postgres store: %w", err)
	}
	defer func() {
		store.Close()
		log.Info("Store successfully closed")
	}()

	publisher, err := publisher.NewPublisher(cfg.KafkaBrokers, cfg.KafkaTopic)
	if err != nil {
		return fmt.Errorf("failed to initialize kafka publisher: %w", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := publisher.Close(closeCtx); err != nil {
			log.Error("Publisher shutdown encountered an error", slog.String("error", err.Error()))
		} else {
			log.Info("Publisher successfully closed")
		}
	}()

	var isReady atomic.Bool
	isReady.Store(true)

	uploadSvc := upload.NewUploadService(store, publisher)
	uploadHandler := rest.NewUploadHandler(uploadSvc)
	router := rest.NewRouter(uploadHandler, &isReady, log)

	srv := server.NewServer(server.ServerConfig{
		Port:              cfg.AppPort,
		Logger:            log,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	})

	errChan := make(chan error, 1)
	go func() {
		log.Info("Starting server", slog.String("port", cfg.AppPort))
		if err := srv.Start(); err != nil {
			// Ignore http.ErrServerClosed, as it's expected during a graceful shutdown
			if !errors.Is(err, http.ErrServerClosed) {
				errChan <- err
			} else {
				errChan <- nil
			}
		}
	}()

	select {
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("server crashed: %w", err)
		}
	case <-ctx.Done():
		log.Info("Received termination signal, starting graceful shutdown")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server forced to shutdown abnormally: %w", err)
	}
	log.Info("Server successfully exited")

	return nil
}
