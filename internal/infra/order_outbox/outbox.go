package order_outbox

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/Staspol216/gh1/internal/ports"
	"github.com/jackc/pgx/v4"
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
			log.Println("Outbox worker finished by context done")
			return
		case <-ticker.C:
			tasks, err := w.LockPending(ctx)

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
		log.Println(err)
	}
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

	return err
}
