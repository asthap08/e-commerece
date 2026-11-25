package payments

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/asthap08/ecommerce/internal/platform/middleware"
)

type Handler struct{ svc *Service }

func NewHandlerWithService(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) CreateIntent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrderID string `json:"order_id"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	response, err := h.svc.CreatePaymentIntent(r.Context(), middleware.UserID(r.Context()), req.OrderID)
	if err != nil {
		switch {
		case errors.Is(err, ErrStripeNotConfigured):
			http.Error(w, "card payments are temporarily unavailable", http.StatusServiceUnavailable)
		case errors.Is(err, ErrPaymentNotFound):
			http.Error(w, "payment not found", http.StatusNotFound)
		case errors.Is(err, ErrInvalidPaymentRequest):
			http.Error(w, "order is required", http.StatusBadRequest)
		case errors.Is(err, ErrPaymentNotPayable):
			http.Error(w, "payment is not payable", http.StatusConflict)
		default:
			http.Error(w, "payment could not be initialized", http.StatusBadGateway)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handler) Webhook(w http.ResponseWriter, r *http.Request) {
	const maxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	if err := h.svc.HandleWebhook(r.Context(), payload, sigHeader); err != nil {
		if errors.Is(err, ErrInvalidWebhook) {
			http.Error(w, "invalid Stripe webhook", http.StatusBadRequest)
		} else {
			http.Error(w, "Stripe webhook processing failed", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}
