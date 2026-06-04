package main

import (
	"containerdgrassland/server"
	"log"

	"go.uber.org/zap"
)

func main() {

	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize log")
		return
	}

	config := &server.Config{
		Logger:  zapLogger,
		Address: "6969",
	}

	server.StartContainerdGrasslandServer(config)
}
