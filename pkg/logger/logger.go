package app_logger

import (
	"log"

	"go.uber.org/zap"
)

var MyLogger *zap.Logger

func init() {
	var err error
	MyLogger, err = zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
}
