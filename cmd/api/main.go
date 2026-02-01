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
	pvz_repository "github.com/Staspol216/gh1/internal/repository/storage"
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

	wg := &sync.WaitGroup{}

	isHTTP := true

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

	db := db.NewDatabase(txManager)

	postgresRepoConfig := &pvz_repository.Config{
		StorageType: pvz_repository.StorageTypePostgres,
		Postgres: &pvz_repository.PostgresConfig{
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

	orderStorage, err := pvz_repository.New(postgresRepoConfig)

	if err != nil {
		log.Fatal("pvz.New: %w", err)
	}

	pvzService := pvz_service.New(orderStorage, txManager)

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
