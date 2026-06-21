package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MK4070/aura/services/ingestion/internal/adapters/embed"
	"github.com/MK4070/aura/services/ingestion/internal/adapters/storage"
	"github.com/MK4070/aura/services/ingestion/internal/adapters/tokenizer"
	"github.com/MK4070/aura/services/ingestion/internal/adapters/vector"
	"github.com/MK4070/aura/services/ingestion/internal/ingest/domain"
	"github.com/MK4070/aura/services/ingestion/internal/ingest/process"
	"github.com/MK4070/aura/services/ingestion/internal/platform/config"
	"github.com/MK4070/aura/services/ingestion/internal/platform/logger"
	"github.com/MK4070/aura/services/ingestion/internal/telemetry"
	"github.com/MK4070/aura/services/ingestion/internal/transport/consumer"
	"golang.org/x/sync/errgroup"
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

	log.Info("Initializing OpenTelemetry Provider")
	shutdownTelemetry := telemetry.InitProvider()
	defer func() {
		log.Info("Shutting down telemetry provider...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(shutdownCtx); err != nil {
			log.Error("Failed to cleanly shutdown telemetry", "error", err)
		}
	}()

	g, gCtx := errgroup.WithContext(ctx)

	store, err := storage.NewPostgresStore(cfg.TableName, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to initialize postgres store: %w", err)
	}
	defer func() {
		store.Close()
		log.Info("Store successfully closed")
	}()

	tokenizer, err := tokenizer.NewSugarmeAdapter()
	if err != nil {
		return fmt.Errorf("failed to initialize tokenizer: %w", err)
	}
	fixedStrategy := domain.NewFixedSize(512, 50, tokenizer)
	chunker, err := process.NewChunker(fixedStrategy)
	if err != nil {
		return fmt.Errorf("failed to initialize chunk service: %w", err)
	}
	embeddingClient, err := embed.NewOllamaEmbedder(cfg.EmbedModel)
	if err != nil {
		return fmt.Errorf("failed to initialize ollama: %w", err)
	}

	vectorStore, err := vector.NewVectorStore(vector.Config{
		Host:           cfg.QdrantHost,
		Port:           cfg.QdrantGRPCPort,
		UseTLS:         false,
		CollectionName: cfg.QdrantCollection,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize vector store: %w", err)
	}
	defer func() {
		vectorStore.Close()
		log.Info("VectorStore successfully closed")
	}()

	processor := process.NewProcessService(store, embeddingClient, chunker, vectorStore)

	consumer, err := consumer.NewConsumer(consumer.Config{
		Brokers:    cfg.KafkaBrokers,
		Topic:      cfg.KafkaTopic,
		GroupID:    fmt.Sprintf("group-%s", cfg.KafkaTopic),
		MaxRetries: cfg.KafkaConsumerMaxRetry,
		NumWorkers: cfg.KafkaConsumerWorkers,
	}, store, log)
	if err != nil {
		return fmt.Errorf("failed to initialize consumer: %w", err)
	}

	g.Go(func() error {
		log.Info("Starting consumer")
		if err := consumer.Start(gCtx, processor); err != nil {
			return fmt.Errorf("consumer crashed: %w", err)
		}

		log.Info("Consumer successfully exited")
		return nil
	})

	return g.Wait()
}
