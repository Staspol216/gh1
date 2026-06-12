package monitoring

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Staspol216/gh1/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

const (
	statusSuccess = "success"
	statusError   = "error"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "route", "status"})

	httpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route", "status"})

	grpcRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "grpc_requests_total",
		Help: "Total number of gRPC requests.",
	}, []string{"method", "status"})

	grpcRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "grpc_request_duration_seconds",
		Help:    "gRPC request duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "status"})

	orderOperationsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "order_operations_total",
		Help: "Total number of order service operations.",
	}, []string{"operation", "status"})

	cacheOperationsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_operations_total",
		Help: "Total number of cache operations.",
	}, []string{"operation", "status"})

	outboxBatchesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "outbox_batches_total",
		Help: "Total number of outbox polling batches.",
	}, []string{"status"})

	outboxTasksTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "outbox_tasks_total",
		Help: "Total number of outbox tasks by operation status.",
	}, []string{"operation", "status"})

	outboxTasksLocked = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "outbox_tasks_locked",
		Help:    "Number of outbox tasks locked per polling batch.",
		Buckets: []float64{0, 1, 5, 10, 25, 50, 100},
	})

	kafkaMessagesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "kafka_messages_total",
		Help: "Total number of Kafka audit messages.",
	}, []string{"operation", "status"})
)

func ObserveHTTPRequest(method, route string, statusCode int, duration time.Duration) {
	status := strconv.Itoa(statusCode)
	httpRequestsTotal.WithLabelValues(method, normalizeRoute(route), status).Inc()
	httpRequestDuration.WithLabelValues(method, normalizeRoute(route), status).Observe(duration.Seconds())
}

func ObserveGRPCRequest(method string, err error, duration time.Duration) {
	status := statusSuccess
	if err != nil {
		status = statusError
	}

	grpcRequestsTotal.WithLabelValues(method, status).Inc()
	grpcRequestDuration.WithLabelValues(method, status).Observe(duration.Seconds())
}

func ObserveOrderOperation(operation string, err error) {
	operationStatus := statusSuccess
	if err != nil {
		operationStatus = statusError
	}

	orderOperationsTotal.WithLabelValues(operation, operationStatus).Inc()
}

func ObserveCacheOperation(operation string, err error) {
	operationStatus := statusSuccess
	if err != nil {
		operationStatus = statusError
	}

	cacheOperationsTotal.WithLabelValues(operation, operationStatus).Inc()
}

func ObserveOutboxBatch(status string, tasksCount int) {
	outboxBatchesTotal.WithLabelValues(status).Inc()
	outboxTasksLocked.Observe(float64(tasksCount))
}

func ObserveOutboxTask(operation string, err error) {
	operationStatus := statusSuccess
	if err != nil {
		operationStatus = statusError
	}

	outboxTasksTotal.WithLabelValues(operation, operationStatus).Inc()
}

func ObserveKafkaMessage(operation string, err error) {
	operationStatus := statusSuccess
	if err != nil {
		operationStatus = statusError
	}

	kafkaMessagesTotal.WithLabelValues(operation, operationStatus).Inc()
}

func SetResponseTimeSummary(responseTime float64) {
	grpcRequestDuration.WithLabelValues("unknown", statusSuccess).Observe(responseTime / 1000)
}

func SetRequestCounter() {
	grpcRequestsTotal.WithLabelValues("unknown", statusSuccess).Inc()
}

func SetErrorCounter() {
	grpcRequestsTotal.WithLabelValues("unknown", statusError).Inc()
}

func RequestDuration(duration float64) {
	grpcRequestDuration.WithLabelValues("unknown", statusSuccess).Observe(duration / 1000)
}

func SetGrpcRequestCountWithStatus(status string) {
	grpcRequestsTotal.WithLabelValues("unknown", status).Inc()
}

func StartMetricsServer(Addr string, path string) {
	mux := http.NewServeMux()
	mux.Handle(path, promhttp.Handler())

	server := &http.Server{
		Addr:              Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		app_logger.MyLogger.Info("metrics server starting", zap.String("address", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app_logger.MyLogger.Error("metrics server error", zap.Error(err))
		}
	}()
}

func normalizeRoute(route string) string {
	if route == "" {
		return "unknown"
	}

	return route
}

func init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		grpcRequestsTotal,
		grpcRequestDuration,
		orderOperationsTotal,
		cacheOperationsTotal,
		outboxBatchesTotal,
		outboxTasksTotal,
		outboxTasksLocked,
		kafkaMessagesTotal,
	)
}
