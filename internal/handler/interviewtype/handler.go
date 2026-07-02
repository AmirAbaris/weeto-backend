package interviewtype

import (
	"errors"
	"net/http"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	"github.com/AmirAbaris/weeto-backend/internal/handler/httputil"
	"github.com/AmirAbaris/weeto-backend/internal/middleware"
	interviewtypesvc "github.com/AmirAbaris/weeto-backend/internal/service/interviewtype"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	svc *interviewtypesvc.Service
}

func NewHandler(svc *interviewtypesvc.Service) *Handler {
	return &Handler{svc: svc}
}

type interviewTypeRequest struct {
	Title           string  `json:"title"`
	Slug            string  `json:"slug"`
	DurationMinutes int32   `json:"duration_minutes"`
	BufferMinutes   int32   `json:"buffer_minutes"`
	MeetingProvider string  `json:"meeting_provider"`
	MeetingURL      *string `json:"meeting_url,omitempty"`
}

type interviewTypeResponse struct {
	ID              string    `json:"id"`
	OrganizationID  string    `json:"organization_id"`
	Title           string    `json:"title"`
	Slug            string    `json:"slug"`
	DurationMinutes int32     `json:"duration_minutes"`
	BufferMinutes   int32     `json:"buffer_minutes"`
	MeetingProvider string    `json:"meeting_provider"`
	MeetingURL      *string   `json:"meeting_url,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req interviewTypeRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.svc.Create(r.Context(), ownerID, toInput(req))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, toResponse(item))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	items, err := h.svc.List(r.Context(), ownerID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	resp := make([]interviewTypeResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, toResponse(item))
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid interview type id")
		return
	}

	var req interviewTypeRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.svc.Update(r.Context(), id, ownerID, toInput(req))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, toResponse(item))
}

func toInput(req interviewTypeRequest) interviewtypesvc.Input {
	return interviewtypesvc.Input{
		Title:           req.Title,
		Slug:            req.Slug,
		DurationMinutes: req.DurationMinutes,
		BufferMinutes:   req.BufferMinutes,
		MeetingProvider: db.MeetingProvider(req.MeetingProvider),
		MeetingURL:      req.MeetingURL,
	}
}

func toResponse(item db.InterviewType) interviewTypeResponse {
	resp := interviewTypeResponse{
		Title:           item.Title,
		Slug:            item.Slug,
		DurationMinutes: item.DurationMinutes,
		BufferMinutes:   item.BufferMinutes,
		MeetingProvider: string(item.MeetingProvider),
	}
	if item.ID.Valid {
		resp.ID = item.ID.String()
	}
	if item.OrganizationID.Valid {
		resp.OrganizationID = item.OrganizationID.String()
	}
	if item.MeetingUrl.Valid {
		url := item.MeetingUrl.String
		resp.MeetingURL = &url
	}
	if item.CreatedAt.Valid {
		resp.CreatedAt = item.CreatedAt.Time
	}
	if item.UpdatedAt.Valid {
		resp.UpdatedAt = item.UpdatedAt.Time
	}
	return resp
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
	case errors.Is(err, interviewtypesvc.ErrNotFound):
		httputil.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, interviewtypesvc.ErrSlugTaken):
		httputil.WriteError(w, http.StatusConflict, err.Error())
	case errors.Is(err, interviewtypesvc.ErrForbidden):
		httputil.WriteError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, interviewtypesvc.ErrOrgRequired):
		httputil.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, interviewtypesvc.ErrGoogleNotConnected):
		httputil.WriteErrorDetail(w, http.StatusUnprocessableEntity, httputil.ErrorDetail{
			Error:     err.Error(),
			Code:      "google_not_connected",
			Action:    "connect_google",
			ActionURL: "/integrations/google/connect",
		})
	case errors.Is(err, interviewtypesvc.ErrPlanLimitInterviewTypes):
		httputil.WriteErrorDetail(w, http.StatusForbidden, httputil.ErrorDetail{
			Error:     err.Error(),
			Code:      "plan_limit_interview_types",
			Action:    "upgrade",
			ActionURL: "/settings/billing",
		})
	case errors.Is(err, interviewtypesvc.ErrInvalidTitle),
		errors.Is(err, interviewtypesvc.ErrInvalidSlug),
		errors.Is(err, interviewtypesvc.ErrInvalidDuration),
		errors.Is(err, interviewtypesvc.ErrInvalidBuffer),
		errors.Is(err, interviewtypesvc.ErrInvalidMeetingProvider),
		errors.Is(err, interviewtypesvc.ErrInvalidMeetingURL):
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
