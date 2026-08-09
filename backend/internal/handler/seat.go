package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nnieru/mini-evvy/internal/dto"
	"github.com/nnieru/mini-evvy/internal/httpx"
	"github.com/nnieru/mini-evvy/internal/middleware"
	"github.com/nnieru/mini-evvy/internal/model"
	"github.com/nnieru/mini-evvy/internal/service"
)

type SeatHandler struct {
	seats *service.SeatService
}

func NewSeatHandler(seats *service.SeatService) *SeatHandler {
	return &SeatHandler{seats: seats}
}

type createSeatItemRequest struct {
	Code        string  `json:"code"`
	CategoryID  string  `json:"category_id"`
	Section     *string `json:"section"`
	Row         *int    `json:"row"`
	Col         *int    `json:"col"`
	Description *string `json:"description"`
	Status      string  `json:"status"`
}

type createSeatsRequest struct {
	Seats []createSeatItemRequest `json:"seats"`
}

type updateSeatRequest struct {
	Code        string  `json:"code"`
	CategoryID  string  `json:"category_id"`
	Section     *string `json:"section"`
	Row         *int    `json:"row"`
	Col         *int    `json:"col"`
	Description *string `json:"description"`
	Status      string  `json:"status"`
}

func (h *SeatHandler) Create(w http.ResponseWriter, r *http.Request) {
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

	var req createSeatsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if len(req.Seats) == 0 {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "seats array is required")
		return
	}

	inputs := make([]service.CreateSeatInput, 0, len(req.Seats))
	for _, item := range req.Seats {
		categoryID, err := uuid.Parse(item.CategoryID)
		if err != nil {
			httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid category_id")
			return
		}
		inputs = append(inputs, service.CreateSeatInput{
			Code:        item.Code,
			CategoryID:  categoryID,
			Section:     item.Section,
			Row:         item.Row,
			Col:         item.Col,
			Description: item.Description,
			Status:      model.SeatStatus(item.Status),
		})
	}

	created, err := h.seats.CreateBatch(r.Context(), userID, eventID, inputs)
	if err != nil {
		h.writeSeatErr(w, err)
		return
	}

	httpx.OK(w, http.StatusCreated, dto.NewSeatListDTO(created))
}

func (h *SeatHandler) List(w http.ResponseWriter, r *http.Request) {
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

	var statusFilter *model.SeatStatus
	if statusParam := r.URL.Query().Get("status"); statusParam != "" {
		s := model.SeatStatus(statusParam)
		statusFilter = &s
	}

	var categoryFilter *uuid.UUID
	if categoryParam := r.URL.Query().Get("category_id"); categoryParam != "" {
		categoryID, err := uuid.Parse(categoryParam)
		if err != nil {
			httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid category_id")
			return
		}
		categoryFilter = &categoryID
	}

	page := httpx.ParsePageParams(r)

	list, err := h.seats.ListByEventPaged(r.Context(), userID, eventID, statusFilter, categoryFilter, page.Page, page.PageSize, page.Q)
	if err != nil {
		h.writeSeatErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewPaginatedSeatListDTO(list.Items, list.Total, list.Page, list.PageSize))
}

func (h *SeatHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	seatID, err := uuid.Parse(chi.URLParam(r, "seatId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid seat ID")
		return
	}

	seat, err := h.seats.Get(r.Context(), userID, seatID)
	if err != nil {
		h.writeSeatErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewSeatResponseDTO(seat))
}

func (h *SeatHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	seatID, err := uuid.Parse(chi.URLParam(r, "seatId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid seat ID")
		return
	}

	var req updateSeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid category_id")
		return
	}

	seat, err := h.seats.Update(r.Context(), userID, seatID, service.UpdateSeatInput{
		Code:        req.Code,
		CategoryID:  categoryID,
		Section:     req.Section,
		Row:         req.Row,
		Col:         req.Col,
		Description: req.Description,
		Status:      model.SeatStatus(req.Status),
	})
	if err != nil {
		h.writeSeatErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewSeatResponseDTO(seat))
}

func (h *SeatHandler) writeSeatErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrSeatNotFound):
		httpx.Fail(w, http.StatusNotFound, "SEAT_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrEventNotFound):
		httpx.Fail(w, http.StatusNotFound, "EVENT_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrCategoryNotFound):
		httpx.Fail(w, http.StatusNotFound, "CATEGORY_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrForbidden):
		httpx.Fail(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, service.ErrInvalidSeatStatus):
		httpx.Fail(w, http.StatusBadRequest, "INVALID_SEAT_STATUS", err.Error())
	default:
		httpx.Fail(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
	}
}
