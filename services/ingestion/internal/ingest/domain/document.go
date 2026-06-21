package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type DocumentStatus string

const (
	StatusPending        DocumentStatus = "PENDING"
	StatusUploaded       DocumentStatus = "UPLOADED"
	StatusProcessing     DocumentStatus = "PROCESSING"
	StatusProcessed      DocumentStatus = "PROCESSED"
	StatusFailed         DocumentStatus = "FAILED"
	MaxDocumentSizeBytes                = 2 * 1024 * 1024
)

type Document struct {
	ID          uuid.UUID
	FileName    string
	ContentType string
	SizeBytes   int64
	Status      DocumentStatus
	UploadedAt  time.Time

	Content string // for downstream retrieval
}

func NewDocument(id uuid.UUID, fileName, contentType string, size int64) (*Document, error) {
	doc := &Document{
		ID:          id,
		FileName:    fileName,
		ContentType: contentType,
		SizeBytes:   size,
		Status:      StatusPending,
		UploadedAt:  time.Now(),
	}
	if err := doc.Validate(); err != nil {
		return nil, err
	}

	return doc, nil
}

func (d *Document) MarkAsUploaded() {
	d.Status = StatusUploaded
}

func (d *Document) Validate() error {
	if d.ID == uuid.Nil {
		return ErrDocumentIDEmpty
	}

	if strings.TrimSpace(d.FileName) == "" {
		return ErrDocumentFileNameEmpty
	}

	contentType := strings.ToLower(strings.TrimSpace(d.ContentType))
	switch contentType {
	case "text/plain":
		// Valid
	default:
		return ErrUnsupportedContentType
	}

	if d.SizeBytes <= 0 {
		return ErrDocumentSizeInvalid
	}

	if d.SizeBytes > MaxDocumentSizeBytes {
		return ErrDocumentSizeTooLarge
	}

	switch d.Status {
	case StatusPending, StatusUploaded, StatusProcessing, StatusProcessed, StatusFailed:
		// Valid
	default:
		return ErrDocumentStatusInvalid
	}

	if d.UploadedAt.IsZero() {
		return ErrDocumentTimeZero
	}

	return nil
}
