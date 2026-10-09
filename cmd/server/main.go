package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"Tree_nity/internal/engine"
	"Tree_nity/internal/hashmap"
	"Tree_nity/internal/topic"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	store := topic.New()
	clientMap := hashmap.New()
	server := engine.NewServer(os.Getpid(), store, clientMap)

	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start server: %v\n", err)
		os.Exit(1)
	}

	<-ctx.Done()

	if err := server.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "error stopping server: %v\n", err)
		os.Exit(1)
	}
}
