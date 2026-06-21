package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/MK4070/aura/services/ingestion/internal/ingest/domain"
	"github.com/MK4070/aura/services/ingestion/internal/ingest/upload"
	"github.com/MK4070/aura/services/ingestion/internal/platform/logger"
	"github.com/google/uuid"
)

type UploadHandler struct {
	uploader upload.Uploader
}

func NewUploadHandler(u upload.Uploader) *UploadHandler {
	return &UploadHandler{uploader: u}
}

func (h *UploadHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	log := logger.GetLogger(r.Context())
	log.Debug("Starting document upload process")
	// fail if > 2MB
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	file, fileHeader, err := r.FormFile("document")
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "error retrieving the file")
		return
	}
	defer file.Close()

	// read first 512 bytes to detect true content-type
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		h.writeError(w, http.StatusInternalServerError, "error reading file content")
		return
	}

	// rewind file pointer back to the beginning
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "error resetting file pointer")
		return
	}

	// detect content type
	contentType := http.DetectContentType(buffer[:n])

	switch {
	case strings.HasPrefix(contentType, "text/plain"):
		h.handleText(w, r, file, fileHeader)
	default:
		h.writeError(w, http.StatusUnsupportedMediaType, fmt.Sprintf("unsupported media type: %s\n\nOnly text/plain is supported", contentType))
	}
}

func (h *UploadHandler) handleText(w http.ResponseWriter, r *http.Request, file multipart.File, fileHeader *multipart.FileHeader) {
	log := logger.GetLogger(r.Context())

	documentID := uuid.New()

	cmd := upload.UploadCommand{
		DocumentID:  documentID,
		FileName:    fileHeader.Filename,
		ContentType: fileHeader.Header.Get("Content-Type"),
		SizeBytes:   fileHeader.Size,
		Content:     file,
	}

	log.Debug("about to enter core system")
	if err := h.uploader.UploadDocument(r.Context(), cmd); err != nil {
		log.Error("failed to process document upload", "error", err)

		switch {
		case errors.Is(err, domain.ErrDocumentFileNameEmpty),
			errors.Is(err, domain.ErrDocumentContentType),
			errors.Is(err, domain.ErrUnsupportedContentType),
			errors.Is(err, domain.ErrDocumentSizeInvalid),
			errors.Is(err, domain.ErrDocumentSizeTooLarge):

			h.writeError(w, http.StatusBadRequest, err.Error())
			return

		default:
			log.Debug(err.Error())
			h.writeError(w, http.StatusInternalServerError, "an internal error occurred")
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	response := map[string]string{
		"message":   "Document uploaded successfully",
		"fileName":  cmd.FileName,
		"sizeBytes": fmt.Sprintf("%d", cmd.SizeBytes),
	}

	json.NewEncoder(w).Encode(response)
}

func (h *UploadHandler) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
