package pvz_worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	pvz_http "github.com/Staspol216/gh1/internal/handlers/http"
)

type ProcessStrategy = string

const (
	Print    ProcessStrategy = "print"
	SaveToDB ProcessStrategy = "saveToDB"
)

type Worker struct {
	ProcessStrategyType ProcessStrategy
	Context             context.Context
	In                  <-chan *pvz_http.AuditLog
	Out                 chan *pvz_http.AuditLog
	Wg                  *sync.WaitGroup
}

func (w *Worker) RunAndServe(index int) {
	w.Wg.Add(2)

	go w.Run(index)
	go w.Serve(index)
}

func (w *Worker) Run(index int) {
	defer w.Wg.Done()

	var timer *time.Timer
	var timeout <-chan time.Time

	const batchCapacity = 5
	batch := make([]*pvz_http.AuditLog, 0, batchCapacity)

	fmt.Printf("Worker %d started\n", index)

	for {
		select {
		case <-w.Context.Done():
			if timer != nil {
				timer.Stop()
				timeout = nil
				timer = nil
			}
			w.work(batch)
			fmt.Printf("Worker %d finished\n", index)
			return
		case <-timeout:
			fmt.Printf("Worker %d done the jobs after timeout\n", index)
			batch = w.work(batch)
			timer = nil
			timeout = nil
		case v := <-w.In:
			fmt.Printf("Worker %d took the job %s\n", index, v.RequestID)

			batch = append(batch, v)
			fmt.Println(len(batch), "len(batch)")

			if len(batch) >= batchCapacity {
				if timer != nil {
					timer.Stop()
					timeout = nil
					timer = nil
				}
				batch = w.work(batch)
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

func (w *Worker) Serve(index int) {
	defer w.Wg.Done()

	fmt.Printf("Serve worker %d started\n", index)

	for {
		select {
		case <-w.Context.Done():
			fmt.Printf("Serve worker %d finished\n", index)
			return
		case j := <-w.Out:
			fmt.Printf("Serve worker %d get job for process\n", index)
			w.proccess(j)
		}
	}
}

func (w *Worker) proccess(job *pvz_http.AuditLog) {
	switch w.ProcessStrategyType {
	case Print:
		w.print(job)
	case SaveToDB:
		w.saveToDB(job)
	}
}

func (w *Worker) print(job *pvz_http.AuditLog) {
	fmt.Printf("Result %#v\n", job)
}

func (w *Worker) saveToDB(job *pvz_http.AuditLog) {
	fmt.Printf("Result %#v\n", job)
}

func (w *Worker) do(job *pvz_http.AuditLog) {
	w.Out <- job
}

func (w *Worker) work(batch []*pvz_http.AuditLog) []*pvz_http.AuditLog {
	count := 0

	for _, job := range batch {
		w.do(job)
		count++
	}

	return batch[count:]
}
