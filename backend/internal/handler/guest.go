package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nnieru/mini-evvy/internal/dto"
	"github.com/nnieru/mini-evvy/internal/httpx"
	"github.com/nnieru/mini-evvy/internal/middleware"
	"github.com/nnieru/mini-evvy/internal/service"
)

type GuestHandler struct {
	guests *service.GuestService
}

func NewGuestHandler(guests *service.GuestService) *GuestHandler {
	return &GuestHandler{guests: guests}
}

type createGuestRequest struct {
	Name        string  `json:"name"`
	Email       string  `json:"email"`
	CategoryID  string  `json:"category_id"`
	TicketCount int     `json:"ticket_count"`
	PaidDate    *string `json:"paid_date"`
}

type updateGuestRequest struct {
	Name        string  `json:"name"`
	Email       string  `json:"email"`
	CategoryID  string  `json:"category_id"`
	TicketCount int     `json:"ticket_count"`
	PaidDate    *string `json:"paid_date"`
}

func parsePaidDate(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (h *GuestHandler) Create(w http.ResponseWriter, r *http.Request) {
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

	var req createGuestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid category_id")
		return
	}

	paidDate, err := parsePaidDate(req.PaidDate)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "paid_date must be RFC3339")
		return
	}

	guest, err := h.guests.Create(r.Context(), userID, eventID, service.CreateGuestInput{
		Name:        req.Name,
		Email:       req.Email,
		CategoryID:  categoryID,
		TicketCount: req.TicketCount,
		PaidDate:    paidDate,
	})
	if err != nil {
		h.writeGuestErr(w, err)
		return
	}

	httpx.OK(w, http.StatusCreated, dto.NewGuestResponseDTO(guest))
}

func (h *GuestHandler) List(w http.ResponseWriter, r *http.Request) {
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

	list, err := h.guests.ListByEventPaged(r.Context(), userID, eventID, page.Page, page.PageSize, page.Q)
	if err != nil {
		h.writeGuestErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewPaginatedGuestListDTO(list.Items, list.Total, list.Page, list.PageSize))
}

func (h *GuestHandler) UnbookedCount(w http.ResponseWriter, r *http.Request) {
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

	total, err := h.guests.CountUnbookedByEvent(r.Context(), userID, eventID)
	if err != nil {
		h.writeGuestErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewUnbookedGuestCountDTO(total))
}

func (h *GuestHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	guestID, err := uuid.Parse(chi.URLParam(r, "guestId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid guest ID")
		return
	}

	guest, err := h.guests.Get(r.Context(), userID, guestID)
	if err != nil {
		h.writeGuestErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewGuestResponseDTO(guest))
}

const maxGuestImportBytes = 5 << 20 // 5 MiB

func (h *GuestHandler) Import(w http.ResponseWriter, r *http.Request) {
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

	if err := r.ParseMultipartForm(maxGuestImportBytes); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "file field required")
		return
	}
	defer file.Close()

	if header.Size > maxGuestImportBytes {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "file too large")
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "failed to read file")
		return
	}

	result, err := h.guests.Import(r.Context(), userID, eventID, data)
	if err != nil {
		h.writeGuestErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewGuestImportResultDTO(result))
}

func (h *GuestHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	guestID, err := uuid.Parse(chi.URLParam(r, "guestId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid guest ID")
		return
	}

	var req updateGuestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid category_id")
		return
	}

	paidDate, err := parsePaidDate(req.PaidDate)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "paid_date must be RFC3339")
		return
	}

	guest, err := h.guests.Update(r.Context(), userID, guestID, service.UpdateGuestInput{
		Name:        req.Name,
		Email:       req.Email,
		CategoryID:  categoryID,
		TicketCount: req.TicketCount,
		PaidDate:    paidDate,
	})
	if err != nil {
		h.writeGuestErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewGuestResponseDTO(guest))
}

func (h *GuestHandler) writeGuestErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrGuestNotFound):
		httpx.Fail(w, http.StatusNotFound, "GUEST_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrEventNotFound):
		httpx.Fail(w, http.StatusNotFound, "EVENT_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrCategoryNotFound):
		httpx.Fail(w, http.StatusNotFound, "CATEGORY_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrForbidden):
		httpx.Fail(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, service.ErrSeatingLocked):
		httpx.Fail(w, http.StatusConflict, "SEATING_LOCKED", err.Error())
	case errors.Is(err, service.ErrInvalidGuestImportFile):
		httpx.Fail(w, http.StatusBadRequest, "INVALID_IMPORT_FILE", err.Error())
	default:
		httpx.Fail(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
	}
}
