package upload_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/MK4070/aura/services/ingestion/internal/ingest/domain"
	"github.com/MK4070/aura/services/ingestion/internal/ingest/upload"
	"github.com/google/uuid"
)

type mockPublisher struct {
	publishFunc func(ctx context.Context, event domain.DocumentUploadedEvent) error
}

func (m *mockPublisher) Publish(ctx context.Context, event domain.DocumentUploadedEvent) error {
	if m.publishFunc != nil {
		return m.publishFunc(ctx, event)
	}
	return nil
}

type mockStore struct {
	uploadDocument func(ctx context.Context, doc domain.Document, content io.Reader) error
}

func (m *mockStore) UploadDocument(ctx context.Context, doc domain.Document, content io.Reader) error {
	if m.uploadDocument != nil {
		return m.uploadDocument(ctx, doc, content)
	}
	return nil
}

func TestService_UploadDocument(t *testing.T) {
	t.Parallel()

	mockID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	validCmd := func() upload.UploadCommand {
		content := "hello streaming world!"
		return upload.UploadCommand{
			DocumentID:  mockID,
			FileName:    "report.txt",
			ContentType: "text/plain",
			SizeBytes:   int64(len(content)),
			Content:     strings.NewReader(content),
		}
	}

	tests := []struct {
		name          string
		mutateCmd     func(*upload.UploadCommand)
		mockStore     *mockStore
		mockPublisher *mockPublisher
		expectedErr   error
	}{
		{
			name:      "successful upload stream",
			mutateCmd: func(cmd *upload.UploadCommand) {},
			mockStore: &mockStore{
				uploadDocument: func(ctx context.Context, doc domain.Document, content io.Reader) error {
					data, _ := io.ReadAll(content)
					if string(data) != "hello streaming world!" {
						t.Errorf("expected stream content to match")
					}
					// Because doc.MarkAsUploaded() is called AFTER store.UploadDocument in the code,
					// the status passed to the DB should be the initial state (Pending).
					if doc.Status != domain.StatusPending {
						t.Errorf("expected status to be PENDING, got %s", doc.Status)
					}
					return nil
				},
			},
			mockPublisher: &mockPublisher{
				publishFunc: func(ctx context.Context, event domain.DocumentUploadedEvent) error {
					// With Thin Events, we only verify that the trigger contains the correct DocumentID.
					// The worker will fetch the latest status directly from the DB.
					if event.Data.DocumentID != mockID {
						t.Errorf("expected event document ID to match mock ID, got %s", event.Data.DocumentID)
					}
					return nil
				},
			},
			expectedErr: nil,
		},
		{
			name: "domain validation catches bad data before streaming",
			mutateCmd: func(cmd *upload.UploadCommand) {
				cmd.FileName = "" // Invalid!
			},
			mockStore: &mockStore{
				uploadDocument: func(ctx context.Context, doc domain.Document, content io.Reader) error {
					t.Fatal("UploadDocument should NOT be called if validation fails")
					return nil
				},
			},
			mockPublisher: &mockPublisher{
				publishFunc: func(ctx context.Context, event domain.DocumentUploadedEvent) error {
					t.Fatal("Publish should NOT be called if validation fails")
					return nil
				},
			},
			expectedErr: domain.ErrDocumentFileNameEmpty,
		},
		{
			name:      "store upload failure halts execution",
			mutateCmd: func(cmd *upload.UploadCommand) {},
			mockStore: &mockStore{
				uploadDocument: func(ctx context.Context, doc domain.Document, content io.Reader) error {
					return errors.New("S3 bucket or database unavailable")
				},
			},
			mockPublisher: &mockPublisher{
				publishFunc: func(ctx context.Context, event domain.DocumentUploadedEvent) error {
					t.Fatal("Publish should NOT be called if store UploadDocument fails")
					return nil
				},
			},
			expectedErr: errors.New("failed to upload document: S3 bucket or database unavailable"),
		},
		{
			name:      "publisher failure after successful save",
			mutateCmd: func(cmd *upload.UploadCommand) {},
			mockStore: &mockStore{
				uploadDocument: func(ctx context.Context, doc domain.Document, content io.Reader) error {
					return nil
				},
			},
			mockPublisher: &mockPublisher{
				publishFunc: func(ctx context.Context, event domain.DocumentUploadedEvent) error {
					return errors.New("kafka broker disconnected")
				},
			},
			expectedErr: errors.New("failed to publish document uploaded event: kafka broker disconnected"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := validCmd()
			tt.mutateCmd(&cmd)

			svc := upload.NewUploadService(tt.mockStore, tt.mockPublisher)
			err := svc.UploadDocument(context.Background(), cmd)

			if tt.expectedErr == nil {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			} else if !errors.Is(err, tt.expectedErr) && err.Error() != tt.expectedErr.Error() {
				t.Errorf("expected error %q, got %q", tt.expectedErr, err)
			}
		})
	}
}
