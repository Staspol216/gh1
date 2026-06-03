//go:generate mockgen -source=outbox.go -destination=mocks/outbox.go -package=mocks

package pvz_order_service

import (
	"context"

	"github.com/Staspol216/gh1/internal/infra/order_outbox"
)

type Outbox interface {
	AddTask(ctx context.Context, task *order_outbox.OrderOutboxTask) (int64, error)
	LockPending(ctx context.Context) ([]order_outbox.OrderOutboxTask, error)
	MarkTaskAsFailed(ctx context.Context, id int64) error
	DeleteTasks(ctx context.Context, ids []int64) error
	DeleteTask(ctx context.Context, id int64) error
}
