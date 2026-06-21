package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

type ServerConfig struct {
	Port              string
	Logger            *slog.Logger
	Handler           http.Handler
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}
type Server struct {
	logger     *slog.Logger
	httpServer *http.Server
}

func NewServer(cfg ServerConfig) *Server {
	return &Server{
		logger: cfg.Logger,
		httpServer: &http.Server{
			Addr:              cfg.Port,
			Handler:           cfg.Handler,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			ReadTimeout:       cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
		},
	}
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Initiating graceful shutdown...")
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
