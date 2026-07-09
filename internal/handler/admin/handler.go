package admin

import (
	"errors"
	"net/http"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	"github.com/AmirAbaris/weeto-backend/internal/handler/httputil"
	"github.com/AmirAbaris/weeto-backend/internal/handler/orgresponse"
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

	withUsage, err := h.orgSvc.GetByIDWithUsageAdmin(r.Context(), org.ID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, orgresponse.FromWithUsage(withUsage))
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
	case errors.Is(err, orgsvc.ErrInvalidPlan):
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
