package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hongminglow/all-in-be/internal/config"
	"github.com/hongminglow/all-in-be/internal/server"
	postgres "github.com/hongminglow/all-in-be/internal/storage/postgres"
	"github.com/joho/godotenv"
)

func main() {
	loadLocalEnv()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	userStore, err := postgres.NewUserStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("init database: %v", err)
	}
	defer userStore.Close()

	srv := server.New(cfg, userStore)

	go func() {
		log.Printf("ALL-IN backend listening on %s", cfg.HTTPAddress())
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Printf("graceful shutdown error: %v", err)
	}
}

func loadLocalEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found; relying on existing environment")
	}
}
