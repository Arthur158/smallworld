package main

import (
	"backend/internal/gateway"
	"context"
	"log"
	"os/signal"
	"syscall"
)

func main() {
	cfg, err := gateway.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	g, err := gateway.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := g.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
