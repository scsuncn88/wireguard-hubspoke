package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wg-hubspoke/wg-hubspoke/common/types"
	"github.com/wg-hubspoke/wg-hubspoke/controller/services"
)

type MonitoringHandler struct {
	monitoringService *services.MonitoringService
}

func NewMonitoringHandler(monitoringService *services.MonitoringService) *MonitoringHandler {
	return &MonitoringHandler{
		monitoringService: monitoringService,
	}
}

// UpdateNodeMetrics godoc
// @Summary Update node metrics
// @Description Update monitoring metrics for a specific node
// @Tags monitoring
// @Accept json
// @Produce json
// @Param node_id path string true "Node ID"
// @Param metrics body map[string]interface{} true "Metrics data"
// @Success 200 {object} types.APIResponse
// @Failure 400 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Router /monitoring/nodes/{node_id}/metrics [post]
func (h *MonitoringHandler) UpdateNodeMetrics(c *gin.Context) {
	nodeIDStr := c.Param("node_id")
	nodeID, err := uuid.Parse(nodeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid node ID format",
		})
		return
	}

	var metrics map[string]interface{}
	if err := c.ShouldBindJSON(&metrics); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	err = h.monitoringService.UpdateNodeMetrics(c.Request.Context(), nodeID, metrics)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Message: "Metrics updated successfully",
	})
}

// GetNodeMetrics godoc
// @Summary Get node metrics
// @Description Get current metrics for a specific node
// @Tags monitoring
// @Accept json
// @Produce json
// @Param node_id path string true "Node ID"
// @Success 200 {object} types.APIResponse{data=services.NodeMetrics}
// @Failure 400 {object} types.APIResponse
// @Failure 404 {object} types.APIResponse
// @Router /monitoring/nodes/{node_id}/metrics [get]
func (h *MonitoringHandler) GetNodeMetrics(c *gin.Context) {
	nodeIDStr := c.Param("node_id")
	nodeID, err := uuid.Parse(nodeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid node ID format",
		})
		return
	}

	metrics, err := h.monitoringService.GetNodeMetrics(c.Request.Context(), nodeID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    metrics,
	})
}

// GetAllNodeMetrics godoc
// @Summary Get all node metrics
// @Description Get current metrics for all nodes
// @Tags monitoring
// @Accept json
// @Produce json
// @Success 200 {object} types.APIResponse{data=map[string]services.NodeMetrics}
// @Failure 500 {object} types.APIResponse
// @Router /monitoring/nodes/metrics [get]
func (h *MonitoringHandler) GetAllNodeMetrics(c *gin.Context) {
	metrics, err := h.monitoringService.GetAllNodeMetrics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    metrics,
	})
}

// GetSystemMetrics godoc
// @Summary Get system metrics
// @Description Get overall system metrics and statistics
// @Tags monitoring
// @Accept json
// @Produce json
// @Success 200 {object} types.APIResponse{data=services.SystemMetrics}
// @Failure 500 {object} types.APIResponse
// @Router /monitoring/system/metrics [get]
func (h *MonitoringHandler) GetSystemMetrics(c *gin.Context) {
	metrics, err := h.monitoringService.GetSystemMetrics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    metrics,
	})
}

// GetNodeHealth godoc
// @Summary Get node health
// @Description Get health status and score for a specific node
// @Tags monitoring
// @Accept json
// @Produce json
// @Param node_id path string true "Node ID"
// @Success 200 {object} types.APIResponse{data=map[string]interface{}}
// @Failure 400 {object} types.APIResponse
// @Failure 404 {object} types.APIResponse
// @Router /monitoring/nodes/{node_id}/health [get]
func (h *MonitoringHandler) GetNodeHealth(c *gin.Context) {
	nodeIDStr := c.Param("node_id")
	nodeID, err := uuid.Parse(nodeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid node ID format",
		})
		return
	}

	health, err := h.monitoringService.GetNodeHealth(c.Request.Context(), nodeID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    health,
	})
}

// GetTopologyHealth godoc
// @Summary Get topology health
// @Description Get overall network topology health status
// @Tags monitoring
// @Accept json
// @Produce json
// @Success 200 {object} types.APIResponse{data=map[string]interface{}}
// @Failure 500 {object} types.APIResponse
// @Router /monitoring/topology/health [get]
func (h *MonitoringHandler) GetTopologyHealth(c *gin.Context) {
	health, err := h.monitoringService.GetTopologyHealth(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    health,
	})
}

// GetMetricsHistory godoc
// @Summary Get metrics history
// @Description Get historical metrics data for a specific node and metric
// @Tags monitoring
// @Accept json
// @Produce json
// @Param node_id path string true "Node ID"
// @Param metric query string true "Metric name"
// @Param duration query string false "Duration" default("24h")
// @Success 200 {object} types.APIResponse{data=[]interface{}}
// @Failure 400 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Router /monitoring/nodes/{node_id}/history [get]
func (h *MonitoringHandler) GetMetricsHistory(c *gin.Context) {
	nodeIDStr := c.Param("node_id")
	nodeID, err := uuid.Parse(nodeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid node ID format",
		})
		return
	}

	metric := c.Query("metric")
	if metric == "" {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Metric parameter is required",
		})
		return
	}

	durationStr := c.DefaultQuery("duration", "24h")
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid duration format",
		})
		return
	}

	history, err := h.monitoringService.GetMetricsHistory(c.Request.Context(), nodeID, metric, duration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    history,
	})
}

// GenerateReport godoc
// @Summary Generate monitoring report
// @Description Generate a comprehensive monitoring report for a time period
// @Tags monitoring
// @Accept json
// @Produce json
// @Param start_time query string true "Start time (RFC3339)"
// @Param end_time query string true "End time (RFC3339)"
// @Success 200 {object} types.APIResponse{data=map[string]interface{}}
// @Failure 400 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Router /monitoring/report [get]
func (h *MonitoringHandler) GenerateReport(c *gin.Context) {
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	if startTimeStr == "" || endTimeStr == "" {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "start_time and end_time are required",
		})
		return
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid start_time format",
		})
		return
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid end_time format",
		})
		return
	}

	report, err := h.monitoringService.GenerateReport(c.Request.Context(), startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    report,
	})
}

// GetPrometheusMetrics godoc
// @Summary Get Prometheus metrics
// @Description Get metrics in Prometheus format for scraping
// @Tags monitoring
// @Accept json
// @Produce text/plain
// @Success 200 {string} string "Prometheus metrics"
// @Router /metrics [get]
func (h *MonitoringHandler) GetPrometheusMetrics(c *gin.Context) {
	metrics, err := h.monitoringService.GetAllNodeMetrics(c.Request.Context())
	if err != nil {
		c.String(http.StatusInternalServerError, "# Error getting metrics\n")
		return
	}

	prometheusMetrics := "# HELP wg_sdwan_node_cpu_usage Node CPU usage percentage\n"
	prometheusMetrics += "# TYPE wg_sdwan_node_cpu_usage gauge\n"

	for nodeID, nodeMetrics := range metrics {
		labels := `{node_id="` + nodeID.String() + `",node_name="` + nodeMetrics.NodeName + `"}`
		prometheusMetrics += "wg_sdwan_node_cpu_usage" + labels + " " + fmt.Sprintf("%.2f", nodeMetrics.CPUUsage) + "\n"
		prometheusMetrics += "wg_sdwan_node_memory_usage" + labels + " " + fmt.Sprintf("%.2f", nodeMetrics.MemoryUsage) + "\n"
		prometheusMetrics += "wg_sdwan_node_network_rx" + labels + " " + fmt.Sprintf("%d", nodeMetrics.NetworkRx) + "\n"
		prometheusMetrics += "wg_sdwan_node_network_tx" + labels + " " + fmt.Sprintf("%d", nodeMetrics.NetworkTx) + "\n"
		prometheusMetrics += "wg_sdwan_node_latency" + labels + " " + fmt.Sprintf("%.2f", nodeMetrics.Latency) + "\n"
		prometheusMetrics += "wg_sdwan_node_packet_loss" + labels + " " + fmt.Sprintf("%.2f", nodeMetrics.PacketLoss) + "\n"
		prometheusMetrics += "wg_sdwan_node_wg_peers" + labels + " " + fmt.Sprintf("%d", nodeMetrics.WGPeers) + "\n"
	}

	// System metrics
	systemMetrics, _ := h.monitoringService.GetSystemMetrics(c.Request.Context())
	prometheusMetrics += "# HELP wg_sdwan_total_nodes Total number of nodes\n"
	prometheusMetrics += "# TYPE wg_sdwan_total_nodes gauge\n"
	prometheusMetrics += "wg_sdwan_total_nodes " + fmt.Sprintf("%d", systemMetrics.TotalNodes) + "\n"
	prometheusMetrics += "wg_sdwan_active_nodes " + fmt.Sprintf("%d", systemMetrics.ActiveNodes) + "\n"
	prometheusMetrics += "wg_sdwan_hub_nodes " + fmt.Sprintf("%d", systemMetrics.HubNodes) + "\n"
	prometheusMetrics += "wg_sdwan_spoke_nodes " + fmt.Sprintf("%d", systemMetrics.SpokeNodes) + "\n"

	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, prometheusMetrics)
}