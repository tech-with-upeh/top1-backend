// @title           Top1 API
// @version         1.0
// @description     Top1 Backend API documentation.
// @host            localhost:8000
// @BasePath        /
// @schemes         http https
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "top1/docs"
	"top1/internal/config"
	"top1/internal/db"
	"top1/internal/handlers"
	authHandlers "top1/internal/handlers/auth"
	"top1/internal/middleware"
	"top1/internal/repo"
	"top1/internal/services"
	"top1/internal/tasks"
	"top1/internal/utils"

	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return
	}

	var pool *pgxpool.Pool
	pool, err = db.Connect(cfg.DbUrl)
	if err != nil {
		slog.Error("Failed to connect to db", "error", err)
		return
	}
	defer pool.Close()

	taskService, err := tasks.NewClient(cfg.RedisUrl)
	if err != nil {
		slog.Error("Failed to establish connection with Redis", "error", err)
		return
	}
	defer taskService.Close()

	redisOptions, err := redis.ParseURL(cfg.RedisUrl)
	if err != nil {
		slog.Error("Invalid Redis URL", "error", err)
		return
	}

	redisClient := redis.NewClient(redisOptions)
	defer redisClient.Close()

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		slog.Error("Redis connection failed", "error", err)
		return
	}

	limiter := redis_rate.NewLimiter(redisClient)
	utilService := utils.NewUtilService(cfg)
	authRepo := repo.NewAuthRepository(pool)
	authService := services.NewAuthService(authRepo, taskService, utilService, cfg)
	authHandler := authHandlers.NewAuthHandler(authService)
	userService := services.NewUserService(authRepo)
	userHandler := handlers.NewUserHandler(userService)

	router := gin.Default()
	router.SetTrustedProxies(nil)
	router.Use(
		middleware.RateLimit(
			limiter,
			redis_rate.PerMinute(60),
		),
	)

	v1 := router.Group("/api/v1")

	v1.GET("/", handlers.HealthCheck)
	v1.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1.POST("/register", authHandler.Register)
	v1.POST("/resend-verification", authHandler.ResendVerification)
	v1.POST("/login", authHandler.Login)
	v1.POST("/logout", authHandler.Logout)
	v1.POST("/refresh", authHandler.Refresh)
	v1.POST("/verify-otp", authHandler.VerifyOTPCode)
	v1.POST("/verify-token", authHandler.VerifyTokenCode)

	users := v1.Group("/users")
	users.Use(middleware.AuthMiddleware(utilService))
	{
		users.GET("/me", userHandler.GetMe)
	}

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	// Start HTTP server.
	go func() {
		slog.Info("Starting HTTP server", "port", cfg.Port)

		if err := server.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			slog.Error("HTTP server failed", "error", err)
		}
	}()

	// Listen for termination signals.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Graceful shutdown failed", "error", err)
	}

	// Close resources.
	slog.Info("Server stopped")
}
