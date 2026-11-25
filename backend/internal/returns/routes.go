package returns

import (
	"github.com/asthap08/ecommerce/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, authenticate middleware.Authenticator) {
	r.Route("/api/v1/returns", func(r chi.Router) {
		r.Use(authenticate)
		r.Post("/", h.Create)
		r.Get("/mine", h.Mine)
		r.Group(func(r chi.Router) {
			r.Use(middleware.AdminOnly)
			r.Get("/", h.ListAll)
			r.Get("/order", h.ListByOrder)
			r.Get("/{id}", h.Get)
			r.Patch("/{id}/status", h.UpdateStatus)
		})
	})
}
