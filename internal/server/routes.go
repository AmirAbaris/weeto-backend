package server

import (
	"net/http"

	authhandler "github.com/AmirAbaris/weeto-backend/internal/handler/auth"
	adminhandler "github.com/AmirAbaris/weeto-backend/internal/handler/admin"
	availabilityhandler "github.com/AmirAbaris/weeto-backend/internal/handler/availability"
	bookinghandler "github.com/AmirAbaris/weeto-backend/internal/handler/booking"
	docshandler "github.com/AmirAbaris/weeto-backend/internal/handler/docs"
	"github.com/AmirAbaris/weeto-backend/internal/handler/health"
	interviewtypehandler "github.com/AmirAbaris/weeto-backend/internal/handler/interviewtype"
	orghandler "github.com/AmirAbaris/weeto-backend/internal/handler/organization"
	publichandler "github.com/AmirAbaris/weeto-backend/internal/handler/public"
	googlehandler "github.com/AmirAbaris/weeto-backend/internal/handler/google"
	"github.com/AmirAbaris/weeto-backend/internal/middleware"
)

type Handlers struct {
	Health        *health.Handler
	Docs          *docshandler.Handler
	Auth          *authhandler.Handler
	Admin         *adminhandler.Handler
	Organization  *orghandler.Handler
	InterviewType *interviewtypehandler.Handler
	Availability  *availabilityhandler.Handler
	Booking       *bookinghandler.Handler
	Public        *publichandler.Handler
	Google        *googlehandler.Handler
}

func Register(mux *http.ServeMux, jwtSecret, adminAPIKey string, h Handlers) {
	mux.HandleFunc("GET /docs", h.Docs.UI)
	mux.HandleFunc("GET /openapi.yaml", h.Docs.Spec)
	mux.HandleFunc("GET /health", h.Health.Live)
	mux.HandleFunc("POST /auth/register", h.Auth.Register)
	mux.HandleFunc("POST /auth/login", h.Auth.Login)
	mux.HandleFunc("POST /auth/refresh", h.Auth.Refresh)
	mux.HandleFunc("POST /auth/logout", h.Auth.Logout)

	mux.Handle("POST /organizations", middleware.WithAuth(jwtSecret, h.Organization.Create))
	mux.Handle("GET /organizations/me", middleware.WithAuth(jwtSecret, h.Organization.GetMine))
	mux.Handle("GET /organizations/{id}", middleware.WithAuth(jwtSecret, h.Organization.GetByID))
	mux.Handle("PUT /organizations/{id}", middleware.WithAuth(jwtSecret, h.Organization.Update))
	mux.Handle("PATCH /organizations/{id}/logo", middleware.WithAuth(jwtSecret, h.Organization.UpdateLogo))
	mux.Handle("DELETE /organizations/{id}", middleware.WithAuth(jwtSecret, h.Organization.Delete))

	if adminAPIKey != "" {
		mux.Handle("PATCH /admin/organizations/{id}/plan", middleware.WithAdminKey(adminAPIKey, h.Admin.UpdatePlan))
	}

	mux.Handle("POST /interview-types", middleware.WithAuth(jwtSecret, h.InterviewType.Create))
	mux.Handle("GET /interview-types", middleware.WithAuth(jwtSecret, h.InterviewType.List))
	mux.Handle("PUT /interview-types/{id}", middleware.WithAuth(jwtSecret, h.InterviewType.Update))
	mux.Handle("DELETE /interview-types/{id}", middleware.WithAuth(jwtSecret, h.InterviewType.Delete))

	mux.Handle("PUT /availability", middleware.WithAuth(jwtSecret, h.Availability.Upsert))
	mux.Handle("GET /availability", middleware.WithAuth(jwtSecret, h.Availability.Get))

	mux.Handle("GET /bookings", middleware.WithAuth(jwtSecret, h.Booking.List))
	mux.Handle("DELETE /bookings/{id}", middleware.WithAuth(jwtSecret, h.Booking.Cancel))

	mux.Handle("GET /integrations/google/connect", middleware.WithAuth(jwtSecret, h.Google.Connect))
	mux.Handle("GET /integrations/google/status", middleware.WithAuth(jwtSecret, h.Google.Status))
	mux.HandleFunc("GET /integrations/google/callback", h.Google.Callback)
	mux.Handle("DELETE /integrations/google", middleware.WithAuth(jwtSecret, h.Google.Disconnect))

	mux.HandleFunc("GET /public/{orgSlug}/{typeSlug}", h.Public.GetMetadata)
	mux.HandleFunc("GET /public/{orgSlug}/{typeSlug}/slots", h.Public.ListSlots)
	mux.HandleFunc("POST /public/{orgSlug}/{typeSlug}/book", h.Public.Book)
	mux.HandleFunc("GET /public/reschedule/{token}", h.Public.GetReschedule)
	mux.HandleFunc("POST /public/reschedule/{token}", h.Public.PostReschedule)
	mux.HandleFunc("GET /public/cancel/{token}", h.Public.GetCancel)
	mux.HandleFunc("POST /public/cancel/{token}", h.Public.PostCancel)
}
