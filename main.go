package main

import (
	"log"

	pvzApp "github.com/Staspol216/gh1/app"
	warehouse "github.com/Staspol216/gh1/storage"
)

func main() {
	warehouse, strError := warehouse.New("storage/warehouse.json")

	if strError != nil {
		log.Fatal("warehouse.New: %w", strError)
	}
		
	app := pvzApp.New(warehouse)
	app.Run()
}
