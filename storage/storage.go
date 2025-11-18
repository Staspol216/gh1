package storage

import (
	"errors"

	"github.com/Staspol216/gh1/models/order"
	inmemory "github.com/Staspol216/gh1/repository"
)

type StorageType string

const (
	StorageTypePostgres StorageType = "pg"
	StorageTypeInmemory StorageType = "inmemory"
)

type InMemoryConfig struct {
	Path string
}

type Config struct {
	StorageType StorageType
	Inmemory    *InMemoryConfig
}

type Storager interface {
	GetOrders() []*order.Order
	SaveOrder(newOrder *order.Order)
	DeleteOrderById(orderId int64)
	FindOrderById(orderId int64) (*order.Order, bool)
	FindRecipientOrdersByIds(orderIds []int64, recipientId int64) []*order.Order
	SaveStorageToFile() error
}

func New(cfg *Config) (Storager, error) {
	switch cfg.StorageType {
	case StorageTypePostgres:
		return inmemory.New(cfg.Inmemory.Path)
	case StorageTypeInmemory:
		return inmemory.New(cfg.Inmemory.Path)
	default:
		return nil, errors.New("unknown storage type")
	}
}
