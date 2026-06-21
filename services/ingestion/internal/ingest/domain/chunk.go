package domain

import "github.com/google/uuid"

type Chunk struct {
	ID             uuid.UUID      `json:"id"`
	DocumentID     uuid.UUID      `json:"documentId"`
	Content        string         `json:"content"`
	SequenceNumber int            `json:"sequenceNumber"`
	StartByte      int            `json:"startByte"`
	EndByte        int            `json:"endByte"`
	TokenCount     int            `json:"tokenCount"`
	Metadata       map[string]any `json:"metadata"`
	Embedding      []float32      `json:"embedding,omitempty"`
}

type Tokenizer interface {
	Count(text string) (int, error)
}
