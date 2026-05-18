package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/neo-vai/go-events/internal/broker"
	"github.com/neo-vai/go-events/internal/config"
	"github.com/neo-vai/go-events/internal/handler/account"
	"github.com/neo-vai/go-events/internal/handler/admin"
	"github.com/neo-vai/go-events/internal/handler/apikey"
	"github.com/neo-vai/go-events/internal/handler/auth"
	"github.com/neo-vai/go-events/internal/handler/event"
	"github.com/neo-vai/go-events/internal/middleware"
	model_account "github.com/neo-vai/go-events/internal/model/account"
	account_service "github.com/neo-vai/go-events/internal/service/account"
	apikey_service "github.com/neo-vai/go-events/internal/service/apikey"
	event_service "github.com/neo-vai/go-events/internal/service/event"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Handlers struct {
	Account *account.Handler
	APIKey  *apikey.Handler
	Event   *event.Handler
	Auth    *auth.Handler
	Admin   *admin.Handler
}

// APIKeyValidatorFunc adapts apiKeyService.ValidateAPIKey to the signature expected by middleware.
// It also fetches the account role and checks account active status.
func APIKeyValidatorFunc(apiKeySvc *apikey_service.APIKeyService, accountSvc *account_service.AccountService) func(ctx *gin.Context, key string) (string, string, string, error) {
	return func(ctx *gin.Context, key string) (string, string, string, error) {
		accountID, _, apiKeyID, err := apiKeySvc.ValidateAPIKey(ctx, key)
		if err != nil {
			return "", "", "", err
		}
		acc, err := accountSvc.GetByID(ctx, accountID)
		if err != nil {
			return "", "", "", err
		}
		if !acc.Active {
			return "", "", "", account_service.ErrAccountInactive
		}
		return accountID, acc.Role, apiKeyID, nil
	}
}

func NewRouter(
	h Handlers,
	apiKeySvc *apikey_service.APIKeyService,
	accountSvc *account_service.AccountService,
	eventSvc *event_service.EventService,
	cfg *config.Config,
	rdb *redis.Client,
	publisher broker.Publisher,
) *gin.Engine {
	r := gin.New()
	// Global middleware (no rate limit yet)
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.StructuredLogger())
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	r.Use(middleware.PrometheusMetrics()) // custom Prometheus metrics
	r.Use(middleware.ValidationErrorHandler())

	// Metrics endpoint - must be exposed for Prometheus scraping
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Routes that should NOT be rate limited: health and swagger
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Apply global Redis rate limiter AFTER static/unprotected routes
	globalLimiter := middleware.GlobalRedisRateLimiter(rdb, cfg.RateLimitGlobal.Requests, cfg.RateLimitGlobal.Per)
	r.Use(globalLimiter)

	// Create event handler with publisher
	eventHandler := event.NewHandler(eventSvc, publisher)

	// Account getter for active check
	accountGetter := func(ctx *gin.Context, accountID string) (*model_account.Account, error) {
		return accountSvc.GetByID(ctx, accountID)
	}

	api := r.Group("/api/v1")
	{
		// Public routes with stricter Redis rate limiter on login
		loginLimiter := middleware.LoginRedisRateLimiter(rdb, cfg.RateLimitLogin.Requests, cfg.RateLimitLogin.Per)
		api.POST("/accounts", h.Account.CreateAccount)
		api.POST("/login", loginLimiter, h.Auth.Login)

		// Protected routes (JWT or API Key)
		protected := api.Group("/")
		protected.Use(middleware.UniversalAuth(
			APIKeyValidatorFunc(apiKeySvc, accountSvc),
			cfg.JWTSecret,
			accountGetter,
		))
		{
			accountGroup := protected.Group("/account")
			{
				accountGroup.GET("", h.Account.GetCurrentAccount)
				accountGroup.PATCH("", h.Account.UpdateCurrentAccount)
				accountGroup.DELETE("", h.Account.DeleteCurrentAccount)

				accountGroup.POST("/keys", h.APIKey.GenerateAPIKeyForCurrentAccount)
				accountGroup.GET("/keys", h.APIKey.ListAPIKeysForCurrentAccount)
				accountGroup.PATCH("/keys/:key_id", h.APIKey.UpdateAPIKeyActiveForCurrentAccount)
				accountGroup.DELETE("/keys/:key_id", h.APIKey.DeleteAPIKeyForCurrentAccount)
			}

			protected.POST("/events", eventHandler.CreateEvent)
			protected.GET("/events", eventHandler.ListEvents)
			protected.GET("/events/:id", eventHandler.GetEvent)
		}

		// Admin routes (JWT only, admin role required)
		adminGroup := api.Group("/admin")
		adminGroup.Use(
			middleware.UniversalAuth(
				APIKeyValidatorFunc(apiKeySvc, accountSvc),
				cfg.JWTSecret,
				accountGetter,
			),
			middleware.RequireAdmin(),
		)
		{
			adminGroup.GET("/accounts", h.Admin.ListAccounts)
			adminGroup.GET("/accounts/:id", h.Admin.GetAccount)
			adminGroup.POST("/accounts", h.Admin.CreateAccount)
			adminGroup.PUT("/accounts/:id", h.Admin.UpdateAccount)
			adminGroup.DELETE("/accounts/:id", h.Admin.DeleteAccount)

			adminGroup.GET("/events", h.Admin.ListEvents)
			adminGroup.GET("/events/:id", h.Admin.GetEvent)

			adminGroup.GET("/api-keys", h.Admin.ListAPIKeys)
			adminGroup.PUT("/api-keys/:id", h.Admin.UpdateAPIKey)
			adminGroup.DELETE("/api-keys/:id", h.Admin.DeleteAPIKey)

			adminGroup.GET("/stats", h.Admin.GetStats)
		}
	}

	return r
}
