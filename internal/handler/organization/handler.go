package organization

import (
	"errors"
	"net/http"

	"github.com/AmirAbaris/weeto-backend/internal/handler/httputil"
	"github.com/AmirAbaris/weeto-backend/internal/handler/orgresponse"
	"github.com/AmirAbaris/weeto-backend/internal/middleware"
	orgsvc "github.com/AmirAbaris/weeto-backend/internal/service/organization"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	svc *orgsvc.Service
}

func NewHandler(svc *orgsvc.Service) *Handler {
	return &Handler{svc: svc}
}

type createOrgRequest struct {
	Name    string  `json:"name"`
	Slug    string  `json:"slug"`
	LogoURL *string `json:"logo_url,omitempty"`
}

type updateOrgRequest struct {
	Name    string  `json:"name"`
	Slug    string  `json:"slug"`
	LogoURL *string `json:"logo_url,omitempty"`
}

type updateLogoRequest struct {
	LogoURL string `json:"logo_url"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createOrgRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	org, err := h.svc.CreateOrg(r.Context(), ownerID, req.Name, req.Slug, req.LogoURL)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	usage, err := h.svc.UsageForOrg(r.Context(), org)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, orgresponse.FromOrganization(org, usage))
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid organization id")
		return
	}

	org, err := h.svc.GetByIDWithUsage(r.Context(), id, ownerID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, orgresponse.FromWithUsage(org))
}

func (h *Handler) GetMine(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	org, err := h.svc.GetByOwnerWithUsage(r.Context(), ownerID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, orgresponse.FromWithUsage(org))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid organization id")
		return
	}

	var req updateOrgRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	org, err := h.svc.UpdateOrg(r.Context(), id, ownerID, req.Name, req.Slug, req.LogoURL)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	usage, err := h.svc.UsageForOrg(r.Context(), org)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, orgresponse.FromOrganization(org, usage))
}

func (h *Handler) UpdateLogo(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid organization id")
		return
	}

	var req updateLogoRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	org, err := h.svc.UpdateLogo(r.Context(), id, ownerID, req.LogoURL)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	usage, err := h.svc.UsageForOrg(r.Context(), org)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, orgresponse.FromOrganization(org, usage))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid organization id")
		return
	}

	if err := h.svc.DeleteOrg(r.Context(), id, ownerID); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(s); err != nil {
		return pgtype.UUID{}, err
	}
	return id, nil
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, orgsvc.ErrOrgNotFound):
		httputil.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, orgsvc.ErrOrgAlreadyExists), errors.Is(err, orgsvc.ErrSlugTaken):
		httputil.WriteError(w, http.StatusConflict, err.Error())
	case errors.Is(err, orgsvc.ErrForbidden):
		httputil.WriteError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, orgsvc.ErrInvalidName),
		errors.Is(err, orgsvc.ErrInvalidSlug),
		errors.Is(err, orgsvc.ErrInvalidPlan),
		errors.Is(err, orgsvc.ErrInvalidOwner):
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
