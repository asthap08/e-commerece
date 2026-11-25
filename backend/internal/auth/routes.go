package auth

import (
	"github.com/asthap08/ecommerce/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, authenticate middleware.Authenticator) {
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/forgot", h.Forgot)
		r.Post("/reset", h.Reset)
		r.With(authenticate).Get("/me", h.Me)
		r.With(authenticate).Put("/me", h.UpdateProfile)
		r.With(authenticate).Put("/password", h.ChangePassword)
		r.With(authenticate).Post("/logout", h.Logout)
	})
}
