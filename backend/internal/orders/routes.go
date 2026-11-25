package orders

import (
	"github.com/asthap08/ecommerce/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, authenticate middleware.Authenticator) {
	r.Route("/api/v1/orders", func(r chi.Router) {
		r.Post("/track", h.Track)
		r.With(authenticate).Get("/", h.List)
		r.With(authenticate).Get("/{id}", h.Get)
		r.With(authenticate).Put("/{id}/cancel", h.Cancel)
		r.With(authenticate, middleware.AdminOnly).Put("/{id}/tracking", h.UpdateTracking)
		r.With(authenticate, middleware.AdminOnly).Patch("/{id}/status", h.UpdateStatus)
	})
}
