package storage

import (
	"context"
	"errors"
	"log"

	"github.com/Staspol216/gh1/db"
	common "github.com/Staspol216/gh1/models"
	"github.com/Staspol216/gh1/models/order"
	"github.com/Staspol216/gh1/repository/inmemory"
	"github.com/Staspol216/gh1/repository/postgresql"
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
	GetList(pagination *common.Pagination) []*order.Order
	Add(newOrder *order.Order) (int64, error)
	AddHistoryRecord(record *order.OrderRecord, orderId int64) (int64, error)
	Delete(orderId int64) error
	Update(updatedOrder *order.Order) error
	GetByID(orderId int64) (*order.Order, error)
	GetByRecipientId(recipientId int64) ([]*order.Order, error)
}

func NewStorage(cfg *Config) (Storager, error) {
	switch cfg.StorageType {
	case StorageTypeInmemory:
		return inmemory.NewOrderRepo(cfg.Inmemory.Path)
	case StorageTypePostgres:
		db, err := db.NewDb(cfg.Postgres.Context)
		if err != nil {
			log.Fatal(err)
		}
		return postgresql.NewOrderRepo(db, cfg.Postgres.Context)
	default:
		return nil, errors.New("unknown storage type")
	}
}
