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

type OrganizationHandler struct {
	org *service.OrganizationService
}

func NewOrganizationHandler(org *service.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{
		org,
	}
}

type CreateOrgRequest struct {
	Name string `json:"name"`
}

func (h *OrganizationHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	var req CreateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	if req.Name == "" {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
	}

	org, err := h.org.Create(r.Context(), userID, req.Name)
	if err != nil {
		h.writeOrgErr(w, err)
		return
	}

	httpx.OK(w, http.StatusCreated, dto.NewOrganizationResponseDTO(org))
}

func (h *OrganizationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	orgs, err := h.org.ListMine(r.Context(), userID)
	if err != nil {
		h.writeOrgErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NeworganizationListDTO(orgs))
}

func (h *OrganizationHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	org, err := h.org.Get(r.Context(), userID, orgID)
	if err != nil {
		h.writeOrgErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewOrganizationResponseDTO(org))
}

func (h *OrganizationHandler) GetMyRole(w http.ResponseWriter, r *http.Request) {
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

	role, err := h.org.GetMyRole(r.Context(), userID, orgID)
	if err != nil {
		h.writeOrgErr(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.MyRoleResponseDTO{Role: role})
}

func (h *OrganizationHandler) writeOrgErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrOrgNotFound):
		httpx.Fail(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrForbidden):
		httpx.Fail(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	default:
		// temporary: surface real error while learning; replace with slog later
		httpx.Fail(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
	}
}

type addMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"` // "admin" | "member"
}

func (h *OrganizationHandler) AddMember(w http.ResponseWriter, r *http.Request) {
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

	var req addMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	if req.Email == "" || req.Role == "" {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "email and role are required")
		return
	}

	member, err := h.org.AddMember(r.Context(), userID, orgID, req.Email, req.Role)

	if err != nil {
		h.writeMemberErr(w, err)
		return
	}

	httpx.OK(w, http.StatusCreated, dto.NewMemberResponseDTO(member))
}

func (h *OrganizationHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user")
		return
	}
	orgID, err := uuid.Parse(chi.URLParam(r, "orgId"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid org id")
		return
	}
	list, err := h.org.ListMembers(r.Context(), userID, orgID)
	if err != nil {
		h.writeOrgErr(w, err)
		return
	}
	httpx.OK(w, http.StatusOK, dto.NewMemberListDTO(list))
}

func (h *OrganizationHandler) writeMemberErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrUserNotFound):
		httpx.Fail(w, http.StatusNotFound, "USER_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrAlreadyMember):
		httpx.Fail(w, http.StatusConflict, "ALREADY_MEMBER", err.Error())
	case errors.Is(err, service.ErrInvalidRole):
		httpx.Fail(w, http.StatusBadRequest, "INVALID_ROLE", err.Error())
	}
}
