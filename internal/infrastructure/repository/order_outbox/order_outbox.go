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

func (r *OrderOutboxRepo) AddTask(ctx context.Context, audit_log *pvz_domain.OrderOutboxTask) (int64, error) {
	query := `INSERT INTO orders_statuses_outbox (
		request_id,
		status,
		created_at,
		updated_at,
		method,
		path,
		remote_address,
        user_agent,
        order_status_details   
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id;`

	row := r.Db.ExecQueryRow(ctx, query,
		audit_log.RequestID,
		audit_log.Status,
		audit_log.CreatedAt,
		audit_log.UpdatedAt,
		audit_log.Method,
		audit_log.Path,
		audit_log.RemoteAddress,
		audit_log.UserAgent,
		audit_log.OrderStatusDetails,
	)

	var id int64
	err := row.Scan(&id)
	if err != nil {
		log.Println(err)
	}
	return id, err
}

func (r *OrderOutboxRepo) GetNewestUnprocessedTask(ctx context.Context) (*pvz_domain.OrderOutboxTask, error) {
	var task pvz_domain.OrderOutboxTask

	err := r.Db.Get(ctx, &task, `SELECT * FROM orders_statuses_outbox WHERE status = 'created' ORDER BY created_at ASC LIMIT 1;`)

	if err != nil {
		return nil, err
	}

	return &task, nil
}

func (r *OrderOutboxRepo) MarkTaskAsProcessing(ctx context.Context, id int64) error {
	_, err := r.Db.Exec(ctx, `
        UPDATE orders_statuses_outbox
        SET status = 'processing', updated_at = NOW()
        WHERE id = $1;
    `, id)

	return err
}

func (r *OrderOutboxRepo) MarkTaskAsFailed(ctx context.Context, id int64) error {
	_, err := r.Db.Exec(ctx, `
        UPDATE orders_statuses_outbox
        SET status = 'failed', updated_at = NOW()
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
