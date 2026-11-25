package returns

import (
	"errors"
	"net/http"

	"github.com/asthap08/ecommerce/internal/platform/httpx"
	"github.com/asthap08/ecommerce/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
)

type Handler struct{ svc *Service }

func NewHandlerWithService(svc *Service) *Handler { return &Handler{svc: svc} }

func fail(w http.ResponseWriter, err error) {
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		httpx.Error(w, http.StatusBadRequest, ve.Msg)
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "return not found")
	case errors.Is(err, ErrNotEligible), errors.Is(err, ErrAlreadyOpen), errors.Is(err, ErrBadTransition):
		httpx.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrBadItems):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, "something went wrong")
	}
}

// Create: a signed-in customer requests a return for their own delivered order.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		OrderID string   `json:"order_id"`
		ItemIDs []string `json:"item_ids"`
		Reason  string   `json:"reason"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	rr, err := h.svc.Request(r.Context(), middleware.UserID(r.Context()), in.OrderID, in.ItemIDs, in.Reason)
	if err != nil {
		fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, rr)
}

func (h *Handler) Mine(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListByUser(r.Context(), middleware.UserID(r.Context()))
	if err != nil {
		fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *Handler) ListAll(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListAll(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *Handler) ListByOrder(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListByOrder(r.Context(), r.URL.Query().Get("order_id"))
	if err != nil {
		fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	rr, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, rr)
}

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status   string `json:"status"`
		Note     string `json:"note"`
		LabelURL string `json:"label_url"`
		AuthCode string `json:"auth_code"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if err := h.svc.SetStatus(r.Context(), chi.URLParam(r, "id"), in.Status, in.Note, in.LabelURL, in.AuthCode); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
