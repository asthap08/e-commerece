package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/asthap08/ecommerce/internal/platform/session"
)

type ctxKey string

const UserKey ctxKey = "user"
const RoleKey ctxKey = "role"

type SessionResolver interface {
	Resolve(context.Context, string, time.Time) (session.Identity, error)
}

type Authenticator func(http.Handler) http.Handler

func Auth(store SessionResolver) Authenticator {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(session.CookieName)
			if err != nil || cookie.Value == "" {
				unauthorizedJSON(w)
				return
			}
			identity, err := store.Resolve(r.Context(), cookie.Value, time.Now())
			if err != nil {
				unauthorizedJSON(w)
				return
			}
			ctx := context.WithValue(r.Context(), UserKey, identity.UserID)
			ctx = context.WithValue(ctx, RoleKey, identity.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value(RoleKey).(string)
		if role != "admin" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func UserID(ctx context.Context) string {
	userID, _ := ctx.Value(UserKey).(string)
	return userID
}

func UserRole(ctx context.Context) string {
	role, _ := ctx.Value(RoleKey).(string)
	return role
}

const sessionCookieName = session.CookieName

func unauthorizedJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
}
