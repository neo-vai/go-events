package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo-vai/go-events/internal/broker"
	"github.com/neo-vai/go-events/internal/config"
	"github.com/neo-vai/go-events/internal/model/event"
	event_repository_pg "github.com/neo-vai/go-events/internal/repository/event/postgres"
	event_service "github.com/neo-vai/go-events/internal/service/event"
)

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

	logger.Info("starting event worker",
		"env", cfg.Env,
		"broker_url", cfg.BrokerURL,
		"broker_subject", cfg.BrokerSubject,
	)

	pool := connectPostgresWithRetry(logger, cfg)
	defer pool.Close()

	eventRepo := event_repository_pg.NewEventRepositoryPG(pool)
	eventService := event_service.NewEventService(eventRepo, nil) // apiKeyRepo not needed in worker

	brokerClient, err := broker.NewNATSClient(cfg.BrokerURL, cfg.BrokerJetStreamEnabled)
	if err != nil {
		logger.Error("failed to connect to broker", "error", err)
		os.Exit(1)
	}
	defer brokerClient.Close()

	queueGroup := "event-workers"
	subscriber := broker.NewNATSSubscriber(brokerClient, cfg.BrokerSubject, queueGroup)

	err = subscriber.Subscribe(func(ev *event.Event) error {
		return eventService.CreateEvent(context.Background(), ev)
	})
	if err != nil {
		logger.Error("failed to subscribe to broker", "error", err)
		os.Exit(1)
	}
	defer subscriber.Close()

	logger.Info("worker started, waiting for messages...")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down worker...")
	time.Sleep(2 * time.Second)
	logger.Info("worker exited")
}

// connectPostgresWithRetry attempts to connect to PostgreSQL with retries.
func connectPostgresWithRetry(logger *slog.Logger, cfg *config.Config) *pgxpool.Pool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

	var pool *pgxpool.Pool
	err = retryConnect(ctx, func() error {
		var err error
		pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
		return err
	}, logger, "PostgreSQL")
	if err != nil {
		logger.Error("failed to connect to PostgreSQL after retries", "error", err)
		os.Exit(1)
	}
	return pool
}

// retryConnect executes a connect function with exponential backoff until success or context timeout.
func retryConnect(ctx context.Context, connectFn func() error, logger *slog.Logger, service string) error {
	backoff := 100 * time.Millisecond
	for {
		err := connectFn()
		if err == nil {
			logger.Info("connected to service", "service", service)
			return nil
		}
		logger.Warn("failed to connect to service, retrying...",
			"service", service,
			"error", err,
			"next_attempt_in", backoff,
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			backoff *= 2
			if backoff > 5*time.Second {
				backoff = 5 * time.Second
			}
		}
	}
}
