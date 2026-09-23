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
	resolver *TrustedProxyResolver
	done     chan struct{}
	once     sync.Once
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, exists := rl.visitors[ip]
	if !exists {
		v = &clientVisitor{tokens: rl.capacity, lastRefill: now}
		rl.visitors[ip] = v
	} else {
		elapsed := now.Sub(v.lastRefill).Seconds()
		v.tokens += elapsed * rl.rate
		if v.tokens > rl.capacity {
			v.tokens = rl.capacity
		}
		v.lastRefill = now
	}

	if v.tokens >= 1 {
		v.tokens -= 1
		return true
	}

	return false
}

func (rl *RateLimiter) runCleanup(ttl time.Duration) {
	ticker := time.NewTicker(ttl)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rl.cleanup(ttl)
		case <-rl.done:
			return
		}
	}
}

func (rl *RateLimiter) cleanup(ttl time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for ip, v := range rl.visitors {
		if time.Since(v.lastRefill) > ttl {
			delete(rl.visitors, ip)
		}
	}
}

func (rl *RateLimiter) Stop() {
	rl.once.Do(func() {
		close(rl.done)
	})
}

func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if rl.resolver != nil {
				if addr := rl.resolver.clientAddr(r); addr.IsValid() {
					ip = addr.String()
				}
			}

			if !rl.allow(ip) {
				response.Error(w, r, http.StatusTooManyRequests, "Batas request terlampaui. Silakan coba sesaat lagi.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func NewRateLimiter(ratePerSec float64, capacity int, resolver *TrustedProxyResolver) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*clientVisitor),
		rate:     ratePerSec,
		capacity: float64(capacity),
		resolver: resolver,
		done:     make(chan struct{}),
	}
	go rl.runCleanup(3 * time.Minute)
	return rl
}
