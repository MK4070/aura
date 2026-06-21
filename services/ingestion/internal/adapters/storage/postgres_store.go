package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/MK4070/aura/services/ingestion/internal/ingest/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	db        *pgxpool.Pool
	tableName string
}

func NewPostgresStore(tableName, connString string) (*PostgresStore, error) {
	dbPool, err := func(dsn string) (*pgxpool.Pool, error) {
		startupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		pool, err := pgxpool.New(startupCtx, dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to parse config or connect: %w", err)
		}

		if err := pool.Ping(startupCtx); err != nil {
			pool.Close()
			return nil, fmt.Errorf("database unreachable: %w", err)
		}

		schema := fmt.Sprintf(`
		-- 1. Main Documents Table
		CREATE TABLE IF NOT EXISTS %[1]s (
			id UUID PRIMARY KEY,
			file_name VARCHAR(255) NOT NULL,
			content_type VARCHAR(50) NOT NULL,
			size_bytes BIGINT NOT NULL,
			status VARCHAR(50) NOT NULL,
			uploaded_at TIMESTAMPTZ NOT NULL,
			content TEXT NOT NULL
		);

		-- Partial index for fast polling of pending documents
		CREATE INDEX IF NOT EXISTS idx_%[1]s_status_uploaded 
		ON %[1]s(status, uploaded_at) 
		WHERE status = 'PENDING';

		-- 2. Error Events Table
		CREATE TABLE IF NOT EXISTS %[1]s_errors (
			id BIGSERIAL PRIMARY KEY,
			document_id UUID NOT NULL UNIQUE REFERENCES %[1]s(id) ON DELETE CASCADE,
			reason TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		-- Index for quickly fetching the error history of a specific document
		CREATE INDEX IF NOT EXISTS idx_%[1]s_errors_doc_id 
		ON %[1]s_errors(document_id);
		`, tableName)

		if _, err := pool.Exec(startupCtx, schema); err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to initialize database schema: %w", err)
		}

		return pool, nil
	}(connString)

	if err != nil {
		return nil, fmt.Errorf("Database initialization failed: %w", err)
	}

	return &PostgresStore{db: dbPool, tableName: tableName}, nil
}

func (s *PostgresStore) UploadDocument(ctx context.Context, doc domain.Document, content io.Reader) error {
	contentBytes, err := io.ReadAll(content)
	if err != nil {
		return fmt.Errorf("failed to read document content: %w", err)
	}

	contentStr := string(contentBytes)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := fmt.Sprintf(`
        INSERT INTO %[1]s (id, file_name, content_type, size_bytes, status, uploaded_at, content)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, s.tableName)

	_, err = s.db.Exec(ctx, query,
		doc.ID,
		doc.FileName,
		doc.ContentType,
		doc.SizeBytes,
		"UPLOADED",
		time.Now(),
		contentStr,
	)

	return err
}

func (s *PostgresStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Document, error) {
	query := fmt.Sprintf(`
		SELECT id, file_name, content_type, size_bytes, status, uploaded_at, content
		FROM %[1]s
		WHERE id = $1
	`, s.tableName)

	var doc domain.Document

	err := s.db.QueryRow(ctx, query, id).Scan(
		&doc.ID,
		&doc.FileName,
		&doc.ContentType,
		&doc.SizeBytes,
		&doc.Status,
		&doc.UploadedAt,
		&doc.Content,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("document with ID %s not found", id)
		}
		return nil, fmt.Errorf("failed to fetch document: %w", err)
	}

	return &doc, nil
}

func (s *PostgresStore) MarkAsProcessed(ctx context.Context, id uuid.UUID) error {
	query := fmt.Sprintf(`
		UPDATE %[1]s 
		SET status = 'PROCESSED' 
		WHERE id = $1
	`, s.tableName)

	commandTag, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return domain.ErrDocumentNotFound
	}

	return nil
}

func (s *PostgresStore) MarkAsFailed(ctx context.Context, id uuid.UUID, reason string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	updateQuery := fmt.Sprintf(`
		UPDATE %[1]s 
		SET status = 'FAILED' 
		WHERE id = $1 AND status != 'PROCESSED'
	`, s.tableName)

	commandTag, err := tx.Exec(ctx, updateQuery, id)
	if err != nil {
		return fmt.Errorf("failed to update document status: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return domain.ErrDocumentNotFound
	}

	insertQuery := fmt.Sprintf(`
		INSERT INTO %[1]s_errors (document_id, reason) 
		VALUES ($1, $2)
		ON CONFLICT (document_id) DO UPDATE 
		SET reason = EXCLUDED.reason
	`, s.tableName)

	if _, err := tx.Exec(ctx, insertQuery, id, reason); err != nil {
		return fmt.Errorf("failed to insert document error log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit failure transaction: %w", err)
	}

	return nil
}

func (s *PostgresStore) Close() {
	if s.db != nil {
		s.db.Close()
	}
}
