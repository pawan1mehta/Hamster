package main

import (
	"containerdgrassland/server"
	"fmt"
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

	err = server.StartContainerdGrasslandServer(config)
	if err != nil {
		fmt.Printf("error while starting the containerd-grassland-server! %v", err)
	}
}
