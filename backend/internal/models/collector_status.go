package models

import (
	"time"

	"github.com/google/uuid"
)

type CollectorStatus struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	CollectorName   string     `json:"collector_name" db:"collector_name"`
	CollectorType   string     `json:"collector_type" db:"collector_type"`
	Status          string     `json:"status" db:"status"`
	LastPollTime    *time.Time `json:"last_poll_time,omitempty" db:"last_poll_time"`
	LastSuccessTime *time.Time `json:"last_success_time,omitempty" db:"last_success_time"`
	LastError       *string    `json:"last_error,omitempty" db:"last_error"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

type CollectorListFilter struct {
	Status        *string
	CollectorType *string
	Page          int
	PageSize      int
}

type CollectorListResponse struct {
	Data []CollectorStatus `json:"data"`
	Meta PaginationMeta    `json:"meta"`
}
