package availability

import (
	"errors"
	"net/http"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/handler/httputil"
	"github.com/AmirAbaris/weeto-backend/internal/middleware"
	availabilitysvc "github.com/AmirAbaris/weeto-backend/internal/service/availability"
)

type Handler struct {
	svc *availabilitysvc.Service
}

func NewHandler(svc *availabilitysvc.Service) *Handler {
	return &Handler{svc: svc}
}

type workingHourRequest struct {
	DayOfWeek int16  `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type breakRequest struct {
	DayOfWeek int16  `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type timeOffRequest struct {
	StartAt string `json:"start_at"`
	EndAt   string `json:"end_at"`
}

type availabilityRequest struct {
	Timezone            string               `json:"timezone"`
	MaxInterviewsPerDay int32                `json:"max_interviews_per_day"`
	BookingHorizonDays  int32                `json:"booking_horizon_days"`
	WorkingHours        []workingHourRequest `json:"working_hours"`
	Breaks              []breakRequest       `json:"breaks"`
	TimeOff             []timeOffRequest     `json:"time_off"`
}

type workingHourResponse struct {
	ID        string    `json:"id,omitempty"`
	DayOfWeek int16     `json:"day_of_week"`
	StartTime string    `json:"start_time"`
	EndTime   string    `json:"end_time"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type breakResponse struct {
	ID        string `json:"id,omitempty"`
	DayOfWeek int16  `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type timeOffResponse struct {
	ID      string    `json:"id,omitempty"`
	StartAt time.Time `json:"start_at"`
	EndAt   time.Time `json:"end_at"`
}

type availabilityResponse struct {
	Timezone            string                `json:"timezone"`
	MaxInterviewsPerDay int32                 `json:"max_interviews_per_day"`
	BookingHorizonDays  int32                 `json:"booking_horizon_days"`
	WorkingHours        []workingHourResponse `json:"working_hours"`
	Breaks              []breakResponse       `json:"breaks"`
	TimeOff             []timeOffResponse     `json:"time_off"`
	UpdatedAt           time.Time             `json:"updated_at,omitempty"`
}

func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req availabilityRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	view, err := h.svc.Upsert(r.Context(), ownerID, toInput(req))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, toResponse(view))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	view, err := h.svc.Get(r.Context(), ownerID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, toResponse(view))
}

func toInput(req availabilityRequest) availabilitysvc.Input {
	in := availabilitysvc.Input{
		Timezone:            req.Timezone,
		MaxInterviewsPerDay: req.MaxInterviewsPerDay,
		BookingHorizonDays:  req.BookingHorizonDays,
		WorkingHours:        make([]availabilitysvc.WorkingHourInput, 0, len(req.WorkingHours)),
		Breaks:              make([]availabilitysvc.BreakInput, 0, len(req.Breaks)),
		TimeOff:             make([]availabilitysvc.TimeOffInput, 0, len(req.TimeOff)),
	}
	for _, h := range req.WorkingHours {
		in.WorkingHours = append(in.WorkingHours, availabilitysvc.WorkingHourInput{
			DayOfWeek: h.DayOfWeek,
			StartTime: h.StartTime,
			EndTime:   h.EndTime,
		})
	}
	for _, b := range req.Breaks {
		in.Breaks = append(in.Breaks, availabilitysvc.BreakInput{
			DayOfWeek: b.DayOfWeek,
			StartTime: b.StartTime,
			EndTime:   b.EndTime,
		})
	}
	for _, t := range req.TimeOff {
		in.TimeOff = append(in.TimeOff, availabilitysvc.TimeOffInput{
			StartAt: t.StartAt,
			EndAt:   t.EndAt,
		})
	}
	return in
}

func toResponse(view availabilitysvc.View) availabilityResponse {
	resp := availabilityResponse{
		Timezone:            view.Timezone,
		MaxInterviewsPerDay: view.MaxInterviewsPerDay,
		BookingHorizonDays:  view.BookingHorizonDays,
		WorkingHours:        make([]workingHourResponse, 0, len(view.WorkingHours)),
		Breaks:              make([]breakResponse, 0, len(view.Breaks)),
		TimeOff:             make([]timeOffResponse, 0, len(view.TimeOff)),
		UpdatedAt:           view.UpdatedAt,
	}
	for _, h := range view.WorkingHours {
		resp.WorkingHours = append(resp.WorkingHours, workingHourResponse{
			ID:        h.ID,
			DayOfWeek: h.DayOfWeek,
			StartTime: h.StartTime,
			EndTime:   h.EndTime,
			CreatedAt: h.CreatedAt,
		})
	}
	for _, b := range view.Breaks {
		resp.Breaks = append(resp.Breaks, breakResponse{
			ID:        b.ID,
			DayOfWeek: b.DayOfWeek,
			StartTime: b.StartTime,
			EndTime:   b.EndTime,
		})
	}
	for _, t := range view.TimeOff {
		resp.TimeOff = append(resp.TimeOff, timeOffResponse{
			ID:      t.ID,
			StartAt: t.StartAt,
			EndAt:   t.EndAt,
		})
	}
	return resp
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, availabilitysvc.ErrNotFound):
		httputil.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, availabilitysvc.ErrForbidden):
		httputil.WriteError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, availabilitysvc.ErrOrgRequired):
		httputil.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, availabilitysvc.ErrInvalidTimezone),
		errors.Is(err, availabilitysvc.ErrInvalidMaxPerDay),
		errors.Is(err, availabilitysvc.ErrInvalidBookingHorizon),
		errors.Is(err, availabilitysvc.ErrInvalidDayOfWeek),
		errors.Is(err, availabilitysvc.ErrInvalidTimeRange),
		errors.Is(err, availabilitysvc.ErrOverlappingHours),
		errors.Is(err, availabilitysvc.ErrBreakOutsideHours),
		errors.Is(err, availabilitysvc.ErrOverlappingTimeOff),
		errors.Is(err, availabilitysvc.ErrInvalidTimeOffRange),
		errors.Is(err, availabilitysvc.ErrInvalidTimeOffTime):
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
