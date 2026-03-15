package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCacheMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("cache miss on first request", func(t *testing.T) {
		cache := NewCache()
		router := gin.New()

		callCount := 0
		router.GET("/test", CacheMiddleware(cache, 1*time.Second), func(c *gin.Context) {
			callCount++
			c.JSON(200, gin.H{"message": "test"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "MISS", w.Header().Get("X-Cache"))
		assert.Equal(t, 1, callCount)
	})

	t.Run("cache hit on second request", func(t *testing.T) {
		cache := NewCache()
		router := gin.New()

		callCount := 0
		router.GET("/test", CacheMiddleware(cache, 1*time.Second), func(c *gin.Context) {
			callCount++
			c.JSON(200, gin.H{"message": "test"})
		})

		// First request
		req1 := httptest.NewRequest("GET", "/test", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		// Second request
		req2 := httptest.NewRequest("GET", "/test", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Equal(t, 200, w2.Code)
		assert.Equal(t, "HIT", w2.Header().Get("X-Cache"))
		assert.Equal(t, 1, callCount) // Handler should only be called once
	})

	t.Run("cache expires after TTL", func(t *testing.T) {
		cache := NewCache()
		router := gin.New()

		callCount := 0
		router.GET("/test", CacheMiddleware(cache, 100*time.Millisecond), func(c *gin.Context) {
			callCount++
			c.JSON(200, gin.H{"message": "test"})
		})

		// First request
		req1 := httptest.NewRequest("GET", "/test", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		// Wait for cache to expire
		time.Sleep(150 * time.Millisecond)

		// Second request after expiration
		req2 := httptest.NewRequest("GET", "/test", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Equal(t, 200, w2.Code)
		assert.Equal(t, "MISS", w2.Header().Get("X-Cache"))
		assert.Equal(t, 2, callCount) // Handler should be called twice
	})

	t.Run("cache only GET requests", func(t *testing.T) {
		cache := NewCache()
		router := gin.New()

		callCount := 0
		router.POST("/test", CacheMiddleware(cache, 1*time.Second), func(c *gin.Context) {
			callCount++
			c.JSON(200, gin.H{"message": "test"})
		})

		// First POST request
		req1 := httptest.NewRequest("POST", "/test", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		// Second POST request
		req2 := httptest.NewRequest("POST", "/test", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Equal(t, 200, w2.Code)
		assert.Empty(t, w2.Header().Get("X-Cache")) // No cache header for POST
		assert.Equal(t, 2, callCount)               // Handler should be called twice
	})

	t.Run("different URLs have different cache keys", func(t *testing.T) {
		cache := NewCache()
		router := gin.New()

		callCount := 0
		router.GET("/test", CacheMiddleware(cache, 1*time.Second), func(c *gin.Context) {
			callCount++
			page := c.Query("page")
			c.JSON(200, gin.H{"page": page})
		})

		// Request with page=1
		req1 := httptest.NewRequest("GET", "/test?page=1", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		// Request with page=2
		req2 := httptest.NewRequest("GET", "/test?page=2", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Equal(t, "MISS", w1.Header().Get("X-Cache"))
		assert.Equal(t, "MISS", w2.Header().Get("X-Cache"))
		assert.Equal(t, 2, callCount) // Different URLs should not share cache
	})

	t.Run("cache cleanup removes expired entries", func(t *testing.T) {
		cache := NewCache()
		router := gin.New()

		router.GET("/test", CacheMiddleware(cache, 100*time.Millisecond), func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "test"})
		})

		// Create cache entry
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Wait for expiration
		time.Sleep(150 * time.Millisecond)

		// Trigger cleanup
		cache.cleanup()

		// Verify cache is empty
		cache.mutex.RLock()
		entryCount := len(cache.entries)
		cache.mutex.RUnlock()

		assert.Equal(t, 0, entryCount)
	})
}

func TestGenerateCacheKey(t *testing.T) {
	t.Run("same URL generates same key", func(t *testing.T) {
		key1 := generateCacheKey("/api/v1/devices?page=1")
		key2 := generateCacheKey("/api/v1/devices?page=1")
		assert.Equal(t, key1, key2)
	})

	t.Run("different URLs generate different keys", func(t *testing.T) {
		key1 := generateCacheKey("/api/v1/devices?page=1")
		key2 := generateCacheKey("/api/v1/devices?page=2")
		assert.NotEqual(t, key1, key2)
	})

	t.Run("cache key is deterministic", func(t *testing.T) {
		url := "/api/v1/devices?page=1&status=healthy"
		key1 := generateCacheKey(url)
		key2 := generateCacheKey(url)
		key3 := generateCacheKey(url)
		assert.Equal(t, key1, key2)
		assert.Equal(t, key2, key3)
	})
}
