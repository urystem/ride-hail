package main

import (
	"context"
	"os"
	"taxi-hailing/pkg"
)

func main() {
	slogger := pkg.CustomSlog("driver-service")
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
	
}
