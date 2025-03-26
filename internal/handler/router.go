package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/neo-vai/go-events/internal/handler/account"
	"github.com/neo-vai/go-events/internal/handler/admin"
	"github.com/neo-vai/go-events/internal/handler/apikey"
	"github.com/neo-vai/go-events/internal/handler/auth"
	"github.com/neo-vai/go-events/internal/handler/event"
	"github.com/neo-vai/go-events/internal/middleware"
	account_service "github.com/neo-vai/go-events/internal/service/account"
	apikey_service "github.com/neo-vai/go-events/internal/service/apikey"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Handlers struct {
	Account *account.Handler
	APIKey  *apikey.Handler
	Event   *event.Handler
	Auth    *auth.Handler
	Admin   *admin.Handler
}

// APIKeyValidatorFunc adapts apiKeyService.ValidateAPIKey to the signature expected by middleware.
// It also fetches the account role from the database.
func APIKeyValidatorFunc(apiKeySvc *apikey_service.APIKeyService, accountSvc *account_service.AccountService) func(ctx *gin.Context, key string) (string, string, error) {
	return func(ctx *gin.Context, key string) (string, string, error) {
		accountID, _, err := apiKeySvc.ValidateAPIKey(ctx, key)
		if err != nil {
			return "", "", err
		}
		// Fetch account to get role
		acc, err := accountSvc.GetByID(ctx, accountID)
		if err != nil {
			return "", "", err
		}
		return accountID, acc.Role, nil
	}
}

func NewRouter(h Handlers, apiKeySvc *apikey_service.APIKeyService, accountSvc *account_service.AccountService) *gin.Engine {
	r := gin.New()
	// Global middleware
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.StructuredLogger())
	r.Use(middleware.CORS())
	r.Use(middleware.RateLimiter(100, 1))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		// Public routes
		api.POST("/accounts", h.Account.CreateAccount)
		api.POST("/login", h.Auth.Login)

		// Protected routes (JWT or API Key)
		protected := api.Group("/")
		protected.Use(middleware.UniversalAuth(APIKeyValidatorFunc(apiKeySvc, accountSvc)))
		{
			// Account endpoints – require ownership
			accountGroup := protected.Group("/accounts/:id")
			accountGroup.Use(middleware.OwnerCheck())
			{
				accountGroup.GET("", h.Account.GetAccount)
				accountGroup.PUT("", h.Account.UpdateAccount)
				accountGroup.DELETE("", h.Account.DeleteAccount)

				accountGroup.POST("/keys", h.APIKey.GenerateAPIKey)
				accountGroup.GET("/keys", h.APIKey.ListAPIKeys)
				accountGroup.PATCH("/keys/:key_id", h.APIKey.UpdateAPIKeyActive)
				accountGroup.DELETE("/keys/:key_id", h.APIKey.DeleteAPIKey)
			}

			// Event endpoints – automatically scoped to authenticated account
			protected.POST("/events", h.Event.CreateEvent)
			protected.GET("/events", h.Event.ListEvents)
			protected.GET("/events/:id", h.Event.GetEvent)
		}

		// Admin routes (JWT only, admin role required)
		adminGroup := api.Group("/admin")
		adminGroup.Use(middleware.UniversalAuth(APIKeyValidatorFunc(apiKeySvc, accountSvc)), middleware.RequireAdmin())
		{
			// Accounts
			adminGroup.GET("/accounts", h.Admin.ListAccounts)
			adminGroup.GET("/accounts/:id", h.Admin.GetAccount)
			adminGroup.POST("/accounts", h.Admin.CreateAccount)
			adminGroup.PUT("/accounts/:id", h.Admin.UpdateAccount)
			adminGroup.DELETE("/accounts/:id", h.Admin.DeleteAccount)

			// Events
			adminGroup.GET("/events", h.Admin.ListEvents)
			adminGroup.GET("/events/:id", h.Admin.GetEvent)

			// API Keys
			adminGroup.GET("/api-keys", h.Admin.ListAPIKeys)
			adminGroup.PUT("/api-keys/:id", h.Admin.UpdateAPIKey)
			adminGroup.DELETE("/api-keys/:id", h.Admin.DeleteAPIKey)

			// Stats
			adminGroup.GET("/stats", h.Admin.GetStats)
		}
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	return r
}
