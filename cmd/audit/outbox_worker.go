package pvz_worker_audit

import (
	"context"
	"errors"
	"log"
	"time"

	pvz_domain "github.com/Staspol216/gh1/internal/domain/order"
	"github.com/Staspol216/gh1/internal/infrastructure/tx_manager"
	"github.com/jackc/pgx/v4"
)

type OutboxWorker struct {
	Context   context.Context
	Repo      AuditLogRepo
	Tasks     chan<- *pvz_domain.OrderOutboxTask
	TxManager *tx_manager.TxManager
}

func (w *OutboxWorker) Run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.Context.Done():
			log.Println("Outbox worker finshed by context done")
			return
		case <-ticker.C:
			w.processTask()
		}
	}
}

func (w *OutboxWorker) processTask() {

	var task *pvz_domain.OrderOutboxTask

	txError := w.TxManager.RunRepeatableRead(func(ctxTx context.Context) error {
		t, err := w.Repo.GetNewestUnprocessedTask(w.Context)

		if err != nil {
			return err
		}

		if err := w.Repo.MarkTaskAsProcessing(w.Context, *t.ID); err != nil {
			log.Printf("MarkTaskAsProcessing: %s", err)
		}

		task = t
		return nil
	})

	if txError != nil {
		if errors.Is(txError, pgx.ErrNoRows) {
			log.Printf("There are no tasks for sending to broker")
			return
		}

		log.Printf("failed to fetch tasks: %v", txError)
		return
	}

	w.Tasks <- task
}
