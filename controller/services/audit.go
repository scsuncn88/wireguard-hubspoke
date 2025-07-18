package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wg-hubspoke/wg-hubspoke/controller/models"
	"gorm.io/gorm"
)

type AuditService struct {
	db *gorm.DB
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{
		db: db,
	}
}

func (s *AuditService) LogAction(ctx context.Context, userID *uuid.UUID, action models.AuditAction, resource string, resourceID *uuid.UUID, description, ipAddress, userAgent string) {
	auditLog := &models.AuditLog{
		UserID:      userID,
		Action:      action,
		Resource:    resource,
		ResourceID:  resourceID,
		Description: description,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	}

	// Don't fail the main operation if audit logging fails
	if err := s.db.Create(auditLog).Error; err != nil {
		// Log error but don't return it
		fmt.Printf("Failed to create audit log: %v\n", err)
	}
}

func (s *AuditService) LogActionWithMetadata(ctx context.Context, userID *uuid.UUID, action models.AuditAction, resource string, resourceID *uuid.UUID, description, ipAddress, userAgent string, metadata map[string]interface{}) {
	var metadataJSON string
	if metadata != nil {
		if data, err := json.Marshal(metadata); err == nil {
			metadataJSON = string(data)
		}
	}

	auditLog := &models.AuditLog{
		UserID:      userID,
		Action:      action,
		Resource:    resource,
		ResourceID:  resourceID,
		Description: description,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Metadata:    metadataJSON,
	}

	if err := s.db.Create(auditLog).Error; err != nil {
		fmt.Printf("Failed to create audit log: %v\n", err)
	}
}

func (s *AuditService) GetAuditLogs(ctx context.Context, page, perPage int, filters map[string]interface{}) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	query := s.db.Model(&models.AuditLog{}).Preload("User")

	// Apply filters
	if userID, ok := filters["user_id"].(uuid.UUID); ok {
		query = query.Where("user_id = ?", userID)
	}

	if action, ok := filters["action"].(string); ok {
		query = query.Where("action = ?", action)
	}

	if resource, ok := filters["resource"].(string); ok {
		query = query.Where("resource = ?", resource)
	}

	if resourceID, ok := filters["resource_id"].(uuid.UUID); ok {
		query = query.Where("resource_id = ?", resourceID)
	}

	if startTime, ok := filters["start_time"].(time.Time); ok {
		query = query.Where("created_at >= ?", startTime)
	}

	if endTime, ok := filters["end_time"].(time.Time); ok {
		query = query.Where("created_at <= ?", endTime)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	offset := (page - 1) * perPage
	if err := query.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get audit logs: %w", err)
	}

	return logs, total, nil
}

func (s *AuditService) GetAuditLog(ctx context.Context, id uuid.UUID) (*models.AuditLog, error) {
	var log models.AuditLog
	if err := s.db.Preload("User").Where("id = ?", id).First(&log).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("audit log not found")
		}
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}

	return &log, nil
}

func (s *AuditService) GetUserActivity(ctx context.Context, userID uuid.UUID, page, perPage int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	query := s.db.Model(&models.AuditLog{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count user activity: %w", err)
	}

	offset := (page - 1) * perPage
	if err := query.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get user activity: %w", err)
	}

	return logs, total, nil
}

func (s *AuditService) GetResourceActivity(ctx context.Context, resource string, resourceID uuid.UUID, page, perPage int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	query := s.db.Model(&models.AuditLog{}).Where("resource = ? AND resource_id = ?", resource, resourceID).Preload("User")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count resource activity: %w", err)
	}

	offset := (page - 1) * perPage
	if err := query.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get resource activity: %w", err)
	}

	return logs, total, nil
}

func (s *AuditService) GetActivitySummary(ctx context.Context, startTime, endTime time.Time) (map[string]interface{}, error) {
	var result []struct {
		Action string `json:"action"`
		Count  int64  `json:"count"`
	}

	query := s.db.Model(&models.AuditLog{}).
		Select("action, COUNT(*) as count").
		Where("created_at >= ? AND created_at <= ?", startTime, endTime).
		Group("action")

	if err := query.Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("failed to get activity summary: %w", err)
	}

	summary := make(map[string]interface{})
	summary["actions"] = result

	// Get total count
	var totalCount int64
	if err := s.db.Model(&models.AuditLog{}).
		Where("created_at >= ? AND created_at <= ?", startTime, endTime).
		Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}
	summary["total_count"] = totalCount

	// Get unique users count
	var uniqueUsers int64
	if err := s.db.Model(&models.AuditLog{}).
		Where("created_at >= ? AND created_at <= ?", startTime, endTime).
		Distinct("user_id").
		Count(&uniqueUsers).Error; err != nil {
		return nil, fmt.Errorf("failed to get unique users count: %w", err)
	}
	summary["unique_users"] = uniqueUsers

	return summary, nil
}

func (s *AuditService) CleanupOldLogs(ctx context.Context, retentionDays int) error {
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	
	result := s.db.Where("created_at < ?", cutoffTime).Delete(&models.AuditLog{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup old audit logs: %w", result.Error)
	}

	fmt.Printf("Cleaned up %d old audit log entries\n", result.RowsAffected)
	return nil
}