package embed

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MK4070/aura/services/ingestion/internal/ingest/domain"
	"github.com/ollama/ollama/api"
)

type OllamaEmbedder struct {
	client *api.Client
	model  string
}

func NewOllamaEmbedder(model string) (*OllamaEmbedder, error) {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, err
	}

	return &OllamaEmbedder{
		client: client,
		model:  model,
	}, nil

}

func (oe *OllamaEmbedder) Embed(ctx context.Context, chunks []domain.Chunk) error {
	if len(chunks) == 0 {
		return nil
	}

	inputs := make([]string, len(chunks))
	for i, chunk := range chunks {
		inputs[i] = chunk.Content
	}

	req := &api.EmbedRequest{
		Model: oe.model,
		Input: inputs,
	}

	resp, err := oe.client.Embed(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to batch embed chunks: %w", err)
	}

	if len(resp.Embeddings) != len(chunks) {
		return fmt.Errorf("embedding mismatch: requested %d, received %d", len(chunks), len(resp.Embeddings))
	}

	for i := range chunks {
		chunks[i].Embedding = resp.Embeddings[i]
	}

	slog.Debug("Successfully embedded chunks",
		slog.Int("count", len(chunks)),
		slog.String("model", oe.model),
	)
	return nil
}
