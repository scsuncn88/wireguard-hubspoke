package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NodeType string

const (
	NodeTypeHub   NodeType = "hub"
	NodeTypeSpoke NodeType = "spoke"
)

type NodeStatus string

const (
	NodeStatusPending    NodeStatus = "pending"
	NodeStatusActive     NodeStatus = "active"
	NodeStatusInactive   NodeStatus = "inactive"
	NodeStatusDisabled   NodeStatus = "disabled"
)

type Node struct {
	ID                uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name              string     `json:"name" gorm:"uniqueIndex;not null"`
	NodeType          NodeType   `json:"node_type" gorm:"not null"`
	PublicKey         string     `json:"public_key" gorm:"not null"`
	PrivateKeyHash    string     `json:"-" gorm:"column:private_key_hash"`
	AllocatedIP       string     `json:"allocated_ip" gorm:"type:inet;not null"`
	Endpoint          string     `json:"endpoint"`
	Port              int        `json:"port"`
	AllowedIPs        []string   `json:"allowed_ips" gorm:"type:text[]"`
	LastHandshake     *time.Time `json:"last_handshake"`
	Status            NodeStatus `json:"status" gorm:"default:pending"`
	PersistentKeepalive *int     `json:"persistent_keepalive"`
	MTU               int        `json:"mtu" gorm:"default:1420"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
}

func (n *Node) BeforeCreate(tx *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}

func (n *Node) IsHub() bool {
	return n.NodeType == NodeTypeHub
}

func (n *Node) IsSpoke() bool {
	return n.NodeType == NodeTypeSpoke
}

func (n *Node) IsActive() bool {
	return n.Status == NodeStatusActive
}

func (n *Node) GetEndpoint() string {
	if n.Endpoint == "" {
		return ""
	}
	if n.Port > 0 {
		return fmt.Sprintf("%s:%d", n.Endpoint, n.Port)
	}
	return n.Endpoint
}

func (n *Node) TableName() string {
	return "nodes"
}