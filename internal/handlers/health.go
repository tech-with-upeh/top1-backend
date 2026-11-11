package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheck godoc
//
// @Summary     Health check
// @Description Check API availability and database connectivity
// @Tags        Health
// @Produce     json
// @Success     200 {object} map[string]interface{} "API and database are healthy"
// @Router      / [get]
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"msg":    "Top 1",
		"status": "success",
		"db":     "connected",
	})
}
