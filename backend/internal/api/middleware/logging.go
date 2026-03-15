package middleware

import (
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

var jsonLogger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

// LoggingMiddleware logs each HTTP request as a JSON structured log entry.
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		jsonLogger.Info("http_request",
			"event", "http_request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status_code", c.Writer.Status(),
			"response_time_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		)
	}
}
