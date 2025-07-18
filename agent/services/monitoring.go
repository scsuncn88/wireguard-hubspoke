package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/wg-hubspoke/wg-hubspoke/agent/client"
	"github.com/wg-hubspoke/wg-hubspoke/agent/config"
	"github.com/wg-hubspoke/wg-hubspoke/agent/wg"
)

type MonitoringService struct {
	config           *config.AgentConfig
	wgManager        *wg.Manager
	controllerClient *client.ControllerClient
	nodeID           uuid.UUID
}

type SystemMetrics struct {
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage float64   `json:"memory_usage"`
	DiskUsage   float64   `json:"disk_usage"`
	NetworkRx   int64     `json:"network_rx"`
	NetworkTx   int64     `json:"network_tx"`
	Timestamp   time.Time `json:"timestamp"`
}

type WireGuardMetrics struct {
	Status          string    `json:"status"`
	Peers           int       `json:"peers"`
	LastHandshake   time.Time `json:"last_handshake"`
	RxBytes         int64     `json:"rx_bytes"`
	TxBytes         int64     `json:"tx_bytes"`
	Latency         float64   `json:"latency_ms"`
	PacketLoss      float64   `json:"packet_loss"`
	InterfaceStatus string    `json:"interface_status"`
}

type NodeMetrics struct {
	NodeID         uuid.UUID        `json:"node_id"`
	SystemMetrics  SystemMetrics    `json:"system_metrics"`
	WGMetrics      WireGuardMetrics `json:"wg_metrics"`
	Errors         []string         `json:"errors"`
	Timestamp      time.Time        `json:"timestamp"`
}

func NewMonitoringService(config *config.AgentConfig, wgManager *wg.Manager, controllerClient *client.ControllerClient) *MonitoringService {
	nodeID, _ := uuid.Parse(config.Node.ID)
	return &MonitoringService{
		config:           config,
		wgManager:        wgManager,
		controllerClient: controllerClient,
		nodeID:           nodeID,
	}
}

func (s *MonitoringService) CollectMetrics(ctx context.Context) (*NodeMetrics, error) {
	metrics := &NodeMetrics{
		NodeID:    s.nodeID,
		Timestamp: time.Now(),
		Errors:    []string{},
	}

	// Collect system metrics
	systemMetrics, err := s.collectSystemMetrics(ctx)
	if err != nil {
		metrics.Errors = append(metrics.Errors, fmt.Sprintf("System metrics error: %v", err))
	} else {
		metrics.SystemMetrics = *systemMetrics
	}

	// Collect WireGuard metrics
	wgMetrics, err := s.collectWireGuardMetrics(ctx)
	if err != nil {
		metrics.Errors = append(metrics.Errors, fmt.Sprintf("WireGuard metrics error: %v", err))
	} else {
		metrics.WGMetrics = *wgMetrics
	}

	return metrics, nil
}

func (s *MonitoringService) collectSystemMetrics(ctx context.Context) (*SystemMetrics, error) {
	metrics := &SystemMetrics{
		Timestamp: time.Now(),
	}

	// CPU usage
	cpuUsage, err := s.getCPUUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU usage: %w", err)
	}
	metrics.CPUUsage = cpuUsage

	// Memory usage
	memUsage, err := s.getMemoryUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory usage: %w", err)
	}
	metrics.MemoryUsage = memUsage

	// Disk usage
	diskUsage, err := s.getDiskUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to get disk usage: %w", err)
	}
	metrics.DiskUsage = diskUsage

	// Network usage
	networkRx, networkTx, err := s.getNetworkUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to get network usage: %w", err)
	}
	metrics.NetworkRx = networkRx
	metrics.NetworkTx = networkTx

	return metrics, nil
}

func (s *MonitoringService) collectWireGuardMetrics(ctx context.Context) (*WireGuardMetrics, error) {
	metrics := &WireGuardMetrics{
		Status:    "unknown",
		Peers:     0,
		RxBytes:   0,
		TxBytes:   0,
		Latency:   0,
		PacketLoss: 0,
	}

	// Check if WireGuard interface is up
	isUp, err := s.wgManager.IsInterfaceUp()
	if err != nil {
		return nil, fmt.Errorf("failed to check interface status: %w", err)
	}

	if !isUp {
		metrics.Status = "down"
		metrics.InterfaceStatus = "down"
		return metrics, nil
	}

	metrics.Status = "up"
	metrics.InterfaceStatus = "up"

	// Get interface status
	status, err := s.wgManager.GetInterfaceStatus()
	if err != nil {
		return nil, fmt.Errorf("failed to get interface status: %w", err)
	}

	metrics.Peers = len(status.Peers)

	// Calculate total traffic and get latest handshake
	var totalRx, totalTx int64
	var latestHandshake time.Time

	for _, peer := range status.Peers {
		totalRx += peer.ReceiveBytes
		totalTx += peer.TransmitBytes

		if peer.LastHandshakeTime.After(latestHandshake) {
			latestHandshake = peer.LastHandshakeTime
		}
	}

	metrics.RxBytes = totalRx
	metrics.TxBytes = totalTx
	metrics.LastHandshake = latestHandshake

	// Measure latency and packet loss to peers
	if len(status.Peers) > 0 {
		latency, packetLoss := s.measureNetworkQuality(status.Peers[0].Endpoint)
		metrics.Latency = latency
		metrics.PacketLoss = packetLoss
	}

	return metrics, nil
}

func (s *MonitoringService) getCPUUsage() (float64, error) {
	if runtime.GOOS == "linux" {
		return s.getLinuxCPUUsage()
	}
	return 0, fmt.Errorf("CPU usage not supported on %s", runtime.GOOS)
}

func (s *MonitoringService) getLinuxCPUUsage() (float64, error) {
	// Read /proc/stat for CPU usage
	cmd := exec.Command("top", "-bn1")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to execute top command: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Cpu(s):") {
			// Parse CPU usage from top output
			parts := strings.Fields(line)
			for i, part := range parts {
				if strings.Contains(part, "us,") {
					cpuStr := strings.TrimSuffix(part, "%us,")
					if cpu, err := strconv.ParseFloat(cpuStr, 64); err == nil {
						return cpu, nil
					}
				}
			}
		}
	}

	return 0, fmt.Errorf("failed to parse CPU usage")
}

func (s *MonitoringService) getMemoryUsage() (float64, error) {
	if runtime.GOOS == "linux" {
		return s.getLinuxMemoryUsage()
	}
	return 0, fmt.Errorf("memory usage not supported on %s", runtime.GOOS)
}

func (s *MonitoringService) getLinuxMemoryUsage() (float64, error) {
	// Read /proc/meminfo
	content, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, fmt.Errorf("failed to read /proc/meminfo: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var memTotal, memFree, memBuffers, memCached int64

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			switch fields[0] {
			case "MemTotal:":
				memTotal, _ = strconv.ParseInt(fields[1], 10, 64)
			case "MemFree:":
				memFree, _ = strconv.ParseInt(fields[1], 10, 64)
			case "Buffers:":
				memBuffers, _ = strconv.ParseInt(fields[1], 10, 64)
			case "Cached:":
				memCached, _ = strconv.ParseInt(fields[1], 10, 64)
			}
		}
	}

	memUsed := memTotal - memFree - memBuffers - memCached
	if memTotal > 0 {
		return float64(memUsed) / float64(memTotal) * 100, nil
	}

	return 0, fmt.Errorf("failed to calculate memory usage")
}

func (s *MonitoringService) getDiskUsage() (float64, error) {
	if runtime.GOOS == "linux" {
		return s.getLinuxDiskUsage()
	}
	return 0, fmt.Errorf("disk usage not supported on %s", runtime.GOOS)
}

func (s *MonitoringService) getLinuxDiskUsage() (float64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return 0, fmt.Errorf("failed to get disk usage: %w", err)
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := total - free

	if total > 0 {
		return float64(used) / float64(total) * 100, nil
	}

	return 0, nil
}

func (s *MonitoringService) getNetworkUsage() (int64, int64, error) {
	if runtime.GOOS == "linux" {
		return s.getLinuxNetworkUsage()
	}
	return 0, 0, fmt.Errorf("network usage not supported on %s", runtime.GOOS)
}

func (s *MonitoringService) getLinuxNetworkUsage() (int64, int64, error) {
	// Read /proc/net/dev
	content, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read /proc/net/dev: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var totalRx, totalTx int64

	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				interfaceName := strings.TrimSpace(parts[0])
				// Skip loopback interface
				if interfaceName == "lo" {
					continue
				}

				fields := strings.Fields(parts[1])
				if len(fields) >= 9 {
					rx, _ := strconv.ParseInt(fields[0], 10, 64)
					tx, _ := strconv.ParseInt(fields[8], 10, 64)
					totalRx += rx
					totalTx += tx
				}
			}
		}
	}

	return totalRx, totalTx, nil
}

func (s *MonitoringService) measureNetworkQuality(endpoint string) (float64, float64) {
	if endpoint == "" {
		return 0, 0
	}

	// Extract IP from endpoint
	parts := strings.Split(endpoint, ":")
	if len(parts) == 0 {
		return 0, 0
	}
	ip := parts[0]

	// Measure latency with ping
	latency := s.measureLatency(ip)
	
	// Measure packet loss
	packetLoss := s.measurePacketLoss(ip)

	return latency, packetLoss
}

func (s *MonitoringService) measureLatency(ip string) float64 {
	cmd := exec.Command("ping", "-c", "3", "-W", "1", ip)
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	// Parse ping output for average latency
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "avg") {
			parts := strings.Split(line, "/")
			if len(parts) >= 5 {
				if latency, err := strconv.ParseFloat(parts[4], 64); err == nil {
					return latency
				}
			}
		}
	}

	return 0
}

func (s *MonitoringService) measurePacketLoss(ip string) float64 {
	cmd := exec.Command("ping", "-c", "10", "-W", "1", ip)
	output, err := cmd.Output()
	if err != nil {
		return 100 // Assume 100% packet loss if ping fails
	}

	// Parse ping output for packet loss
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "packet loss") {
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.Contains(part, "%") {
					lossStr := strings.TrimSuffix(part, "%")
					if loss, err := strconv.ParseFloat(lossStr, 64); err == nil {
						return loss
					}
				}
			}
		}
	}

	return 0
}

func (s *MonitoringService) SendMetrics(ctx context.Context, metrics *NodeMetrics) error {
	// Convert metrics to map for API call
	metricsMap := map[string]interface{}{
		"cpu_usage":         metrics.SystemMetrics.CPUUsage,
		"memory_usage":      metrics.SystemMetrics.MemoryUsage,
		"disk_usage":        metrics.SystemMetrics.DiskUsage,
		"network_rx":        metrics.SystemMetrics.NetworkRx,
		"network_tx":        metrics.SystemMetrics.NetworkTx,
		"wg_status":         metrics.WGMetrics.Status,
		"wg_peers":          metrics.WGMetrics.Peers,
		"wg_last_handshake": metrics.WGMetrics.LastHandshake,
		"wg_rx_bytes":       metrics.WGMetrics.RxBytes,
		"wg_tx_bytes":       metrics.WGMetrics.TxBytes,
		"latency_ms":        metrics.WGMetrics.Latency,
		"packet_loss":       metrics.WGMetrics.PacketLoss,
		"errors":            metrics.Errors,
		"timestamp":         metrics.Timestamp,
	}

	// Send metrics to controller
	url := fmt.Sprintf("%s/monitoring/nodes/%s/metrics", s.config.Controller.URL, s.nodeID.String())
	
	// This would be implemented in the controller client
	// For now, we'll just log the metrics
	fmt.Printf("Sending metrics to controller: %+v\n", metricsMap)

	return nil
}

func (s *MonitoringService) StartPeriodicCollection(ctx context.Context) {
	ticker := time.NewTicker(s.config.Monitoring.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics, err := s.CollectMetrics(ctx)
			if err != nil {
				fmt.Printf("Failed to collect metrics: %v\n", err)
				continue
			}

			if err := s.SendMetrics(ctx, metrics); err != nil {
				fmt.Printf("Failed to send metrics: %v\n", err)
			}
		}
	}
}

func (s *MonitoringService) GetCurrentMetrics(ctx context.Context) (*NodeMetrics, error) {
	return s.CollectMetrics(ctx)
}

func (s *MonitoringService) GetSystemInfo() map[string]interface{} {
	return map[string]interface{}{
		"os":           runtime.GOOS,
		"arch":         runtime.GOARCH,
		"go_version":   runtime.Version(),
		"num_cpu":      runtime.NumCPU(),
		"node_id":      s.nodeID.String(),
		"node_name":    s.config.Node.Name,
		"node_type":    s.config.Node.Type,
		"wg_interface": s.config.WireGuard.Interface,
	}
}