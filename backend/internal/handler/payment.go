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

type PaymentHandler struct {
	payments *service.PaymentService
}

func NewPaymentHandler(payments *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{payments: payments}
}

type createPaymentRequest struct {
	Amount     string  `json:"amount"`
	Currency   string  `json:"currency"`
	Method     *string `json:"method"`
	GatewayRef *string `json:"gateway_ref"`
	Status     string  `json:"status"`
}

func (h *PaymentHandler) Create(w http.ResponseWriter, r *http.Request) {
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

	var req createPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Amount == "" {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "amount is required")
		return
	}

	payment, err := h.payments.Create(r.Context(), userID, bookingID, service.CreatePaymentInput{
		Amount:     req.Amount,
		Currency:   req.Currency,
		Method:     req.Method,
		GatewayRef: req.GatewayRef,
		Status:     model.PaymentStatus(req.Status),
	})
	if err != nil {
		h.writePaymentErr(w, err)
		return
	}

	httpx.OK(w, http.StatusCreated, dto.NewPaymentResponseDTO(payment))
}

func (h *PaymentHandler) List(w http.ResponseWriter, r *http.Request) {
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

	list, err := h.payments.ListByBooking(r.Context(), userID, bookingID)
	if err != nil {
		h.writePaymentErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewPaymentListDTO(list))
}

func (h *PaymentHandler) writePaymentErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrBookingNotFound):
		httpx.Fail(w, http.StatusNotFound, "BOOKING_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrEventNotFound):
		httpx.Fail(w, http.StatusNotFound, "EVENT_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrGuestNotFound):
		httpx.Fail(w, http.StatusNotFound, "GUEST_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrForbidden):
		httpx.Fail(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, service.ErrBookingNotPayable):
		httpx.Fail(w, http.StatusConflict, "BOOKING_NOT_PAYABLE", err.Error())
	case errors.Is(err, service.ErrInvalidPaymentAmount):
		httpx.Fail(w, http.StatusBadRequest, "INVALID_PAYMENT_AMOUNT", err.Error())
	case errors.Is(err, service.ErrInvalidCurrency):
		httpx.Fail(w, http.StatusBadRequest, "INVALID_CURRENCY", err.Error())
	case errors.Is(err, service.ErrInvalidPaymentStatus):
		httpx.Fail(w, http.StatusBadRequest, "INVALID_PAYMENT_STATUS", err.Error())
	default:
		httpx.Fail(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
	}
}
