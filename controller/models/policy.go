package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PolicyAction string

const (
	PolicyActionAllow PolicyAction = "allow"
	PolicyActionDeny  PolicyAction = "deny"
)

type Policy struct {
	ID                uuid.UUID     `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name              string        `json:"name" gorm:"not null"`
	Description       string        `json:"description"`
	SourceNodeID      *uuid.UUID    `json:"source_node_id" gorm:"type:uuid"`
	DestinationNodeID *uuid.UUID    `json:"destination_node_id" gorm:"type:uuid"`
	SourceNode        *Node         `json:"source_node,omitempty" gorm:"foreignKey:SourceNodeID"`
	DestinationNode   *Node         `json:"destination_node,omitempty" gorm:"foreignKey:DestinationNodeID"`
	SourceCIDR        string        `json:"source_cidr"`
	DestinationCIDR   string        `json:"destination_cidr"`
	Protocol          string        `json:"protocol"`
	Port              *int          `json:"port"`
	Action            PolicyAction  `json:"action" gorm:"not null"`
	Priority          int           `json:"priority" gorm:"default:100"`
	Enabled           bool          `json:"enabled" gorm:"default:true"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
}

func (p *Policy) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

func (p *Policy) IsAllow() bool {
	return p.Action == PolicyActionAllow
}

func (p *Policy) IsDeny() bool {
	return p.Action == PolicyActionDeny
}

func (p *Policy) TableName() string {
	return "policies"
}