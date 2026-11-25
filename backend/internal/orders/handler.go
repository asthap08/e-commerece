package orders

import (
	"errors"
	"net/http"
	"strings"

	"github.com/asthap08/ecommerce/internal/platform/httpx"
	"github.com/asthap08/ecommerce/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
)

type Handler struct{ svc *Service }

func NewHandlerWithService(svc *Service) *Handler { return &Handler{svc: svc} }

func fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrOrderNotFound):
		httpx.Error(w, http.StatusNotFound, "order not found")
	case errors.Is(err, ErrInvalidStatusTransition):
		httpx.Error(w, http.StatusConflict, "this order can no longer be changed to that status")
	case errors.Is(err, ErrInvalidTracking):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, "something went wrong")
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := httpx.Page(r, 50, 100)
	items, err := h.svc.ListByUser(r.Context(), middleware.UserID(r.Context()), limit, offset)
	if err != nil {
		fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"data": items, "limit": limit, "offset": offset})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	o, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		fail(w, err)
		return
	}
	if o.UserID != middleware.UserID(r.Context()) && middleware.UserRole(r.Context()) != "admin" {
		httpx.Error(w, http.StatusNotFound, "order not found")
		return
	}
	httpx.JSON(w, http.StatusOK, o)
}

// Cancel lets the signed-in customer cancel THEIR OWN pending order. Previously
// any signed-in user could cancel any order (even shipped ones) by id.
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.CancelOwn(r.Context(), chi.URLParam(r, "id"), middleware.UserID(r.Context())); err != nil {
		fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// Track is public: it needs the order id AND the e-mail used at checkout.
func (h *Handler) Track(w http.ResponseWriter, r *http.Request) {
	var in struct {
		OrderID string `json:"order_id"`
		Email   string `json:"email"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.OrderID) == "" || strings.TrimSpace(in.Email) == "" {
		httpx.Error(w, http.StatusBadRequest, "order id and email are required")
		return
	}
	t, err := h.svc.Track(r.Context(), in.OrderID, in.Email)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "no order matches that id and email")
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

func (h *Handler) UpdateTracking(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TrackingNumber  string `json:"tracking_number"`
		TrackingCarrier string `json:"tracking_carrier"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if err := h.svc.UpdateTracking(r.Context(), chi.URLParam(r, "id"), in.TrackingNumber, in.TrackingCarrier); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status string `json:"status"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	switch in.Status = strings.TrimSpace(in.Status); in.Status {
	case "processing", "shipped", "delivered", "cancelled":
	default:
		httpx.Error(w, http.StatusBadRequest, "invalid fulfillment status")
		return
	}
	if err := h.svc.UpdateFulfillmentStatus(r.Context(), chi.URLParam(r, "id"), in.Status); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
