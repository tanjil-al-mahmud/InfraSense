package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Email        *string   `json:"email,omitempty" db:"email"`
	Role         string    `json:"role" db:"role"`
	Enabled      bool      `json:"enabled" db:"enabled"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type UserCreateRequest struct {
	Username string  `json:"username" binding:"required"`
	Password string  `json:"password" binding:"required,min=12"`
	Email    *string `json:"email,omitempty"`
	Role     string  `json:"role" binding:"required"`
}

const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleViewer   = "viewer"
)
