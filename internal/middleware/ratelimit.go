package middleware

import (
	"net/http"
	"sync"
	"time"

	"go.internal/business-data-api/pkg/response"
)

type clientVisitor struct {
	tokens     float64
	lastRefill time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*clientVisitor
	rate     float64
	capacity float64
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, exists := rl.visitors[ip]
	if !exists {
		rl.visitors[ip] = &clientVisitor{
			tokens:     rl.capacity - 1,
			lastRefill: now,
		}
		return true
	}

	elapsed := now.Sub(v.lastRefill).Seconds()
	v.tokens += elapsed * rl.rate
	if v.tokens > rl.capacity {
		v.tokens = rl.capacity
	}
	v.lastRefill = now

	if v.tokens >= 1 {
		v.tokens -= 1
		return true
	}

	return false
}

func (rl *RateLimiter) cleanup(ttl time.Duration) {
	ticker := time.NewTicker(ttl)
	for range ticker.C {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastRefill) > ttl {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.Header.Get("X-Real-IP")
			if ip == "" {
				ip = r.Header.Get("X-Forwarded-For")
			}
			if ip == "" {
				ip = r.RemoteAddr
			}

			if !rl.allow(ip) {
				response.Error(w, r, http.StatusTooManyRequests, "Batas request terlampaui. Silakan coba sesaat lagi.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func NewRateLimiter(ratePerSec float64, capacity int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*clientVisitor),
		rate:     ratePerSec,
		capacity: float64(capacity),
	}
	go rl.cleanup(3 * time.Minute)
	return rl
}
