package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"sync"
	"syscall"

	pvz_worker_audit "github.com/Staspol216/gh1/cmd/audit"
	"github.com/Staspol216/gh1/internal/db"
	pvz_cli "github.com/Staspol216/gh1/internal/handlers/cli"
	pvz_http "github.com/Staspol216/gh1/internal/handlers/http"
	"github.com/Staspol216/gh1/internal/repository/postgresql"
	"github.com/Staspol216/gh1/internal/repository/tx_manager"
	pvz_service "github.com/Staspol216/gh1/internal/service"
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

	jobs := make(chan *pvz_http.AuditLog, jobsCount)
	results := make(chan *pvz_http.AuditLog, jobsCount)

	pool, err := db.NewPool(sigCtx)

	if err != nil {
		log.Fatal(err)
	}

	txManager := tx_manager.New(pool, sigCtx)

	orderCache := pvz_service.NewCache()

	defer orderCache.Rdb.Close()

	if redisPingErr := orderCache.Healthcheck(sigCtx); redisPingErr != nil {
		log.Fatal(redisPingErr)
	}

	db := db.NewDatabase(txManager)

	postgresRepoConfig := &pvz_service.Config{
		StorageType: pvz_service.StorageTypePostgres,
		Postgres: &pvz_service.PostgresConfig{
			Db:      db,
			Context: sigCtx,
		},
	}

	auditLogRepo := &postgresql.AuditLogRepo{
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

	orderStorage, err := pvz_service.NewStorage(postgresRepoConfig)

	if err != nil {
		log.Fatal("pvz.New: %w", err)
	}

	populateCacheErr := orderCache.PopulateOrders(sigCtx, orderStorage, 0)

	if populateCacheErr != nil {
		log.Fatal("PopulateOrdersCache: %w", populateCacheErr)
	}

	pvzService := pvz_service.New(orderStorage, orderCache, txManager)

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
