package tokenizer

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
)

//go:embed tokenizer.json
var nomicTokenizerBytes []byte

type SugarmeAdapter struct {
	tk *tokenizer.Tokenizer
}

func NewSugarmeAdapter() (*SugarmeAdapter, error) {
	tmpFile, err := os.CreateTemp("", "nomic-tokenizer-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file for tokenizer: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(nomicTokenizerBytes); err != nil {
		return nil, fmt.Errorf("failed to write embedded tokenizer bytes: %w", err)
	}

	tk, err := pretrained.FromFile(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to load tokenizer from file: %w", err)
	}

	return &SugarmeAdapter{tk: tk}, nil
}

func (s *SugarmeAdapter) Count(text string) (int, error) {
	if text == "" {
		return 0, nil
	}

	encoding, err := s.tk.EncodeSingle(text, false)
	if err != nil {
		return 0, fmt.Errorf("failed to encode text: %w", err)
	}

	return len(encoding.Ids), nil
}
