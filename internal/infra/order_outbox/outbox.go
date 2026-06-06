package order_outbox

import (
	"context"
	"errors"
	"time"

	"github.com/Staspol216/gh1/internal/ports"
	"github.com/Staspol216/gh1/pkg/logger"
	"github.com/Staspol216/gh1/pkg/monitoring"
	"github.com/jackc/pgx/v4"
	"go.uber.org/zap"
)

type OrderOutbox struct {
	Db    pvz_ports.DB
	Tasks chan<- []OrderOutboxTask
}

func (w *OrderOutbox) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			app_logger.MyLogger.Info("outbox worker finished by context done")
			return
		case <-ticker.C:
			tasks, err := w.LockPending(ctx)

			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					app_logger.MyLogger.Info("there are no tasks for sending to broker")
					monitoring.ObserveOutboxBatch("empty", 0)
					return
				}

				app_logger.MyLogger.Error("failed to fetch outbox tasks", zap.Error(err))
				monitoring.ObserveOutboxBatch("error", 0)
				return
			}

			monitoring.ObserveOutboxBatch("success", len(tasks))
			w.Tasks <- tasks
		}
	}
}

func (w *OrderOutbox) AddTask(ctx context.Context, task *OrderOutboxTask) (int64, error) {
	query := `
	INSERT INTO orders_statuses_outbox (
		status,
		created_at,
		order_status,
		description,
		timestamp
	) VALUES ($1, $2, $3, $4, $5) RETURNING id;`

	row := w.Db.ExecQueryRow(ctx, query,
		task.Status,
		task.CreatedAt,
		task.OrderStatus,
		task.Description,
		task.Timestamp,
	)

	var id int64
	err := row.Scan(&id)
	if err != nil {
		app_logger.MyLogger.Error("add outbox task", zap.Error(err))
	}
	monitoring.ObserveOutboxTask("add", err)
	return id, err
}

func (w *OrderOutbox) LockPending(ctx context.Context) ([]OrderOutboxTask, error) {
	var tasks []OrderOutboxTask

	query := `WITH picked AS (
		SELECT * FROM orders_statuses_outbox 
		WHERE status = 'created'
		ORDER BY created_at 
		LIMIT 100 
		FOR UPDATE SKIP LOCKED
	)
	UPDATE orders_statuses_outbox AS o
	SET status = 'processing'
	FROM picked
	WHERE o.id = picked.id
	RETURNING
		o.id,
		o.status,
		o.created_at;
	`

	err := w.Db.Select(ctx, &tasks, query)

	if err != nil {
		return nil, err
	}

	return tasks, nil
}

func (w *OrderOutbox) MarkTaskAsFailed(ctx context.Context, id int64) error {
	_, err := w.Db.Exec(ctx, `
    	UPDATE orders_statuses_outbox
        SET status = 'failed'
        WHERE id = $1;
    `, id)

	monitoring.ObserveOutboxTask("mark_failed", err)
	return err
}

func (w *OrderOutbox) DeleteTasks(ctx context.Context, ids []int64) error {

	for _, id := range ids {
		err := w.DeleteTask(ctx, id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *OrderOutbox) DeleteTask(ctx context.Context, id int64) error {
	_, err := w.Db.Exec(ctx, `DELETE FROM orders_statuses_outbox WHERE id = $1;`, id)

	monitoring.ObserveOutboxTask("delete", err)
	return err
}
