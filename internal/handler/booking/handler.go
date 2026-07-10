package booking

import (
	"errors"
	"net/http"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/handler/httputil"
	"github.com/AmirAbaris/weeto-backend/internal/middleware"
	bookingsvc "github.com/AmirAbaris/weeto-backend/internal/service/booking"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	svc *bookingsvc.Service
}

func NewHandler(svc *bookingsvc.Service) *Handler {
	return &Handler{svc: svc}
}

type bookingResponse struct {
	ID             string    `json:"id"`
	SlotID         string    `json:"slot_id"`
	Name           string    `json:"name"`
	Phone          string    `json:"phone"`
	Email          string    `json:"email"`
	Status         string    `json:"status"`
	StartAt        time.Time `json:"start_at"`
	EndAt          time.Time `json:"end_at"`
	InterviewTitle string    `json:"interview_title"`
}

type listResponse struct {
	Today    []bookingResponse `json:"today"`
	Upcoming []bookingResponse `json:"upcoming"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	result, err := h.svc.List(r.Context(), ownerID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, listResponse{
		Today:    toResponses(result.Today),
		Upcoming: toResponses(result.Upcoming),
	})
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid booking id")
		return
	}

	if err := h.svc.Cancel(r.Context(), ownerID, id); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toResponses(views []bookingsvc.BookingView) []bookingResponse {
	resp := make([]bookingResponse, 0, len(views))
	for _, v := range views {
		resp = append(resp, bookingResponse{
			ID:             v.ID,
			SlotID:         v.SlotID,
			Name:           v.Name,
			Phone:          v.Phone,
			Email:          v.Email,
			Status:         v.Status,
			StartAt:        v.StartAt,
			EndAt:          v.EndAt,
			InterviewTitle: v.InterviewTitle,
		})
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
	case errors.Is(err, bookingsvc.ErrBookingNotFound),
		errors.Is(err, bookingsvc.ErrOrgRequired):
		httputil.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, bookingsvc.ErrForbidden):
		httputil.WriteError(w, http.StatusForbidden, err.Error())
	default:
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
