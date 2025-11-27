package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"taxi-hailing/intenal/broker"
	"taxi-hailing/intenal/repo"
	"taxi-hailing/intenal/server"
	"taxi-hailing/intenal/service"
	"taxi-hailing/intenal/ws"
	"taxi-hailing/pkg"
)

func main() {
	slogger := pkg.CustomSlog("ride-service")
	cfg, err := pkg.ParseConfig()
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
	db := repo.NewRideRepo(pool)
	rabbit, err := broker.NewRideRabbit(cfg.RabbitMQCfg, slogger)
	if err != nil {
		slogger.Error("cannot create connection to rabbitMQ", "action", "connect to rabbitMQ", "error", err)
		os.Exit(1)
	}
	defer rabbit.CloseRabbit()
	ws := ws.NewWebSocket(slogger, cfg.WebSocketCfg.Port, db)
	myService := service.NewRideService(context.Background(), slogger, db, rabbit, ws)
	myServer := server.NewRideServer(cfg.RideService, cfg.ServicesCfg.Secret, myService)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		slogger.Info("starting the server", "action", "start the server")
		err := myServer.StartServer()
		slog.Error("server stopped", "error", err)
		quit <- nil
	}()
	<-quit
	myServer.ShutDownServer(context.Background())
}
