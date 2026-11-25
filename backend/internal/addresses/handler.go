package addresses

import (
	"errors"
	"net/http"

	"github.com/asthap08/ecommerce/internal/platform/httpx"
	"github.com/asthap08/ecommerce/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func fail(w http.ResponseWriter, err error) {
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		httpx.Error(w, http.StatusBadRequest, ve.Msg)
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "address not found")
	default:
		httpx.Error(w, http.StatusInternalServerError, "something went wrong")
	}
}

// The user id comes from the authenticated session (middleware.UserID). The old
// code read ctx.Value("user_id") - a key nothing ever set - and panicked.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var a Address
	if !httpx.Decode(w, r, &a) {
		return
	}
	a.ID, a.UserID = "", middleware.UserID(r.Context())
	created, err := h.svc.Create(r.Context(), a)
	if err != nil {
		fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, created)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListByUser(r.Context(), middleware.UserID(r.Context()))
	if err != nil {
		fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id"), middleware.UserID(r.Context())); err != nil {
		fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) SetDefault(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.SetDefault(r.Context(), chi.URLParam(r, "id"), middleware.UserID(r.Context())); err != nil {
		fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
