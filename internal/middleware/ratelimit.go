package middleware

import (
	"log/slog"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
)

func RateLimit(
	limiter *redis_rate.Limiter,
	limit redis_rate.Limit,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "api:" + c.ClientIP() + ":" + c.FullPath()

		result, err := limiter.Allow(
			c.Request.Context(),
			key,
			limit,
		)
		if err != nil {
			slog.Error("Rate limiter failed", "error", err)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "Rate limiter unavailable",
			})
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(limit.Rate))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))

		if result.Allowed == 0 {
			retryAfter := int(math.Ceil(result.RetryAfter.Seconds()))
			c.Header("Retry-After", strconv.Itoa(retryAfter))

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests",
			})
			return
		}

		c.Next()
	}
}
