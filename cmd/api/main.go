package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/neo-vai/go-events/docs"
	"github.com/neo-vai/go-events/internal/broker"
	"github.com/neo-vai/go-events/internal/config"
	"github.com/neo-vai/go-events/internal/handler"
	accountHandler "github.com/neo-vai/go-events/internal/handler/account"
	adminHandler "github.com/neo-vai/go-events/internal/handler/admin"
	apikeyHandler "github.com/neo-vai/go-events/internal/handler/apikey"
	authHandler "github.com/neo-vai/go-events/internal/handler/auth"
	account_repository_pg "github.com/neo-vai/go-events/internal/repository/account/postgres"
	apikey_repository_pg "github.com/neo-vai/go-events/internal/repository/apikey/postgres"
	"github.com/neo-vai/go-events/internal/repository/cache"
	event_repository_pg "github.com/neo-vai/go-events/internal/repository/event/postgres"
	account_service "github.com/neo-vai/go-events/internal/service/account"
	apikey_service "github.com/neo-vai/go-events/internal/service/apikey"
	event_service "github.com/neo-vai/go-events/internal/service/event"
	"github.com/neo-vai/go-events/internal/validator"
	ginprometheus "github.com/zsais/go-gin-prometheus"
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
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	logLevel := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	logger.Info("starting server",
		"env", cfg.Env,
		"port", cfg.Port,
		"log_level", cfg.LogLevel,
	)

	// Connect to Redis
	redisClient, err := cache.NewRedisClient(
		cfg.RedisURL,
		cfg.RedisPassword,
		cfg.RedisDB,
		cfg.RedisMaxRetries,
		cfg.RedisPoolSize,
		cfg.RedisMinIdleConns,
		cfg.RedisDialTimeout,
		cfg.RedisReadTimeout,
		cfg.RedisWriteTimeout,
		cfg.RedisPoolTimeout,
		cfg.RedisIdleTimeout,
	)
	if err != nil {
		logger.Error("failed to initialize Redis client", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		logger.Error("unable to parse database URL", "error", err)
		os.Exit(1)
	}
	poolConfig.MaxConns = int32(cfg.DBMaxConns)
	poolConfig.MinConns = int32(cfg.DBMinConns)
	poolConfig.MaxConnLifetime = cfg.DBMaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.DBMaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		logger.Error("unable to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	validator.RegisterCustomValidators()

	// Initialize broker client and publisher
	brokerClient, err := broker.NewNATSClient(cfg.BrokerURL, cfg.BrokerJetStreamEnabled)
	if err != nil {
		logger.Error("failed to connect to broker", "error", err)
		os.Exit(1)
	}
	defer brokerClient.Close()

	publisher := broker.NewNATSPublisher(brokerClient, cfg.BrokerSubject)

	accountRepo := account_repository_pg.NewAccountRepositoryPG(pool)
	apiKeyRepo := apikey_repository_pg.NewAPIKeyRepositoryPG(pool)
	eventRepo := event_repository_pg.NewEventRepositoryPG(pool)

	passwordHasher := account_service.NewBcryptHasher(cfg.BcryptCost)
	accountService := account_service.NewAccountService(accountRepo, passwordHasher)

	apiKeyCache := cache.NewAPIKeyCache(redisClient.Client())
	apiKeyService := apikey_service.NewAPIKeyService(apiKeyRepo, apiKeyCache, cfg.APIKeyLength)

	eventService := event_service.NewEventService(eventRepo, apiKeyRepo)

	accountH := accountHandler.NewHandler(accountService)
	apiKeyH := apikeyHandler.NewHandler(apiKeyService)
	authH := authHandler.NewHandler(accountService, cfg.JWTSecret, cfg.JWTExpiresHours)

	statsSvc := &simpleStatsService{db: pool}
	adminH := adminHandler.NewHandler(accountService, eventService, apiKeyService, statsSvc)

	router := handler.NewRouter(handler.Handlers{
		Account: accountH,
		APIKey:  apiKeyH,
		Auth:    authH,
		Admin:   adminH,
	}, apiKeyService, accountService, eventService, cfg, redisClient.Client(), publisher)

	// Add Prometheus metrics middleware
	p := ginprometheus.NewPrometheus("gin")
	p.Use(router)

	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		logger.Warn("failed to set trusted proxies", "error", err)
	}

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  cfg.HTTPTimeout,
		WriteTimeout: cfg.HTTPTimeout,
	}

	srv.RegisterOnShutdown(func() {
		logger.Info("server shutdown initiated, waiting for active connections to finish")
	})

	go func() {
		logger.Info("server started", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down server...")

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}
	logger.Info("server exited properly")
}

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
