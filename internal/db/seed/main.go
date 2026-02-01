package main

import (
	"context"
	"log"

	"github.com/Staspol216/gh1/internal/db"
	"github.com/Staspol216/gh1/internal/repository/postgresql"
	"github.com/Staspol216/gh1/internal/repository/tx_manager"
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

	repo, repoErr := postgresql.NewOrderRepo(db, context)

	if repoErr != nil {
		log.Fatal(repoErr)
	}

	repo.SeedOrders(context)

}
