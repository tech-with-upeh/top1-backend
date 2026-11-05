// @title           Top1 API
// @version         1.0
// @description     Top1 Backend API documentation.
// @host            localhost:8000
// @BasePath        /
// @schemes         http https
package main

import (
	"log/slog"

	_ "top1/docs"
	"top1/internal/config"
	"top1/internal/db"
	"top1/internal/handlers"
	authHandlers "top1/internal/handlers/auth"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	var cfg *config.Config
	var err error
	cfg, err = config.Load()
	if err != nil {
		slog.Error("Failed to load config: ", err)
	}

	var pool *pgxpool.Pool
	pool, err = db.Connect(cfg.DbUrl)
	if err != nil {
		slog.Error("Failed to connect to db: ", err)
	}
	defer pool.Close()

	var router *gin.Engine = gin.Default()
	router.SetTrustedProxies(nil)

	router.GET("/", handlers.HealthCheck)

	router.POST("/register", authHandlers.CreateUserHandler(pool))

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	router.Run(":" + cfg.Port)
}
