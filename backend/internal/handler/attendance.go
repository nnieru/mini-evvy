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
	"github.com/nnieru/mini-evvy/internal/model"
	"github.com/nnieru/mini-evvy/internal/service"
)

type AttendanceHandler struct {
	attendance *service.AttendanceService
}

func NewAttendanceHandler(attendance *service.AttendanceService) *AttendanceHandler {
	return &AttendanceHandler{attendance: attendance}
}

type createAttendanceRequest struct {
	Barcode string  `json:"barcode"`
	GuestID string  `json:"guest_id"`
	SeatID  string  `json:"seat_id"`
	Message *string `json:"message"`
}

func (h *AttendanceHandler) Create(w http.ResponseWriter, r *http.Request) {
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

	var req createAttendanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	in := service.CreateAttendanceInput{
		Barcode: strings.TrimSpace(req.Barcode),
		Message: req.Message,
	}

	if in.Barcode == "" {
		if req.GuestID == "" || req.SeatID == "" {
			httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "guest_id and seat_id or barcode required")
			return
		}
		guestID, err := uuid.Parse(req.GuestID)
		if err != nil {
			httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid guest_id")
			return
		}
		seatID, err := uuid.Parse(req.SeatID)
		if err != nil {
			httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid seat_id")
			return
		}
		in.GuestID = guestID
		in.SeatID = seatID
	}

	log, err := h.attendance.Create(r.Context(), userID, eventID, in)
	if err != nil {
		h.writeAttendanceErr(w, err)
		return
	}

	httpx.OK(w, http.StatusCreated, dto.NewAttendanceResponseDTO(log))
}

func (h *AttendanceHandler) List(w http.ResponseWriter, r *http.Request) {
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

	list, err := h.attendance.ListByEventPaged(r.Context(), userID, eventID, page.Page, page.PageSize, page.Q)
	if err != nil {
		h.writeAttendanceErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewPaginatedAttendanceListDTO(list.Items, list.Total, list.Page, list.PageSize))
}

type updateAttendanceRequest struct {
	Status  string  `json:"status"`
	Message *string `json:"message"`
}

func (h *AttendanceHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	attendanceID, err := uuid.Parse(chi.URLParam(r, "attendanceId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid attendance ID")
		return
	}

	log, err := h.attendance.Get(r.Context(), userID, attendanceID)
	if err != nil {
		h.writeAttendanceErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewAttendanceResponseDTO(log))
}

func (h *AttendanceHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	attendanceID, err := uuid.Parse(chi.URLParam(r, "attendanceId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid attendance ID")
		return
	}

	var req updateAttendanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Status == "" {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "status is required")
		return
	}

	log, err := h.attendance.Update(r.Context(), userID, attendanceID, service.UpdateAttendanceInput{
		Status:  model.AttendanceStatus(req.Status),
		Message: req.Message,
	})
	if err != nil {
		h.writeAttendanceErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewAttendanceResponseDTO(log))
}

func (h *AttendanceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	attendanceID, err := uuid.Parse(chi.URLParam(r, "attendanceId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid attendance ID")
		return
	}

	if err := h.attendance.Delete(r.Context(), userID, attendanceID); err != nil {
		h.writeAttendanceErr(w, err)
		return
	}

	httpx.OK(w, http.StatusNoContent, nil)
}

func (h *AttendanceHandler) writeAttendanceErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrEventNotFound):
		httpx.Fail(w, http.StatusNotFound, "EVENT_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrGuestNotFound):
		httpx.Fail(w, http.StatusNotFound, "GUEST_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrSeatNotFound):
		httpx.Fail(w, http.StatusNotFound, "SEAT_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrPaidBookingNotFound):
		httpx.Fail(w, http.StatusNotFound, "PAID_BOOKING_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrInvalidBarcode):
		httpx.Fail(w, http.StatusBadRequest, "INVALID_BARCODE", err.Error())
	case errors.Is(err, service.ErrAlreadyCheckedIn):
		httpx.Fail(w, http.StatusConflict, "ALREADY_CHECKED_IN", err.Error())
	case errors.Is(err, service.ErrAttendanceNotFound):
		httpx.Fail(w, http.StatusNotFound, "ATTENDANCE_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrInvalidAttendanceStatus):
		httpx.Fail(w, http.StatusBadRequest, "INVALID_ATTENDANCE_STATUS", err.Error())
	case errors.Is(err, service.ErrInvalidAttendanceTransition):
		httpx.Fail(w, http.StatusBadRequest, "INVALID_ATTENDANCE_TRANSITION", err.Error())
	case errors.Is(err, service.ErrForbidden):
		httpx.Fail(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	default:
		httpx.Fail(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
	}
}
