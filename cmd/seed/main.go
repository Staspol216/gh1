package main

import (
	"context"
	"log"

	db "github.com/Staspol216/gh1/internal/infrastructure/postgres"
	psql_order_repo "github.com/Staspol216/gh1/internal/infrastructure/repository/order/postgres"
	"github.com/Staspol216/gh1/internal/infrastructure/tx_manager"
)

func main() {

	context, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.NewPool(context)

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
