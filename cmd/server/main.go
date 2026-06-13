package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/Staspol216/gh1/internal/config"
	"github.com/Staspol216/gh1/internal/handlers/grpc"
	"github.com/Staspol216/gh1/internal/handlers/http/orders"
	"github.com/Staspol216/gh1/internal/infra/order_outbox"
	"github.com/Staspol216/gh1/internal/infra/postgres"
	"github.com/Staspol216/gh1/internal/infra/repository/order"
	"github.com/Staspol216/gh1/internal/infra/tx_manager"
	"github.com/Staspol216/gh1/internal/service/order"
	"github.com/Staspol216/gh1/internal/service/order_audit"
	"github.com/Staspol216/gh1/pkg/api/common.proto"
	"github.com/Staspol216/gh1/pkg/api/orders.proto"
	"github.com/Staspol216/gh1/pkg/logger"
	"github.com/Staspol216/gh1/pkg/monitoring"
	"github.com/Staspol216/gh1/pkg/tracing"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func main() {
	const (
		jobsCount = 5
	)
	defer app_logger.MyLogger.Sync()

	cfg, err := pvz_config.Load()
	if err != nil {
		app_logger.MyLogger.Fatal("load config error", zap.Error(err))
	}

	monitoring.StartMetricsServer(cfg.PrometheusAddr(), cfg.PrometheusPath)

	_, tracingCloser := tracing.InitTracer(cfg.AppName, cfg.JaegerCollectorEndpoint())
	defer tracingCloser.Close()

	wg := &sync.WaitGroup{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.Connect(ctx, cfg.DBConnString())
	if err != nil {
		app_logger.MyLogger.Fatal("connect to postgres", zap.Error(err))
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr(),
	})
	defer rdb.Close()

	txManager := tx_manager.New(pool, sigCtx)

	orderCache := order.NewOrderCache(rdb)

	if redisPingErr := orderCache.Healthcheck(sigCtx); redisPingErr != nil {
		app_logger.MyLogger.Fatal("redis healthcheck failed", zap.Error(redisPingErr))
	}

	database := db.NewDatabase(txManager)

	tasks := make(chan []order_outbox.OrderOutboxTask, jobsCount)
	defer close(tasks)

	orderOutbox := &order_outbox.OrderOutbox{
		Db:    database,
		Tasks: tasks,
	}

	wg.Go(func() {
		orderOutbox.Run(sigCtx, 60*time.Second)
	})

	producer, err := sarama.NewSyncProducer([]string{cfg.KafkaAddr()}, nil)
	if err != nil {
		app_logger.MyLogger.Fatal("create kafka producer", zap.Error(err))
	}
	defer producer.Close()

	orderAuditLogProducer := order_audit.OrderAuditLogProducer{
		Context:  sigCtx,
		Producer: producer,
		Tasks:    tasks,
		Outbox:   orderOutbox,
	}

	consumer, err := sarama.NewConsumer([]string{cfg.KafkaAddr()}, nil)
	if err != nil {
		app_logger.MyLogger.Fatal("create kafka consumer", zap.Error(err))
	}
	defer consumer.Close()

	partConsumer, err := consumer.ConsumePartition("order_audit_logs", 0, sarama.OffsetNewest)
	if err != nil {
		app_logger.MyLogger.Fatal("consume kafka partition", zap.Error(err))
	}
	defer partConsumer.Close()

	orderAuditLogConsumer := order_audit.OrderAuditLogPartitionConsumer{
		Context:   sigCtx,
		Partition: partConsumer,
	}

	wg.Go(func() {
		orderAuditLogProducer.Run()
	})

	wg.Go(func() {
		orderAuditLogConsumer.Run()
	})

	orderRepo, err := order.NewOrderRepo(database)

	if err != nil {
		app_logger.MyLogger.Fatal("create order repository", zap.Error(err))
	}

	pvzService := pvz_order_service.NewPvzService(orderRepo, orderOutbox, orderCache, txManager)

	tcpListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.BackendGRPCPort))

	if err != nil {
		app_logger.MyLogger.Fatal("listen tcp", zap.Error(err), zap.Int("port", cfg.BackendGRPCPort))
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(tracing.UnaryInterceptor))

	grpcHandler := pvz_grpc.New(pvzService)

	orders_proto.RegisterOrdersServiceServer(grpcServer, grpcHandler)

	app_logger.MyLogger.Info("gRPC server listening", zap.String("address", tcpListener.Addr().String()))

	wg.Go(func() {
		if err := grpcServer.Serve(tcpListener); err != nil {
			app_logger.MyLogger.Error("gRPC server error", zap.Error(err))
		}
	})

	httpServer, err := newGatewayHTTPServer(sigCtx, cfg, grpcHandler)
	if err != nil {
		app_logger.MyLogger.Fatal("create HTTP gateway server", zap.Error(err))
	}

	wg.Go(func() {
		app_logger.MyLogger.Info("HTTP gateway server listening", zap.String("address", cfg.HTTPAddr()))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app_logger.MyLogger.Error("HTTP gateway server error", zap.Error(err))
		}
	})

	<-sigCtx.Done()

	app_logger.MyLogger.Info("shutdown signal received, gracefully shutting down gRPC server")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		app_logger.MyLogger.Error("HTTP gateway server shutdown error", zap.Error(err))
	}

	grpcServer.GracefulStop()

	wg.Wait()

	app_logger.MyLogger.Info("all servers shut down successfully")
}

func newGatewayHTTPServer(ctx context.Context, cfg *pvz_config.Config, ordersServer orders_proto.OrdersServiceServer) (*http.Server, error) {
	mux := runtime.NewServeMux(
		runtime.WithErrorHandler(gatewayErrorHandler),
		runtime.WithForwardResponseOption(gatewayStatusCodeForwarder),
	)

	if err := orders_proto.RegisterOrdersServiceHandlerServer(ctx, mux, ordersServer); err != nil {
		return nil, err
	}

	return &http.Server{
		Addr:    cfg.HTTPAddr(),
		Handler: orders_http_handler.NewGatewayRouter(gatewayHTTPHandler(mux)),
	}, nil
}

func gatewayHTTPHandler(gateway http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/ping" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte("pong"))
			return
		}

		gateway.ServeHTTP(w, r)
	})
}

func gatewayStatusCodeForwarder(ctx context.Context, w http.ResponseWriter, _ proto.Message) error {
	method, ok := runtime.RPCMethod(ctx)
	if !ok {
		return nil
	}

	switch method {
	case orders_proto.OrdersService_CreateOrder_FullMethodName:
		w.WriteHeader(http.StatusCreated)
	case orders_proto.OrdersService_UpdateOrders_FullMethodName,
		orders_proto.OrdersService_DeleteOrder_FullMethodName:
		w.WriteHeader(http.StatusNoContent)
	}

	return nil
}

func gatewayErrorHandler(_ context.Context, _ *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, _ *http.Request, err error) {
	var customStatus *runtime.HTTPStatusError
	if errors.As(err, &customStatus) {
		err = customStatus.Err
	}

	st := status.Convert(err)
	httpStatus := runtime.HTTPStatusFromCode(st.Code())
	if customStatus != nil {
		httpStatus = customStatus.HTTPStatus
	}

	body := &common_proto.ApiError{
		Error: &common_proto.BaseError{
			Code:    gatewayAPIErrorCode(httpStatus, st.Code()),
			Message: st.Message(),
		},
	}

	buf, marshalErr := marshaler.Marshal(body)
	if marshalErr != nil {
		app_logger.MyLogger.Error("marshal gateway error response", zap.Error(marshalErr))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Del("Trailer")
	w.Header().Del("Transfer-Encoding")
	w.Header().Set("Content-Type", marshaler.ContentType(body))
	w.WriteHeader(httpStatus)
	if _, writeErr := w.Write(buf); writeErr != nil {
		app_logger.MyLogger.Error("write gateway error response", zap.Error(writeErr))
	}
}

func gatewayAPIErrorCode(httpStatus int, grpcCode codes.Code) common_proto.ApiErrorCode {
	switch httpStatus {
	case http.StatusBadRequest:
		return common_proto.ApiErrorCode_bad_request
	case http.StatusUnauthorized:
		return common_proto.ApiErrorCode_unauthorized
	case http.StatusForbidden:
		return common_proto.ApiErrorCode_forbidden
	case http.StatusNotFound:
		return common_proto.ApiErrorCode_not_found
	case http.StatusMethodNotAllowed:
		return common_proto.ApiErrorCode_method_not_allowed
	case http.StatusNotAcceptable:
		return common_proto.ApiErrorCode_not_acceptable
	case http.StatusRequestTimeout:
		return common_proto.ApiErrorCode_request_timeout
	case http.StatusConflict:
		return common_proto.ApiErrorCode_state_conflict
	case http.StatusNotImplemented:
		return common_proto.ApiErrorCode_not_implemented
	case 499:
		return common_proto.ApiErrorCode_request_cancelled
	}

	if grpcCode == codes.Canceled {
		return common_proto.ApiErrorCode_request_cancelled
	}

	return common_proto.ApiErrorCode_internal
}
