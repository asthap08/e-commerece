package addresses

import (
	"github.com/asthap08/ecommerce/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, authenticate middleware.Authenticator) {
	r.Route("/api/v1/addresses", func(r chi.Router) {
		r.With(authenticate).Get("/", h.List)
		r.With(authenticate).Post("/", h.Create)
		r.With(authenticate).Delete("/{id}", h.Delete)
		r.With(authenticate).Put("/{id}/default", h.SetDefault)
	})
}
