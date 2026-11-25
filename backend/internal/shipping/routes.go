package shipping

import (
	"github.com/asthap08/ecommerce/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, authenticate middleware.Authenticator) {
	r.Route("/api/v1/shipments", func(r chi.Router) {
		r.With(authenticate, middleware.AdminOnly).Post("/", h.Create)
	})
}
