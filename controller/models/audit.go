package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuditAction string

const (
	AuditActionCreate AuditAction = "create"
	AuditActionUpdate AuditAction = "update"
	AuditActionDelete AuditAction = "delete"
	AuditActionLogin  AuditAction = "login"
	AuditActionLogout AuditAction = "logout"
)

type AuditLog struct {
	ID          uuid.UUID   `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID      *uuid.UUID  `json:"user_id" gorm:"type:uuid"`
	User        *User       `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Action      AuditAction `json:"action" gorm:"not null"`
	Resource    string      `json:"resource"`
	ResourceID  *uuid.UUID  `json:"resource_id" gorm:"type:uuid"`
	Description string      `json:"description"`
	IPAddress   string      `json:"ip_address"`
	UserAgent   string      `json:"user_agent"`
	Metadata    string      `json:"metadata" gorm:"type:jsonb"`
	CreatedAt   time.Time   `json:"created_at"`
}

func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

func (a *AuditLog) TableName() string {
	return "audit_logs"
}