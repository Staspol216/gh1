package pvz_repository

import (
	"context"
	"errors"
	"log"

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
	Context context.Context
}

type Config struct {
	StorageType StorageType
	Inmemory    *InMemoryConfig
	Postgres    *PostgresConfig
}

type Storager interface {
	GetList(pagination *pvz_model.Pagination) []*pvz_model.Order
	Add(newOrder *pvz_model.Order) (int64, error)
	AddHistoryRecord(record *pvz_model.OrderRecord, orderId int64) (int64, error)
	Delete(orderId int64) error
	Update(updatedOrder *pvz_model.Order) error
	GetByID(orderId int64) (*pvz_model.Order, error)
	GetByIDs(orderIds []int64) ([]*pvz_model.Order, error)
	GetByRecipientId(recipientId int64) ([]*pvz_model.Order, error)
}

func NewStorage(cfg *Config) (Storager, *db.Database, error) {
	switch cfg.StorageType {
	case StorageTypeInmemory:
		repo, err := inmemory.NewOrderRepo(cfg.Inmemory.Path)
		return repo, nil, err
	case StorageTypePostgres:
		db, err := db.NewDb(cfg.Postgres.Context)
		if err != nil {
			log.Fatal(err)
		}
		repo, err := postgresql.NewOrderRepo(db, cfg.Postgres.Context)
		if err != nil {
			return nil, nil, err
		}
		return repo, db, nil
	default:
		return nil, nil, errors.New("unknown storage type")
	}
}
