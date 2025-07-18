package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Topology struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	HubID     uuid.UUID `json:"hub_id" gorm:"type:uuid;not null"`
	SpokeID   uuid.UUID `json:"spoke_id" gorm:"type:uuid;not null"`
	Hub       Node      `json:"hub" gorm:"foreignKey:HubID"`
	Spoke     Node      `json:"spoke" gorm:"foreignKey:SpokeID"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (t *Topology) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

func (t *Topology) TableName() string {
	return "topology"
}