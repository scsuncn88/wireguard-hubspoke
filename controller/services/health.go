package services

import (
	"context"
	"time"

	"github.com/wg-hubspoke/wg-hubspoke/common/types"
	"gorm.io/gorm"
)

type HealthService struct {
	db      *gorm.DB
	version string
}

func NewHealthService(db *gorm.DB, version string) *HealthService {
	return &HealthService{
		db:      db,
		version: version,
	}
}

func (s *HealthService) GetHealthStatus(ctx context.Context) *types.HealthStatus {
	status := &types.HealthStatus{
		Version:   s.version,
		Timestamp: time.Now(),
		Services:  make(map[string]string),
	}

	// Check database
	if err := s.checkDatabase(ctx); err != nil {
		status.Status = "unhealthy"
		status.Services["database"] = "unhealthy: " + err.Error()
	} else {
		status.Services["database"] = "healthy"
	}

	// Overall status
	if status.Status == "" {
		status.Status = "healthy"
	}

	return status
}

func (s *HealthService) IsReady(ctx context.Context) bool {
	return s.checkDatabase(ctx) == nil
}

func (s *HealthService) checkDatabase(ctx context.Context) error {
	// Simple database connection check
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}

	return sqlDB.PingContext(ctx)
}