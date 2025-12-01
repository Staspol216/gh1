package main

import (
	"context"
	"log"

	pvzApp "github.com/Staspol216/gh1/app"
	PvzSerivce "github.com/Staspol216/gh1/service"
	"github.com/Staspol216/gh1/storage"
	"github.com/joho/godotenv"
)

func main() {
	// config := &storage.Config{
	// 	StorageType: storage.StorageTypeInmemory,
	// 	Inmemory: &storage.InMemoryConfig{
	// 		Path: "data/pvz.json",
	// 	},
	// }

	if err := godotenv.Load(); err != nil {
		log.Println("no .env file loaded")
	}

	context, cancel := context.WithCancel(context.Background())
	defer cancel()

	postgresConfig := &storage.Config{
		StorageType: storage.StorageTypePostgres,
		Postgres: &storage.PostgresConfig{
			Context: context,
		},
	}

	orderStorage, err := storage.NewStorage(postgresConfig)

	if err != nil {
		log.Fatal("pvz.New: %w", err)
	}

	pvzService := PvzSerivce.New(orderStorage)

	app := pvzApp.New(pvzService)

	app.Run()
}
