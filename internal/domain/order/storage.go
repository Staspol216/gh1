package pvz_domain

import "context"

// Storager defines storage operations for orders used across the application.
// Placing this in the domain layer keeps the dependency direction inward.
type OrderStorager interface {
	GetAll(ctx context.Context) ([]*Order, error)
	GetList(ctx context.Context, pagination *Pagination) ([]*Order, error)
	Add(ctx context.Context, newOrder *Order) (int64, error)
	AddHistoryRecord(ctx context.Context, record *OrderRecord, orderId int64) (int64, error)
	Delete(ctx context.Context, orderId int64) error
	Update(ctx context.Context, updatedOrder *Order) error
	GetByID(ctx context.Context, orderId int64) (*Order, error)
	GetByIDs(ctx context.Context, orderIds []int64) ([]*Order, error)
}
