package main

import (
	"os"

	cmd "github.com/MrSnakeDoc/keg/internal"
	"github.com/MrSnakeDoc/keg/internal/logger"
)

func main() {
	if err := cmd.Execute(); err != nil {
		logger.LogError("%v", err)
		os.Exit(1)
	}
}
