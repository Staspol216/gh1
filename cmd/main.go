package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	pvz_worker_audit "github.com/Staspol216/gh1/cmd/audit"
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

	wg := &sync.WaitGroup{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(sigCtx)

	if err != nil {
		log.Fatal(err)
	}

	txManager := tx_manager.New(pool, sigCtx)

	orderCache := cache_order_repo.New()

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

	kafkaHost := os.Getenv("KAFKA_HOST")
	kafkaPort := os.Getenv("KAFKA_PORT")
	kafkaAddr := fmt.Sprintf("%s:%s", kafkaHost, kafkaPort)

	producer, err := sarama.NewSyncProducer([]string{kafkaAddr}, nil)
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

	// Создание консьюмера Kafka
	consumer, err := sarama.NewConsumer([]string{kafkaAddr}, nil)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Подписка на партицию "order_audit_logs" в Kafka
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

	grpcPort := os.Getenv("BACKEND_GRPC_PORT")
	tcpListener, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))

	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	grcpHandler := pvz_grpc.New(sigCtx, pvzService)

	orders_proto.RegisterOrdersServiceServer(grpcServer, grcpHandler)

	log.Printf("Server listening at %v", tcpListener.Addr())

	grpcServeErr := grpcServer.Serve(tcpListener)

	if grpcServeErr != nil {
		log.Fatalf("Failed to grpcServer.Serve: %v", grpcServeErr)
	}

	httpServeErr := httpHandler.Serve()

	if httpServeErr != nil {
		log.Printf("Failed to httpHandler.Serve: %v", httpServeErr)
	}

	wg.Wait()

	fmt.Println("Main done")
}
