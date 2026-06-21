package rest

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/MK4070/aura/services/ingestion/internal/platform/logger"
	"github.com/google/uuid"
)

type loggingWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lw *loggingWriter) WriteHeader(code int) {
	lw.statusCode = code
	lw.ResponseWriter.WriteHeader(code)
}

func RequestLogger(baseLogger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			traceID := uuid.NewString()

			reqLogger := baseLogger.With(
				"trace_id", traceID,
				"method", r.Method,
				"path", r.URL.Path,
			)

			ctx := logger.WithLogger(r.Context(), reqLogger)
			r = r.WithContext(ctx)

			lw := &loggingWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(lw, r)

			reqLogger.Info("HTTP Request Completed",
				"status", lw.statusCode,
				"duration", time.Since(start).String(),
			)
		})
	}
}

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8000")

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
