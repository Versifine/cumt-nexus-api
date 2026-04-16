package main

import (
	"fmt"
	"os"

	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
)

func main() {

	fmt.Println("项目初始化")
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Configuration loaded successfully: %+v\n", cfg)
}
