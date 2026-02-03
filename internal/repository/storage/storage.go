package pvz_repository

import (
	"context"
	"errors"

	"github.com/Staspol216/gh1/internal/db"
	pvz_model "github.com/Staspol216/gh1/internal/models/order"
	"github.com/Staspol216/gh1/internal/repository/inmemory"
	"github.com/Staspol216/gh1/internal/repository/postgresql"
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
	GetList(ctx context.Context, pagination *pvz_model.Pagination) []*pvz_model.Order
	Add(ctx context.Context, newOrder *pvz_model.Order) (int64, error)
	AddHistoryRecord(ctx context.Context, record *pvz_model.OrderRecord, orderId int64) (int64, error)
	Delete(ctx context.Context, orderId int64) error
	Update(ctx context.Context, updatedOrder *pvz_model.Order) error
	GetByID(ctx context.Context, orderId int64, queryOption string) (*pvz_model.Order, error)
	GetByIDs(ctx context.Context, orderIds []int64) ([]*pvz_model.Order, error)
	GetByIdAndRecipientId(ctx context.Context, recipientId int64, orderId int64) (*pvz_model.Order, error)
}

func New(cfg *Config) (Storager, error) {
	switch cfg.StorageType {
	case StorageTypeInmemory:
		return inmemory.NewOrderRepo(cfg.Inmemory.Path)
	case StorageTypePostgres:
		return postgresql.NewOrderRepo(cfg.Postgres.Db, cfg.Postgres.Context)
	default:
		return nil, errors.New("unknown storage type")
	}
}
