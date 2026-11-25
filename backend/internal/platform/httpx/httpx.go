// Package httpx holds small helpers shared by every HTTP handler so that
// responses, errors and request-body handling are consistent.
package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
)

// MaxBody is the default request body limit (1 MiB).
const MaxBody = 1 << 20

// JSON writes v as JSON with the given status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// Error writes {"error": msg}. Never pass raw database errors as msg.
func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"error": msg})
}

// Decode reads a JSON body (size limited, unknown fields allowed) into dst.
func Decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBody)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			Error(w, http.StatusRequestEntityTooLarge, "request body too large")
		} else {
			Error(w, http.StatusBadRequest, "invalid request body")
		}
		return false
	}
	return true
}

// Page parses limit/offset query params with sane bounds.
func Page(r *http.Request, def, max int) (limit, offset int) {
	limit, offset = def, 0
	q := r.URL.Query()
	if v := atoi(q.Get("limit")); v > 0 && v <= max {
		limit = v
	}
	if v := atoi(q.Get("offset")); v > 0 {
		offset = v
	}
	return
}

func atoi(s string) int {
	n := 0
	if s == "" || len(s) > 9 {
		return 0
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
