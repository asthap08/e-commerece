package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

// CacheControl marks anonymous catalogue reads as publicly cacheable for a
// short time and everything else (carts, orders, auth, admin, any request that
// carries a session cookie, every write) as non-cacheable. Previously every GET
// was "public, max-age=300", which let browsers/CDNs serve stale carts and
// keep a logged-out user looking logged in.
func CacheControl(maxAge int) func(http.Handler) http.Handler {
	public := "public, max-age=" + strconv.Itoa(maxAge)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, noCookie := r.Cookie(sessionCookieName)
			cacheable := r.Method == http.MethodGet && noCookie != nil &&
				strings.HasPrefix(r.URL.Path, "/api/v1/products") &&
				!strings.HasSuffix(strings.TrimRight(r.URL.Path, "/"), "/reviews")
			if cacheable {
				w.Header().Set("Cache-Control", public)
				w.Header().Add("Vary", "Accept-Encoding")
			} else {
				w.Header().Set("Cache-Control", "no-store")
			}
			next.ServeHTTP(w, r)
		})
	}
}
