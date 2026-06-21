package domain

import (
	"time"

	"github.com/google/uuid"
)

type DocumentUploadedEvent struct {
	EventID   uuid.UUID `json:"eventId"`
	EventType string    `json:"eventType"`
	Timestamp time.Time `json:"timestamp"`

	Data DocumentPayload `json:"data"`
}

type DocumentPayload struct {
	DocumentID uuid.UUID `json:"documentId"`

	// StoragePath string `json:"storagePath"`
}

func NewDocumentUploadedEvent(doc *Document, eventID uuid.UUID) DocumentUploadedEvent {
	return DocumentUploadedEvent{
		EventID:   eventID,
		EventType: "DocumentUploaded",
		Timestamp: time.Now().UTC(),
		Data: DocumentPayload{
			DocumentID: doc.ID,

			// StoragePath: doc.StoragePath,
		},
	}
}
