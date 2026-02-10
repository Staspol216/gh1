package pvz_service

import (
	"context"
	"errors"

	db "github.com/Staspol216/gh1/internal/db/postgres"
	pvz_model "github.com/Staspol216/gh1/internal/models/order"
	inmemory_order_repo "github.com/Staspol216/gh1/internal/repository/order/inmemory"
	psql_order_repo "github.com/Staspol216/gh1/internal/repository/order/postgres"
)

type StorageType string

const (
	StorageTypePostgres StorageType = "pg"
	StorageTypeInmemory StorageType = "inmemory"
)

type InMemoryConfig struct {
	Path string
}

type PostgresConfig struct {
	Db      *db.Database
	Context context.Context
}

type Config struct {
	StorageType StorageType
	Inmemory    *InMemoryConfig
	Postgres    *PostgresConfig
}

type Storager interface {
	GetAll(ctx context.Context) ([]*pvz_model.Order, error)
	GetList(ctx context.Context, pagination *pvz_model.Pagination) ([]*pvz_model.Order, error)
	Add(ctx context.Context, newOrder *pvz_model.Order) (int64, error)
	AddHistoryRecord(ctx context.Context, record *pvz_model.OrderRecord, orderId int64) (int64, error)
	Delete(ctx context.Context, orderId int64) error
	Update(ctx context.Context, updatedOrder *pvz_model.Order) error
	GetByID(ctx context.Context, orderId int64) (*pvz_model.Order, error)
	GetByIDs(ctx context.Context, orderIds []int64) ([]*pvz_model.Order, error)
}

func NewStorage(cfg *Config) (Storager, error) {
	switch cfg.StorageType {
	case StorageTypeInmemory:
		return inmemory_order_repo.NewOrderRepo(cfg.Inmemory.Path)
	case StorageTypePostgres:
		return psql_order_repo.NewOrderRepo(cfg.Postgres.Db, cfg.Postgres.Context)
	default:
		return nil, errors.New("unknown storage type")
	}
}
