package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/wg-hubspoke/wg-hubspoke/common/types"
	"github.com/wg-hubspoke/wg-hubspoke/controller/models"
	"golang.org/x/crypto/curve25519"
	"gorm.io/gorm"
)

var (
	ErrNodeNotFound     = errors.New("node not found")
	ErrNodeExists       = errors.New("node already exists")
	ErrInvalidNodeType  = errors.New("invalid node type")
	ErrInvalidPublicKey = errors.New("invalid public key")
)

type NodeService struct {
	db     *gorm.DB
	config *types.Config
}

func NewNodeService(db *gorm.DB, config *types.Config) *NodeService {
	return &NodeService{
		db:     db,
		config: config,
	}
}

func (s *NodeService) RegisterNode(ctx context.Context, req types.NodeRegistrationRequest) (*models.Node, error) {
	// Validate node type
	if req.NodeType != "hub" && req.NodeType != "spoke" {
		return nil, ErrInvalidNodeType
	}

	// Validate public key
	if !s.isValidPublicKey(req.PublicKey) {
		return nil, ErrInvalidPublicKey
	}

	// Check if node already exists
	var existingNode models.Node
	if err := s.db.Where("name = ?", req.Name).First(&existingNode).Error; err == nil {
		return nil, ErrNodeExists
	}

	// Allocate IP address
	allocatedIP, err := s.allocateIP(ctx, models.NodeType(req.NodeType))
	if err != nil {
		return nil, fmt.Errorf("failed to allocate IP: %w", err)
	}

	// Create node
	node := &models.Node{
		Name:        req.Name,
		NodeType:    models.NodeType(req.NodeType),
		PublicKey:   req.PublicKey,
		AllocatedIP: allocatedIP,
		Endpoint:    req.Endpoint,
		Port:        req.Port,
		AllowedIPs:  req.AllowedIPs,
		Status:      models.NodeStatusPending,
		MTU:         s.config.WG.MTU,
	}

	if s.config.WG.PersistentKeepalive > 0 {
		node.PersistentKeepalive = &s.config.WG.PersistentKeepalive
	}

	if err := s.db.Create(node).Error; err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	// Update topology if it's a spoke node
	if node.NodeType == models.NodeTypeSpoke {
		if err := s.updateTopology(ctx, node); err != nil {
			return nil, fmt.Errorf("failed to update topology: %w", err)
		}
	}

	return node, nil
}

func (s *NodeService) GetNodes(ctx context.Context, page, perPage int, nodeType, status string) ([]models.Node, int64, error) {
	var nodes []models.Node
	var total int64

	query := s.db.Model(&models.Node{})

	if nodeType != "" {
		query = query.Where("node_type = ?", nodeType)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count nodes: %w", err)
	}

	offset := (page - 1) * perPage
	if err := query.Offset(offset).Limit(perPage).Find(&nodes).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get nodes: %w", err)
	}

	return nodes, total, nil
}

func (s *NodeService) GetNode(ctx context.Context, id uuid.UUID) (*models.Node, error) {
	var node models.Node
	if err := s.db.Where("id = ?", id).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNodeNotFound
		}
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	return &node, nil
}

func (s *NodeService) UpdateNode(ctx context.Context, id uuid.UUID, req types.NodeUpdateRequest) (*models.Node, error) {
	var node models.Node
	if err := s.db.Where("id = ?", id).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNodeNotFound
		}
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Endpoint != nil {
		updates["endpoint"] = *req.Endpoint
	}
	if req.Port != nil {
		updates["port"] = *req.Port
	}
	if req.AllowedIPs != nil {
		updates["allowed_ips"] = req.AllowedIPs
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if len(updates) > 0 {
		if err := s.db.Model(&node).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("failed to update node: %w", err)
		}
	}

	return &node, nil
}

func (s *NodeService) DeleteNode(ctx context.Context, id uuid.UUID) error {
	var node models.Node
	if err := s.db.Where("id = ?", id).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNodeNotFound
		}
		return fmt.Errorf("failed to get node: %w", err)
	}

	if err := s.db.Delete(&node).Error; err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	return nil
}

func (s *NodeService) GetNodeConfig(ctx context.Context, id uuid.UUID) (*types.NodeConfigResponse, error) {
	var node models.Node
	if err := s.db.Where("id = ?", id).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNodeNotFound
		}
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	// Generate private key for the node (this should be done during registration in practice)
	privateKey, err := s.generatePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Get peers for this node
	peers, err := s.getPeersForNode(ctx, &node)
	if err != nil {
		return nil, fmt.Errorf("failed to get peers: %w", err)
	}

	config := &types.NodeConfigResponse{
		Interface: types.WGInterface{
			PrivateKey: privateKey,
			Address:    []string{node.AllocatedIP},
			ListenPort: node.Port,
			MTU:        node.MTU,
		},
		Peers:       peers,
		GeneratedAt: time.Now(),
	}

	return config, nil
}

func (s *NodeService) isValidPublicKey(publicKey string) bool {
	decoded, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return false
	}
	return len(decoded) == 32
}

func (s *NodeService) generatePrivateKey() (string, error) {
	var privateKey [32]byte
	if _, err := rand.Read(privateKey[:]); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(privateKey[:]), nil
}

func (s *NodeService) generatePublicKey(privateKey string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		return "", err
	}

	var privateKeyBytes [32]byte
	copy(privateKeyBytes[:], decoded)

	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &privateKeyBytes)

	return base64.StdEncoding.EncodeToString(publicKey[:]), nil
}

func (s *NodeService) allocateIP(ctx context.Context, nodeType models.NodeType) (string, error) {
	// Parse subnet
	_, subnet, err := net.ParseCIDR(s.config.WG.Subnet)
	if err != nil {
		return "", fmt.Errorf("invalid subnet: %w", err)
	}

	// Get all allocated IPs
	var nodes []models.Node
	if err := s.db.Select("allocated_ip").Find(&nodes).Error; err != nil {
		return "", fmt.Errorf("failed to get allocated IPs: %w", err)
	}

	allocatedIPs := make(map[string]bool)
	for _, node := range nodes {
		allocatedIPs[node.AllocatedIP] = true
	}

	// Find available IP
	ip := subnet.IP
	for subnet.Contains(ip) {
		ipStr := ip.String()
		if !allocatedIPs[ipStr] && !ip.Equal(subnet.IP) {
			// Add subnet mask
			ones, _ := subnet.Mask.Size()
			return fmt.Sprintf("%s/%d", ipStr, ones), nil
		}
		// Increment IP
		for i := len(ip) - 1; i >= 0; i-- {
			ip[i]++
			if ip[i] != 0 {
				break
			}
		}
	}

	return "", errors.New("no available IP addresses")
}

func (s *NodeService) updateTopology(ctx context.Context, spokeNode *models.Node) error {
	// Find active hub nodes
	var hubNodes []models.Node
	if err := s.db.Where("node_type = ? AND status = ?", models.NodeTypeHub, models.NodeStatusActive).Find(&hubNodes).Error; err != nil {
		return fmt.Errorf("failed to get hub nodes: %w", err)
	}

	// Connect spoke to first available hub
	if len(hubNodes) > 0 {
		topology := &models.Topology{
			HubID:   hubNodes[0].ID,
			SpokeID: spokeNode.ID,
		}

		if err := s.db.Create(topology).Error; err != nil {
			return fmt.Errorf("failed to create topology: %w", err)
		}
	}

	return nil
}

func (s *NodeService) getPeersForNode(ctx context.Context, node *models.Node) ([]types.WGPeer, error) {
	var peers []types.WGPeer

	if node.NodeType == models.NodeTypeHub {
		// For hub nodes, get all connected spoke nodes
		var spokes []models.Node
		if err := s.db.Raw(`
			SELECT n.* FROM nodes n
			JOIN topology t ON n.id = t.spoke_id
			WHERE t.hub_id = ? AND n.status = ?
		`, node.ID, models.NodeStatusActive).Scan(&spokes).Error; err != nil {
			return nil, fmt.Errorf("failed to get spoke nodes: %w", err)
		}

		for _, spoke := range spokes {
			peer := types.WGPeer{
				PublicKey:  spoke.PublicKey,
				AllowedIPs: []string{spoke.AllocatedIP},
			}
			if spoke.PersistentKeepalive != nil {
				peer.PersistentKeepalive = *spoke.PersistentKeepalive
			}
			peers = append(peers, peer)
		}
	} else {
		// For spoke nodes, get their hub node
		var hub models.Node
		if err := s.db.Raw(`
			SELECT n.* FROM nodes n
			JOIN topology t ON n.id = t.hub_id
			WHERE t.spoke_id = ? AND n.status = ?
		`, node.ID, models.NodeStatusActive).Scan(&hub).Error; err != nil {
			return nil, fmt.Errorf("failed to get hub node: %w", err)
		}

		if hub.ID != uuid.Nil {
			peer := types.WGPeer{
				PublicKey:  hub.PublicKey,
				AllowedIPs: []string{"0.0.0.0/0"},
				Endpoint:   hub.GetEndpoint(),
			}
			if hub.PersistentKeepalive != nil {
				peer.PersistentKeepalive = *hub.PersistentKeepalive
			}
			peers = append(peers, peer)
		}
	}

	return peers, nil
}