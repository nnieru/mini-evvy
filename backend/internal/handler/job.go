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

type JobHandler struct {
	jobs *service.JobQueryService
}

func NewJobHandler(jobs *service.JobQueryService) *JobHandler {
	return &JobHandler{jobs: jobs}
}

func (h *JobHandler) ListByEvent(w http.ResponseWriter, r *http.Request) {
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

	list, err := h.jobs.ListByEvent(r.Context(), userID, eventID)
	if err != nil {
		h.writeJobErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewJobListDTO(list))
}

func (h *JobHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(r, "jobId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid job ID")
		return
	}

	job, err := h.jobs.Get(r.Context(), userID, jobID)
	if err != nil {
		h.writeJobErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewJobResponseDTO(job))
}

func (h *JobHandler) writeJobErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrEventNotFound):
		httpx.Fail(w, http.StatusNotFound, "EVENT_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrJobNotFound):
		httpx.Fail(w, http.StatusNotFound, "JOB_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrForbidden):
		httpx.Fail(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	default:
		httpx.Fail(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
	}
}
