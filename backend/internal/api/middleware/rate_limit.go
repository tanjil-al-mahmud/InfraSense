package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}

	// Cleanup old entries every minute
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, timestamps := range rl.requests {
		// Remove timestamps older than window
		valid := []time.Time{}
		for _, ts := range timestamps {
			if now.Sub(ts) < rl.window {
				valid = append(valid, ts)
			}
		}

		if len(valid) == 0 {
			delete(rl.requests, ip)
		} else {
			rl.requests[ip] = valid
		}
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Get existing timestamps for this IP
	timestamps, exists := rl.requests[ip]
	if !exists {
		rl.requests[ip] = []time.Time{now}
		return true
	}

	// Remove timestamps older than window
	valid := []time.Time{}
	for _, ts := range timestamps {
		if now.Sub(ts) < rl.window {
			valid = append(valid, ts)
		}
	}

	// Check if limit exceeded
	if len(valid) >= rl.limit {
		return false
	}

	// Add current timestamp
	valid = append(valid, now)
	rl.requests[ip] = valid

	return true
}

// RateLimitMiddleware limits requests per IP
func RateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	limiter := newRateLimiter(limit, window)

	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !limiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"code":  "RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
