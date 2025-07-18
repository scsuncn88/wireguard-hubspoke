package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wg-hubspoke/wg-hubspoke/common/types"
	"github.com/wg-hubspoke/wg-hubspoke/controller/services"
)

type HealthHandler struct {
	healthService *services.HealthService
	version       string
}

func NewHealthHandler(healthService *services.HealthService, version string) *HealthHandler {
	return &HealthHandler{
		healthService: healthService,
		version:       version,
	}
}

// HealthCheck godoc
// @Summary Health check endpoint
// @Description Check the health status of the service and its dependencies
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} types.APIResponse{data=types.HealthStatus}
// @Failure 503 {object} types.APIResponse{data=types.HealthStatus}
// @Router /health [get]
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	status := h.healthService.GetHealthStatus(c.Request.Context())
	
	httpStatus := http.StatusOK
	if status.Status != "healthy" {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, types.APIResponse{
		Success: status.Status == "healthy",
		Data:    status,
	})
}

// ReadinessCheck godoc
// @Summary Readiness check endpoint
// @Description Check if the service is ready to serve requests
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} types.APIResponse
// @Failure 503 {object} types.APIResponse
// @Router /ready [get]
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	ready := h.healthService.IsReady(c.Request.Context())
	
	httpStatus := http.StatusOK
	if !ready {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, types.APIResponse{
		Success: ready,
		Data: map[string]interface{}{
			"ready":     ready,
			"timestamp": time.Now(),
		},
	})
}

// LivenessCheck godoc
// @Summary Liveness check endpoint
// @Description Check if the service is alive
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} types.APIResponse
// @Router /live [get]
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"alive":     true,
			"timestamp": time.Now(),
			"version":   h.version,
		},
	})
}