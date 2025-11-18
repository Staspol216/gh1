package main

import (
	"log"

	pvzApp "github.com/Staspol216/gh1/app"
	PvzSerivce "github.com/Staspol216/gh1/service"
	"github.com/Staspol216/gh1/storage"
)

func main() {
	config := &storage.Config{
		StorageType: storage.StorageTypeInmemory,
		Inmemory: &storage.InMemoryConfig{
			Path: "data/pvz.json",
		},
	}

	orderStorage, strError := storage.New(config)

	if strError != nil {
		log.Fatal("pvz.New: %w", strError)
	}

	pvzService := PvzSerivce.New(orderStorage)

	app := pvzApp.New(pvzService)
	app.Run()
}
