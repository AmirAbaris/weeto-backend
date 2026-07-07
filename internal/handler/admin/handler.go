package admin

import (
	"errors"
	"net/http"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	"github.com/AmirAbaris/weeto-backend/internal/handler/httputil"
	orgsvc "github.com/AmirAbaris/weeto-backend/internal/service/organization"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	orgSvc *orgsvc.Service
}

func NewHandler(orgSvc *orgsvc.Service) *Handler {
	return &Handler{orgSvc: orgSvc}
}

type updatePlanRequest struct {
	Plan db.PlanType `json:"plan"`
}

type organizationResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	LogoURL   *string   `json:"logo_url,omitempty"`
	OwnerID   string    `json:"owner_id"`
	Plan      string    `json:"plan"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (h *Handler) UpdatePlan(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid organization id")
		return
	}

	var req updatePlanRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	org, err := h.orgSvc.UpdatePlan(r.Context(), id, req.Plan)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, toOrganizationResponse(org))
}

func parseUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(s); err != nil {
		return pgtype.UUID{}, err
	}
	return id, nil
}

func toOrganizationResponse(org db.Organization) organizationResponse {
	resp := organizationResponse{
		Name: org.Name,
		Slug: org.Slug,
		Plan: string(org.Plan),
	}
	if org.ID.Valid {
		resp.ID = org.ID.String()
	}
	if org.OwnerID.Valid {
		resp.OwnerID = org.OwnerID.String()
	}
	if org.LogoUrl.Valid {
		logo := org.LogoUrl.String
		resp.LogoURL = &logo
	}
	if org.CreatedAt.Valid {
		resp.CreatedAt = org.CreatedAt.Time
	}
	if org.UpdatedAt.Valid {
		resp.UpdatedAt = org.UpdatedAt.Time
	}
	return resp
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, orgsvc.ErrOrgNotFound):
		httputil.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, orgsvc.ErrInvalidPlan):
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
