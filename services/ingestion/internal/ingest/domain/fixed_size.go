package domain

import "github.com/google/uuid"

type FixedSize struct {
	chunkSize uint16
	overlap   uint16
	tokenizer Tokenizer
}

func NewFixedSize(chunkSize, overlap uint16, t Tokenizer) *FixedSize {
	return &FixedSize{
		chunkSize: chunkSize,
		overlap:   overlap,
		tokenizer: t,
	}
}

func (s *FixedSize) Chunk(doc Document) ([]Chunk, error) {
	if s.chunkSize <= s.overlap {
		return nil, ErrInvalidOverlap
	}
	if s.chunkSize == 0 {
		return nil, ErrInvalidChunkSize
	}

	// in-memory allocation
	runes := []rune(doc.Content)
	var chunks []Chunk

	if len(runes) == 0 {
		return chunks, nil
	}

	step := int(s.chunkSize - s.overlap)
	seqNumber := 0
	currentByteOffset := 0

	for i := 0; i < len(runes); i += step {
		endRune := min(i+int(s.chunkSize), len(runes))
		chunkContent := string(runes[i:endRune])
		chunkByteLen := len(chunkContent)

		tokenCount := 0
		if s.tokenizer != nil {
			count, err := s.tokenizer.Count(chunkContent)
			if err != nil {
				return nil, err
			}
			tokenCount = count
		}

		// Duplicate the metadata map so chunks don't share the same memory reference
		chunkMetadata := map[string]any{
			"fileName":   doc.FileName,
			"uploadedAt": doc.UploadedAt.String(),
			"sourceType": doc.ContentType,
		}

		chunks = append(chunks, Chunk{
			ID:             uuid.New(),
			DocumentID:     doc.ID,
			Content:        chunkContent,
			SequenceNumber: seqNumber,
			StartByte:      currentByteOffset,
			EndByte:        currentByteOffset + chunkByteLen,
			TokenCount:     tokenCount,
			Metadata:       chunkMetadata,
		})

		seqNumber++
		if endRune == len(runes) {
			break
		}

		stepEndRune := min(i+step, len(runes))
		stepContent := string(runes[i:stepEndRune])
		currentByteOffset += len(stepContent)

	}

	return chunks, nil
}
