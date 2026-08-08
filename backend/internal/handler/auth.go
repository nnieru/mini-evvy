package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/nnieru/mini-evvy/internal/dto"
	"github.com/nnieru/mini-evvy/internal/httpx"
	"github.com/nnieru/mini-evvy/internal/middleware"
	"github.com/nnieru/mini-evvy/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type registerRequest struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	isPasswordConfirmed := strings.EqualFold(req.Password, req.ConfirmPassword)
	if req.Name == "" || req.Email == "" || req.Password == "" || req.ConfirmPassword == "" || !isPasswordConfirmed {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION ERROR", "invalid request body")
		return
	}

	result, err := h.auth.Register(r.Context(), req.Name, req.Email, req.Password)

	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	httpx.OK(w, http.StatusCreated, dto.NewAuthResponseDTO(result.User, result.Token))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		httpx.Fail(w, http.StatusBadRequest, "VALIDATION ERROR", "invalid request body")
		return
	}

	result, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewAuthResponseDTO(result.User, result.Token))
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user credentials")
		return
	}

	user, err := h.auth.Me(r.Context(), userID)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	httpx.OK(w, http.StatusOK, dto.NewUserResponseDTO(user))
}

func (h *AuthHandler) writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials):
		httpx.Fail(w, http.StatusConflict, "EMAIL_TAKEN", err.Error())
	case errors.Is(err, service.ErrInvalidCredentials):
		httpx.Fail(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", err.Error())
	case errors.Is(err, service.ErrUserDisabled):
		httpx.Fail(w, http.StatusForbidden, "USER_DISABLED", err.Error())
	case errors.Is(err, service.ErrUserPending):
		httpx.Fail(w, http.StatusForbidden, "USER_PENDING", err.Error())

	default:
		httpx.Fail(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "something went wrong")
	}
}
