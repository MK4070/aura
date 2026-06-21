package process

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/MK4070/aura/services/ingestion/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	apiMetric "go.opentelemetry.io/otel/metric"
)

type ProcessService struct {
	store    DocumentStore
	vector   VectorRepository
	embedder EmbeddingClient
	chunker  DocumentChunker
}

func NewProcessService(
	s DocumentStore,
	e EmbeddingClient,
	c DocumentChunker,
	v VectorRepository,
) *ProcessService {
	return &ProcessService{
		store:    s,
		embedder: e,
		chunker:  c,
		vector:   v,
	}

}

func (s *ProcessService) ProcessDocument(ctx context.Context, cmd ProcessCommand) error {
	startTime := time.Now()

	doc, err := s.store.GetByID(ctx, cmd.DocumentID)
	if err != nil {
		return fmt.Errorf("failed to find document: %w", err)
	}

	slog.Debug("Document fetched from store",
		"file_name", doc.FileName,
		"id", doc.ID,
		"status", doc.Status,
		"size", doc.SizeBytes,
	)

	if doc.Status == "PROCESSED" {
		slog.Info("Document already processed", "doc_id", cmd.DocumentID)
		// Return nil (success) so the worker knows it can safely commit the Kafka offset
		return nil
	}

	// 2. Chunk
	chunks, err := s.chunker.Process(*doc)
	if err != nil {
		return fmt.Errorf("failed to chunk: %w", err)
	}

	// 3. Embed
	if err := s.embedder.Embed(ctx, chunks); err != nil {
		return fmt.Errorf("failed to embed chunks: %w", err)
	}

	// 4. Save to vector DB
	err = s.vector.Upsert(ctx, chunks)
	duration := time.Since(startTime).Seconds()
	if err != nil {
		telemetry.ChunksProcessed.Add(ctx, 1, apiMetric.WithAttributes(attribute.String("status", "error")))

		telemetry.IngestionDuration.Record(ctx, duration, apiMetric.WithAttributes(attribute.String("status", "error")))

		return fmt.Errorf("failed to upsert chunks to vector repository: %w", err)
	}

	slog.Info("Successfully processed document",
		"document_id", doc.ID,
		"chunk_count", len(chunks),
	)

	telemetry.ChunksProcessed.Add(ctx, int64(len(chunks)), apiMetric.WithAttributes(attribute.String("status", "success")))

	telemetry.IngestionDuration.Record(ctx, duration, apiMetric.WithAttributes(attribute.String("status", "success")))

	return nil
}
