package payments

import (
	"github.com/asthap08/ecommerce/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, authenticate middleware.Authenticator) {
	r.Route("/api/v1/payments", func(r chi.Router) {
		r.Post("/webhook", h.Webhook)
		r.With(authenticate).Post("/intent", h.CreateIntent)
	})
}
