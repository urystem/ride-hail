package main

import (
	"context"
	"fmt"
	"taxi-hailing/pkg"
)

func main() {
	cfg, err := pkg.ParseConfig()
	if err != nil {

	}

	pool, err := pkg.NewDB(context.Background(), &cfg.DatabaseCfg)
	if err != nil {
	}

}
func fff() {
	fmt.Println()
}
