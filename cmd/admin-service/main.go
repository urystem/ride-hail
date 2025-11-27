package main

import (
	"fmt"
	"taxi-hailing/config"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(*cfg)
}
