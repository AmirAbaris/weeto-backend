package auth

import (
	"errors"
	"net/http"

	"github.com/AmirAbaris/weeto-backend/internal/handler/httputil"
	authsvc "github.com/AmirAbaris/weeto-backend/internal/service/auth"
)

type Handler struct {
	svc *authsvc.Service
}

func NewHandler(svc *authsvc.Service) *Handler {
	return &Handler{svc: svc}
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pair, err := h.svc.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, toTokenResponse(pair))
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pair, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, toTokenResponse(pair))
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pair, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, toTokenResponse(pair))
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toTokenResponse(tokens authsvc.AuthTokens) tokenResponse {
	return tokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		TokenType:    "Bearer",
	}
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, authsvc.ErrEmailAlreadyExists):
		httputil.WriteError(w, http.StatusConflict, err.Error())
	case errors.Is(err, authsvc.ErrInvalidCredentials):
		httputil.WriteError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, authsvc.ErrWeakPassword), errors.Is(err, authsvc.ErrInvalidEmail):
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, authsvc.ErrInvalidToken):
		httputil.WriteError(w, http.StatusUnauthorized, err.Error())
	default:
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
