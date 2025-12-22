package pvz_worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	pvz_http "github.com/Staspol216/gh1/internal/handlers/http"
)

type Worker struct {
	Context context.Context
	In      <-chan *pvz_http.AuditLog
	Out     chan<- *pvz_http.AuditLog
	Wg      *sync.WaitGroup
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
			fmt.Printf("Worker %d finished\n", index)
			return
		case <-timeout:
			fmt.Printf("Worker %d done the jobs after timeout\n", index)
			batch = w.Work(batch)
			timer = nil
			timeout = nil
		case v := <-w.In:
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

func (w *Worker) Work(batch []*pvz_http.AuditLog) []*pvz_http.AuditLog {
	count := 0

	for _, job := range batch {
		w.Out <- job
		count++
	}

	return batch[count:]
}
