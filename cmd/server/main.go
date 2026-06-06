package main

import (
	"context"
	"fmt"
	"net"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/Staspol216/gh1/internal/config"
	"github.com/Staspol216/gh1/internal/handlers/grpc"
	"github.com/Staspol216/gh1/internal/handlers/http"
	"github.com/Staspol216/gh1/internal/infra/order_outbox"
	"github.com/Staspol216/gh1/internal/infra/postgres"
	"github.com/Staspol216/gh1/internal/infra/repository/order"
	"github.com/Staspol216/gh1/internal/infra/tx_manager"
	"github.com/Staspol216/gh1/internal/service/order"
	"github.com/Staspol216/gh1/internal/service/order_audit"
	"github.com/Staspol216/gh1/pkg/api/orders.proto"
	"github.com/Staspol216/gh1/pkg/logger"
	"github.com/Staspol216/gh1/pkg/monitoring"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	const (
		jobsCount = 5
	)
	defer app_logger.MyLogger.Sync()
	monitoring.StartMetricsServer()

	cfg, err := pvz_config.Load()

	if err != nil {
		app_logger.MyLogger.Fatal("load config error", zap.Error(err))
	}

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

	httpHandler := pvz_http.New(sigCtx, pvzService)

	tcpListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.BackendGRPCPort))

	if err != nil {
		app_logger.MyLogger.Fatal("listen tcp", zap.Error(err), zap.Int("port", cfg.BackendGRPCPort))
	}

	grpcServer := grpc.NewServer()

	grcpHandler := pvz_grpc.New(pvzService)

	orders_proto.RegisterOrdersServiceServer(grpcServer, grcpHandler)

	app_logger.MyLogger.Info("gRPC server listening", zap.String("address", tcpListener.Addr().String()))

	wg.Go(func() {
		if err := grpcServer.Serve(tcpListener); err != nil {
			app_logger.MyLogger.Error("gRPC server error", zap.Error(err))
		}
	})

	wg.Go(func() {
		if err := httpHandler.Serve(cfg); err != nil {
			app_logger.MyLogger.Error("HTTP server error", zap.Error(err))
		}
	})

	<-sigCtx.Done()

	app_logger.MyLogger.Info("shutdown signal received, gracefully shutting down gRPC server")

	grpcServer.GracefulStop()

	wg.Wait()

	app_logger.MyLogger.Info("all servers shut down successfully")
}
