package public

import (
	"errors"
	"net/http"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/handler/httputil"
	bookingsvc "github.com/AmirAbaris/weeto-backend/internal/service/booking"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	svc *bookingsvc.Service
}

func NewHandler(svc *bookingsvc.Service) *Handler {
	return &Handler{svc: svc}
}

type metadataResponse struct {
	Organization  organizationView  `json:"organization"`
	InterviewType interviewTypeView `json:"interview_type"`
}

type organizationView struct {
	Name    string  `json:"name"`
	Slug    string  `json:"slug"`
	LogoURL *string `json:"logo_url,omitempty"`
}

type interviewTypeView struct {
	Title           string `json:"title"`
	Slug            string `json:"slug"`
	DurationMinutes int32  `json:"duration_minutes"`
	BufferMinutes   int32  `json:"buffer_minutes"`
	MeetingProvider string `json:"meeting_provider"`
}

type slotsResponse struct {
	Slots []bookingsvc.SlotView `json:"slots"`
}

type bookRequest struct {
	SlotID string `json:"slot_id"`
	Name   string `json:"name"`
	Phone  string `json:"phone"`
	Email  string `json:"email"`
}

type bookResponse struct {
	ID               string    `json:"id"`
	SlotID           string    `json:"slot_id"`
	Name             string    `json:"name"`
	Phone            string    `json:"phone"`
	Email            string    `json:"email"`
	Status           string    `json:"status"`
	StartAt          time.Time `json:"start_at"`
	EndAt            time.Time `json:"end_at"`
	InterviewTitle   string    `json:"interview_title"`
	OrganizationName string    `json:"organization_name"`
	MeetLink         *string   `json:"meet_link,omitempty"`
}

func (h *Handler) GetMetadata(w http.ResponseWriter, r *http.Request) {
	meta, err := h.svc.ResolveType(r.Context(), r.PathValue("orgSlug"), r.PathValue("typeSlug"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	resp := metadataResponse{
		Organization: organizationView{
			Name: meta.Organization.Name,
			Slug: meta.Organization.Slug,
		},
		InterviewType: interviewTypeView{
			Title:           meta.InterviewType.Title,
			Slug:            meta.InterviewType.Slug,
			DurationMinutes: meta.InterviewType.DurationMinutes,
			BufferMinutes:   meta.InterviewType.BufferMinutes,
			MeetingProvider: string(meta.InterviewType.MeetingProvider),
		},
	}
	if meta.Organization.LogoUrl.Valid {
		logo := meta.Organization.LogoUrl.String
		resp.Organization.LogoURL = &logo
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListSlots(w http.ResponseWriter, r *http.Request) {
	slots, err := h.svc.ListAvailableSlots(r.Context(), r.PathValue("orgSlug"), r.PathValue("typeSlug"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, slotsResponse{Slots: slots})
}

func (h *Handler) Book(w http.ResponseWriter, r *http.Request) {
	var req bookRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	slotID, err := parseUUID(req.SlotID)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid slot id")
		return
	}

	orgSlug := r.PathValue("orgSlug")
	typeSlug := r.PathValue("typeSlug")

	meta, err := h.svc.ResolveType(r.Context(), orgSlug, typeSlug)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	result, err := h.svc.Book(r.Context(), orgSlug, typeSlug, bookingsvc.BookInput{
		SlotID: slotID,
		Name:   req.Name,
		Phone:  req.Phone,
		Email:  req.Email,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	resp := bookResponse{
		ID:               result.Booking.ID.String(),
		SlotID:           result.Booking.SlotID.String(),
		Name:             result.Booking.CandidateName,
		Phone:            result.Booking.CandidatePhone,
		Email:            result.Booking.CandidateEmail,
		Status:           string(result.Booking.Status),
		StartAt:          result.Slot.StartAt.Time.UTC(),
		EndAt:            result.Slot.EndAt.Time.UTC(),
		InterviewTitle:   meta.InterviewType.Title,
		OrganizationName: meta.Organization.Name,
	}
	if result.Booking.MeetLink.Valid {
		meetLink := result.Booking.MeetLink.String
		resp.MeetLink = &meetLink
	}

	httputil.WriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) GetReschedule(w http.ResponseWriter, r *http.Request) {
	ctx, err := h.svc.GetRescheduleContext(r.Context(), r.PathValue("token"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, ctx)
}

type rescheduleRequest struct {
	SlotID string `json:"slot_id"`
}

func (h *Handler) PostReschedule(w http.ResponseWriter, r *http.Request) {
	var req rescheduleRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	slotID, err := parseUUID(req.SlotID)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid slot id")
		return
	}

	result, orgName, interviewTitle, err := h.svc.Reschedule(r.Context(), r.PathValue("token"), slotID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	resp := bookResponse{
		ID:               result.Booking.ID.String(),
		SlotID:           result.Booking.SlotID.String(),
		Name:             result.Booking.CandidateName,
		Phone:            result.Booking.CandidatePhone,
		Email:            result.Booking.CandidateEmail,
		Status:           string(result.Booking.Status),
		StartAt:          result.Slot.StartAt.Time.UTC(),
		EndAt:            result.Slot.EndAt.Time.UTC(),
		InterviewTitle:   interviewTitle,
		OrganizationName: orgName,
	}
	if result.Booking.MeetLink.Valid {
		meetLink := result.Booking.MeetLink.String
		resp.MeetLink = &meetLink
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetCancel(w http.ResponseWriter, r *http.Request) {
	ctx, err := h.svc.GetCancelContext(r.Context(), r.PathValue("token"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, ctx)
}

func (h *Handler) PostCancel(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.CancelByToken(r.Context(), r.PathValue("token")); err != nil {
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
	case errors.Is(err, bookingsvc.ErrOrgNotFound),
		errors.Is(err, bookingsvc.ErrTypeNotFound):
		httputil.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, bookingsvc.ErrSlotNotFound),
		errors.Is(err, bookingsvc.ErrSlotUnavailable):
		httputil.WriteError(w, http.StatusConflict, err.Error())
	case errors.Is(err, bookingsvc.ErrInvalidName),
		errors.Is(err, bookingsvc.ErrInvalidPhone),
		errors.Is(err, bookingsvc.ErrInvalidEmail),
		errors.Is(err, bookingsvc.ErrInvalidSlotID):
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, bookingsvc.ErrGoogleNotConnected):
		httputil.WriteErrorDetail(w, http.StatusUnprocessableEntity, httputil.ErrorDetail{
			Error:     err.Error(),
			Code:      "google_not_connected",
			Action:    "connect_google",
			ActionURL: "/integrations/google/connect",
		})
	case errors.Is(err, bookingsvc.ErrMeetLinkLimitReached):
		httputil.WriteErrorDetail(w, http.StatusForbidden, httputil.ErrorDetail{
			Error: err.Error(),
			Code:  "plan_limit_meet_links",
		})
	case errors.Is(err, bookingsvc.ErrGoogleCalendarFailed):
		httputil.WriteErrorDetail(w, http.StatusUnprocessableEntity, httputil.ErrorDetail{
			Error: err.Error(),
			Code:  "google_calendar_failed",
		})
	case errors.Is(err, bookingsvc.ErrTokenNotFound),
		errors.Is(err, bookingsvc.ErrBookingNotFound):
		httputil.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, bookingsvc.ErrModifyCutoff):
		httputil.WriteErrorDetail(w, http.StatusForbidden, httputil.ErrorDetail{
			Error: err.Error(),
			Code:  "modify_cutoff",
		})
	case errors.Is(err, bookingsvc.ErrBookingNotModifiable):
		httputil.WriteError(w, http.StatusConflict, err.Error())
	case errors.Is(err, bookingsvc.ErrSameSlot):
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
