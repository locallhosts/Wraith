// Package ratelimit implements a simple per-API-key token bucket rate
// limiter. Deliberately in-memory and per-instance rather than a
// Redis-backed distributed limiter — the tradeoff (limits are per-replica,
// not global) is documented in docs/ENTERPRISE.md and is the right choice
// for a single-digit-replica deployment; swap for Redis+Lua if you scale
// out the API horizontally.
package ratelimit

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type bucket struct {
	tokens     float64
	lastRefill time.Time
}

type Limiter struct {
	mu                    sync.Mutex
	buckets               map[string]*bucket
	ratePerMin            float64
	capacity              float64
	playgroundRatePerMin  float64
	playgroundCapacity    float64
	lastCleanup           time.Time
}

func New(requestsPerMinute int, playgroundRequestsPerMinute ...int) *Limiter {
	if requestsPerMinute < 1 {
		requestsPerMinute = 1
	}
	playgroundRPM := requestsPerMinute
	if len(playgroundRequestsPerMinute) > 0 && playgroundRequestsPerMinute[0] > 0 {
		playgroundRPM = playgroundRequestsPerMinute[0]
	}
	return &Limiter{
		buckets:              map[string]*bucket{},
		ratePerMin:           float64(requestsPerMinute),
		capacity:              float64(requestsPerMinute),
		playgroundRatePerMin: float64(playgroundRPM),
		playgroundCapacity:   float64(playgroundRPM),
		lastCleanup:          time.Now(),
	}
}

func (l *Limiter) allow(key string, ratePerMin, capacity float64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if now.Sub(l.lastCleanup) >= 5*time.Minute && len(l.buckets) > 10000 {
		cutoff := now.Add(-10 * time.Minute)
		for k, b := range l.buckets {
			if b.lastRefill.Before(cutoff) {
				delete(l.buckets, k)
			}
		}
		l.lastCleanup = now
	}

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: capacity - 1, lastRefill: now}
		l.buckets[key] = b
		return true
	}

	elapsed := now.Sub(b.lastRefill).Minutes()
	b.tokens += elapsed * ratePerMin
	if b.tokens > capacity {
		b.tokens = capacity
	}
	b.lastRefill = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Middleware keys the bucket by authenticated identity label when
// available, falling back to remote IP for unauthenticated routes
// (e.g. the webhook endpoint).
func (l *Limiter) Middleware(identityKey func(c *gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := identityKey(c)
		if key == "" {
			key = c.ClientIP()
		}
		if c.Request.URL.Path == "/playground/validate" {
			key = "playground:" + c.ClientIP()
			if !l.allow(key, l.playgroundRatePerMin, l.playgroundCapacity) {
				c.Header("Retry-After", "60")
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error": "Playground rate limit exceeded, slow down",
				})
				return
			}
			c.Next()
			return
		}
		if !l.allow(key, l.ratePerMin, l.capacity) {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded, slow down",
			})
			return
		}
		c.Next()
	}
}
