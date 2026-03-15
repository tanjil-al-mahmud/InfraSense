package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type cacheEntry struct {
	statusCode  int
	contentType string
	body        []byte
	expiresAt   time.Time
}

type Cache struct {
	entries map[string]*cacheEntry
	mutex   sync.RWMutex
}

func NewCache() *Cache {
	cache := &Cache{
		entries: make(map[string]*cacheEntry),
	}

	// Start cleanup goroutine
	go cache.cleanupLoop()

	return cache
}

func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

func (c *Cache) cleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, key)
		}
	}
}

func (c *Cache) get(key string) (*cacheEntry, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry, true
}

func (c *Cache) set(key string, entry *cacheEntry) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.entries[key] = entry
}

// CacheMiddleware creates a caching middleware with specified TTL
func CacheMiddleware(cache *Cache, ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only cache GET requests
		if c.Request.Method != "GET" {
			c.Next()
			return
		}

		// Generate cache key from URL and query parameters
		cacheKey := generateCacheKey(c.Request.URL.String())

		// Check if cached response exists
		if entry, exists := cache.get(cacheKey); exists {
			c.Header("X-Cache", "HIT")
			c.Data(entry.statusCode, entry.contentType, entry.body)
			c.Abort()
			return
		}

		// Create a response writer wrapper to capture the response
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = writer

		c.Header("X-Cache", "MISS")
		c.Next()

		// Cache the response if status is 200
		if writer.Status() == 200 {
			entry := &cacheEntry{
				statusCode:  writer.Status(),
				contentType: writer.Header().Get("Content-Type"),
				body:        writer.body.Bytes(),
				expiresAt:   time.Now().Add(ttl),
			}
			cache.set(cacheKey, entry)
		}
	}
}

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

func generateCacheKey(url string) string {
	hash := sha256.Sum256([]byte(url))
	return hex.EncodeToString(hash[:])
}
