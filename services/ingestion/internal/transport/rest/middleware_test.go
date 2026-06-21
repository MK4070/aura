package rest_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MK4070/aura/services/ingestion/internal/transport/rest"
)

func TestRequestLoggerAndCORS(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		handlerFunc    http.HandlerFunc
		expectedStatus int
	}{
		{
			name:   "Implicit 200 OK",
			method: http.MethodGet,
			path:   "/users",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("success"))
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Explicit 400 Bad Request",
			method: http.MethodPost,
			path:   "/upload",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "invalid payload", http.StatusBadRequest)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Explicit 500 Internal Server Error",
			method: http.MethodDelete,
			path:   "/data/123",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "CORS Preflight OPTIONS",
			method: http.MethodOptions,
			path:   "/api/v1/query",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				// This should NOT be reached because the CORS middleware returns early
				t.Fatal("Handler should not be reached on OPTIONS preflight")
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			jsonHandler := slog.NewJSONHandler(&logBuf, nil)
			testLogger := slog.New(jsonHandler)

			corsHandler := rest.CorsMiddleware(tt.handlerFunc)

			wrappedHandler := rest.RequestLogger(testLogger)(corsHandler)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Client received status %d, expected %d", rr.Code, tt.expectedStatus)
			}

			if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:8000" {
				t.Errorf("Expected CORS origin http://localhost:8000, got %q", origin)
			}
			if methods := rr.Header().Get("Access-Control-Allow-Methods"); methods != "GET, POST, OPTIONS" {
				t.Errorf("Expected CORS methods GET, POST, OPTIONS, got %q", methods)
			}
			if headers := rr.Header().Get("Access-Control-Allow-Headers"); headers != "Content-Type" {
				t.Errorf("Expected CORS headers Content-Type, got %q", headers)
			}

			logOutput := logBuf.String()
			if logOutput == "" {
				t.Fatal("Expected log output, but got none")
			}

			var logEntry map[string]any
			if err := json.Unmarshal([]byte(logOutput), &logEntry); err != nil {
				t.Fatalf("Failed to parse log JSON: %v", err)
			}

			if msg := logEntry["msg"]; msg != "HTTP Request Completed" {
				t.Errorf("Expected msg 'HTTP Request Completed', got %v", msg)
			}

			if method := logEntry["method"]; method != tt.method {
				t.Errorf("Expected method %s, got %v", tt.method, method)
			}

			if path := logEntry["path"]; path != tt.path {
				t.Errorf("Expected path %s, got %v", tt.path, path)
			}

			if status := logEntry["status"]; status != float64(tt.expectedStatus) {
				t.Errorf("Expected logged status %d, got %v", tt.expectedStatus, status)
			}

			if traceID, ok := logEntry["trace_id"].(string); !ok || traceID == "" {
				t.Error("Expected trace_id to be populated in logs")
			}

			if duration, ok := logEntry["duration"].(string); !ok || duration == "" {
				t.Error("Expected duration to be populated in logs")
			}
		})
	}
}
