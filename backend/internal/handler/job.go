package handler

import (
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

	page := httpx.ParsePageParams(r)
	jobType := r.URL.Query().Get("type")
	status := r.URL.Query().Get("status")

	list, err := h.jobs.ListByEventPaged(
		r.Context(),
		userID,
		eventID,
		page.Page,
		page.PageSize,
		page.Q,
		jobType,
		status,
	)
	if err != nil {
		h.writeJobErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewPaginatedJobListDTO(list.Items, list.Total, list.Page, list.PageSize))
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

func (h *JobHandler) ExportInvitationEmails(w http.ResponseWriter, r *http.Request) {
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
	status := r.URL.Query().Get("status")
	data, err := h.jobs.ExportInvitationEmails(r.Context(), userID, eventID, q, status)
	if err != nil {
		h.writeJobErr(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="invitation-emails-%s.xlsx"`, eventID))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
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
