package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/neo-vai/go-events/internal/handler/account"
	"github.com/neo-vai/go-events/internal/handler/apikey"
	"github.com/neo-vai/go-events/internal/handler/event"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Handlers struct {
	Account *account.Handler
	APIKey  *apikey.Handler
	Event   *event.Handler
}

func NewRouter(h Handlers) *gin.Engine {
	r := gin.Default()

	api := r.Group("/api/v1")
	{
		// Account routes
		api.POST("/accounts", h.Account.CreateAccount)
		api.GET("/accounts/:id", h.Account.GetAccount)
		api.PUT("/accounts/:id", h.Account.UpdateAccount)
		api.DELETE("/accounts/:id", h.Account.DeleteAccount)

		// API Key routes (более длинные пути)
		api.POST("/accounts/:id/keys", h.APIKey.GenerateAPIKey)
		api.GET("/accounts/:id/keys", h.APIKey.ListAPIKeys)
		api.PATCH("/accounts/:id/keys/:key_id", h.APIKey.UpdateAPIKeyActive)
		api.DELETE("/accounts/:id/keys/:key_id", h.APIKey.DeleteAPIKey)

		// Event routes
		api.POST("/events", h.Event.CreateEvent)
		api.GET("/events", h.Event.ListEvents)
		api.GET("/events/:id", h.Event.GetEvent)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
