package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nnieru/mini-evvy/internal/dto"
	"github.com/nnieru/mini-evvy/internal/httpx"
	"github.com/nnieru/mini-evvy/internal/middleware"
	"github.com/nnieru/mini-evvy/internal/service"
)

type EventJobsHandler struct {
	finalize *service.FinalizeService
}

func NewEventJobsHandler(finalize *service.FinalizeService) *EventJobsHandler {
	return &EventJobsHandler{finalize: finalize}
}

type jobResponse struct {
	JobID string `json:"job_id"`
}

func (h *EventJobsHandler) FinalizeSeating(w http.ResponseWriter, r *http.Request) {
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

	job, err := h.finalize.RequestFinalize(r.Context(), userID, eventID)
	if err != nil {
		h.writeFinalizeErr(w, err)
		return
	}

	httpx.OK(w, http.StatusAccepted, jobResponse{JobID: job.ID.String()})
}

func (h *EventJobsHandler) GetSeatingPreview(w http.ResponseWriter, r *http.Request) {
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

	rows, err := h.finalize.GetSeatingPreview(r.Context(), userID, eventID)
	if err != nil {
		h.writeSeatingErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewSeatingPreviewListDTO(rows))
}

func (h *EventJobsHandler) ApproveSeating(w http.ResponseWriter, r *http.Request) {
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

	if err := h.finalize.ApproveSeating(r.Context(), userID, eventID); err != nil {
		h.writeSeatingErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (h *EventJobsHandler) RejectSeating(w http.ResponseWriter, r *http.Request) {
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

	if err := h.finalize.RejectSeating(r.Context(), userID, eventID); err != nil {
		h.writeSeatingErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, map[string]string{"status": "open"})
}

func (h *EventJobsHandler) writeFinalizeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrEventNotFound):
		httpx.Fail(w, http.StatusNotFound, "EVENT_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrForbidden):
		httpx.Fail(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, service.ErrFinalizeInProgress):
		httpx.Fail(w, http.StatusConflict, "FINALIZE_IN_PROGRESS", err.Error())
	case errors.Is(err, service.ErrSeatingNotOpen):
		httpx.Fail(w, http.StatusConflict, "SEATING_NOT_OPEN", err.Error())
	default:
		httpx.Fail(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
	}
}

func (h *EventJobsHandler) writeSeatingErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrEventNotFound):
		httpx.Fail(w, http.StatusNotFound, "EVENT_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrForbidden):
		httpx.Fail(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, service.ErrSeatingNotPreview):
		httpx.Fail(w, http.StatusNotFound, "SEATING_PREVIEW_NOT_AVAILABLE", err.Error())
	default:
		httpx.Fail(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
	}
}
