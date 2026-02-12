package pvz_domain

import "time"

type AuditLog struct {
	Timestamp time.Time `json:"timestamp"`

	RequestID     string `json:"request_id,omitempty"`
	Method        string `json:"method,omitempty"`
	Path          string `json:"path,omitempty"`
	RemoteAddress string `json:"remote_address,omitempty"`
	UserAgent     string `json:"user_agent,omitempty"`

	StatusResponse int           `json:"status_response,omitempty"`
	DurationMs     time.Duration `json:"duration_ms,omitempty"`

	Details interface{} `json:"details,omitempty"`
}
