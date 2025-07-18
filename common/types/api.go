package types

import (
	"time"

	"github.com/google/uuid"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

type PaginatedResponse struct {
	APIResponse
	Pagination PaginationInfo `json:"pagination"`
}

type PaginationInfo struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type NodeRegistrationRequest struct {
	Name       string   `json:"name" binding:"required"`
	NodeType   string   `json:"node_type" binding:"required,oneof=hub spoke"`
	PublicKey  string   `json:"public_key" binding:"required"`
	Endpoint   string   `json:"endpoint"`
	Port       int      `json:"port"`
	AllowedIPs []string `json:"allowed_ips"`
}

type NodeUpdateRequest struct {
	Name       *string  `json:"name,omitempty"`
	Endpoint   *string  `json:"endpoint,omitempty"`
	Port       *int     `json:"port,omitempty"`
	AllowedIPs []string `json:"allowed_ips,omitempty"`
	Status     *string  `json:"status,omitempty"`
}

type NodeConfigResponse struct {
	Interface   WGInterface `json:"interface"`
	Peers       []WGPeer    `json:"peers"`
	GeneratedAt time.Time   `json:"generated_at"`
}

type WGInterface struct {
	PrivateKey string   `json:"private_key"`
	Address    []string `json:"address"`
	ListenPort int      `json:"listen_port"`
	MTU        int      `json:"mtu"`
}

type WGPeer struct {
	PublicKey           string   `json:"public_key"`
	AllowedIPs          []string `json:"allowed_ips"`
	Endpoint            string   `json:"endpoint,omitempty"`
	PersistentKeepalive int      `json:"persistent_keepalive,omitempty"`
}

type PolicyRequest struct {
	Name              string     `json:"name" binding:"required"`
	Description       string     `json:"description"`
	SourceNodeID      *uuid.UUID `json:"source_node_id"`
	DestinationNodeID *uuid.UUID `json:"destination_node_id"`
	SourceCIDR        string     `json:"source_cidr"`
	DestinationCIDR   string     `json:"destination_cidr"`
	Protocol          string     `json:"protocol"`
	Port              *int       `json:"port"`
	Action            string     `json:"action" binding:"required,oneof=allow deny"`
	Priority          int        `json:"priority"`
	Enabled           bool       `json:"enabled"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserInfo  `json:"user"`
}

type UserInfo struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	IsActive bool      `json:"is_active"`
}

type TopologyResponse struct {
	Nodes []NodeInfo       `json:"nodes"`
	Links []TopologyLink   `json:"links"`
}

type NodeInfo struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	NodeType    string    `json:"node_type"`
	Status      string    `json:"status"`
	AllocatedIP string    `json:"allocated_ip"`
	Endpoint    string    `json:"endpoint"`
	IsOnline    bool      `json:"is_online"`
}

type TopologyLink struct {
	Source uuid.UUID `json:"source"`
	Target uuid.UUID `json:"target"`
	Type   string    `json:"type"`
}

type HealthStatus struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

type MetricsResponse struct {
	NodesTotal    int                    `json:"nodes_total"`
	NodesActive   int                    `json:"nodes_active"`
	HubsTotal     int                    `json:"hubs_total"`
	SpokesTotal   int                    `json:"spokes_total"`
	PoliciesTotal int                    `json:"policies_total"`
	TrafficStats  map[string]interface{} `json:"traffic_stats"`
}