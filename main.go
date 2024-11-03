package main

import (
	"context"
	"grind/services"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Setup signal handling for graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create channels
	tokenChan := make(chan services.RaydiumPair, 100)

	// Start services
	go services.ProcessNewTokens(tokenChan)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")
}
