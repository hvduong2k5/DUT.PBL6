package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/omamx/care-service/internal/domain"
	"github.com/omamx/care-service/internal/media"
)

type MediaHandler struct {
	service *media.Service
}

func NewMediaHandler(service *media.Service) *MediaHandler {
	return &MediaHandler{service: service}
}

type createUploadRequest struct {
	Purpose        domain.MediaPurpose `json:"purpose"`
	Filename       string              `json:"filename"`
	MIMEType       string              `json:"mime_type"`
	SizeBytes      int64               `json:"size_bytes"`
	ChecksumSHA256 string              `json:"checksum_sha256,omitempty"`
}

func (h *MediaHandler) CreateUploadSession(w http.ResponseWriter, r *http.Request) {
	actor, _ := AuthenticatedActor(r.Context())
	var request createUploadRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "The request body is invalid", nil)
		return
	}
	session, err := h.service.CreateUploadSession(r.Context(), media.CreateUploadInput{
		OwnerID: actor.ID, Purpose: request.Purpose, Filename: request.Filename,
		MIMEType: request.MIMEType, SizeBytes: request.SizeBytes, ChecksumSHA256: request.ChecksumSHA256,
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	status := http.StatusCreated
	if !session.Created {
		status = http.StatusOK
	}
	writeJSON(w, status, map[string]any{"data": map[string]any{
		"file_id":               session.File.ID,
		"purpose":               session.File.Purpose,
		"status":                session.File.Status,
		"upload_url":            session.UploadURL,
		"method":                session.Method,
		"required_headers":      session.Headers,
		"upload_url_expires_at": session.URLExpiresAt,
	}})
}

func (h *MediaHandler) Complete(w http.ResponseWriter, r *http.Request) {
	actor, _ := AuthenticatedActor(r.Context())
	fileID, err := uuid.Parse(chi.URLParam(r, "fileID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_FILE_ID", "The file ID is invalid", nil)
		return
	}
	file, err := h.service.Complete(r.Context(), actor.ID, fileID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": mediaDTO(file)})
}

func (h *MediaHandler) Access(w http.ResponseWriter, r *http.Request) {
	actor, _ := AuthenticatedActor(r.Context())
	fileID, err := uuid.Parse(chi.URLParam(r, "fileID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_FILE_ID", "The file ID is invalid", nil)
		return
	}
	accessURL, expiresAt, err := h.service.Access(r.Context(), actor.ID, fileID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{
		"file_id": fileID, "access_url": accessURL, "access_url_expires_at": expiresAt,
	}})
}

func (h *MediaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actor, _ := AuthenticatedActor(r.Context())
	fileID, err := uuid.Parse(chi.URLParam(r, "fileID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_FILE_ID", "The file ID is invalid", nil)
		return
	}
	if err := h.service.DeleteUnattached(r.Context(), actor.ID, fileID); err != nil {
		writeDomainError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func mediaDTO(file domain.MediaFile) map[string]any {
	return map[string]any{
		"file_id": file.ID, "purpose": file.Purpose, "filename": file.OriginalFilename,
		"mime_type": file.DetectedMIMEType, "size_bytes": file.ActualSize,
		"status": file.Status, "scan_status": file.ScanStatus,
		"created_at": file.CreatedAt, "uploaded_at": file.UploadedAt,
	}
}
