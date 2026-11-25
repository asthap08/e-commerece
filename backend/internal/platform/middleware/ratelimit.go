package middleware

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimiter is a fixed-window per-client limiter. The key is the client IP
// (without the ephemeral port – the old code keyed on "ip:port" so every new
// TCP connection got a fresh bucket and the limit never triggered), and the
// window start is fixed, so busy clients are no longer blocked forever.
type RateLimiter struct {
	mu      sync.Mutex
	hits    map[string]*bucket
	limit   int
	window  time.Duration
	methods map[string]bool // nil = all methods
	paths   map[string]bool // nil = all paths
}

type bucket struct {
	count int
	reset time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{hits: map[string]*bucket{}, limit: limit, window: window}
	go func() {
		t := time.NewTicker(window)
		defer t.Stop()
		for now := range t.C {
			rl.mu.Lock()
			for k, b := range rl.hits {
				if now.After(b.reset) {
					delete(rl.hits, k)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

// OnlyPaths restricts the limiter to POSTs on the given exact paths
// (used for a much stricter login/register limit).
func (rl *RateLimiter) OnlyPaths(paths ...string) *RateLimiter {
	rl.paths = map[string]bool{}
	for _, p := range paths {
		rl.paths[p] = true
	}
	rl.methods = map[string]bool{http.MethodPost: true}
	return rl
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rl.paths != nil && !(rl.paths[r.URL.Path] && rl.methods[r.Method]) {
			next.ServeHTTP(w, r)
			return
		}
		now := time.Now()
		key := clientIP(r)
		rl.mu.Lock()
		b := rl.hits[key]
		if b == nil || now.After(b.reset) {
			b = &bucket{reset: now.Add(rl.window)}
			rl.hits[key] = b
		}
		b.count++
		over := b.count > rl.limit
		wait := int(time.Until(b.reset).Seconds()) + 1
		rl.mu.Unlock()
		if over {
			w.Header().Set("Retry-After", strconv.Itoa(wait))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"too many requests, please slow down"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
