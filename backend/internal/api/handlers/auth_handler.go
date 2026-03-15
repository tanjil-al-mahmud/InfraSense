package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/api/response"
	"github.com/infrasense/backend/internal/auth"
	"github.com/infrasense/backend/internal/db"
	"github.com/infrasense/backend/internal/models"
	"github.com/infrasense/backend/internal/services"
)

type AuthHandler struct {
	userRepo     *db.UserRepository
	jwtService   *auth.JWTService
	auditService *services.AuditService
}

func NewAuthHandler(userRepo *db.UserRepository, jwtService *auth.JWTService, auditService *services.AuditService) *AuthHandler {
	return &AuthHandler{
		userRepo:     userRepo,
		jwtService:   jwtService,
		auditService: auditService,
	}
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), "INVALID_REQUEST")
		return
	}

	// Get user by username
	user, err := h.userRepo.GetByUsername(c.Request.Context(), req.Username)
	if err != nil {
		// Log failed login attempt
		h.auditService.LogUserLoginFailed(c.Request.Context(), req.Username, c.ClientIP())
		// Don't reveal if user exists or not
		response.Unauthorized(c, "Invalid username or password")
		return
	}

	// Check if user is enabled
	if !user.Enabled {
		h.auditService.LogUserLoginFailed(c.Request.Context(), req.Username, c.ClientIP())
		response.Error(c, http.StatusUnauthorized, "User account is disabled", "ACCOUNT_DISABLED")
		return
	}

	// Verify password
	if err := auth.VerifyPassword(req.Password, user.PasswordHash); err != nil {
		h.auditService.LogUserLoginFailed(c.Request.Context(), req.Username, c.ClientIP())
		response.Unauthorized(c, "Invalid username or password")
		return
	}

	// Generate JWT token
	token, err := h.jwtService.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		response.InternalError(c, "Failed to generate token")
		return
	}

	// Log successful login
	h.auditService.LogUserLogin(c.Request.Context(), user.ID, user.Username, c.ClientIP())

	// Remove password hash from response
	user.PasswordHash = ""

	// Login response is returned directly (not wrapped in data) per spec
	c.JSON(http.StatusOK, models.LoginResponse{
		Token: token,
		User:  *user,
	})
}

// Logout handles POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// In a stateless JWT system, logout is handled client-side by removing the token
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// Me handles GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}

	// Remove password hash from response
	user.PasswordHash = ""

	response.Success(c, user)
}

// CreateUser handles POST /api/v1/users (admin only)
func (h *AuthHandler) CreateUser(c *gin.Context) {
	var req models.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), "INVALID_REQUEST")
		return
	}

	// Validate role
	validRoles := map[string]bool{
		models.RoleAdmin:    true,
		models.RoleOperator: true,
		models.RoleViewer:   true,
	}
	if !validRoles[req.Role] {
		response.BadRequest(c, "Invalid role", "INVALID_ROLE")
		return
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		response.InternalError(c, "Failed to hash password")
		return
	}

	// Create user
	user := &models.User{
		ID:           uuid.New(),
		Username:     req.Username,
		PasswordHash: passwordHash,
		Email:        req.Email,
		Role:         req.Role,
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := h.userRepo.Create(c.Request.Context(), user); err != nil {
		response.InternalError(c, "Failed to create user")
		return
	}

	// Remove password hash from response
	user.PasswordHash = ""

	response.Created(c, user)
}
