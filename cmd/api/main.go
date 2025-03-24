package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Postgres connection string
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	// Create connection pool
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	fmt.Println("Connected to Postgres")

	//// Initialize repositories
	//accountRepo := account_repository_pg.NewAccountRepositoryPG(pool)
	//apiKeyRepo := apikey_repository_pg.NewAPIKeyRepositoryPG(pool)
	//eventRepo := event_repository_pg.NewEventRepositoryPG(pool)
	//
	//accountSerice = accout_service.NewAccountService(accountRepo)
	//apiKeyService = apikey_service.NewAPIKeyService(apiKeyRepo)
	//eventService = event_service.NewEventService(eventRepo)

}
