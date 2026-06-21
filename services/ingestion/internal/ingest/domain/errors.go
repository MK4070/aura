package domain

import "errors"

var (
	ErrInvalidDocument        = errors.New("invalid document parameters")
	ErrDocumentIDEmpty        = errors.New("document ID cannot be empty")
	ErrDocumentFileNameEmpty  = errors.New("file name cannot be empty")
	ErrDocumentContentType    = errors.New("content type cannot be empty")
	ErrDocumentSizeInvalid    = errors.New("document size must be strictly greater than zero")
	ErrDocumentSizeTooLarge   = errors.New("document size exceeds the maximum allowed limit")
	ErrDocumentStatusInvalid  = errors.New("invalid document status")
	ErrDocumentTimeZero       = errors.New("uploaded timestamp cannot be empty/zero")
	ErrUnsupportedContentType = errors.New("unsupported content type: only text/plain is allowed")

	ErrDocumentNotFound  = errors.New("document not found")
	ErrDuplicateDocument = errors.New("document already exists")

	ErrChunkingStrategyNotFound = errors.New("chunking strategy not set")
	ErrInvalidOverlap           = errors.New("overlap must be strictly less than chunk size")
	ErrInvalidChunkSize         = errors.New("chunk size must be greater than zero")
)
