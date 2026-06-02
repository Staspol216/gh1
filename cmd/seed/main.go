package main

import (
	"context"
	"log"

	"github.com/Staspol216/gh1/internal/config"
	"github.com/Staspol216/gh1/internal/infrastructure/postgres"
	"github.com/Staspol216/gh1/internal/infrastructure/repository/order/postgres"
	"github.com/Staspol216/gh1/internal/infrastructure/tx_manager"
	"github.com/jackc/pgx/v4/pgxpool"
)

func main() {

	cfg, err := pvz_config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.Connect(ctx, cfg.DBConnString())

	if err != nil {
		log.Fatal(err)
	}

	txManager := tx_manager.New(pool, ctx)

	database := db.NewDatabase(txManager)

	repo, repoErr := order_repo.New(database)

	if repoErr != nil {
		log.Fatal(repoErr)
	}

	repo.SeedOrders(ctx)

}
