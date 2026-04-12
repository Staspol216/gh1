package main

import (
	"context"
	"log"

	pvz_config "github.com/Staspol216/gh1/internal/config"
	db "github.com/Staspol216/gh1/internal/infrastructure/postgres"
	psql_order_repo "github.com/Staspol216/gh1/internal/infrastructure/repository/order/postgres"
	"github.com/Staspol216/gh1/internal/infrastructure/tx_manager"
	"github.com/jackc/pgx/v4/pgxpool"
)

func main() {

	cfg, err := pvz_config.Load()

	context, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.Connect(context, cfg.DBConnString())

	if err != nil {
		log.Fatal(err)
	}

	txManager := tx_manager.New(pool, context)

	db := db.NewDatabase(txManager)

	repo, repoErr := psql_order_repo.New(db, context)

	if repoErr != nil {
		log.Fatal(repoErr)
	}

	repo.SeedOrders(context)

}
