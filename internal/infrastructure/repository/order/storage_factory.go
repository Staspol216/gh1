package pvz_order_storage

import (
	"context"
	"errors"

	pvz_domain "github.com/Staspol216/gh1/internal/domain/order"
	db "github.com/Staspol216/gh1/internal/infrastructure/postgres"
	inmemory_order_repo "github.com/Staspol216/gh1/internal/infrastructure/repository/order/inmemory"
	psql_order_repo "github.com/Staspol216/gh1/internal/infrastructure/repository/order/postgres"
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

func New(cfg *Config) (pvz_domain.OrderStorager, error) {
	switch cfg.StorageType {
	case StorageTypeInmemory:
		return inmemory_order_repo.New(cfg.Inmemory.Path)
	case StorageTypePostgres:
		return psql_order_repo.New(cfg.Postgres.Db, cfg.Postgres.Context)
	default:
		return nil, errors.New("unknown storage type")
	}
}
