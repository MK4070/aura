package rest_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/MK4070/aura/services/ingestion/internal/transport/rest"
)

func TestNewRouter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	uploadHandler := &rest.UploadHandler{}

	tests := []struct {
		name           string
		method         string
		path           string
		isReady        bool
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Health Route - Server is Ready",
			method:         http.MethodGet,
			path:           "/health",
			isReady:        true,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"ok"}`,
		},
		{
			name:           "Health Route - Server is Shutting Down",
			method:         http.MethodGet,
			path:           "/health",
			isReady:        false,
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   "Server is shutting down",
		},
		{
			name:           "Health Route - Wrong HTTP Method",
			method:         http.MethodPost,
			path:           "/health",
			isReady:        true,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "",
		},
		{
			name:           "Document Route - Wrong HTTP Method",
			method:         http.MethodGet,
			path:           "/document",
			isReady:        true,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "",
		},
		{
			name:           "Unknown Route - Returns 404",
			method:         http.MethodGet,
			path:           "/unknown-path",
			isReady:        true,
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
		{
			name:           "CORS Preflight - Intercepts OPTIONS request",
			method:         http.MethodOptions,
			path:           "/document",
			isReady:        true,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ready atomic.Bool
			ready.Store(tt.isReady)

			router := rest.NewRouter(uploadHandler, &ready, logger)

			req, err := http.NewRequest(tt.method, tt.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedBody != "" && !strings.Contains(rr.Body.String(), tt.expectedBody) {
				t.Errorf("Expected body to contain %q, got %q", tt.expectedBody, rr.Body.String())
			}

			// 3. Verify CORS Headers are present on ALL responses
			if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:8000" {
				t.Errorf("Expected CORS origin http://localhost:8000, got %q", origin)
			}
			if methods := rr.Header().Get("Access-Control-Allow-Methods"); methods != "GET, POST, OPTIONS" {
				t.Errorf("Expected CORS methods GET, POST, OPTIONS, got %q", methods)
			}
			if headers := rr.Header().Get("Access-Control-Allow-Headers"); headers != "Content-Type" {
				t.Errorf("Expected CORS headers Content-Type, got %q", headers)
			}
		})
	}
}
