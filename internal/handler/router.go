package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/neo-vai/go-events/internal/handler/account"
	"github.com/neo-vai/go-events/internal/handler/apikey"
	"github.com/neo-vai/go-events/internal/handler/auth"
	"github.com/neo-vai/go-events/internal/handler/event"
	"github.com/neo-vai/go-events/internal/middleware"
	apikey_service "github.com/neo-vai/go-events/internal/service/apikey"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Handlers struct {
	Account *account.Handler
	APIKey  *apikey.Handler
	Event   *event.Handler
	Auth    *auth.Handler
}

// APIKeyValidatorFunc adapts apiKeyService.ValidateAPIKey to the signature expected by middleware.
func APIKeyValidatorFunc(apiKeySvc *apikey_service.APIKeyService) func(ctx *gin.Context, key string) (string, error) {
	return func(ctx *gin.Context, key string) (string, error) {
		return apiKeySvc.ValidateAPIKey(ctx, key)
	}
}

func NewRouter(h Handlers, apiKeySvc *apikey_service.APIKeyService) *gin.Engine {
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
		protected.Use(middleware.UniversalAuth(APIKeyValidatorFunc(apiKeySvc)))
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
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	return r
}
