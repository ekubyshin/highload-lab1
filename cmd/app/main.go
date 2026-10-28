package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"highload/lab1/internal/config"
	"highload/lab1/internal/httpapi"
	"highload/lab1/internal/item"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "app: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	health := httpapi.NewHealth()
	api, err := httpapi.New(item.NewStore(), health, cfg.MaxBodyBytes)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.Handler(),
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}
	log.Printf("http server starting on %s", cfg.HTTPAddr)
	return httpapi.ListenAndServe(ctx, server, health, cfg.ShutdownTimeout)
}
