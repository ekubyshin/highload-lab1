package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"highload/lab1/internal/dashboard"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "dashboard-generator: %s\n", err.Error())
		os.Exit(1)
	}
}

func run() error {
	cfg, err := dashboard.ConfigFromEnv(os.Getenv)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	result, err := dashboard.Generate(ctx, nil, cfg)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "dashboard-generator: saved dashboard %s (%d panels)\n", result.UID, result.Panels)
	return nil
}
