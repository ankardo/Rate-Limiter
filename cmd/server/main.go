package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/ankardo/Rate-Limiter/config"
	"github.com/ankardo/Rate-Limiter/config/logger"
	"github.com/ankardo/Rate-Limiter/internal/app/limiter"
	"github.com/ankardo/Rate-Limiter/internal/app/middleware"
	"github.com/ankardo/Rate-Limiter/internal/domain"
	"github.com/ankardo/Rate-Limiter/internal/infrastructure/persistence"
	"github.com/ankardo/Rate-Limiter/internal/infrastructure/webserver"
)

func main() {
	cfg := config.LoadConfig("./.env")
	ctx := context.Background()
	var rateLimiter domain.Limiter

	if os.Getenv("USE_MEMORY_STORE") == "true" {
		rateLimiter = limiter.NewMemoryRateLimiter(domain.LimiterConfig{
			MaxRequests:      cfg.MaxRequests,
			TokenMaxRequests: cfg.TokenMaxRequests,
			BlockDuration:    int64(cfg.BlockDuration),
		})
		logger.Info("Using in-memory rate limiter")
	} else {
		redisClient, err := persistence.NewRedisClient(ctx, cfg.RedisAddr, cfg.RedisPassword)
		if err != nil {
			logger.Error("Failed to connect to Redis: %v", err)
		}
		redisStore := persistence.NewRedisStore(redisClient)
		rateLimiter = limiter.NewRedisRateLimiter(redisStore, domain.LimiterConfig{
			MaxRequests:      cfg.MaxRequests,
			TokenMaxRequests: cfg.TokenMaxRequests,
			BlockDuration:    int64(cfg.BlockDuration),
			TTLExpiration:    int64(cfg.TTLExpiration),
		})
		logger.Info("Using Redis rate limiter")
	}

	rateLimiterMiddleware := middleware.RateLimiterMiddleware(rateLimiter)
	mux := webserver.NewRouter(rateLimiterMiddleware)

	logger.Info("Server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
