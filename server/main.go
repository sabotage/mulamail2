package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"mulamail/api"
	"mulamail/blockchain"
	"mulamail/config"
	"mulamail/db"
	"mulamail/vault"
)

func main() {
	cfg := config.Load()

	// MongoDB
	dbClient, err := db.Connect(cfg.MongoURI, cfg.MongoDBName)
	if err != nil {
		log.Fatalf("MongoDB connect: %v", err)
	}
	defer dbClient.Close()

	// Solana RPC
	solanaClient := blockchain.NewClient(cfg.SolanaRPC)

	// Storage (local or S3)
	var storage vault.Storage
	switch cfg.StorageType {
	case "s3":
		log.Printf("Using S3 storage: region=%s bucket=%s", cfg.AWSRegion, cfg.S3Bucket)
		s3Client, err := vault.NewS3Client(cfg.AWSRegion, cfg.S3Bucket)
		if err != nil {
			log.Fatalf("S3 init: %v", err)
		}
		storage = s3Client
	case "local":
		log.Printf("Using local storage: path=%s", cfg.LocalDataPath)
		localStorage, err := vault.NewLocalStorage(cfg.LocalDataPath)
		if err != nil {
			log.Fatalf("Local storage init: %v", err)
		}
		storage = localStorage
	default:
		log.Fatalf("Invalid storage type: %s (must be 'local' or 's3')", cfg.StorageType)
	}

	// HTTP server
	mux := api.NewRouter(dbClient, solanaClient, storage, cfg)
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	// Graceful shutdown on SIGINT / SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("MulaMail server listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down…")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Println("stopped")
}
