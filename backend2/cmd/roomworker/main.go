package main

import (
	"backend/internal/roomworker"
	"context"
	"log"
	"os/signal"
	"syscall"
)

func main() {
	cfg, err := roomworker.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	worker, err := roomworker.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer worker.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := worker.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
