package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	pvz_worker_audit "github.com/Staspol216/gh1/cmd/audit"
	pvz_domain "github.com/Staspol216/gh1/internal/domain/order"
	pvz_http "github.com/Staspol216/gh1/internal/handlers/http"
	db "github.com/Staspol216/gh1/internal/infrastructure/postgres"
	pvz_order_storage "github.com/Staspol216/gh1/internal/infrastructure/repository/order"
	cache_order_repo "github.com/Staspol216/gh1/internal/infrastructure/repository/order/redis"
	psql_order_outbox_repo "github.com/Staspol216/gh1/internal/infrastructure/repository/order_outbox"
	"github.com/Staspol216/gh1/internal/infrastructure/tx_manager"
	pvz_order_service "github.com/Staspol216/gh1/internal/service/order"
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

	kafkaAddr := os.Getenv("KAFKA_BROKERS")

	producer, err := sarama.NewSyncProducer([]string{kafkaAddr}, nil)
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

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

	debugger := &pvz_worker_audit.AuditDebugger{}
	writer := pvz_worker_audit.OrderAuditLogPartitionWriter{
		Context:    sigCtx,
		Producer:   producer,
		Tasks:      tasks,
		OutboxRepo: orderOutboxRepo,
		Debugger:   debugger,
	}

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

	populateCacheErr := orderCache.PopulateOrders(sigCtx, orderStorage, 0)

	if populateCacheErr != nil {
		log.Fatal("PopulateOrdersCache: %w", populateCacheErr)
	}

	pvzService := pvz_order_service.New(orderStorage, *orderOutboxRepo, orderCache, txManager)

	handler := pvz_http.New(sigCtx, pvzService)

	wg.Go(func() {
		if err := handler.Serve(); err != nil {
			log.Printf("server shutdown error: %v", err)
		}
	})

	wg.Wait()

	fmt.Println("Main done")
}
