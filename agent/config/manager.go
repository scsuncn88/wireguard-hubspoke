package config

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/wg-hubspoke/wg-hubspoke/common/types"
	"gopkg.in/yaml.v3"
)

type Manager struct {
	configPath string
	config     *AgentConfig
}

type AgentConfig struct {
	Controller ControllerConfig `yaml:"controller"`
	Node       NodeConfig       `yaml:"node"`
	WireGuard  WGConfig         `yaml:"wireguard"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
}

type ControllerConfig struct {
	URL             string        `yaml:"url"`
	Token           string        `yaml:"token"`
	RetryAttempts   int           `yaml:"retry_attempts"`
	RetryDelay      time.Duration `yaml:"retry_delay"`
	RequestTimeout  time.Duration `yaml:"request_timeout"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
	ConfigRefreshInterval time.Duration `yaml:"config_refresh_interval"`
}

type NodeConfig struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Endpoint string `yaml:"endpoint"`
	Port     int    `yaml:"port"`
}

type WGConfig struct {
	Interface    string `yaml:"interface"`
	ConfigPath   string `yaml:"config_path"`
	PrivateKey   string `yaml:"private_key"`
	PublicKey    string `yaml:"public_key"`
	MTU          int    `yaml:"mtu"`
}

type MonitoringConfig struct {
	Enabled         bool          `yaml:"enabled"`
	Interval        time.Duration `yaml:"interval"`
	HealthCheckPort int           `yaml:"health_check_port"`
	MetricsPath     string        `yaml:"metrics_path"`
}

func NewManager(configPath string) *Manager {
	return &Manager{
		configPath: configPath,
	}
}

func (m *Manager) LoadConfig() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	config := &AgentConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set defaults
	m.setDefaults(config)

	m.config = config
	return nil
}

func (m *Manager) SaveConfig() error {
	if m.config == nil {
		return fmt.Errorf("no config to save")
	}

	data, err := yaml.Marshal(m.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (m *Manager) GetConfig() *AgentConfig {
	return m.config
}

func (m *Manager) UpdateNodeConfig(nodeID string, privateKey string, publicKey string) error {
	if m.config == nil {
		return fmt.Errorf("config not loaded")
	}

	m.config.Node.ID = nodeID
	m.config.WireGuard.PrivateKey = privateKey
	m.config.WireGuard.PublicKey = publicKey

	return m.SaveConfig()
}

func (m *Manager) GenerateWireGuardConfig(ctx context.Context, nodeConfig *types.NodeConfigResponse) (string, error) {
	var config string

	// Interface section
	config += "[Interface]\n"
	config += fmt.Sprintf("PrivateKey = %s\n", nodeConfig.Interface.PrivateKey)
	
	for _, addr := range nodeConfig.Interface.Address {
		config += fmt.Sprintf("Address = %s\n", addr)
	}
	
	if nodeConfig.Interface.ListenPort > 0 {
		config += fmt.Sprintf("ListenPort = %d\n", nodeConfig.Interface.ListenPort)
	}
	
	if nodeConfig.Interface.MTU > 0 {
		config += fmt.Sprintf("MTU = %d\n", nodeConfig.Interface.MTU)
	}

	// Peers section
	for _, peer := range nodeConfig.Peers {
		config += "\n[Peer]\n"
		config += fmt.Sprintf("PublicKey = %s\n", peer.PublicKey)
		
		for _, allowedIP := range peer.AllowedIPs {
			config += fmt.Sprintf("AllowedIPs = %s\n", allowedIP)
		}
		
		if peer.Endpoint != "" {
			config += fmt.Sprintf("Endpoint = %s\n", peer.Endpoint)
		}
		
		if peer.PersistentKeepalive > 0 {
			config += fmt.Sprintf("PersistentKeepalive = %d\n", peer.PersistentKeepalive)
		}
	}

	return config, nil
}

func (m *Manager) WriteWireGuardConfig(config string) error {
	if m.config == nil {
		return fmt.Errorf("config not loaded")
	}

	configPath := m.config.WireGuard.ConfigPath
	if configPath == "" {
		configPath = fmt.Sprintf("/etc/wireguard/%s.conf", m.config.WireGuard.Interface)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create wireguard config directory: %w", err)
	}

	// Write config with restrictive permissions
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		return fmt.Errorf("failed to write wireguard config: %w", err)
	}

	return nil
}

func (m *Manager) setDefaults(config *AgentConfig) {
	// Controller defaults
	if config.Controller.RetryAttempts == 0 {
		config.Controller.RetryAttempts = 3
	}
	if config.Controller.RetryDelay == 0 {
		config.Controller.RetryDelay = 10 * time.Second
	}
	if config.Controller.RequestTimeout == 0 {
		config.Controller.RequestTimeout = 30 * time.Second
	}
	if config.Controller.HeartbeatInterval == 0 {
		config.Controller.HeartbeatInterval = 30 * time.Second
	}
	if config.Controller.ConfigRefreshInterval == 0 {
		config.Controller.ConfigRefreshInterval = 5 * time.Minute
	}

	// WireGuard defaults
	if config.WireGuard.Interface == "" {
		config.WireGuard.Interface = "wg0"
	}
	if config.WireGuard.ConfigPath == "" {
		config.WireGuard.ConfigPath = fmt.Sprintf("/etc/wireguard/%s.conf", config.WireGuard.Interface)
	}
	if config.WireGuard.MTU == 0 {
		config.WireGuard.MTU = 1420
	}

	// Monitoring defaults
	if config.Monitoring.Interval == 0 {
		config.Monitoring.Interval = 30 * time.Second
	}
	if config.Monitoring.HealthCheckPort == 0 {
		config.Monitoring.HealthCheckPort = 8081
	}
	if config.Monitoring.MetricsPath == "" {
		config.Monitoring.MetricsPath = "/metrics"
	}
}

func (m *Manager) ConfigExists() bool {
	_, err := os.Stat(m.configPath)
	return !os.IsNotExist(err)
}

func (m *Manager) CreateDefaultConfig(controllerURL, nodeName, nodeType string) error {
	config := &AgentConfig{
		Controller: ControllerConfig{
			URL: controllerURL,
		},
		Node: NodeConfig{
			Name: nodeName,
			Type: nodeType,
		},
		WireGuard: WGConfig{
			Interface: "wg0",
		},
		Monitoring: MonitoringConfig{
			Enabled: true,
		},
	}

	m.setDefaults(config)
	m.config = config

	return m.SaveConfig()
}