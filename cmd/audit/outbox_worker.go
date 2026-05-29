package pvz_worker_audit

import (
	"context"
	"errors"
	"log"
	"time"

	pvz_domain "github.com/Staspol216/gh1/internal/domain/order"
	"github.com/jackc/pgx/v4"
)

type OutboxWorker struct {
	Repo  AuditLogRepo
	Tasks chan<- []pvz_domain.OrderOutboxTask
}

func (w *OutboxWorker) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Outbox worker finshed by context done")
			return
		case <-ticker.C:
			tasks, err := w.Repo.LockPending(ctx)

			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					log.Printf("There are no tasks for sending to broker")
					return
				}

				log.Printf("failed to fetch tasks: %v", err)
				return
			}

			w.Tasks <- tasks
		}
	}
}
