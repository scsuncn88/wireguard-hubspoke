package services

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wg-hubspoke/wg-hubspoke/common/types"
	"github.com/wg-hubspoke/wg-hubspoke/controller/models"
	"gorm.io/gorm"
)

type HAService struct {
	db           *gorm.DB
	config       *types.Config
	nodeID       string
	clusterID    string
	isLeader     bool
	peerNodes    map[string]*PeerNode
	mutex        sync.RWMutex
	leaderChan   chan bool
	healthTicker *time.Ticker
	httpClient   *http.Client
}

type PeerNode struct {
	ID       string    `json:"id"`
	Address  string    `json:"address"`
	Port     int       `json:"port"`
	Status   string    `json:"status"`
	LastSeen time.Time `json:"last_seen"`
	IsLeader bool      `json:"is_leader"`
	Version  string    `json:"version"`
}

type ClusterStatus struct {
	ClusterID    string               `json:"cluster_id"`
	Leader       string               `json:"leader"`
	Nodes        map[string]*PeerNode `json:"nodes"`
	Healthy      bool                 `json:"healthy"`
	LastElection time.Time            `json:"last_election"`
}

type HealthResponse struct {
	NodeID    string    `json:"node_id"`
	Status    string    `json:"status"`
	IsLeader  bool      `json:"is_leader"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

type LeaderElectionRequest struct {
	NodeID       string    `json:"node_id"`
	ClusterID    string    `json:"cluster_id"`
	Term         int64     `json:"term"`
	LastLogIndex int64     `json:"last_log_index"`
	Timestamp    time.Time `json:"timestamp"`
}

type LeaderElectionResponse struct {
	Success   bool   `json:"success"`
	Term      int64  `json:"term"`
	VoterID   string `json:"voter_id"`
	Timestamp time.Time `json:"timestamp"`
}

func NewHAService(db *gorm.DB, config *types.Config) *HAService {
	nodeID := uuid.New().String()
	if config.HA.NodeID != "" {
		nodeID = config.HA.NodeID
	}

	return &HAService{
		db:        db,
		config:    config,
		nodeID:    nodeID,
		clusterID: config.HA.ClusterID,
		isLeader:  false,
		peerNodes: make(map[string]*PeerNode),
		leaderChan: make(chan bool, 1),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

func (s *HAService) Start(ctx context.Context) error {
	if !s.config.HA.Enabled {
		fmt.Println("HA not enabled, running in single node mode")
		s.isLeader = true
		return nil
	}

	fmt.Printf("Starting HA service for node %s in cluster %s\n", s.nodeID, s.clusterID)

	// Start health check ticker
	s.healthTicker = time.NewTicker(s.config.HA.HeartbeatInterval)

	// Start peer discovery
	go s.startPeerDiscovery(ctx)

	// Start health monitoring
	go s.startHealthMonitoring(ctx)

	// Start leader election
	go s.startLeaderElection(ctx)

	return nil
}

func (s *HAService) Stop() error {
	if s.healthTicker != nil {
		s.healthTicker.Stop()
	}
	return nil
}

func (s *HAService) IsLeader() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.isLeader
}

func (s *HAService) GetLeader() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	for _, peer := range s.peerNodes {
		if peer.IsLeader {
			return peer.ID
		}
	}
	
	if s.isLeader {
		return s.nodeID
	}
	
	return ""
}

func (s *HAService) GetClusterStatus() *ClusterStatus {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	status := &ClusterStatus{
		ClusterID: s.clusterID,
		Leader:    s.GetLeader(),
		Nodes:     make(map[string]*PeerNode),
		Healthy:   true,
	}

	// Add self
	status.Nodes[s.nodeID] = &PeerNode{
		ID:       s.nodeID,
		Address:  "localhost",
		Port:     s.config.Server.Port,
		Status:   "healthy",
		LastSeen: time.Now(),
		IsLeader: s.isLeader,
		Version:  "1.0.0",
	}

	// Add peers
	for id, peer := range s.peerNodes {
		status.Nodes[id] = peer
		if time.Since(peer.LastSeen) > 2*s.config.HA.HeartbeatInterval {
			status.Healthy = false
		}
	}

	return status
}

func (s *HAService) startPeerDiscovery(ctx context.Context) {
	// In a real implementation, this would discover peers via:
	// - Service discovery (Consul, etcd)
	// - Configuration file
	// - DNS discovery
	// - Kubernetes API
	
	// For now, we'll use a simple static configuration
	if len(s.config.HA.PeerNodes) > 0 {
		for _, peerAddr := range s.config.HA.PeerNodes {
			peerID := uuid.New().String()
			s.mutex.Lock()
			s.peerNodes[peerID] = &PeerNode{
				ID:       peerID,
				Address:  peerAddr,
				Port:     s.config.Server.Port,
				Status:   "unknown",
				LastSeen: time.Now(),
				IsLeader: false,
				Version:  "1.0.0",
			}
			s.mutex.Unlock()
		}
	}
}

func (s *HAService) startHealthMonitoring(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.healthTicker.C:
			s.checkPeerHealth(ctx)
		}
	}
}

func (s *HAService) checkPeerHealth(ctx context.Context) {
	s.mutex.RLock()
	peers := make(map[string]*PeerNode)
	for id, peer := range s.peerNodes {
		peers[id] = peer
	}
	s.mutex.RUnlock()

	for id, peer := range peers {
		go func(peerID string, peerNode *PeerNode) {
			health := s.checkSinglePeerHealth(ctx, peerNode)
			
			s.mutex.Lock()
			if health != nil {
				peerNode.Status = health.Status
				peerNode.IsLeader = health.IsLeader
				peerNode.LastSeen = health.Timestamp
				peerNode.Version = health.Version
			} else {
				peerNode.Status = "unhealthy"
				peerNode.LastSeen = time.Now()
			}
			s.mutex.Unlock()
		}(id, peer)
	}
}

func (s *HAService) checkSinglePeerHealth(ctx context.Context, peer *PeerNode) *HealthResponse {
	url := fmt.Sprintf("http://%s:%d/ha/health", peer.Address, peer.Port)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil
	}

	return &health
}

func (s *HAService) startLeaderElection(ctx context.Context) {
	// Initial election delay
	time.Sleep(time.Duration(s.nodeID[0]%10) * time.Second)

	electionTicker := time.NewTicker(s.config.HA.ElectionTimeout)
	defer electionTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-electionTicker.C:
			if !s.hasHealthyLeader() {
				s.attemptLeaderElection(ctx)
			}
		}
	}
}

func (s *HAService) hasHealthyLeader() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.isLeader {
		return true
	}

	for _, peer := range s.peerNodes {
		if peer.IsLeader && peer.Status == "healthy" && time.Since(peer.LastSeen) < 2*s.config.HA.HeartbeatInterval {
			return true
		}
	}

	return false
}

func (s *HAService) attemptLeaderElection(ctx context.Context) {
	fmt.Printf("Node %s attempting leader election\n", s.nodeID)

	s.mutex.RLock()
	peers := make(map[string]*PeerNode)
	for id, peer := range s.peerNodes {
		peers[id] = peer
	}
	s.mutex.RUnlock()

	votes := 1 // Vote for self
	totalNodes := len(peers) + 1
	requiredVotes := (totalNodes / 2) + 1

	// Send election requests to peers
	for _, peer := range peers {
		if peer.Status == "healthy" {
			if s.requestVote(ctx, peer) {
				votes++
			}
		}
	}

	// Check if we have majority
	if votes >= requiredVotes {
		s.becomeLeader()
		fmt.Printf("Node %s elected as leader with %d/%d votes\n", s.nodeID, votes, totalNodes)
	} else {
		fmt.Printf("Node %s failed to get majority: %d/%d votes\n", s.nodeID, votes, totalNodes)
	}
}

func (s *HAService) requestVote(ctx context.Context, peer *PeerNode) bool {
	url := fmt.Sprintf("http://%s:%d/ha/election", peer.Address, peer.Port)
	
	request := LeaderElectionRequest{
		NodeID:       s.nodeID,
		ClusterID:    s.clusterID,
		Term:         time.Now().Unix(),
		LastLogIndex: 0, // Simplified
		Timestamp:    time.Now(),
	}

	body, err := json.Marshal(request)
	if err != nil {
		return false
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	var response LeaderElectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return false
	}

	return response.Success
}

func (s *HAService) becomeLeader() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isLeader {
		s.isLeader = true
		
		// Notify other components
		select {
		case s.leaderChan <- true:
		default:
		}

		// Start leader-specific tasks
		go s.startLeaderTasks()
	}
}

func (s *HAService) stepDownAsLeader() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.isLeader {
		s.isLeader = false
		
		// Notify other components
		select {
		case s.leaderChan <- false:
		default:
		}

		fmt.Printf("Node %s stepped down as leader\n", s.nodeID)
	}
}

func (s *HAService) startLeaderTasks() {
	fmt.Printf("Node %s starting leader tasks\n", s.nodeID)

	// Leader-specific tasks:
	// 1. Configuration synchronization
	// 2. Node health monitoring
	// 3. Policy enforcement
	// 4. Certificate management
	// 5. Database maintenance
	
	// For now, just periodic health announcements
	ticker := time.NewTicker(s.config.HA.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !s.IsLeader() {
				return
			}
			s.announceLeadership()
		}
	}
}

func (s *HAService) announceLeadership() {
	s.mutex.RLock()
	peers := make(map[string]*PeerNode)
	for id, peer := range s.peerNodes {
		peers[id] = peer
	}
	s.mutex.RUnlock()

	for _, peer := range peers {
		go func(p *PeerNode) {
			url := fmt.Sprintf("http://%s:%d/ha/leader", p.Address, p.Port)
			
			announcement := map[string]interface{}{
				"leader_id":   s.nodeID,
				"cluster_id":  s.clusterID,
				"term":        time.Now().Unix(),
				"timestamp":   time.Now(),
			}

			body, err := json.Marshal(announcement)
			if err != nil {
				return
			}

			req, _ := http.NewRequest("POST", url, strings.NewReader(string(body)))
			req.Header.Set("Content-Type", "application/json")
			
			s.httpClient.Do(req)
		}(peer)
	}
}

func (s *HAService) HandleVoteRequest(request *LeaderElectionRequest) *LeaderElectionResponse {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	response := &LeaderElectionResponse{
		Success:   false,
		Term:      time.Now().Unix(),
		VoterID:   s.nodeID,
		Timestamp: time.Now(),
	}

	// Simple voting logic
	if request.ClusterID == s.clusterID && !s.isLeader {
		response.Success = true
	}

	return response
}

func (s *HAService) HandleLeaderAnnouncement(announcement map[string]interface{}) {
	leaderID, ok := announcement["leader_id"].(string)
	if !ok {
		return
	}

	clusterID, ok := announcement["cluster_id"].(string)
	if !ok || clusterID != s.clusterID {
		return
	}

	// If we're the leader but someone else is announcing, step down
	if s.isLeader && leaderID != s.nodeID {
		s.stepDownAsLeader()
	}

	// Update peer information
	s.mutex.Lock()
	for id, peer := range s.peerNodes {
		if id == leaderID {
			peer.IsLeader = true
			peer.LastSeen = time.Now()
			peer.Status = "healthy"
		} else {
			peer.IsLeader = false
		}
	}
	s.mutex.Unlock()
}

func (s *HAService) GetHealthStatus() *HealthResponse {
	return &HealthResponse{
		NodeID:    s.nodeID,
		Status:    "healthy",
		IsLeader:  s.IsLeader(),
		Timestamp: time.Now(),
		Version:   "1.0.0",
	}
}

func (s *HAService) SyncConfiguration(ctx context.Context) error {
	if !s.IsLeader() {
		return fmt.Errorf("only leader can sync configuration")
	}

	// Sync critical configuration to all peers
	// This would include:
	// - Node registrations
	// - Policy updates
	// - Certificate rotations
	// - System configuration changes

	fmt.Printf("Leader %s syncing configuration to peers\n", s.nodeID)
	return nil
}

func (s *HAService) GetLeaderChannel() <-chan bool {
	return s.leaderChan
}

func (s *HAService) EnsureLeaderOrProxy(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.IsLeader() {
			next(w, r)
			return
		}

		// Find leader and proxy request
		leader := s.GetLeader()
		if leader == "" {
			http.Error(w, "No leader available", http.StatusServiceUnavailable)
			return
		}

		// In a real implementation, this would proxy the request to the leader
		http.Error(w, "Not the leader, redirect to leader", http.StatusTemporaryRedirect)
	}
}