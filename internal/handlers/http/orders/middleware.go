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
	chi_middleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"go.uber.org/zap"
)

func NewGatewayRouter(gateway http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(chi_middleware.RequestID)
	r.Use(chi_middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(tracingMiddleware)
	r.Use(metricsMiddleware)

	loggedGateway := requestLoggingMiddleware(gateway)
	r.Method(http.MethodPost, "/orders", loggedGateway)
	r.Method(http.MethodPatch, "/orders", loggedGateway)
	r.Method(http.MethodDelete, "/orders/{orderID}", loggedGateway)

	r.Method(http.MethodGet, "/ping", gateway)
	r.Method(http.MethodGet, "/orders", gateway)
	r.Method(http.MethodGet, "/orders/{orderID}", gateway)
	r.Method(http.MethodGet, "/orders/refunds", gateway)
	r.Method(http.MethodGet, "/orders-history", gateway)

	r.NotFound(gateway.ServeHTTP)
	r.MethodNotAllowed(gateway.ServeHTTP)

	return r
}

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

func requestLoggingMiddleware(next http.Handler) http.Handler {
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
