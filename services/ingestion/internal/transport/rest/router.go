package rest

import (
	"log/slog"
	"net/http"
	"sync/atomic"
)

func NewRouter(uploadHandler *UploadHandler, ready *atomic.Bool, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /document", uploadHandler.HandleUpload)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if !ready.Load() {
			http.Error(w, "Server is shutting down", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	corsHandler := CorsMiddleware(mux)
	finalHandler := RequestLogger(logger)(corsHandler)

	return finalHandler
}
