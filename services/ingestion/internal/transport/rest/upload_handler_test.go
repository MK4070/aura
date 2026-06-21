package rest_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MK4070/aura/services/ingestion/internal/ingest/domain"
	"github.com/MK4070/aura/services/ingestion/internal/ingest/upload"
	"github.com/MK4070/aura/services/ingestion/internal/platform/logger"
	"github.com/MK4070/aura/services/ingestion/internal/transport/rest"
)

type mockUploader struct {
	err error
}

func (m *mockUploader) UploadDocument(ctx context.Context, cmd upload.UploadCommand) error {
	return m.err
}

func createMultipartRequest(t *testing.T, fieldName, fileName, contentType, content string) *http.Request {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}

	_, err = part.Write([]byte(content))
	if err != nil {
		t.Fatalf("failed to write form file: %v", err)
	}

	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/document", &body)

	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req
}

func TestUploadHandler_HandleUpload(t *testing.T) {
	silentLog := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name           string
		buildReq       func(t *testing.T) *http.Request
		mockUploader   *mockUploader
		expectedStatus int
	}{
		{
			name: "successful multipart upload",
			buildReq: func(t *testing.T) *http.Request {
				return createMultipartRequest(t, "document", "data.txt", "text/plain", "file content here")
			},
			mockUploader:   &mockUploader{err: nil},
			expectedStatus: http.StatusAccepted, // 202
		},
		{
			name: "missing document field in form",
			buildReq: func(t *testing.T) *http.Request {
				// We send the form, but with the wrong field name ("wrong_field" instead of "document")
				return createMultipartRequest(t, "wrong_field", "data.txt", "text/plain", "content")
			},
			mockUploader:   &mockUploader{err: nil},
			expectedStatus: http.StatusBadRequest, // 400
		},
		{
			name: "core service rejects file type (400)",
			buildReq: func(t *testing.T) *http.Request {
				return createMultipartRequest(t, "document", "data.pdf", "application/pdf", "fake pdf bytes")
			},
			mockUploader:   &mockUploader{err: domain.ErrUnsupportedContentType},
			expectedStatus: http.StatusBadRequest, // 400
		},
		{
			name: "core service crashes (500)",
			buildReq: func(t *testing.T) *http.Request {
				return createMultipartRequest(t, "document", "data.txt", "text/plain", "good content")
			},
			mockUploader:   &mockUploader{err: errors.New("database disconnected")},
			expectedStatus: http.StatusInternalServerError, // 500
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := rest.NewUploadHandler(tt.mockUploader)

			req := tt.buildReq(t)

			ctx := logger.WithLogger(req.Context(), silentLog)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.HandleUpload(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}
		})
	}
}
