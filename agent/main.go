package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/wg-hubspoke/wg-hubspoke/agent/client"
	"github.com/wg-hubspoke/wg-hubspoke/agent/config"
	"github.com/wg-hubspoke/wg-hubspoke/agent/wg"
	"github.com/wg-hubspoke/wg-hubspoke/common/types"
	"golang.org/x/crypto/curve25519"
)

var (
	version    = "dev"
	buildTime  = "unknown"
	commitHash = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "wg-sdwan-agent",
		Short: "WireGuard SD-WAN Agent",
		Long:  "Agent daemon for WireGuard SD-WAN hub-and-spoke network",
		Run:   runAgent,
	}

	rootCmd.Flags().StringP("config", "c", "/etc/wireguard-sdwan/agent.yaml", "Configuration file path")
	rootCmd.Flags().StringP("controller-url", "u", "", "Controller URL")
	rootCmd.Flags().StringP("node-name", "n", "", "Node name")
	rootCmd.Flags().StringP("node-type", "t", "spoke", "Node type (hub or spoke)")
	rootCmd.Flags().StringP("endpoint", "e", "", "Node endpoint")
	rootCmd.Flags().IntP("port", "p", 0, "Node port")
	rootCmd.Flags().BoolP("daemon", "d", false, "Run as daemon")
	rootCmd.Flags().BoolP("version", "v", false, "Show version")

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func runAgent(cmd *cobra.Command, args []string) {
	showVersion, _ := cmd.Flags().GetBool("version")
	if showVersion {
		fmt.Printf("wg-sdwan-agent %s (built %s, commit %s)\n", version, buildTime, commitHash)
		return
	}

	configPath, _ := cmd.Flags().GetString("config")
	controllerURL, _ := cmd.Flags().GetString("controller-url")
	nodeName, _ := cmd.Flags().GetString("node-name")
	nodeType, _ := cmd.Flags().GetString("node-type")
	endpoint, _ := cmd.Flags().GetString("endpoint")
	port, _ := cmd.Flags().GetInt("port")
	daemon, _ := cmd.Flags().GetBool("daemon")

	// Initialize configuration manager
	configManager := config.NewManager(configPath)

	// Load or create configuration
	if !configManager.ConfigExists() {
		if controllerURL == "" || nodeName == "" {
			log.Fatal("Configuration file not found. Please provide --controller-url and --node-name")
		}

		if err := configManager.CreateDefaultConfig(controllerURL, nodeName, nodeType); err != nil {
			log.Fatalf("Failed to create default configuration: %v", err)
		}
	}

	if err := configManager.LoadConfig(); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	agentConfig := configManager.GetConfig()

	// Override configuration with command line flags
	if controllerURL != "" {
		agentConfig.Controller.URL = controllerURL
	}
	if nodeName != "" {
		agentConfig.Node.Name = nodeName
	}
	if nodeType != "" {
		agentConfig.Node.Type = nodeType
	}
	if endpoint != "" {
		agentConfig.Node.Endpoint = endpoint
	}
	if port > 0 {
		agentConfig.Node.Port = port
	}

	// Initialize WireGuard manager
	wgManager, err := wg.NewManager(agentConfig.WireGuard.Interface)
	if err != nil {
		log.Fatalf("Failed to initialize WireGuard manager: %v", err)
	}
	defer wgManager.Close()

	// Check if WireGuard is available
	if err := wgManager.IsWireGuardAvailable(); err != nil {
		log.Fatalf("WireGuard not available: %v", err)
	}

	// Initialize controller client
	controllerClient := client.NewControllerClient(agentConfig.Controller.URL)

	// Start agent
	agent := &Agent{
		config:           agentConfig,
		configManager:    configManager,
		wgManager:        wgManager,
		controllerClient: controllerClient,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down agent...")
		cancel()
	}()

	// Run agent
	if daemon {
		if err := agent.RunDaemon(ctx); err != nil {
			log.Fatalf("Agent daemon failed: %v", err)
		}
	} else {
		if err := agent.RunOnce(ctx); err != nil {
			log.Fatalf("Agent run failed: %v", err)
		}
	}
}

type Agent struct {
	config           *config.AgentConfig
	configManager    *config.Manager
	wgManager        *wg.Manager
	controllerClient *client.ControllerClient
}

func (a *Agent) RunOnce(ctx context.Context) error {
	// Register or update node
	if err := a.registerNode(ctx); err != nil {
		return fmt.Errorf("failed to register node: %w", err)
	}

	// Get configuration
	if err := a.updateConfiguration(ctx); err != nil {
		return fmt.Errorf("failed to update configuration: %w", err)
	}

	// Apply configuration
	if err := a.applyConfiguration(ctx); err != nil {
		return fmt.Errorf("failed to apply configuration: %w", err)
	}

	return nil
}

func (a *Agent) RunDaemon(ctx context.Context) error {
	log.Printf("Starting WireGuard SD-WAN agent daemon (version %s)", version)

	// Initial setup
	if err := a.RunOnce(ctx); err != nil {
		return fmt.Errorf("initial setup failed: %w", err)
	}

	// Start periodic tasks
	heartbeatTicker := time.NewTicker(a.config.Controller.HeartbeatInterval)
	defer heartbeatTicker.Stop()

	configTicker := time.NewTicker(a.config.Controller.ConfigRefreshInterval)
	defer configTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-heartbeatTicker.C:
			if err := a.heartbeat(ctx); err != nil {
				log.Printf("Heartbeat failed: %v", err)
			}
		case <-configTicker.C:
			if err := a.updateConfiguration(ctx); err != nil {
				log.Printf("Config update failed: %v", err)
			} else {
				if err := a.applyConfiguration(ctx); err != nil {
					log.Printf("Config apply failed: %v", err)
				}
			}
		}
	}
}

func (a *Agent) registerNode(ctx context.Context) error {
	// Generate key pair if not exists
	if a.config.WireGuard.PrivateKey == "" || a.config.WireGuard.PublicKey == "" {
		privateKey, publicKey, err := a.generateKeyPair()
		if err != nil {
			return fmt.Errorf("failed to generate key pair: %w", err)
		}

		a.config.WireGuard.PrivateKey = privateKey
		a.config.WireGuard.PublicKey = publicKey

		if err := a.configManager.SaveConfig(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	// Register with controller
	req := types.NodeRegistrationRequest{
		Name:      a.config.Node.Name,
		NodeType:  a.config.Node.Type,
		PublicKey: a.config.WireGuard.PublicKey,
		Endpoint:  a.config.Node.Endpoint,
		Port:      a.config.Node.Port,
	}

	resp, err := a.controllerClient.RegisterNode(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to register node: %w", err)
	}

	// Extract node ID from response
	if nodeData, ok := resp.Data.(map[string]interface{}); ok {
		if nodeID, ok := nodeData["id"].(string); ok {
			a.config.Node.ID = nodeID
			if err := a.configManager.SaveConfig(); err != nil {
				return fmt.Errorf("failed to save node ID: %w", err)
			}
		}
	}

	log.Printf("Node registered successfully: %s", a.config.Node.Name)
	return nil
}

func (a *Agent) updateConfiguration(ctx context.Context) error {
	if a.config.Node.ID == "" {
		return fmt.Errorf("node ID not set")
	}

	config, err := a.controllerClient.GetNodeConfig(ctx, a.config.Node.ID)
	if err != nil {
		return fmt.Errorf("failed to get node config: %w", err)
	}

	// Generate WireGuard configuration
	wgConfig, err := a.configManager.GenerateWireGuardConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to generate WireGuard config: %w", err)
	}

	// Write configuration to file
	if err := a.configManager.WriteWireGuardConfig(wgConfig); err != nil {
		return fmt.Errorf("failed to write WireGuard config: %w", err)
	}

	log.Printf("Configuration updated successfully")
	return nil
}

func (a *Agent) applyConfiguration(ctx context.Context) error {
	configPath := a.config.WireGuard.ConfigPath
	if configPath == "" {
		configPath = fmt.Sprintf("/etc/wireguard/%s.conf", a.config.WireGuard.Interface)
	}

	// Validate configuration
	if err := a.wgManager.ValidateConfig(configPath); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Check if interface is already up
	isUp, err := a.wgManager.IsInterfaceUp()
	if err != nil {
		return fmt.Errorf("failed to check interface status: %w", err)
	}

	if isUp {
		// Restart interface with new configuration
		if err := a.wgManager.RestartInterface(ctx, configPath); err != nil {
			return fmt.Errorf("failed to restart interface: %w", err)
		}
	} else {
		// Start interface
		if err := a.wgManager.ApplyConfig(ctx, configPath); err != nil {
			return fmt.Errorf("failed to apply configuration: %w", err)
		}
	}

	// Update node status to active
	if err := a.controllerClient.UpdateNodeStatus(ctx, a.config.Node.ID, "active"); err != nil {
		log.Printf("Failed to update node status: %v", err)
	}

	log.Printf("WireGuard configuration applied successfully")
	return nil
}

func (a *Agent) heartbeat(ctx context.Context) error {
	// Check controller health
	_, err := a.controllerClient.HealthCheck(ctx)
	if err != nil {
		return fmt.Errorf("controller health check failed: %w", err)
	}

	// Update node status
	if a.config.Node.ID != "" {
		if err := a.controllerClient.UpdateNodeStatus(ctx, a.config.Node.ID, "active"); err != nil {
			return fmt.Errorf("failed to update node status: %w", err)
		}
	}

	return nil
}

func (a *Agent) generateKeyPair() (string, string, error) {
	// Generate private key
	var privateKey [32]byte
	if _, err := rand.Read(privateKey[:]); err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// Generate public key
	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	privateKeyB64 := base64.StdEncoding.EncodeToString(privateKey[:])
	publicKeyB64 := base64.StdEncoding.EncodeToString(publicKey[:])

	return privateKeyB64, publicKeyB64, nil
}