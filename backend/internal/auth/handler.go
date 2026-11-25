package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/asthap08/ecommerce/internal/platform/httpx"
	"github.com/asthap08/ecommerce/internal/platform/middleware"
	"github.com/asthap08/ecommerce/internal/platform/session"
)

type Handler struct {
	svc          *Service
	sessions     *session.Store
	cookieSecure bool
	sameSite     http.SameSite
}

func NewHandler(svc *Service, sessions *session.Store, cookieSecure bool, sameSite http.SameSite) *Handler {
	return &Handler{svc: svc, sessions: sessions, cookieSecure: cookieSecure, sameSite: sameSite}
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		httpx.Error(w, http.StatusBadRequest, ve.Msg)
	case errors.Is(err, ErrEmailTaken):
		httpx.Error(w, http.StatusConflict, "an account with this email already exists")
	case errors.Is(err, ErrBadLogin):
		httpx.Error(w, http.StatusUnauthorized, "email or password is incorrect")
	case errors.Is(err, ErrLocked):
		httpx.Error(w, http.StatusTooManyRequests, err.Error())
	case errors.Is(err, ErrBadResetToken):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrWrongPassword):
		httpx.Error(w, http.StatusForbidden, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, "something went wrong, please try again")
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	// "role" is intentionally NOT part of the request: it used to be accepted
	// from the client, so anyone could register as an administrator.
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	u, err := h.svc.Register(r.Context(), in.Email, in.Password, in.FullName)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.startSession(w, r, u, http.StatusCreated)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	u, err := h.svc.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.startSession(w, r, u, http.StatusOK)
}

// Me returns the signed-in user (id, email, role, profile).
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	u, err := h.svc.Get(r.Context(), middleware.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var in struct {
		FullName string `json:"full_name"`
		Phone    string `json:"phone"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	u, err := h.svc.UpdateProfile(r.Context(), middleware.UserID(r.Context()), in.FullName, in.Phone)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	keep := ""
	if c, err := r.Cookie(session.CookieName); err == nil {
		keep = session.HashToken(c.Value)
	}
	if err := h.svc.ChangePassword(r.Context(), middleware.UserID(r.Context()), in.CurrentPassword, in.NewPassword, keep); err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "password updated"})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(session.CookieName); err == nil {
		if err := h.sessions.Revoke(r.Context(), cookie.Value, time.Now()); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "could not end session")
			return
		}
	}
	h.setCookie(w, "", time.Time{}, -1)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) setCookie(w http.ResponseWriter, value string, expires time.Time, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name: session.CookieName, Value: value, Path: "/", Expires: expires, MaxAge: maxAge,
		HttpOnly: true, Secure: h.cookieSecure, SameSite: h.sameSite,
	})
}

func (h *Handler) startSession(w http.ResponseWriter, r *http.Request, u User, status int) {
	token, expiresAt, err := h.sessions.Create(r.Context(), u.ID, time.Now())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not create session")
		return
	}
	h.setCookie(w, token, expiresAt, int(time.Until(expiresAt).Seconds()))
	httpx.JSON(w, status, map[string]any{"user": u})
}

// Forgot always answers 200 (no account enumeration).
func (h *Handler) Forgot(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	h.svc.RequestReset(r.Context(), in.Email)
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "If that email has an account, a reset link is on its way."})
}

func (h *Handler) Reset(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if err := h.svc.ResetPassword(r.Context(), in.Token, in.NewPassword); err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "password updated"})
}
