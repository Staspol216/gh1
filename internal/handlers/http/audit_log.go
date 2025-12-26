package pvz_http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

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

type AuditLogger struct {
	handler *HTTPHandler
}

func CreateOrdersStatusesMap(md *AuditLogger, orderIds []int64) (map[int64]string, error) {
	orders, err := md.handler.pvz.GetOrdersByIDs(orderIds)

	if err != nil {
		return nil, errors.New("cannot get orders in audit log")
	}

	orderStatusesMap := make(map[int64]string, len(orders))

	for _, order := range orders {
		orderStatusesMap[order.ID] = string(order.Status)
	}

	return orderStatusesMap, nil
}

func (md *AuditLogger) LogRequestResponseAndStatusChangeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Print(3333)
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		r.Body = io.NopCloser(bytes.NewBuffer(body))

		var req OrderUpdateRequest
		if len(body) > 0 {
			_ = json.Unmarshal(body, &req)
		}
		log.Printf("Body: %s", string(body))

		timestamp := time.Now()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		oldOrdersStatusesMap, err := CreateOrdersStatusesMap(md, req.OrderIDs)

		if err != nil {
			log.Println(err)
			next.ServeHTTP(ww, r)
			return
		}

		next.ServeHTTP(ww, r)

		latency := time.Since(timestamp)

		newOrdersStatusesMap, err := CreateOrdersStatusesMap(md, req.OrderIDs)

		if err != nil {
			log.Println(err)
			next.ServeHTTP(ww, r)
			return
		}

		log := &AuditLog{
			Timestamp:      timestamp,
			RequestID:      uuid.NewString(),
			Method:         r.Method,
			Path:           r.URL.Path,
			RemoteAddress:  r.RemoteAddr,
			UserAgent:      r.UserAgent(),
			StatusResponse: ww.Status(),
			DurationMs:     latency,

			Details: struct {
				OldOrdersStatuses map[int64]string
				NewOrdersStatuses map[int64]string
			}{
				OldOrdersStatuses: oldOrdersStatusesMap,
				NewOrdersStatuses: newOrdersStatusesMap,
			},
		}

		md.handler.WriteAuditLog(log)
	})
}

func (md *AuditLogger) LogRequestResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		timestamp := time.Now()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		latency := time.Since(timestamp)

		log := &AuditLog{
			Timestamp:      timestamp,
			RequestID:      uuid.NewString(),
			Method:         r.Method,
			Path:           r.URL.Path,
			RemoteAddress:  r.RemoteAddr,
			UserAgent:      r.UserAgent(),
			StatusResponse: ww.Status(),
			DurationMs:     latency,
		}

		md.handler.WriteAuditLog(log)
	})
}
