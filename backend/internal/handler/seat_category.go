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
	"github.com/nnieru/mini-evvy/internal/service"
)

type SeatCategoryHandler struct {
	categories *service.SeatCategoryService
}

func NewSeatCategoryHandler(categories *service.SeatCategoryService) *SeatCategoryHandler {
	return &SeatCategoryHandler{categories: categories}
}

type createCategoryRequest struct {
	Name     string   `json:"name"`
	Code     *string  `json:"code"`
	Price    float64  `json:"price"`
	Currency string   `json:"currency"`
}

type updateCategoryRequest struct {
	Name     string   `json:"name"`
	Code     *string  `json:"code"`
	Price    float64  `json:"price"`
	Currency string   `json:"currency"`
}

func (h *SeatCategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
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

	var req createCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Name == "" {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}

	category, err := h.categories.Create(r.Context(), userID, eventID, service.CreateCategoryInput{
		Name:     req.Name,
		Code:     req.Code,
		Price:    req.Price,
		Currency: req.Currency,
	})
	if err != nil {
		h.writeCategoryErr(w, err)
		return
	}

	httpx.OK(w, http.StatusCreated, dto.NewSeatCategoryResponseDTO(category))
}

func (h *SeatCategoryHandler) List(w http.ResponseWriter, r *http.Request) {
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

	list, err := h.categories.ListByEvent(r.Context(), userID, eventID)
	if err != nil {
		h.writeCategoryErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewSeatCategoryListDTO(list))
}

func (h *SeatCategoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	categoryID, err := uuid.Parse(chi.URLParam(r, "categoryId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid category ID")
		return
	}

	category, err := h.categories.Get(r.Context(), userID, categoryID)
	if err != nil {
		h.writeCategoryErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewSeatCategoryResponseDTO(category))
}

func (h *SeatCategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	categoryID, err := uuid.Parse(chi.URLParam(r, "categoryId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid category ID")
		return
	}

	var req updateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Name == "" {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}

	category, err := h.categories.Update(r.Context(), userID, categoryID, service.UpdateCategoryInput{
		Name:     req.Name,
		Code:     req.Code,
		Price:    req.Price,
		Currency: req.Currency,
	})
	if err != nil {
		h.writeCategoryErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewSeatCategoryResponseDTO(category))
}

func (h *SeatCategoryHandler) writeCategoryErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrCategoryNotFound):
		httpx.Fail(w, http.StatusNotFound, "CATEGORY_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrEventNotFound):
		httpx.Fail(w, http.StatusNotFound, "EVENT_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrForbidden):
		httpx.Fail(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	default:
		httpx.Fail(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
	}
}
