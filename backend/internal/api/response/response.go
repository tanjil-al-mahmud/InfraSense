package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/infrasense/backend/internal/models"
)

// Success returns 200 with {"data": data}
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"data": data})
}

// Created returns 201 with {"data": data}
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{"data": data})
}

// Paginated returns 200 with {"data": data, "meta": meta}
func Paginated(c *gin.Context, data interface{}, meta models.PaginationMeta) {
	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": meta,
	})
}

// Error returns the given status with {"error": message, "code": code}
func Error(c *gin.Context, status int, message string, code string) {
	c.JSON(status, gin.H{"error": message, "code": code})
}

// NotFound returns 404 with {"error": message, "code": "NOT_FOUND"}
func NotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, gin.H{"error": message, "code": "NOT_FOUND"})
}

// BadRequest returns 400 with {"error": message, "code": code}
func BadRequest(c *gin.Context, message string, code string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": message, "code": code})
}

// Unauthorized returns 401 with {"error": message, "code": "UNAUTHORIZED"}
func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, gin.H{"error": message, "code": "UNAUTHORIZED"})
}

// InternalError returns 500 with {"error": message, "code": "INTERNAL_ERROR"}
func InternalError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": message, "code": "INTERNAL_ERROR"})
}
