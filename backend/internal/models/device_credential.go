package models

import (
	"time"

	"github.com/google/uuid"
)

type DeviceCredential struct {
	ID                       uuid.UUID `json:"id" db:"id"`
	DeviceID                 uuid.UUID `json:"device_id" db:"device_id"`
	Protocol                 string    `json:"protocol" db:"protocol"`
	Username                 *string   `json:"username,omitempty" db:"username"`
	PasswordEncrypted        []byte    `json:"-" db:"password_encrypted"`
	CommunityStringEncrypted []byte    `json:"-" db:"community_string_encrypted"`
	AuthProtocol             *string   `json:"auth_protocol,omitempty" db:"auth_protocol"`
	PrivProtocol             *string   `json:"priv_protocol,omitempty" db:"priv_protocol"`
	// Connection settings
	Port            *int      `json:"port,omitempty" db:"port"`
	HTTPScheme      string    `json:"http_scheme" db:"http_scheme"`
	SSLVerify       bool      `json:"ssl_verify" db:"ssl_verify"`
	PollingInterval int       `json:"polling_interval" db:"polling_interval"`
	TimeoutSeconds  int       `json:"timeout_seconds" db:"timeout_seconds"`
	RetryAttempts   int       `json:"retry_attempts" db:"retry_attempts"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

type DeviceCredentialRequest struct {
	Protocol        string  `json:"protocol" binding:"required"`
	Username        *string `json:"username,omitempty"`
	Password        *string `json:"password,omitempty"`
	CommunityString *string `json:"community_string,omitempty"`
	AuthProtocol    *string `json:"auth_protocol,omitempty"`
	PrivProtocol    *string `json:"priv_protocol,omitempty"`
	// Connection settings
	Port            *int    `json:"port,omitempty"`
	HTTPScheme      *string `json:"http_scheme,omitempty"`
	SSLVerify       *bool   `json:"ssl_verify,omitempty"`
	PollingInterval *int    `json:"polling_interval,omitempty"`
	TimeoutSeconds  *int    `json:"timeout_seconds,omitempty"`
	RetryAttempts   *int    `json:"retry_attempts,omitempty"`
}

const (
	ProtocolIPMI    = "ipmi"
	ProtocolRedfish = "redfish"
	ProtocolSNMPv2c = "snmp_v2c"
	ProtocolSNMPv3  = "snmp_v3"
)
