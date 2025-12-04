package main

import (
	"context"
	"log"

	"github.com/Staspol216/gh1/handlers/cli"
	"github.com/Staspol216/gh1/handlers/http"
	PvzSerivce "github.com/Staspol216/gh1/service"
	"github.com/Staspol216/gh1/storage"
	"github.com/joho/godotenv"
)

type Handler interface {
	Serve()
}

func main() {
	// config := &storage.Config{
	// 	StorageType: storage.StorageTypeInmemory,
	// 	Inmemory: &storage.InMemoryConfig{
	// 		Path: "data/pvz.json",
	// 	},
	// }

	isHTTP := true

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

	var handler Handler

	if isHTTP {
		handler = http.New(pvzService)
	} else {
		handler = cli.New(pvzService)
	}

	handler.Serve()
}
