package domain_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/MK4070/aura/services/ingestion/internal/ingest/domain"
	"github.com/google/uuid"
)

type MockTokenizer struct {
	CountFunc func(text string) (int, error)
}

func (m *MockTokenizer) Count(text string) (int, error) {
	if m.CountFunc != nil {
		return m.CountFunc(text)
	}
	// Default behavior: 1 token per character
	return len(text), nil
}

func TestFixedSize_Chunk(t *testing.T) {
	docID := uuid.New()
	uploadTime := time.Now()

	baseDoc := domain.Document{
		ID:          docID,
		FileName:    "test.txt",
		ContentType: "text/plain",
		UploadedAt:  uploadTime,
	}

	expectedMetadata := map[string]any{
		"fileName":   "test.txt",
		"sourceType": "text/plain",
		"uploadedAt": uploadTime.String(),
	}

	tests := []struct {
		name      string
		chunkSize uint16
		overlap   uint16
		t         domain.Tokenizer
		doc       domain.Document
		want      []domain.Chunk
		wantErr   bool
	}{
		{
			name:      "Invalid overlap (overlap >= chunkSize)",
			chunkSize: 10,
			overlap:   10,
			t:         nil,
			doc:       baseDoc,
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "Empty document content",
			chunkSize: 10,
			overlap:   2,
			t:         nil,
			doc: func() domain.Document {
				d := baseDoc
				d.Content = ""
				return d
			}(),
			want:    nil,
			wantErr: false,
		},
		{
			name:      "Single chunk (content smaller than chunk size)",
			chunkSize: 100,
			overlap:   10,
			t:         nil,
			doc: func() domain.Document {
				d := baseDoc
				d.Content = "Short text"
				return d
			}(),
			want: []domain.Chunk{
				{
					DocumentID:     docID,
					Content:        "Short text",
					SequenceNumber: 0,
					StartByte:      0,
					EndByte:        10,
					TokenCount:     0,
					Metadata:       expectedMetadata,
				},
			},
			wantErr: false,
		},
		{
			name:      "Multiple chunks with overlap",
			chunkSize: 5,
			overlap:   2,
			t:         nil,
			doc: func() domain.Document {
				d := baseDoc
				d.Content = "123456789"
				return d
			}(),
			want: []domain.Chunk{
				{
					DocumentID:     docID,
					Content:        "12345",
					SequenceNumber: 0,
					StartByte:      0,
					EndByte:        5,
					TokenCount:     0,
					Metadata:       expectedMetadata,
				},
				{
					// Step is (5-2) = 3. Next starts at index 3 ("4")
					DocumentID:     docID,
					Content:        "45678",
					SequenceNumber: 1,
					StartByte:      3,
					EndByte:        8,
					TokenCount:     0,
					Metadata:       expectedMetadata,
				},
				{
					// Step is 3. Next starts at index 6 ("7")
					DocumentID:     docID,
					Content:        "789",
					SequenceNumber: 2,
					StartByte:      6,
					EndByte:        9,
					TokenCount:     0,
					Metadata:       expectedMetadata,
				},
			},
			wantErr: false,
		},
		{
			name:      "Multi-byte characters (Emojis) byte tracking",
			chunkSize: 7,
			overlap:   2,
			t:         nil,
			doc: func() domain.Document {
				d := baseDoc
				// 8 runes total. Earth emoji takes 4 bytes.
				// "Hello " (6 bytes) + "🌍" (4 bytes) + "!" (1 byte) = 11 bytes total
				d.Content = "Hello 🌍!"
				return d
			}(),
			want: []domain.Chunk{
				{
					DocumentID:     docID,
					Content:        "Hello 🌍",
					SequenceNumber: 0,
					StartByte:      0,
					EndByte:        10, // 5 + 1 + 4
					TokenCount:     0,
					Metadata:       expectedMetadata,
				},
				{
					// Step = 5 runes ("Hello"). Byte length = 5
					DocumentID:     docID,
					Content:        " 🌍!", // 3 runes
					SequenceNumber: 1,
					StartByte:      5,  // Starts after "Hello"
					EndByte:        11, // 5 + (1 + 4 + 1)
					TokenCount:     0,
					Metadata:       expectedMetadata,
				},
			},
			wantErr: false,
		},
		{
			name:      "With Mock Tokenizer",
			chunkSize: 4,
			overlap:   0,
			t: &MockTokenizer{
				CountFunc: func(text string) (int, error) {
					return 42, nil
				},
			},
			doc: func() domain.Document {
				d := baseDoc
				d.Content = "abcd"
				return d
			}(),
			want: []domain.Chunk{
				{
					DocumentID:     docID,
					Content:        "abcd",
					SequenceNumber: 0,
					StartByte:      0,
					EndByte:        4,
					TokenCount:     42,
					Metadata:       expectedMetadata,
				},
			},
			wantErr: false,
		},
		{
			name:      "Tokenizer Returns Error",
			chunkSize: 4,
			overlap:   0,
			t: &MockTokenizer{
				CountFunc: func(text string) (int, error) {
					return 0, errors.New("tokenizer failure")
				},
			},
			doc: func() domain.Document {
				d := baseDoc
				d.Content = "abcd"
				return d
			}(),
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := domain.NewFixedSize(tt.chunkSize, tt.overlap, tt.t)
			got, gotErr := s.Chunk(tt.doc)

			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Chunk() failed unexpectedly: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Chunk() succeeded unexpectedly, expected an error")
			}

			if len(got) != len(tt.want) {
				t.Fatalf("Chunk() returned %d chunks, want %d", len(got), len(tt.want))
			}

			for i := range got {
				// We don't want to fail tests because uuid.New() generated random IDs
				if got[i].ID == uuid.Nil {
					t.Errorf("Chunk[%d].ID was nil, expected a generated UUID", i)
				}

				// Zero out IDs so reflect.DeepEqual can pass based on content alone
				got[i].ID = uuid.Nil
				tt.want[i].ID = uuid.Nil
			}

			// Deep equal comparison
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Chunk() mismatch:\nGot:  %+v\nWant: %+v", got, tt.want)
			}
		})
	}
}
