package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"sync"
	"syscall"

	pvz_worker "github.com/Staspol216/gh1/cmd/worker"
	pvz_cli "github.com/Staspol216/gh1/internal/handlers/cli"
	pvz_http "github.com/Staspol216/gh1/internal/handlers/http"
	pvz_repository "github.com/Staspol216/gh1/internal/repository/storage"
	pvz_service "github.com/Staspol216/gh1/internal/service"
	"github.com/davecgh/go-spew/spew"
	"github.com/joho/godotenv"
)

type Handler interface {
	Serve(ctx context.Context) error
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

	go func() {
		for res := range results {
			spew.Dump("Result: \n", res)
		}
	}()

	wg.Add(workersCount)
	for i := 0; i < workersCount; i++ {
		worker := &pvz_worker.Worker{Context: sigCtx, In: jobs, Out: results, Wg: wg}
		go worker.Run(i)
	}

	postgresConfig := &pvz_repository.Config{
		StorageType: pvz_repository.StorageTypePostgres,
		Postgres: &pvz_repository.PostgresConfig{
			Context: sigCtx,
		},
	}

	orderStorage, err := pvz_repository.NewStorage(postgresConfig)

	if err != nil {
		log.Fatal("pvz.New: %w", err)
	}

	pvzService := pvz_service.New(orderStorage)

	var handler Handler

	if isHTTP {
		handler = pvz_http.New(pvzService, jobs)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := handler.Serve(sigCtx); err != nil {
				log.Printf("server shutdown error: %v", err)
			}
		}()
	} else {
		handler = pvz_cli.New(pvzService)
		handler.Serve(sigCtx)
	}

	wg.Wait()

	close(results)
	close(jobs)

	fmt.Println("Main done")
}
