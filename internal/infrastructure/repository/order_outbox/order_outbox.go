package psql_order_outbox_repo

import (
	"context"
	"log"

	pvz_domain "github.com/Staspol216/gh1/internal/domain/order"
	db "github.com/Staspol216/gh1/internal/infrastructure/postgres"
)

type OrderOutboxRepo struct {
	Db db.DB
}

func (r *OrderOutboxRepo) AddTask(ctx context.Context, auditLog *pvz_domain.OrderOutboxTask) (int64, error) {
	query := `
	INSERT INTO orders_statuses_outbox (
		status,
		created_at,
		order_status,
		description,
		timestamp
	) VALUES ($1, $2, $3, $4, $5) RETURNING id;`

	row := r.Db.ExecQueryRow(ctx, query,
		auditLog.Status,
		auditLog.CreatedAt,
		auditLog.OrderStatus,
		auditLog.Description,
		auditLog.Timestamp,
	)

	var id int64
	err := row.Scan(&id)
	if err != nil {
		log.Println(err)
	}
	return id, err
}

func (r *OrderOutboxRepo) LockPending(ctx context.Context) ([]pvz_domain.OrderOutboxTask, error) {
	var tasks []pvz_domain.OrderOutboxTask

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

	err := r.Db.Select(ctx, &tasks, query)

	if err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *OrderOutboxRepo) MarkTaskAsFailed(ctx context.Context, id int64) error {
	_, err := r.Db.Exec(ctx, `
    	UPDATE orders_statuses_outbox
        SET status = 'failed'
        WHERE id = $1;
    `, id)

	return err
}

func (r *OrderOutboxRepo) DeleteTasks(ctx context.Context, ids []int64) error {

	for _, id := range ids {
		err := r.DeleteTask(ctx, id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *OrderOutboxRepo) DeleteTask(ctx context.Context, id int64) error {
	_, err := r.Db.Exec(ctx, `DELETE FROM orders_statuses_outbox WHERE id = $1;`, id)

	return err
}
