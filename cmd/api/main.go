package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"sync"
	"syscall"

	pvz_worker_audit "github.com/Staspol216/gh1/cmd/audit"
	pvz_domain "github.com/Staspol216/gh1/internal/domain/audit_log"
	pvz_cli "github.com/Staspol216/gh1/internal/handlers/cli"
	pvz_http "github.com/Staspol216/gh1/internal/handlers/http"
	db "github.com/Staspol216/gh1/internal/infrastructure/postgres"
	psql_audit_log_repo "github.com/Staspol216/gh1/internal/infrastructure/repository/audit_log"
	pvz_order_storage "github.com/Staspol216/gh1/internal/infrastructure/repository/order"
	cache_order_repo "github.com/Staspol216/gh1/internal/infrastructure/repository/order/redis"
	"github.com/Staspol216/gh1/internal/infrastructure/tx_manager"
	pvz_order_service "github.com/Staspol216/gh1/internal/service/order"
	"github.com/joho/godotenv"
)

type Handler interface {
	Serve() error
}

func main() {
	const (
		jobsCount    = 5
		workersCount = 2
	)

	isHTTP := true

	wg := &sync.WaitGroup{}

	if err := godotenv.Load(); err != nil {
		log.Println("no .env file loaded")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	jobs := make(chan *pvz_domain.AuditLog, jobsCount)
	results := make(chan *pvz_domain.AuditLog, jobsCount)

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

	auditLogRepo := &psql_audit_log_repo.AuditLogRepo{
		Db:      db,
		Context: sigCtx,
	}

	for i := 0; i < workersCount; i++ {
		worker := &pvz_worker_audit.Worker{
			ProcessStrategy: pvz_worker_audit.SaveToDB,
			Context:         sigCtx,
			In:              jobs,
			Out:             results,
			Wg:              wg,
			Repo:            auditLogRepo,
		}
		worker.RunAndServe(i)
	}

	orderStorage, err := pvz_order_storage.New(postgresRepoConfig)

	if err != nil {
		log.Fatal("pvz.New: %w", err)
	}

	populateCacheErr := orderCache.PopulateOrders(sigCtx, orderStorage, 0)

	if populateCacheErr != nil {
		log.Fatal("PopulateOrdersCache: %w", populateCacheErr)
	}

	pvzService := pvz_order_service.New(orderStorage, orderCache, txManager)

	var handler Handler

	if isHTTP {
		handler = pvz_http.New(sigCtx, pvzService, jobs)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := handler.Serve(); err != nil {
				log.Printf("server shutdown error: %v", err)
			}
		}()
	} else {
		handler = pvz_cli.New(pvzService)
		handler.Serve()
	}

	wg.Wait()

	close(results)
	close(jobs)

	fmt.Println("Main done")
}
