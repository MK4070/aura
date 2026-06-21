package vector

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MK4070/aura/services/ingestion/internal/ingest/domain"
	"github.com/qdrant/go-client/qdrant"
)

type VectorStore struct {
	qdrant         *qdrant.Client
	collectionName string

	wg      sync.WaitGroup
	closing atomic.Bool
}

type Config struct {
	Host           string
	Port           int
	APIKey         string
	UseTLS         bool
	CollectionName string
}

func NewVectorStore(cfg Config) (*VectorStore, error) {
	qClient, err := qdrant.NewClient(&qdrant.Config{
		Host:   cfg.Host,
		Port:   cfg.Port,
		APIKey: cfg.APIKey,
		UseTLS: cfg.UseTLS,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize qdrant gRPC client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := qClient.CollectionExists(ctx, cfg.CollectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if collection exists: %w", err)
	}

	if !exists {
		err = qClient.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: cfg.CollectionName,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     768, // Standard dimensionality for nomic-embed-text
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create collection %q: %w", cfg.CollectionName, err)
		}
	}

	return &VectorStore{
		qdrant:         qClient,
		collectionName: cfg.CollectionName,
	}, nil
}

func (c *VectorStore) Close() error {
	c.closing.Store(true)
	c.wg.Wait()
	if c.qdrant != nil {
		return c.qdrant.Close()
	}
	return nil
}

func (c *VectorStore) Upsert(ctx context.Context, chunks []domain.Chunk) error {
	if c.closing.Load() {
		return fmt.Errorf("qdrant client is shutting down, refusing new upserts")
	}

	c.wg.Add(1)
	defer c.wg.Done()

	if len(chunks) == 0 {
		return nil
	}

	points := make([]*qdrant.PointStruct, 0, len(chunks))

	for _, chunk := range chunks {
		if len(chunk.Embedding) == 0 {
			return fmt.Errorf("chunk %s has no embedding vector", chunk.ID)
		}

		payload := map[string]any{
			"documentId":     chunk.DocumentID.String(),
			"content":        chunk.Content,
			"sequenceNumber": chunk.SequenceNumber,
			"startByte":      chunk.StartByte,
			"endByte":        chunk.EndByte,
			"tokenCount":     chunk.TokenCount,
		}

		if chunk.Metadata != nil {
			for k, v := range chunk.Metadata {
				if _, exists := payload[k]; !exists {
					payload[k] = v
				}
			}
		}

		points = append(points, &qdrant.PointStruct{
			Id:      qdrant.NewID(chunk.ID.String()),
			Vectors: qdrant.NewVectors(chunk.Embedding...),
			Payload: qdrant.NewValueMap(payload),
		})
	}

	waitFlag := true

	_, err := c.qdrant.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: c.collectionName,
		Points:         points,
		Wait:           &waitFlag,
	})

	if err != nil {
		return fmt.Errorf("failed to upsert %d chunks to collection %s: %w", len(chunks), c.collectionName, err)
	}

	return nil
}
