package main

import (
	"context"
	"log"

	db "github.com/Staspol216/gh1/internal/db/postgres"
	"github.com/Staspol216/gh1/internal/db/tx_manager"
	psql_order_repo "github.com/Staspol216/gh1/internal/repository/order/postgres"
	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Fatal("no .env file loaded")
	}

	context, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.NewPool(context)

	if err != nil {
		log.Fatal(err)
	}

	txManager := tx_manager.New(pool, context)

	db := db.NewDatabase(txManager)

	repo, repoErr := psql_order_repo.NewOrderRepo(db, context)

	if repoErr != nil {
		log.Fatal(repoErr)
	}

	repo.SeedOrders(context)

}
