package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nnieru/mini-evvy/internal/dto"
	"github.com/nnieru/mini-evvy/internal/httpx"
	"github.com/nnieru/mini-evvy/internal/mailer/invitation"
	"github.com/nnieru/mini-evvy/internal/middleware"
	"github.com/nnieru/mini-evvy/internal/service"
)

const maxBannerUploadBytes = 2 << 20 // 2 MiB

type EmailTemplateHandler struct {
	templates *service.EmailTemplateService
}

func NewEmailTemplateHandler(templates *service.EmailTemplateService) *EmailTemplateHandler {
	return &EmailTemplateHandler{templates: templates}
}

func (h *EmailTemplateHandler) GetInvitation(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "eventId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid event ID")
		return
	}

	view, err := h.templates.GetInvitation(r.Context(), userID, eventID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewInvitationEmailTemplateDTO(view))
}

func (h *EmailTemplateHandler) UpsertInvitation(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "eventId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid event ID")
		return
	}

	var req dto.InvitationEmailTemplateDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	view, err := h.templates.UpsertInvitation(r.Context(), userID, eventID, dto.InvitationConfigFromDTO(req))
	if err != nil {
		h.writeErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewInvitationEmailTemplateDTO(view))
}

func (h *EmailTemplateHandler) ResetInvitation(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "eventId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid event ID")
		return
	}

	if err := h.templates.ResetInvitation(r.Context(), userID, eventID); err != nil {
		h.writeErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *EmailTemplateHandler) PreviewInvitation(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "eventId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid event ID")
		return
	}

	var cfg *invitation.Config
	if r.ContentLength > 0 {
		var req dto.InvitationEmailTemplateDTO
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
			return
		}
		c := dto.InvitationConfigFromDTO(req)
		cfg = &c
	}

	preview, err := h.templates.PreviewInvitation(r.Context(), userID, eventID, cfg)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewInvitationEmailPreviewDTO(preview))
}

func (h *EmailTemplateHandler) TestSendInvitation(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "eventId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid event ID")
		return
	}

	var cfg *invitation.Config
	if r.ContentLength > 0 {
		var req dto.InvitationEmailTemplateDTO
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
			return
		}
		c := dto.InvitationConfigFromDTO(req)
		cfg = &c
	}

	if err := h.templates.TestSendInvitation(r.Context(), userID, eventID, cfg); err != nil {
		h.writeErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *EmailTemplateHandler) UploadBanner(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "eventId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid event ID")
		return
	}

	if err := r.ParseMultipartForm(maxBannerUploadBytes); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "file field required")
		return
	}
	defer file.Close()

	if header.Size > maxBannerUploadBytes {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "file too large (max 2 MiB)")
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "failed to read file")
		return
	}

	url, err := h.templates.UploadBanner(r.Context(), userID, eventID, header.Filename, data)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	httpx.OK(w, http.StatusCreated, dto.NewBannerUploadResponseDTO(url))
}

func (h *EmailTemplateHandler) writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrEventNotFound):
		httpx.Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrForbidden):
		httpx.Fail(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, service.ErrTestSendRateLimited):
		httpx.Fail(w, http.StatusTooManyRequests, "RATE_LIMITED", err.Error())
	case errors.Is(err, service.ErrValidation):
		msg := strings.TrimPrefix(err.Error(), "validation error: ")
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", msg)
	default:
		httpx.Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
