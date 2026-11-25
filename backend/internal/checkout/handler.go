package checkout

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/asthap08/ecommerce/internal/platform/httpx"
	"github.com/asthap08/ecommerce/internal/platform/middleware"
)

type Handler struct {
	svc      *Service
	enabled  bool
	onPlaced func(email, name, orderID string, totalMinor int64)
}

// OnPlaced registers a callback fired (best effort) after an order is created.
func (h *Handler) OnPlaced(f func(email, name, orderID string, totalMinor int64)) { h.onPlaced = f }

func NewHandler() *Handler { return &Handler{} }

func NewHandlerWithService(svc *Service, enabled bool) *Handler {
	return &Handler{svc: svc, enabled: enabled}
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"module": "checkout", "status": "ok"})
}

func (h *Handler) Checkout(w http.ResponseWriter, r *http.Request) {
	if !h.enabled {
		httpx.Error(w, http.StatusServiceUnavailable, "checkout is temporarily unavailable")
		return
	}
	var req Request
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.UserID = middleware.UserID(r.Context())
	req.IdempotencyKey = r.Header.Get("Idempotency-Key")
	resp, err := h.svc.Checkout(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidRequest):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrCartEmpty), errors.Is(err, ErrInsufficientStock), errors.Is(err, ErrIdempotencyConflict):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, "checkout could not be completed", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
	if h.onPlaced != nil {
		h.onPlaced(req.CustomerEmail, req.CustomerName, resp.OrderID, resp.TotalMinor)
	}
}

// cleanMsg strips the sentinel prefix ("invalid checkout request: ...") so the
// shopper sees just the useful part.
func cleanMsg(err error) string {
	msg := err.Error()
	for _, p := range []string{ErrInvalidRequest.Error() + ": ", ErrCoupon.Error() + ": "} {
		msg = strings.TrimPrefix(msg, p)
	}
	return msg
}
