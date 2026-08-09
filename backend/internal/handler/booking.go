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

type BookingHandler struct {
	bookings *service.BookingService
	jobs     *service.JobQueryService
}

func NewBookingHandler(bookings *service.BookingService, jobs *service.JobQueryService) *BookingHandler {
	return &BookingHandler{bookings: bookings, jobs: jobs}
}

type createBookingRequest struct {
	GuestID    string  `json:"guest_id"`
	SeatID     string  `json:"seat_id"`
	Source     string  `json:"source"`
	Notes      *string `json:"notes"`
	Name       string  `json:"name"`
	Email      string  `json:"email"`
	CategoryID string  `json:"category_id"`
}

type updateBookingRequest struct {
	Status string  `json:"status"`
	Notes  *string `json:"notes"`
}

type createBookingBatchItemRequest struct {
	SeatID string `json:"seat_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}

type createBookingBatchRequest struct {
	Notes *string                       `json:"notes"`
	Items []createBookingBatchItemRequest `json:"items"`
}

func (h *BookingHandler) Create(w http.ResponseWriter, r *http.Request) {
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

	var req createBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	if req.SeatID == "" {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "seat_id is required")
		return
	}
	seatID, err := uuid.Parse(req.SeatID)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid seat_id")
		return
	}

	input := service.CreateBookingInput{
		SeatID: seatID,
		Source: model.BookingSource(req.Source),
		Notes:  req.Notes,
		Name:   req.Name,
		Email:  req.Email,
	}

	hasGuestID := req.GuestID != ""
	hasInlineGuest := req.Name != "" || req.Email != "" || req.CategoryID != ""
	if hasGuestID && hasInlineGuest {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "provide either guest_id or name, email, and category_id")
		return
	}
	if !hasGuestID && !hasInlineGuest {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "provide either guest_id or name, email, and category_id")
		return
	}

	if hasGuestID {
		guestID, err := uuid.Parse(req.GuestID)
		if err != nil {
			httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid guest_id")
			return
		}
		input.GuestID = &guestID
	} else {
		if req.Name == "" || req.Email == "" || req.CategoryID == "" {
			httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "name, email, and category_id are required")
			return
		}
		categoryID, err := uuid.Parse(req.CategoryID)
		if err != nil {
			httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid category_id")
			return
		}
		input.CategoryID = categoryID
	}

	booking, err := h.bookings.Create(r.Context(), userID, eventID, input)
	if err != nil {
		h.writeBookingErr(w, err)
		return
	}

	httpx.OK(w, http.StatusCreated, dto.NewBookingResponseDTO(booking))
}

func (h *BookingHandler) CreateBatch(w http.ResponseWriter, r *http.Request) {
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

	var req createBookingBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if len(req.Items) == 0 {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "items is required")
		return
	}

	items := make([]service.CreateBookingBatchItemInput, 0, len(req.Items))
	for _, item := range req.Items {
		if item.SeatID == "" {
			httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "seat_id is required for each item")
			return
		}
		seatID, err := uuid.Parse(item.SeatID)
		if err != nil {
			httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid seat_id")
			return
		}
		if item.Name == "" || item.Email == "" {
			httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "name and email are required for each item")
			return
		}
		items = append(items, service.CreateBookingBatchItemInput{
			SeatID: seatID,
			Name:   item.Name,
			Email:  item.Email,
		})
	}

	bookings, err := h.bookings.CreateBatch(r.Context(), userID, eventID, service.CreateBookingBatchInput{
		Notes: req.Notes,
		Items: items,
	})
	if err != nil {
		h.writeBookingErr(w, err)
		return
	}

	httpx.OK(w, http.StatusCreated, dto.NewBookingListDTO(bookings))
}

func (h *BookingHandler) List(w http.ResponseWriter, r *http.Request) {
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
	paymentStatus := r.URL.Query().Get("payment_status")

	result, err := h.bookings.ListByEventPaged(r.Context(), userID, eventID, service.ListBookingsFilter{
		PaymentStatus: paymentStatus,
		Q:             page.Q,
		Page:          page.Page,
		PageSize:      page.PageSize,
	})
	if err != nil {
		h.writeBookingErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewPaginatedBookingListDTO(result.Items, result.Total, result.Page, result.PageSize))
}

func (h *BookingHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	bookingID, err := uuid.Parse(chi.URLParam(r, "bookingId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid booking ID")
		return
	}

	booking, err := h.bookings.Get(r.Context(), userID, bookingID)
	if err != nil {
		h.writeBookingErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewBookingResponseDTO(booking))
}

func (h *BookingHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	bookingID, err := uuid.Parse(chi.URLParam(r, "bookingId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid booking ID")
		return
	}

	var req updateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Status == "" {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "status is required")
		return
	}

	booking, err := h.bookings.Update(r.Context(), userID, bookingID, service.UpdateBookingInput{
		Status: model.BookingStatus(req.Status),
		Notes:  req.Notes,
	})
	if err != nil {
		h.writeBookingErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewBookingResponseDTO(booking))
}

func (h *BookingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	bookingID, err := uuid.Parse(chi.URLParam(r, "bookingId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid booking ID")
		return
	}

	if err := h.bookings.Delete(r.Context(), userID, bookingID); err != nil {
		h.writeBookingErr(w, err)
		return
	}

	httpx.OK(w, http.StatusNoContent, nil)
}

func (h *BookingHandler) ResendInvitation(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	bookingID, err := uuid.Parse(chi.URLParam(r, "bookingId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid booking ID")
		return
	}

	job, err := h.jobs.ResendInvitation(r.Context(), userID, bookingID)
	if err != nil {
		h.writeResendErr(w, err)
		return
	}

	httpx.OK(w, http.StatusAccepted, jobResponse{JobID: job.ID.String()})
}

func (h *BookingHandler) writeResendErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrBookingNotFound):
		httpx.Fail(w, http.StatusNotFound, "BOOKING_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrEventNotFound):
		httpx.Fail(w, http.StatusNotFound, "EVENT_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrGuestNotFound):
		httpx.Fail(w, http.StatusNotFound, "GUEST_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrGuestEmailMissing):
		httpx.Fail(w, http.StatusBadRequest, "GUEST_EMAIL_MISSING", err.Error())
	case errors.Is(err, service.ErrBookingNotResendable):
		httpx.Fail(w, http.StatusBadRequest, "BOOKING_NOT_RESENDABLE", err.Error())
	case errors.Is(err, service.ErrForbidden):
		httpx.Fail(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	default:
		httpx.Fail(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
	}
}

func (h *BookingHandler) writeBookingErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrBookingNotFound):
		httpx.Fail(w, http.StatusNotFound, "BOOKING_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrEventNotFound):
		httpx.Fail(w, http.StatusNotFound, "EVENT_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrGuestNotFound):
		httpx.Fail(w, http.StatusNotFound, "GUEST_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrSeatNotFound):
		httpx.Fail(w, http.StatusNotFound, "SEAT_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrSeatNotAvailable):
		httpx.Fail(w, http.StatusConflict, "SEAT_NOT_AVAILABLE", err.Error())
	case errors.Is(err, service.ErrCategoryMismatch):
		httpx.Fail(w, http.StatusBadRequest, "CATEGORY_MISMATCH", err.Error())
	case errors.Is(err, service.ErrCategoryNotFound):
		httpx.Fail(w, http.StatusNotFound, "CATEGORY_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrInvalidBookingRequest):
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, service.ErrSeatsDifferentCategory):
		httpx.Fail(w, http.StatusBadRequest, "SEATS_DIFFERENT_CATEGORY", err.Error())
	case errors.Is(err, service.ErrSeatingLocked):
		httpx.Fail(w, http.StatusConflict, "SEATING_LOCKED", err.Error())
	case errors.Is(err, service.ErrInvalidBookingStatus):
		httpx.Fail(w, http.StatusBadRequest, "INVALID_BOOKING_STATUS", err.Error())
	case errors.Is(err, service.ErrInvalidBookingSource):
		httpx.Fail(w, http.StatusBadRequest, "INVALID_BOOKING_SOURCE", err.Error())
	case errors.Is(err, service.ErrInvalidStatusTransition):
		httpx.Fail(w, http.StatusBadRequest, "INVALID_STATUS_TRANSITION", err.Error())
	case errors.Is(err, service.ErrForbidden):
		httpx.Fail(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	default:
		httpx.Fail(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
	}
}
