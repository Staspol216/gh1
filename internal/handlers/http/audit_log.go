package pvz_http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type AuditLog struct {
	Timestamp time.Time `json:"timestamp"`

	RequestID  string `json:"request_id,omitempty"`
	Method     string `json:"method,omitempty"`
	Path       string `json:"path,omitempty"`
	RemoteAddr string `json:"remote_addr,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`

	Status     int           `json:"status,omitempty"`
	DurationMs time.Duration `json:"duration_ms,omitempty"`
}

func AuditLogger(next http.Handler, h *HTTPHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		timestamp := time.Now()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(w, r)

		latency := time.Since(timestamp)

		log := &AuditLog{
			Timestamp:  timestamp,
			RequestID:  uuid.NewString(),
			Method:     r.Method,
			Path:       r.URL.Path,
			RemoteAddr: r.RemoteAddr,
			UserAgent:  r.UserAgent(),
			Status:     ww.Status(),
			DurationMs: latency,
		}

		h.WriteAuditLog(log)
	})
}
