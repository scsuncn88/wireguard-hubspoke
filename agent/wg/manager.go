package wg

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type Manager struct {
	client    *wgctrl.Client
	interface string
}

type InterfaceStatus struct {
	Name         string
	Type         string
	PublicKey    string
	ListenPort   int
	FirewallMark int
	Peers        []PeerStatus
}

type PeerStatus struct {
	PublicKey                   string
	Endpoint                    string
	AllowedIPs                  []string
	LastHandshakeTime           time.Time
	ReceiveBytes                int64
	TransmitBytes               int64
	PersistentKeepaliveInterval time.Duration
}

func NewManager(interfaceName string) (*Manager, error) {
	client, err := wgctrl.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create WireGuard client: %w", err)
	}

	return &Manager{
		client:    client,
		interface: interfaceName,
	}, nil
}

func (m *Manager) Close() error {
	return m.client.Close()
}

func (m *Manager) IsInterfaceUp() (bool, error) {
	_, err := m.client.Device(m.interface)
	if err != nil {
		if strings.Contains(err.Error(), "no such device") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check interface: %w", err)
	}
	return true, nil
}

func (m *Manager) GetInterfaceStatus() (*InterfaceStatus, error) {
	device, err := m.client.Device(m.interface)
	if err != nil {
		return nil, fmt.Errorf("failed to get device info: %w", err)
	}

	status := &InterfaceStatus{
		Name:         device.Name,
		Type:         device.Type.String(),
		PublicKey:    device.PublicKey.String(),
		ListenPort:   device.ListenPort,
		FirewallMark: device.FirewallMark,
		Peers:        make([]PeerStatus, 0, len(device.Peers)),
	}

	for _, peer := range device.Peers {
		allowedIPs := make([]string, 0, len(peer.AllowedIPs))
		for _, ip := range peer.AllowedIPs {
			allowedIPs = append(allowedIPs, ip.String())
		}

		peerStatus := PeerStatus{
			PublicKey:                   peer.PublicKey.String(),
			AllowedIPs:                  allowedIPs,
			LastHandshakeTime:           peer.LastHandshakeTime,
			ReceiveBytes:                peer.ReceiveBytes,
			TransmitBytes:               peer.TransmitBytes,
			PersistentKeepaliveInterval: peer.PersistentKeepaliveInterval,
		}

		if peer.Endpoint != nil {
			peerStatus.Endpoint = peer.Endpoint.String()
		}

		status.Peers = append(status.Peers, peerStatus)
	}

	return status, nil
}

func (m *Manager) ApplyConfig(ctx context.Context, configPath string) error {
	cmd := exec.CommandContext(ctx, "wg-quick", "down", m.interface)
	cmd.Run() // Ignore errors, interface might not be up

	cmd = exec.CommandContext(ctx, "wg-quick", "up", configPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to apply WireGuard config: %w", err)
	}

	return nil
}

func (m *Manager) StopInterface(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "wg-quick", "down", m.interface)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop WireGuard interface: %w", err)
	}

	return nil
}

func (m *Manager) RestartInterface(ctx context.Context, configPath string) error {
	if err := m.StopInterface(ctx); err != nil {
		return fmt.Errorf("failed to stop interface: %w", err)
	}

	time.Sleep(1 * time.Second)

	if err := m.ApplyConfig(ctx, configPath); err != nil {
		return fmt.Errorf("failed to start interface: %w", err)
	}

	return nil
}

func (m *Manager) UpdatePeerEndpoint(ctx context.Context, publicKey, endpoint string) error {
	pubKey, err := wgtypes.ParseKey(publicKey)
	if err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}

	udpAddr, err := wgtypes.ParseEndpoint(endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint: %w", err)
	}

	peerConfig := wgtypes.PeerConfig{
		PublicKey:    pubKey,
		UpdateOnly:   true,
		ReplaceAllowedIPs: false,
		Endpoint:     &udpAddr,
	}

	config := wgtypes.Config{
		Peers: []wgtypes.PeerConfig{peerConfig},
	}

	if err := m.client.ConfigureDevice(m.interface, config); err != nil {
		return fmt.Errorf("failed to update peer endpoint: %w", err)
	}

	return nil
}

func (m *Manager) RemovePeer(ctx context.Context, publicKey string) error {
	pubKey, err := wgtypes.ParseKey(publicKey)
	if err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}

	peerConfig := wgtypes.PeerConfig{
		PublicKey: pubKey,
		Remove:    true,
	}

	config := wgtypes.Config{
		Peers: []wgtypes.PeerConfig{peerConfig},
	}

	if err := m.client.ConfigureDevice(m.interface, config); err != nil {
		return fmt.Errorf("failed to remove peer: %w", err)
	}

	return nil
}

func (m *Manager) GetPeerStats(publicKey string) (*PeerStatus, error) {
	status, err := m.GetInterfaceStatus()
	if err != nil {
		return nil, err
	}

	for _, peer := range status.Peers {
		if peer.PublicKey == publicKey {
			return &peer, nil
		}
	}

	return nil, fmt.Errorf("peer not found")
}

func (m *Manager) IsWireGuardAvailable() error {
	// Check if WireGuard is available
	cmd := exec.Command("wg", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("WireGuard not available: %w", err)
	}

	// Check if wg-quick is available
	cmd = exec.Command("wg-quick", "--help")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wg-quick not available: %w", err)
	}

	return nil
}

func (m *Manager) ValidateConfig(configPath string) error {
	// Use wg-quick to validate the configuration
	cmd := exec.Command("wg-quick", "strip", configPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("invalid WireGuard configuration: %w", err)
	}

	return nil
}