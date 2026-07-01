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

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
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

	setRefreshTokenCookie(w, pair)

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

	setRefreshTokenCookie(w, pair)

	httputil.WriteJSON(w, http.StatusOK, toTokenResponse(pair))
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "missing or invalid refresh token")
		return
	}

	pair, err := h.svc.Refresh(r.Context(), cookie.Value)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	setRefreshTokenCookie(w, pair)

	httputil.WriteJSON(w, http.StatusOK, toTokenResponse(pair))
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "missing or invalid refresh token")
		return
	}

	if err := h.svc.Logout(r.Context(), cookie.Value); err != nil {
		writeServiceError(w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		MaxAge:   0,
	})

	w.WriteHeader(http.StatusNoContent)
}

func setRefreshTokenCookie(w http.ResponseWriter, tokens authsvc.AuthTokens) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens.RefreshToken,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		MaxAge:   tokens.RefreshExpiresIn,
	})
}

func toTokenResponse(tokens authsvc.AuthTokens) tokenResponse {
	return tokenResponse{
		AccessToken: tokens.AccessToken,
		ExpiresIn:   tokens.ExpiresIn,
		TokenType:   "Bearer",
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
