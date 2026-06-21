package upload

import (
	"context"
	"io"

	"github.com/MK4070/aura/services/ingestion/internal/ingest/domain"
	"github.com/google/uuid"
)

// == inbound ports ==

type UploadCommand struct {
	DocumentID  uuid.UUID
	FileName    string
	ContentType string
	SizeBytes   int64
	Content     io.Reader
}

type Uploader interface {
	UploadDocument(ctx context.Context, cmd UploadCommand) error
}

// == outbound ports ==

type DocumentStore interface {
	UploadDocument(ctx context.Context, doc domain.Document, content io.Reader) error
}

type EventPublisher interface {
	Publish(ctx context.Context, event domain.DocumentUploadedEvent) error
}
