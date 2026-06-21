package process

import (
	"context"

	"github.com/MK4070/aura/services/ingestion/internal/ingest/domain"
	"github.com/google/uuid"
)

// == inbound ports ==

type ProcessCommand struct {
	DocumentID uuid.UUID
}

type Processor interface {
	ProcessDocument(ctx context.Context, cmd ProcessCommand) error
}

type DocumentChunker interface {
	Process(doc domain.Document) ([]domain.Chunk, error)
}

type ChunkingStrategy interface {
	Chunk(doc domain.Document) ([]domain.Chunk, error)
}

// == outbound ports ==

type DocumentStore interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Document, error)

	MarkAsProcessed(ctx context.Context, id uuid.UUID) error
	MarkAsFailed(ctx context.Context, id uuid.UUID, reason string) error
}

type VectorRepository interface {
	Upsert(ctx context.Context, chunks []domain.Chunk) error
}

type EmbeddingClient interface {
	Embed(ctx context.Context, chunks []domain.Chunk) error
}
