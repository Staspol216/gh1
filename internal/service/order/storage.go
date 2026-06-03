//go:generate mockgen -source=storage.go -destination=mocks/storage.go -package=mocks

package pvz_order_service

import (
	"context"
	"time"

	"github.com/Staspol216/gh1/internal/domain/order"
)

// OrderStorage defines storage operations for orders used across the application.
// Placing this in the domain layer keeps the dependency direction inward.
type OrderStorage interface {
	GetAll(ctx context.Context) ([]*pvz_domain.Order, error)
	GetList(ctx context.Context, pagination *pvz_domain.Pagination) ([]*pvz_domain.Order, error)
	Add(ctx context.Context, newOrder *pvz_domain.Order) (int64, error)
	AddHistoryRecord(ctx context.Context, record *pvz_domain.OrderRecord, orderId int64) (int64, error)
	Delete(ctx context.Context, orderId int64) error
	Update(ctx context.Context, updatedOrder *pvz_domain.Order) error
	GetByID(ctx context.Context, orderId int64) (*pvz_domain.Order, error)
	GetByIDs(ctx context.Context, orderIds []int64) ([]*pvz_domain.Order, error)
}

type OrdersCache interface {
	GetOrder(ctx context.Context, id interface{}) (*pvz_domain.Order, error)
	SetOrder(ctx context.Context, order *pvz_domain.Order, ttl time.Duration) error
	DeleteOrder(ctx context.Context, orderId int64) error
}
