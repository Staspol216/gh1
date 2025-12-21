package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Staspol216/gh1/handlers/cli"
	"github.com/Staspol216/gh1/handlers/http"
	PvzSerivce "github.com/Staspol216/gh1/service"
	"github.com/Staspol216/gh1/storage"
	"github.com/davecgh/go-spew/spew"
	"github.com/joho/godotenv"
)

type Handler interface {
	Serve(ctx context.Context) error
}

type Worker struct {
	ctx context.Context
	in  <-chan *http.AuditLog
	out chan<- *http.AuditLog
	wg  *sync.WaitGroup
}

func (w *Worker) Run(index int) {
	defer w.wg.Done()

	var timer *time.Timer
	var timeout <-chan time.Time

	const batchCapacity = 5
	batch := make([]*http.AuditLog, 0, batchCapacity)

	fmt.Printf("Worker %d started\n", index)

	for {
		select {
		case <-w.ctx.Done():
			if timer != nil {
				timer.Stop()
				timeout = nil
				timer = nil
			}
			fmt.Printf("Worker %d finished\n", index)
			return
		case <-timeout:
			fmt.Printf("Worker %d done the jobs after timeout\n", index)
			batch = w.Work(batch)
			timer = nil
			timeout = nil
		case v := <-w.in:
			fmt.Printf("Worker %d took the job %d\n", index, v)

			batch = append(batch, v)
			fmt.Println(len(batch), "len(batch)")

			if len(batch) >= batchCapacity {
				if timer != nil {
					timer.Stop()
					timeout = nil
					timer = nil
				}
				batch = w.Work(batch)
				fmt.Printf("Worker %d done the jobs by reaching batch limit\n", index)
				continue
			}

			if timer == nil {
				timer = time.NewTimer(5 * time.Second)
				timeout = timer.C
			} else {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(5 * time.Second)
				timeout = timer.C
			}
		}
	}
}

func (w *Worker) Work(batch []*http.AuditLog) []*http.AuditLog {
	count := 0

	for _, job := range batch {
		w.out <- job
		count++
	}

	return batch[count:]
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

	jobs := make(chan *http.AuditLog, jobsCount)
	results := make(chan *http.AuditLog, jobsCount)

	go func() {
		for res := range results {
			spew.Dump("Result: \n", res)
		}
	}()

	wg.Add(workersCount)
	for i := 0; i < workersCount; i++ {
		worker := &Worker{sigCtx, jobs, results, wg}
		go worker.Run(i)
	}

	postgresConfig := &storage.Config{
		StorageType: storage.StorageTypePostgres,
		Postgres: &storage.PostgresConfig{
			Context: sigCtx,
		},
	}

	orderStorage, err := storage.NewStorage(postgresConfig)

	if err != nil {
		log.Fatal("pvz.New: %w", err)
	}

	pvzService := PvzSerivce.New(orderStorage)

	var handler Handler

	if isHTTP {
		handler = http.New(pvzService, jobs)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := handler.Serve(sigCtx); err != nil {
				log.Printf("server shutdown error: %v", err)
			}
		}()
	} else {
		handler = cli.New(pvzService)
		handler.Serve(sigCtx)
	}

	wg.Wait()

	close(results)
	close(jobs)

	fmt.Println("Main done")
}
