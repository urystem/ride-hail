package main

import (
	"context"
	"os"
	"taxi-hailing/config"
	"taxi-hailing/intenal/broker"
	"taxi-hailing/intenal/repo"
	"taxi-hailing/pkg"
)

func main() {
	slogger := pkg.CustomSlog("driver-service")
	cfg, err := config.LoadConfig()
	if err != nil {
		slogger.Error("cannot parse config", "action", "parse config", "error", err)
		os.Exit(1)
	}
	pool, err := pkg.NewDB(context.Background(), &cfg.DatabaseCfg)
	if err != nil {
		slogger.Error("cannot create connection to db", "action", "connect to db", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	db := repo.NewDriverRepo(pool)
	rabbit, err := broker.NewDriverRabbit(cfg.RabbitMQCfg, slogger)
	if err != nil {
		slogger.Error("cannot create connection to rabbitMQ", "action", "connect to rabbitMQ", "error", err)
		os.Exit(1)
	}
	defer rabbit.CloseRabbit()
}

