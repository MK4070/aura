package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/MK4070/aura/services/ingestion/internal/ingest/domain"
	"github.com/google/uuid"
)

func TestDocument_Validate(t *testing.T) {
	t.Parallel()

	validTime := time.Now()

	validDoc := func() domain.Document {
		return domain.Document{
			ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			FileName:    "report.txt",
			ContentType: "text/plain",
			SizeBytes:   12,
			Status:      domain.StatusPending,
			UploadedAt:  validTime,
		}
	}

	tests := []struct {
		name        string
		mutate      func(*domain.Document)
		expectedErr error
	}{
		{
			name:        "valid document",
			mutate:      func(d *domain.Document) {},
			expectedErr: nil,
		},
		{
			name: "empty ID",
			mutate: func(d *domain.Document) {
				d.ID = uuid.Nil
			},
			expectedErr: domain.ErrDocumentIDEmpty,
		},
		{
			name: "empty file name",
			mutate: func(d *domain.Document) {
				d.FileName = ""
			},
			expectedErr: domain.ErrDocumentFileNameEmpty,
		},
		{
			name: "unsupported content type (pdf)",
			mutate: func(d *domain.Document) {
				d.ContentType = "application/pdf"
			},
			expectedErr: domain.ErrUnsupportedContentType,
		},
		{
			name: "valid content type with weird casing and spaces",
			mutate: func(d *domain.Document) {
				d.ContentType = "  Text/Plain  " // Should pass!
			},
			expectedErr: nil,
		},
		{
			name: "size zero",
			mutate: func(d *domain.Document) {
				d.SizeBytes = 0
			},
			expectedErr: domain.ErrDocumentSizeInvalid,
		},
		{
			name: "size exceeds maximum",
			mutate: func(d *domain.Document) {
				d.SizeBytes = domain.MaxDocumentSizeBytes + 1
			},
			expectedErr: domain.ErrDocumentSizeTooLarge,
		},
		{
			name: "invalid status enum",
			mutate: func(d *domain.Document) {
				d.Status = "UPLOADING" // Not a defined constant
			},
			expectedErr: domain.ErrDocumentStatusInvalid,
		},
		{
			name: "zero uploaded time",
			mutate: func(d *domain.Document) {
				d.UploadedAt = time.Time{}
			},
			expectedErr: domain.ErrDocumentTimeZero,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := validDoc()

			tt.mutate(&doc)

			err := doc.Validate()

			if tt.expectedErr == nil {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.expectedErr)
				} else if !errors.Is(err, tt.expectedErr) && err.Error() != tt.expectedErr.Error() {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
			}
		})
	}
}
