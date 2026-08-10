package handler

import (
	"encoding/json"
	"errors"
	"fmt"
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

	page := httpx.ParsePageParams(r)

	result, err := h.finalize.GetSeatingPreview(r.Context(), userID, eventID, page.Page, page.PageSize, page.Q)
	if err != nil {
		h.writeSeatingErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewPaginatedSeatingPreviewListDTO(result.Items, result.Total, result.Page, result.PageSize))
}

func (h *EventJobsHandler) ExportSeatingPreview(w http.ResponseWriter, r *http.Request) {
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

	q := httpx.ParsePageParams(r).Q
	data, err := h.finalize.ExportSeatingPreview(r.Context(), userID, eventID, q)
	if err != nil {
		h.writeSeatingErr(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="seating-preview-%s.xlsx"`, eventID))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
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

type reassignDraftItemRequest struct {
	SeatID string `json:"seat_id"`
}

func (h *EventJobsHandler) ReassignDraftItem(w http.ResponseWriter, r *http.Request) {
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

	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid draft item ID")
		return
	}

	var body reassignDraftItemRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	seatID, err := uuid.Parse(body.SeatID)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid seat ID")
		return
	}

	if err := h.finalize.ReassignDraftItem(r.Context(), userID, eventID, itemID, seatID); err != nil {
		h.writeSeatingErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, map[string]string{"status": "updated"})
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
	case errors.Is(err, service.ErrNoOpenDraft):
		httpx.Fail(w, http.StatusNotFound, "NO_OPEN_DRAFT", err.Error())
	case errors.Is(err, service.ErrDraftItemNotFound):
		httpx.Fail(w, http.StatusNotFound, "DRAFT_ITEM_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrSeatNotFound):
		httpx.Fail(w, http.StatusNotFound, "SEAT_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrSeatNotAvailable):
		httpx.Fail(w, http.StatusConflict, "SEAT_NOT_AVAILABLE", err.Error())
	case errors.Is(err, service.ErrCategoryMismatch):
		httpx.Fail(w, http.StatusBadRequest, "CATEGORY_MISMATCH", err.Error())
	default:
		httpx.Fail(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
	}
}
