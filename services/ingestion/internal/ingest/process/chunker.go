package process

import (
	"github.com/MK4070/aura/services/ingestion/internal/ingest/domain"
)

type Chunker struct {
	strategy ChunkingStrategy
}

func NewChunker(s ChunkingStrategy) (*Chunker, error) {
	if s == nil {
		return nil, domain.ErrChunkingStrategyNotFound
	}
	return &Chunker{strategy: s}, nil
}

func (c *Chunker) SetStrategy(s ChunkingStrategy) {
	c.strategy = s
}

func (c *Chunker) Process(doc domain.Document) ([]domain.Chunk, error) {
	if c.strategy == nil {
		return nil, domain.ErrChunkingStrategyNotFound
	}
	return c.strategy.Chunk(doc)
}
