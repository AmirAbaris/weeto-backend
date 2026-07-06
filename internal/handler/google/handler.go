package google

import (
	"errors"
	"net/http"

	"github.com/AmirAbaris/weeto-backend/internal/config"
	"github.com/AmirAbaris/weeto-backend/internal/handler/httputil"
	"github.com/AmirAbaris/weeto-backend/internal/middleware"
	googlesvc "github.com/AmirAbaris/weeto-backend/internal/service/google"
)

type Handler struct {
	cfg *config.Config
	svc *googlesvc.Service
}

func NewHandler(cfg *config.Config, svc *googlesvc.Service) *Handler {
	return &Handler{cfg: cfg, svc: svc}
}

func (h *Handler) Connect(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	url, err := h.svc.ConnectURL(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	http.Redirect(w, r, url, http.StatusFound)
}

func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		http.Redirect(w, r, h.frontendRedirect("error", errParam), http.StatusFound)
		return
	}

	if _, err := h.svc.HandleCallback(r.Context(), code, state); err != nil {
		http.Redirect(w, r, h.frontendRedirect("error", "callback_failed"), http.StatusFound)
		return
	}

	http.Redirect(w, r, h.frontendRedirect("connected", ""), http.StatusFound)
}

func (h *Handler) Disconnect(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.svc.Disconnect(r.Context(), userID); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) frontendRedirect(status, detail string) string {
	base := h.cfg.FrontendURL + "/dashboard/settings"
	if detail != "" {
		return base + "?google=" + status + "&detail=" + detail
	}
	return base + "?google=" + status
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, googlesvc.ErrGoogleNotConfigured):
		httputil.WriteError(w, http.StatusServiceUnavailable, err.Error())
	case errors.Is(err, googlesvc.ErrInvalidOAuthState):
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
