package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter creates a rate limiter middleware that uses Redis to coordinate limits across instances.
// The limit is defined as 'requests' allowed per 'per' duration.
func RedisRateLimiter(rdb *redis.Client, requests int, per time.Duration, keyFunc func(c *gin.Context) string) gin.HandlerFunc {
	limiter := redis_rate.NewLimiter(rdb)
	return func(c *gin.Context) {
		key := keyFunc(c)
		if key == "" {
			c.Next()
			return
		}

		limit := redis_rate.Limit{
			Rate:   requests,
			Burst:  requests,
			Period: per,
		}

		res, err := limiter.Allow(c.Request.Context(), key, limit)
		if err != nil {
			c.Next()
			return
		}
		if res.Allowed == 0 {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

// GlobalRedisRateLimiter is a convenience wrapper that uses client IP as the key.
func GlobalRedisRateLimiter(rdb *redis.Client, requests int, per time.Duration) gin.HandlerFunc {
	return RedisRateLimiter(rdb, requests, per, func(c *gin.Context) string {
		return "rate:global:ip:" + c.ClientIP()
	})
}

// LoginRedisRateLimiter uses client IP for the login endpoint.
func LoginRedisRateLimiter(rdb *redis.Client, requests int, per time.Duration) gin.HandlerFunc {
	return RedisRateLimiter(rdb, requests, per, func(c *gin.Context) string {
		return "rate:login:ip:" + c.ClientIP()
	})
}
