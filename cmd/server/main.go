package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	pvz_worker_audit "github.com/Staspol216/gh1/cmd/audit"
	pvz_config "github.com/Staspol216/gh1/internal/config"
	pvz_domain "github.com/Staspol216/gh1/internal/domain/order"
	pvz_grpc "github.com/Staspol216/gh1/internal/handlers/grpc"
	pvz_http "github.com/Staspol216/gh1/internal/handlers/http"
	db "github.com/Staspol216/gh1/internal/infrastructure/postgres"
	pvz_order_storage "github.com/Staspol216/gh1/internal/infrastructure/repository/order"
	cache_order_repo "github.com/Staspol216/gh1/internal/infrastructure/repository/order/redis"
	psql_order_outbox_repo "github.com/Staspol216/gh1/internal/infrastructure/repository/order_outbox"
	"github.com/Staspol216/gh1/internal/infrastructure/tx_manager"
	pvz_order_service "github.com/Staspol216/gh1/internal/service/order"
	orders_proto "github.com/Staspol216/gh1/pkg/api/orders.proto"
	"github.com/jackc/pgx/v4/pgxpool"
	"google.golang.org/grpc"
)

type Handler interface {
	Serve() error
}

func main() {
	const (
		jobsCount    = 5
		workersCount = 2
	)

	cfg, err := pvz_config.Load()

	if err != nil {
		log.Fatalf("Load config error: %v", err)
	}

	wg := &sync.WaitGroup{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.Connect(ctx, cfg.DBConnString())

	if err != nil {
		log.Fatal(err)
	}

	txManager := tx_manager.New(pool, sigCtx)

	orderCache := cache_order_repo.New(cfg)

	defer orderCache.Rdb.Close()

	if redisPingErr := orderCache.Healthcheck(sigCtx); redisPingErr != nil {
		log.Fatal(redisPingErr)
	}

	db := db.NewDatabase(txManager)

	postgresRepoConfig := &pvz_order_storage.Config{
		StorageType: pvz_order_storage.StorageTypePostgres,
		Postgres: &pvz_order_storage.PostgresConfig{
			Db:      db,
			Context: sigCtx,
		},
	}

	orderOutboxRepo := &psql_order_outbox_repo.OrderOutboxRepo{
		Db: db,
	}

	tasks := make(chan *pvz_domain.OrderOutboxTask, jobsCount)
	defer close(tasks)

	worker := &pvz_worker_audit.OutboxWorker{
		Context:   sigCtx,
		Repo:      orderOutboxRepo,
		Tasks:     tasks,
		TxManager: txManager,
	}

	wg.Go(func() {
		worker.Run(1 * time.Second)
	})

	producer, err := sarama.NewSyncProducer([]string{cfg.KafkaAddr()}, nil)
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	debugger := &pvz_worker_audit.AuditDebugger{}
	writer := pvz_worker_audit.OrderAuditLogPartitionWriter{
		Context:    sigCtx,
		Producer:   producer,
		Tasks:      tasks,
		OutboxRepo: orderOutboxRepo,
		Debugger:   debugger,
	}

	consumer, err := sarama.NewConsumer([]string{cfg.KafkaAddr()}, nil)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	partConsumer, err := consumer.ConsumePartition("order_audit_logs", 0, sarama.OffsetNewest)
	if err != nil {
		log.Fatalf("Failed to consume partition: %v", err)
	}
	defer partConsumer.Close()

	reader := pvz_worker_audit.OrderAuditLogPartitionReader{
		Context:   sigCtx,
		Partition: partConsumer,
	}

	wg.Go(func() {
		writer.Run()
	})

	wg.Go(func() {
		reader.Run()
	})

	orderStorage, err := pvz_order_storage.New(postgresRepoConfig)

	if err != nil {
		log.Fatal("pvz.New: %w", err)
	}

	populateOrdersErr := orderCache.PopulateOrders(sigCtx, orderStorage, 0)

	if populateOrdersErr != nil {
		log.Fatal("PopulateOrders: %w", populateOrdersErr)
	}

	pvzService := pvz_order_service.New(orderStorage, *orderOutboxRepo, orderCache, txManager)

	httpHandler := pvz_http.New(sigCtx, pvzService)

	tcpListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.BackendGRPCPort))

	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	grcpHandler := pvz_grpc.New(sigCtx, pvzService)

	orders_proto.RegisterOrdersServiceServer(grpcServer, grcpHandler)

	log.Printf("gRPC Server listening at %v", tcpListener.Addr())

	wg.Go(func() {
		if err := grpcServer.Serve(tcpListener); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	})

	wg.Go(func() {
		if err := httpHandler.Serve(cfg); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	})

	<-sigCtx.Done()

	log.Println("Shutdown signal received, gracefully shutting down GRPC server...")

	grpcServer.GracefulStop()

	wg.Wait()

	fmt.Println("✓ All servers shut down successfully")
}
