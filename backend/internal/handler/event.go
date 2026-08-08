package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nnieru/mini-evvy/internal/dto"
	"github.com/nnieru/mini-evvy/internal/httpx"
	"github.com/nnieru/mini-evvy/internal/middleware"
	"github.com/nnieru/mini-evvy/internal/model"
	"github.com/nnieru/mini-evvy/internal/service"
)

type EventHandler struct {
	events *service.EventService
}

func NewEventHandler(events *service.EventService) *EventHandler {
	return &EventHandler{events: events}
}

type createEventRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date"`
	StartTime   *string `json:"start_time"`
	EndTime     *string `json:"end_time"`
}

type updateEventRequest struct {
	Name        string  `json:"name"`
	Status      string  `json:"status"`
	Description *string `json:"description"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date"`
	StartTime   *string `json:"start_time"`
	EndTime     *string `json:"end_time"`
}

func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	orgID, err := uuid.Parse(chi.URLParam(r, "orgId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid organization ID")
		return
	}

	var req createEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Name == "" || req.StartDate == "" {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "name and start_date are required")
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "start_date must be YYYY-MM-DD")
		return
	}

	var endDate *time.Time
	if req.EndDate != nil && *req.EndDate != "" {
		parsed, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "end_date must be YYYY-MM-DD")
			return
		}
		endDate = &parsed
	}

	event, err := h.events.Create(r.Context(), userID, orgID, service.CreateEventInput{
		Name:        req.Name,
		Description: req.Description,
		StartDate:   startDate,
		EndDate:     endDate,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
	})
	if err != nil {
		h.writeEventErr(w, err)
		return
	}

	httpx.OK(w, http.StatusCreated, dto.NewEventResponseDTO(event))
}

func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	orgID, err := uuid.Parse(chi.URLParam(r, "orgId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid organization ID")
		return
	}

	list, err := h.events.ListByOrg(r.Context(), userID, orgID)
	if err != nil {
		h.writeEventErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewEventListDTO(list))
}

func (h *EventHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	list, err := h.events.ListMine(r.Context(), userID)
	if err != nil {
		h.writeEventErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewMyEventListDTO(list))
}

func (h *EventHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	event, err := h.events.Get(r.Context(), userID, eventID)
	if err != nil {
		h.writeEventErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewEventResponseDTO(event))
}

func (h *EventHandler) Update(w http.ResponseWriter, r *http.Request) {
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

	var req updateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Name == "" || req.StartDate == "" {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "name and start_date are required")
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "start_date must be YYYY-MM-DD")
		return
	}

	var endDate *time.Time
	if req.EndDate != nil && *req.EndDate != "" {
		parsed, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "end_date must be YYYY-MM-DD")
			return
		}
		endDate = &parsed
	}

	event, err := h.events.Update(r.Context(), userID, eventID, service.UpdateEventInput{
		Name:        req.Name,
		Status:      model.EventStatus(req.Status),
		Description: req.Description,
		StartDate:   startDate,
		EndDate:     endDate,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
	})
	if err != nil {
		h.writeEventErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewEventResponseDTO(event))
}

func (h *EventHandler) writeEventErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrEventNotFound):
		httpx.Fail(w, http.StatusNotFound, "EVENT_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrForbidden):
		httpx.Fail(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	default:
		httpx.Fail(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
	}
}
