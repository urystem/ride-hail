package main

import (
	"fmt"
	"taxi-hailing/pkg"
)

func main() {
	cfg, err := pkg.ParseConfig()
	fmt.Println(cfg, err)
}
