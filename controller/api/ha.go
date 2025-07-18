package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wg-hubspoke/wg-hubspoke/common/types"
	"github.com/wg-hubspoke/wg-hubspoke/controller/services"
)

type HAHandler struct {
	haService *services.HAService
}

func NewHAHandler(haService *services.HAService) *HAHandler {
	return &HAHandler{
		haService: haService,
	}
}

// GetClusterStatus godoc
// @Summary Get cluster status
// @Description Get current cluster status and node information
// @Tags ha
// @Accept json
// @Produce json
// @Success 200 {object} types.APIResponse{data=services.ClusterStatus}
// @Router /ha/status [get]
func (h *HAHandler) GetClusterStatus(c *gin.Context) {
	status := h.haService.GetClusterStatus()
	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    status,
	})
}

// GetHealthStatus godoc
// @Summary Get node health status
// @Description Get current node health status for HA
// @Tags ha
// @Accept json
// @Produce json
// @Success 200 {object} types.APIResponse{data=services.HealthResponse}
// @Router /ha/health [get]
func (h *HAHandler) GetHealthStatus(c *gin.Context) {
	health := h.haService.GetHealthStatus()
	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    health,
	})
}

// HandleVoteRequest godoc
// @Summary Handle leader election vote
// @Description Handle leader election vote request from peer nodes
// @Tags ha
// @Accept json
// @Produce json
// @Param request body services.LeaderElectionRequest true "Election request"
// @Success 200 {object} types.APIResponse{data=services.LeaderElectionResponse}
// @Router /ha/election [post]
func (h *HAHandler) HandleVoteRequest(c *gin.Context) {
	var request services.LeaderElectionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	response := h.haService.HandleVoteRequest(&request)
	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    response,
	})
}

// HandleLeaderAnnouncement godoc
// @Summary Handle leader announcement
// @Description Handle leader announcement from elected leader
// @Tags ha
// @Accept json
// @Produce json
// @Param announcement body map[string]interface{} true "Leader announcement"
// @Success 200 {object} types.APIResponse
// @Router /ha/leader [post]
func (h *HAHandler) HandleLeaderAnnouncement(c *gin.Context) {
	var announcement map[string]interface{}
	if err := c.ShouldBindJSON(&announcement); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	h.haService.HandleLeaderAnnouncement(announcement)
	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Message: "Leader announcement processed",
	})
}

// SyncConfiguration godoc
// @Summary Sync configuration
// @Description Sync configuration across cluster nodes (leader only)
// @Tags ha
// @Accept json
// @Produce json
// @Success 200 {object} types.APIResponse
// @Failure 403 {object} types.APIResponse
// @Router /ha/sync [post]
func (h *HAHandler) SyncConfiguration(c *gin.Context) {
	if err := h.haService.SyncConfiguration(c.Request.Context()); err != nil {
		c.JSON(http.StatusForbidden, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Message: "Configuration synchronized successfully",
	})
}