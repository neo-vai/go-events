package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/neo-vai/go-events/internal/handler"
	accountHandler "github.com/neo-vai/go-events/internal/handler/account"
	adminHandler "github.com/neo-vai/go-events/internal/handler/admin"
	apikeyHandler "github.com/neo-vai/go-events/internal/handler/apikey"
	authHandler "github.com/neo-vai/go-events/internal/handler/auth"
	eventHandler "github.com/neo-vai/go-events/internal/handler/event"

	account_repository_pg "github.com/neo-vai/go-events/internal/repository/account/postgres"
	apikey_repository_pg "github.com/neo-vai/go-events/internal/repository/apikey/postgres"
	event_repository_pg "github.com/neo-vai/go-events/internal/repository/event/postgres"

	account_service "github.com/neo-vai/go-events/internal/service/account"
	apikey_service "github.com/neo-vai/go-events/internal/service/apikey"
	event_service "github.com/neo-vai/go-events/internal/service/event"

	_ "github.com/neo-vai/go-events/docs"
)

// @title           Event Tracking API
// @version         1.0
// @description     API for tracking events with JWT authentication
// @BasePath        /api/v1
//
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 JWT token (format: "Bearer <token>")
//
// @securityDefinitions.apikey  ApiKeyAuth
// @in                          header
// @name                        X-API-Key
// @description                 API key for authentication
func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	accountRepo := account_repository_pg.NewAccountRepositoryPG(pool)
	apiKeyRepo := apikey_repository_pg.NewAPIKeyRepositoryPG(pool)
	eventRepo := event_repository_pg.NewEventRepositoryPG(pool)

	passwordHasher := account_service.NewBcryptHasher(0)
	accountService := account_service.NewAccountService(accountRepo, passwordHasher)
	apiKeyService := apikey_service.NewAPIKeyService(apiKeyRepo)
	eventService := event_service.NewEventService(eventRepo)
	authService := accountService

	accountH := accountHandler.NewHandler(accountService)
	apiKeyH := apikeyHandler.NewHandler(apiKeyService)
	eventH := eventHandler.NewHandler(eventService)
	authH := authHandler.NewHandler(authService)

	// For admin handlers, we need services with admin capabilities.
	// We'll reuse the same services but they need to implement admin interfaces.
	// We'll create a stats service (simple implementation).
	statsSvc := &simpleStatsService{db: pool}
	adminH := adminHandler.NewHandler(accountService, eventService, apiKeyService, statsSvc)

	router := handler.NewRouter(handler.Handlers{
		Account: accountH,
		APIKey:  apiKeyH,
		Event:   eventH,
		Auth:    authH,
		Admin:   adminH,
	}, apiKeyService, accountService)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Server started on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited properly")
}

// simpleStatsService implements admin.StatsService
type simpleStatsService struct {
	db *pgxpool.Pool
}

func (s *simpleStatsService) GetStats(ctx context.Context) (*adminHandler.StatsResponse, error) {
	var stats adminHandler.StatsResponse
	err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM accounts").Scan(&stats.TotalAccounts)
	if err != nil {
		return nil, err
	}
	err = s.db.QueryRow(ctx, "SELECT COUNT(*) FROM accounts WHERE active = true").Scan(&stats.ActiveAccounts)
	if err != nil {
		return nil, err
	}
	err = s.db.QueryRow(ctx, "SELECT COUNT(*) FROM events").Scan(&stats.TotalEvents)
	if err != nil {
		return nil, err
	}
	err = s.db.QueryRow(ctx, "SELECT COUNT(*) FROM events WHERE created_at >= CURRENT_DATE").Scan(&stats.EventsToday)
	if err != nil {
		return nil, err
	}
	err = s.db.QueryRow(ctx, "SELECT COUNT(*) FROM api_keys").Scan(&stats.TotalAPIKeys)
	if err != nil {
		return nil, err
	}
	err = s.db.QueryRow(ctx, "SELECT COUNT(*) FROM api_keys WHERE active = true").Scan(&stats.ActiveAPIKeys)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}
