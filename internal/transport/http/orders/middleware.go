package orders_http_handler

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/Staspol216/gh1/pkg/logger"
	"github.com/Staspol216/gh1/pkg/monitoring"
	"github.com/Staspol216/gh1/pkg/tracing"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		recorder := &responseStatusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(recorder, r)

		route := chi.RouteContext(r.Context()).RoutePattern()
		monitoring.ObserveHTTPRequest(r.Method, route, recorder.statusCode, time.Since(startTime))
	})
}

func tracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		span, ctx := tracing.StartSpanFromContext(r.Context(), "HTTP request")
		span.SetTag("component", "http")
		span.SetTag("http.method", r.Method)
		span.SetTag("http.url", r.URL.Path)

		recorder := &responseStatusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(recorder, r.WithContext(ctx))

		route := chi.RouteContext(ctx).RoutePattern()
		span.SetTag("http.route", route)
		span.SetTag("http.status_code", recorder.statusCode)
		if recorder.statusCode >= http.StatusInternalServerError {
			span.SetTag("error", true)
		}
		tracing.FinishSpan(span, startTime, nil)
	})
}

func shouldLogRequest(r *http.Request) bool {
	routePattern := chi.RouteContext(r.Context()).RoutePattern()

	switch r.Method {
	case http.MethodPost, http.MethodPatch:
		return routePattern == "/orders"
	case http.MethodDelete:
		return routePattern == "/orders/{orderID}"
	default:
		return false
	}
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)

		defer func(body io.ReadCloser) {
			if closeErr := body.Close(); closeErr != nil {
				return
			}
		}(r.Body)

		if err != nil {
			app_logger.MyLogger.Error("read request body", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		r.Body = io.NopCloser(bytes.NewBuffer(body))

		fields := []zap.Field{
			zap.String("remote", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("proto", r.Proto),
		}

		if len(body) > 0 {
			fields = append(fields, zap.ByteString("body", body))
		}

		app_logger.MyLogger.Info("HTTP request", fields...)

		next.ServeHTTP(w, r)
	})
}

type responseStatusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseStatusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseStatusRecorder) Write(body []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}

	return r.ResponseWriter.Write(body)
}
