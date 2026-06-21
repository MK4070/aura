package upload

import (
	"context"
	"fmt"

	"github.com/MK4070/aura/services/ingestion/internal/ingest/domain"
	"github.com/google/uuid"
)

type UploadService struct {
	store     DocumentStore
	publisher EventPublisher
}

func NewUploadService(s DocumentStore, p EventPublisher) *UploadService {
	return &UploadService{store: s, publisher: p}
}

func (s *UploadService) UploadDocument(ctx context.Context, cmd UploadCommand) error {
	doc, err := domain.NewDocument(
		cmd.DocumentID,
		cmd.FileName,
		cmd.ContentType,
		cmd.SizeBytes,
	)
	if err != nil {
		return err
	}

	if err := s.store.UploadDocument(ctx, *doc, cmd.Content); err != nil {
		return fmt.Errorf("failed to upload document: %w", err)
	}

	doc.MarkAsUploaded()

	eventId := uuid.New()
	event := domain.NewDocumentUploadedEvent(doc, eventId)

	if err := s.publisher.Publish(ctx, event); err != nil {
		return fmt.Errorf("failed to publish document uploaded event: %w", err)
	}

	return nil
}
