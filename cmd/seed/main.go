package main

import (
	"context"

	"github.com/Staspol216/gh1/internal/config"
	"github.com/Staspol216/gh1/internal/infra/postgres"
	"github.com/Staspol216/gh1/internal/infra/repository/order"
	"github.com/Staspol216/gh1/internal/infra/tx_manager"
	"github.com/Staspol216/gh1/pkg/logger"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
)

func main() {
	defer app_logger.MyLogger.Sync()

	cfg, err := pvz_config.Load()
	if err != nil {
		app_logger.MyLogger.Fatal("load config error", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.Connect(ctx, cfg.DBConnString())

	if err != nil {
		app_logger.MyLogger.Fatal("connect to postgres", zap.Error(err))
	}

	txManager := tx_manager.New(pool, ctx)

	database := db.NewDatabase(txManager)

	repo, repoErr := order.NewOrderRepo(database)

	if repoErr != nil {
		app_logger.MyLogger.Fatal("create order repository", zap.Error(repoErr))
	}

	repo.SeedOrders(ctx)

}
