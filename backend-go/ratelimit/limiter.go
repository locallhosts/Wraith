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
	mu         sync.Mutex
	buckets    map[string]*bucket
	ratePerMin float64
	capacity   float64
}

func New(requestsPerMinute int) *Limiter {
	return &Limiter{
		buckets:    map[string]*bucket{},
		ratePerMin: float64(requestsPerMinute),
		capacity:   float64(requestsPerMinute),
	}
}

func (l *Limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	now := time.Now()
	if !ok {
		b = &bucket{tokens: l.capacity - 1, lastRefill: now}
		l.buckets[key] = b
		return true
	}

	elapsed := now.Sub(b.lastRefill).Minutes()
	b.tokens += elapsed * l.ratePerMin
	if b.tokens > l.capacity {
		b.tokens = l.capacity
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
		if !l.allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded, slow down",
			})
			return
		}
		c.Next()
	}
}
