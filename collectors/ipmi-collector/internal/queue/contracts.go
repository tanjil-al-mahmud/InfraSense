package queue

import "time"

type MetricSample struct {
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels"`
	Timestamp time.Time         `json:"timestamp"`
}

type MetricsBatch struct {
	SchemaVersion string        `json:"schema_version"`
	DeviceID      string        `json:"device_id"`
	Source        string        `json:"source"` // ipmi
	CollectedAt   time.Time     `json:"collected_at"`
	Samples       []MetricSample`json:"samples"`
}

type HardwareEvent struct {
	SchemaVersion  string                 `json:"schema_version"`
	DeviceID       string                 `json:"device_id"`
	OccurredAt     *time.Time             `json:"occurred_at,omitempty"`
	ObservedAt     time.Time              `json:"observed_at"`
	SourceProtocol string                 `json:"source_protocol"` // ipmi
	Component      string                 `json:"component"`
	EventType      string                 `json:"event_type"`
	Severity       string                 `json:"severity"`
	Message        string                 `json:"message"`
	Classification map[string]any         `json:"classification,omitempty"`
	Raw            map[string]any         `json:"raw,omitempty"`
	DedupeKey      string                 `json:"dedupe_key"`
}

