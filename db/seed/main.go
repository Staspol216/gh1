package main

import (
	"context"
	"log"

	"github.com/Staspol216/gh1/db"
	"github.com/Staspol216/gh1/repository/postgresql"
	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Fatal("no .env file loaded")
	}

	context, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, dbErr := db.NewDb(context)

	if dbErr != nil {
		log.Fatal(dbErr)
	}

	repo, repoErr := postgresql.NewOrderRepo(db, context)

	if repoErr != nil {
		log.Fatal(repoErr)
	}

	repo.SeedOrders()

}
