package pricing

import (
	"github.com/asthap08/ecommerce/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, authenticate middleware.Authenticator) {
	r.Route("/api/v1/coupons", func(r chi.Router) {
		// Anonymous coupon lookup used to return full coupon records, which let
		// anyone enumerate codes. Listing/creating is admin-only; validating
		// needs a signed-in shopper and returns only the discount.
		r.With(authenticate, middleware.AdminOnly).Get("/", h.List)
		r.With(authenticate, middleware.AdminOnly).Post("/", h.Create)
		r.With(authenticate, middleware.AdminOnly).Patch("/{id}", h.SetActive)
		r.With(authenticate).Get("/validate", h.Validate)
	})
}
