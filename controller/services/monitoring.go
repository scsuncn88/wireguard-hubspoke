package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wg-hubspoke/wg-hubspoke/common/types"
	"github.com/wg-hubspoke/wg-hubspoke/controller/models"
	"gorm.io/gorm"
)

type MonitoringService struct {
	db           *gorm.DB
	nodeMetrics  sync.Map
	systemMetrics *SystemMetrics
	mutex        sync.RWMutex
}

type NodeMetrics struct {
	NodeID         uuid.UUID `json:"node_id"`
	NodeName       string    `json:"node_name"`
	Status         string    `json:"status"`
	LastSeen       time.Time `json:"last_seen"`
	CPUUsage       float64   `json:"cpu_usage"`
	MemoryUsage    float64   `json:"memory_usage"`
	DiskUsage      float64   `json:"disk_usage"`
	NetworkRx      int64     `json:"network_rx"`
	NetworkTx      int64     `json:"network_tx"`
	WGPeers        int       `json:"wg_peers"`
	WGStatus       string    `json:"wg_status"`
	WGLastHandshake time.Time `json:"wg_last_handshake"`
	Latency        float64   `json:"latency_ms"`
	PacketLoss     float64   `json:"packet_loss"`
	Bandwidth      int64     `json:"bandwidth_bps"`
	Errors         []string  `json:"errors"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type SystemMetrics struct {
	TotalNodes     int64     `json:"total_nodes"`
	ActiveNodes    int64     `json:"active_nodes"`
	InactiveNodes  int64     `json:"inactive_nodes"`
	HubNodes       int64     `json:"hub_nodes"`
	SpokeNodes     int64     `json:"spoke_nodes"`
	TotalTraffic   int64     `json:"total_traffic"`
	ActivePolicies int64     `json:"active_policies"`
	LastUpdated    time.Time `json:"last_updated"`
}

type MetricsReport struct {
	NodeID    uuid.UUID              `json:"node_id"`
	Timestamp time.Time              `json:"timestamp"`
	Metrics   map[string]interface{} `json:"metrics"`
}

type AlertRule struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Metric      string    `json:"metric"`
	Operator    string    `json:"operator"`
	Threshold   float64   `json:"threshold"`
	Severity    string    `json:"severity"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

type Alert struct {
	ID          uuid.UUID `json:"id"`
	RuleID      uuid.UUID `json:"rule_id"`
	NodeID      uuid.UUID `json:"node_id"`
	Message     string    `json:"message"`
	Severity    string    `json:"severity"`
	Status      string    `json:"status"`
	TriggeredAt time.Time `json:"triggered_at"`
	ResolvedAt  *time.Time `json:"resolved_at"`
}

func NewMonitoringService(db *gorm.DB) *MonitoringService {
	return &MonitoringService{
		db: db,
		systemMetrics: &SystemMetrics{
			LastUpdated: time.Now(),
		},
	}
}

func (s *MonitoringService) UpdateNodeMetrics(ctx context.Context, nodeID uuid.UUID, metrics map[string]interface{}) error {
	// Get node info
	var node models.Node
	if err := s.db.Where("id = ?", nodeID).First(&node).Error; err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	// Parse metrics
	nodeMetrics := &NodeMetrics{
		NodeID:    nodeID,
		NodeName:  node.Name,
		Status:    string(node.Status),
		LastSeen:  time.Now(),
		UpdatedAt: time.Now(),
	}

	// Extract metrics from map
	if cpu, ok := metrics["cpu_usage"].(float64); ok {
		nodeMetrics.CPUUsage = cpu
	}
	if memory, ok := metrics["memory_usage"].(float64); ok {
		nodeMetrics.MemoryUsage = memory
	}
	if disk, ok := metrics["disk_usage"].(float64); ok {
		nodeMetrics.DiskUsage = disk
	}
	if rx, ok := metrics["network_rx"].(int64); ok {
		nodeMetrics.NetworkRx = rx
	}
	if tx, ok := metrics["network_tx"].(int64); ok {
		nodeMetrics.NetworkTx = tx
	}
	if peers, ok := metrics["wg_peers"].(int); ok {
		nodeMetrics.WGPeers = peers
	}
	if status, ok := metrics["wg_status"].(string); ok {
		nodeMetrics.WGStatus = status
	}
	if handshake, ok := metrics["wg_last_handshake"].(time.Time); ok {
		nodeMetrics.WGLastHandshake = handshake
	}
	if latency, ok := metrics["latency_ms"].(float64); ok {
		nodeMetrics.Latency = latency
	}
	if loss, ok := metrics["packet_loss"].(float64); ok {
		nodeMetrics.PacketLoss = loss
	}
	if bandwidth, ok := metrics["bandwidth_bps"].(int64); ok {
		nodeMetrics.Bandwidth = bandwidth
	}
	if errors, ok := metrics["errors"].([]string); ok {
		nodeMetrics.Errors = errors
	}

	// Store metrics
	s.nodeMetrics.Store(nodeID, nodeMetrics)

	// Update node last handshake in database
	if !nodeMetrics.WGLastHandshake.IsZero() {
		s.db.Model(&node).Update("last_handshake", nodeMetrics.WGLastHandshake)
	}

	// Check alert rules
	s.checkAlerts(ctx, nodeMetrics)

	return nil
}

func (s *MonitoringService) GetNodeMetrics(ctx context.Context, nodeID uuid.UUID) (*NodeMetrics, error) {
	if metrics, ok := s.nodeMetrics.Load(nodeID); ok {
		return metrics.(*NodeMetrics), nil
	}
	return nil, fmt.Errorf("metrics not found for node %s", nodeID)
}

func (s *MonitoringService) GetAllNodeMetrics(ctx context.Context) (map[uuid.UUID]*NodeMetrics, error) {
	result := make(map[uuid.UUID]*NodeMetrics)
	
	s.nodeMetrics.Range(func(key, value interface{}) bool {
		nodeID := key.(uuid.UUID)
		metrics := value.(*NodeMetrics)
		result[nodeID] = metrics
		return true
	})

	return result, nil
}

func (s *MonitoringService) GetSystemMetrics(ctx context.Context) (*SystemMetrics, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Update system metrics
	s.updateSystemMetrics(ctx)

	return s.systemMetrics, nil
}

func (s *MonitoringService) updateSystemMetrics(ctx context.Context) {
	var totalNodes, activeNodes, inactiveNodes, hubNodes, spokeNodes, activePolicies int64
	var totalTraffic int64

	// Count nodes
	s.db.Model(&models.Node{}).Count(&totalNodes)
	s.db.Model(&models.Node{}).Where("status = ?", models.NodeStatusActive).Count(&activeNodes)
	s.db.Model(&models.Node{}).Where("status = ?", models.NodeStatusInactive).Count(&inactiveNodes)
	s.db.Model(&models.Node{}).Where("node_type = ?", models.NodeTypeHub).Count(&hubNodes)
	s.db.Model(&models.Node{}).Where("node_type = ?", models.NodeTypeSpoke).Count(&spokeNodes)

	// Count policies
	s.db.Model(&models.Policy{}).Where("enabled = ?", true).Count(&activePolicies)

	// Calculate total traffic
	s.nodeMetrics.Range(func(key, value interface{}) bool {
		metrics := value.(*NodeMetrics)
		totalTraffic += metrics.NetworkRx + metrics.NetworkTx
		return true
	})

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.systemMetrics.TotalNodes = totalNodes
	s.systemMetrics.ActiveNodes = activeNodes
	s.systemMetrics.InactiveNodes = inactiveNodes
	s.systemMetrics.HubNodes = hubNodes
	s.systemMetrics.SpokeNodes = spokeNodes
	s.systemMetrics.TotalTraffic = totalTraffic
	s.systemMetrics.ActivePolicies = activePolicies
	s.systemMetrics.LastUpdated = time.Now()
}

func (s *MonitoringService) GetNodeHealth(ctx context.Context, nodeID uuid.UUID) (map[string]interface{}, error) {
	metrics, err := s.GetNodeMetrics(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	health := make(map[string]interface{})
	health["node_id"] = nodeID
	health["node_name"] = metrics.NodeName
	health["status"] = metrics.Status
	health["last_seen"] = metrics.LastSeen
	health["is_online"] = time.Since(metrics.LastSeen) < 5*time.Minute

	// Health scores
	healthScore := 100.0
	issues := []string{}

	if metrics.CPUUsage > 80 {
		healthScore -= 20
		issues = append(issues, "High CPU usage")
	}
	if metrics.MemoryUsage > 80 {
		healthScore -= 20
		issues = append(issues, "High memory usage")
	}
	if metrics.DiskUsage > 90 {
		healthScore -= 30
		issues = append(issues, "High disk usage")
	}
	if metrics.PacketLoss > 5 {
		healthScore -= 25
		issues = append(issues, "High packet loss")
	}
	if metrics.Latency > 100 {
		healthScore -= 15
		issues = append(issues, "High latency")
	}
	if len(metrics.Errors) > 0 {
		healthScore -= 10
		issues = append(issues, "System errors detected")
	}

	health["health_score"] = healthScore
	health["issues"] = issues
	health["metrics"] = metrics

	return health, nil
}

func (s *MonitoringService) GetTopologyHealth(ctx context.Context) (map[string]interface{}, error) {
	systemMetrics, err := s.GetSystemMetrics(ctx)
	if err != nil {
		return nil, err
	}

	allMetrics, err := s.GetAllNodeMetrics(ctx)
	if err != nil {
		return nil, err
	}

	health := make(map[string]interface{})
	health["system_metrics"] = systemMetrics

	// Calculate network health
	var totalHealthScore float64
	var onlineNodes int
	nodeHealth := make(map[string]interface{})

	for nodeID, metrics := range allMetrics {
		isOnline := time.Since(metrics.LastSeen) < 5*time.Minute
		if isOnline {
			onlineNodes++
		}

		// Calculate node health score
		healthScore := 100.0
		if metrics.CPUUsage > 80 {
			healthScore -= 20
		}
		if metrics.MemoryUsage > 80 {
			healthScore -= 20
		}
		if metrics.PacketLoss > 5 {
			healthScore -= 25
		}

		totalHealthScore += healthScore
		nodeHealth[nodeID.String()] = map[string]interface{}{
			"health_score": healthScore,
			"is_online":    isOnline,
			"last_seen":    metrics.LastSeen,
		}
	}

	health["node_health"] = nodeHealth
	health["online_nodes"] = onlineNodes
	health["total_nodes"] = len(allMetrics)
	
	if len(allMetrics) > 0 {
		health["average_health_score"] = totalHealthScore / float64(len(allMetrics))
	}

	return health, nil
}

func (s *MonitoringService) checkAlerts(ctx context.Context, metrics *NodeMetrics) {
	// Simple alert checking - in production, this would be more sophisticated
	if metrics.CPUUsage > 90 {
		s.triggerAlert(ctx, "high_cpu", metrics.NodeID, fmt.Sprintf("CPU usage is %.2f%%", metrics.CPUUsage), "warning")
	}
	if metrics.MemoryUsage > 90 {
		s.triggerAlert(ctx, "high_memory", metrics.NodeID, fmt.Sprintf("Memory usage is %.2f%%", metrics.MemoryUsage), "warning")
	}
	if metrics.PacketLoss > 10 {
		s.triggerAlert(ctx, "high_packet_loss", metrics.NodeID, fmt.Sprintf("Packet loss is %.2f%%", metrics.PacketLoss), "critical")
	}
	if time.Since(metrics.LastSeen) > 10*time.Minute {
		s.triggerAlert(ctx, "node_offline", metrics.NodeID, "Node has been offline for more than 10 minutes", "critical")
	}
}

func (s *MonitoringService) triggerAlert(ctx context.Context, alertType string, nodeID uuid.UUID, message, severity string) {
	// In a real implementation, this would send notifications, store alerts, etc.
	fmt.Printf("ALERT [%s]: %s - %s\n", severity, alertType, message)
}

func (s *MonitoringService) GetMetricsHistory(ctx context.Context, nodeID uuid.UUID, metric string, duration time.Duration) ([]interface{}, error) {
	// This would typically query a time-series database
	// For now, return mock data
	return []interface{}{}, nil
}

func (s *MonitoringService) CleanupOldMetrics(ctx context.Context, retentionDays int) error {
	// Clean up old metrics data
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	
	// Remove metrics older than retention period
	s.nodeMetrics.Range(func(key, value interface{}) bool {
		metrics := value.(*NodeMetrics)
		if metrics.UpdatedAt.Before(cutoffTime) {
			s.nodeMetrics.Delete(key)
		}
		return true
	})

	return nil
}

func (s *MonitoringService) GenerateReport(ctx context.Context, startTime, endTime time.Time) (map[string]interface{}, error) {
	report := make(map[string]interface{})
	report["period"] = map[string]interface{}{
		"start": startTime,
		"end":   endTime,
	}

	// System overview
	systemMetrics, _ := s.GetSystemMetrics(ctx)
	report["system_overview"] = systemMetrics

	// Node summary
	allMetrics, _ := s.GetAllNodeMetrics(ctx)
	nodeSummary := make(map[string]interface{})
	
	for nodeID, metrics := range allMetrics {
		nodeSummary[nodeID.String()] = map[string]interface{}{
			"name":         metrics.NodeName,
			"status":       metrics.Status,
			"last_seen":    metrics.LastSeen,
			"cpu_usage":    metrics.CPUUsage,
			"memory_usage": metrics.MemoryUsage,
			"network_rx":   metrics.NetworkRx,
			"network_tx":   metrics.NetworkTx,
			"wg_peers":     metrics.WGPeers,
			"latency":      metrics.Latency,
			"packet_loss":  metrics.PacketLoss,
		}
	}

	report["node_summary"] = nodeSummary
	report["generated_at"] = time.Now()

	return report, nil
}