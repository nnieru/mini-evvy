package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nnieru/mini-evvy/internal/dto"
	"github.com/nnieru/mini-evvy/internal/httpx"
	"github.com/nnieru/mini-evvy/internal/middleware"
	"github.com/nnieru/mini-evvy/internal/service"
)

type EventImportHandler struct {
	imports *service.EventImportService
}

func NewEventImportHandler(imports *service.EventImportService) *EventImportHandler {
	return &EventImportHandler{imports: imports}
}

func (h *EventImportHandler) ImportConfig(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	targetEventID, err := uuid.Parse(chi.URLParam(r, "eventId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid event ID")
		return
	}

	var req dto.ImportEventConfigRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	sourceEventID, err := uuid.Parse(req.SourceEventID)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "source_event_id must be a valid UUID")
		return
	}

	result, err := h.imports.ImportConfig(r.Context(), userID, targetEventID, service.ImportConfigInput{
		SourceEventID:        sourceEventID,
		IncludeCategories:    req.IncludeCategories,
		IncludeSeats:         req.IncludeSeats,
		IncludeEmailTemplate: req.IncludeEmailTemplate,
	})
	if err != nil {
		h.writeErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewImportEventConfigResultDTO(result))
}

func (h *EventImportHandler) writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrEventNotFound):
		httpx.Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrForbidden):
		httpx.Fail(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, service.ErrTargetNotEmpty):
		httpx.Fail(w, http.StatusConflict, "TARGET_NOT_EMPTY", err.Error())
	case errors.Is(err, service.ErrValidation):
		msg := strings.TrimPrefix(err.Error(), "validation error: ")
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", msg)
	default:
		httpx.Fail(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
