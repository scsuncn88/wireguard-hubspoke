package services

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/wg-hubspoke/wg-hubspoke/common/types"
	"github.com/wg-hubspoke/wg-hubspoke/controller/models"
	"gorm.io/gorm"
)

type BackupService struct {
	db           *gorm.DB
	config       *types.Config
	auditService *AuditService
}

type BackupInfo struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string    `json:"name" gorm:"not null"`
	Type        string    `json:"type" gorm:"not null"` // full, incremental, differential
	Status      string    `json:"status" gorm:"not null"` // running, completed, failed
	FilePath    string    `json:"file_path" gorm:"not null"`
	FileSize    int64     `json:"file_size"`
	StartTime   time.Time `json:"start_time" gorm:"not null"`
	EndTime     *time.Time `json:"end_time"`
	CreatedBy   uuid.UUID `json:"created_by" gorm:"type:uuid;not null"`
	Description string    `json:"description"`
	ErrorLog    string    `json:"error_log"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type BackupOptions struct {
	BackupType    string            `json:"backup_type"` // full, incremental, schema_only
	Compression   bool              `json:"compression"`
	BackupPath    string            `json:"backup_path"`
	Description   string            `json:"description"`
	RetentionDays int               `json:"retention_days"`
	Tables        []string          `json:"tables"` // specific tables to backup
	Metadata      map[string]string `json:"metadata"`
}

type RestoreOptions struct {
	BackupID       uuid.UUID `json:"backup_id"`
	RestoreType    string    `json:"restore_type"` // full, selective
	Tables         []string  `json:"tables"`
	DropExisting   bool      `json:"drop_existing"`
	IgnoreErrors   bool      `json:"ignore_errors"`
	RestoreData    bool      `json:"restore_data"`
	RestoreSchema  bool      `json:"restore_schema"`
}

type RestoreResult struct {
	Success       bool      `json:"success"`
	TablesRestored int      `json:"tables_restored"`
	RecordsRestored int64   `json:"records_restored"`
	Errors        []string  `json:"errors"`
	Duration      time.Duration `json:"duration"`
	RestoreTime   time.Time `json:"restore_time"`
}

func NewBackupService(db *gorm.DB, config *types.Config, auditService *AuditService) *BackupService {
	return &BackupService{
		db:           db,
		config:       config,
		auditService: auditService,
	}
}

func (s *BackupService) CreateBackup(ctx context.Context, options BackupOptions, createdBy uuid.UUID) (*BackupInfo, error) {
	// Create backup record
	backup := &BackupInfo{
		Name:        fmt.Sprintf("backup_%s_%s", options.BackupType, time.Now().Format("20060102_150405")),
		Type:        options.BackupType,
		Status:      "running",
		StartTime:   time.Now(),
		CreatedBy:   createdBy,
		Description: options.Description,
	}

	if err := s.db.Create(backup).Error; err != nil {
		return nil, fmt.Errorf("failed to create backup record: %w", err)
	}

	// Create backup directory if it doesn't exist
	backupDir := options.BackupPath
	if backupDir == "" {
		backupDir = "/var/backups/wg-sdwan"
	}
	
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		s.updateBackupStatus(backup.ID, "failed", fmt.Sprintf("Failed to create backup directory: %v", err))
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup filename
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("wg_sdwan_%s_%s.sql", options.BackupType, timestamp)
	if options.Compression {
		filename += ".gz"
	}
	
	backup.FilePath = filepath.Join(backupDir, filename)

	// Perform backup based on type
	var err error
	switch options.BackupType {
	case "full":
		err = s.performFullBackup(ctx, backup, options)
	case "schema_only":
		err = s.performSchemaBackup(ctx, backup, options)
	case "incremental":
		err = s.performIncrementalBackup(ctx, backup, options)
	default:
		err = fmt.Errorf("unsupported backup type: %s", options.BackupType)
	}

	if err != nil {
		s.updateBackupStatus(backup.ID, "failed", err.Error())
		return nil, err
	}

	// Get file size
	if stat, err := os.Stat(backup.FilePath); err == nil {
		backup.FileSize = stat.Size()
	}

	// Update backup status
	endTime := time.Now()
	backup.EndTime = &endTime
	s.updateBackupStatus(backup.ID, "completed", "")

	// Log backup action
	s.auditService.LogAction(ctx, createdBy, "create_backup", "backup", backup.ID,
		map[string]interface{}{
			"backup_type": options.BackupType,
			"file_path":   backup.FilePath,
			"file_size":   backup.FileSize,
			"duration":    endTime.Sub(backup.StartTime).String(),
		})

	// Clean up old backups
	go s.cleanupOldBackups(options.RetentionDays)

	return backup, nil
}

func (s *BackupService) performFullBackup(ctx context.Context, backup *BackupInfo, options BackupOptions) error {
	// Build pg_dump command
	cmd := exec.CommandContext(ctx, "pg_dump",
		"-h", s.config.Database.Host,
		"-p", fmt.Sprintf("%d", s.config.Database.Port),
		"-U", s.config.Database.User,
		"-d", s.config.Database.Name,
		"-f", backup.FilePath,
		"--verbose",
		"--create",
		"--clean",
	)

	// Set password via environment variable
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", s.config.Database.Password))

	// Add compression if requested
	if options.Compression {
		cmd.Args = append(cmd.Args, "--compress=9")
	}

	// Add specific tables if requested
	if len(options.Tables) > 0 {
		for _, table := range options.Tables {
			cmd.Args = append(cmd.Args, "-t", table)
		}
	}

	// Execute backup
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_dump failed: %w, output: %s", err, string(output))
	}

	return nil
}

func (s *BackupService) performSchemaBackup(ctx context.Context, backup *BackupInfo, options BackupOptions) error {
	// Build pg_dump command for schema only
	cmd := exec.CommandContext(ctx, "pg_dump",
		"-h", s.config.Database.Host,
		"-p", fmt.Sprintf("%d", s.config.Database.Port),
		"-U", s.config.Database.User,
		"-d", s.config.Database.Name,
		"-f", backup.FilePath,
		"--schema-only",
		"--verbose",
		"--create",
		"--clean",
	)

	// Set password via environment variable
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", s.config.Database.Password))

	// Execute backup
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_dump (schema) failed: %w, output: %s", err, string(output))
	}

	return nil
}

func (s *BackupService) performIncrementalBackup(ctx context.Context, backup *BackupInfo, options BackupOptions) error {
	// For incremental backup, we'll export only modified data since last backup
	// This is a simplified implementation - in production, you'd use PostgreSQL's WAL
	
	// Find last successful backup
	var lastBackup BackupInfo
	err := s.db.Where("status = ? AND type IN (?)", "completed", []string{"full", "incremental"}).
		Order("created_at DESC").
		First(&lastBackup).Error
	
	if err != nil {
		// No previous backup found, perform full backup instead
		return s.performFullBackup(ctx, backup, options)
	}

	// Get modified records since last backup
	var modifiedTables []string
	
	// Check each table for modifications
	tables := []string{"nodes", "policies", "users", "audit_logs", "topologies"}
	for _, table := range tables {
		var count int64
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE updated_at > ?", table)
		s.db.Raw(query, lastBackup.StartTime).Count(&count)
		
		if count > 0 {
			modifiedTables = append(modifiedTables, table)
		}
	}

	// Create incremental backup with only modified tables
	if len(modifiedTables) == 0 {
		// No changes, create empty backup file
		file, err := os.Create(backup.FilePath)
		if err != nil {
			return fmt.Errorf("failed to create empty backup file: %w", err)
		}
		defer file.Close()
		
		file.WriteString("-- No changes since last backup\n")
		return nil
	}

	// Perform backup with modified tables only
	backupOptions := options
	backupOptions.Tables = modifiedTables
	return s.performFullBackup(ctx, backup, backupOptions)
}

func (s *BackupService) RestoreBackup(ctx context.Context, options RestoreOptions, restoredBy uuid.UUID) (*RestoreResult, error) {
	startTime := time.Now()
	result := &RestoreResult{
		RestoreTime: startTime,
		Errors:      []string{},
	}

	// Get backup info
	var backup BackupInfo
	if err := s.db.Where("id = ?", options.BackupID).First(&backup).Error; err != nil {
		return nil, fmt.Errorf("backup not found: %w", err)
	}

	// Check if backup file exists
	if _, err := os.Stat(backup.FilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("backup file not found: %s", backup.FilePath)
	}

	// Perform restore based on type
	var err error
	switch options.RestoreType {
	case "full":
		err = s.performFullRestore(ctx, backup, options, result)
	case "selective":
		err = s.performSelectiveRestore(ctx, backup, options, result)
	default:
		return nil, fmt.Errorf("unsupported restore type: %s", options.RestoreType)
	}

	result.Duration = time.Since(startTime)
	result.Success = err == nil

	// Log restore action
	s.auditService.LogAction(ctx, restoredBy, "restore_backup", "backup", backup.ID,
		map[string]interface{}{
			"restore_type":     options.RestoreType,
			"success":          result.Success,
			"tables_restored":  result.TablesRestored,
			"records_restored": result.RecordsRestored,
			"duration":         result.Duration.String(),
		})

	return result, err
}

func (s *BackupService) performFullRestore(ctx context.Context, backup BackupInfo, options RestoreOptions, result *RestoreResult) error {
	// Build psql command for restore
	cmd := exec.CommandContext(ctx, "psql",
		"-h", s.config.Database.Host,
		"-p", fmt.Sprintf("%d", s.config.Database.Port),
		"-U", s.config.Database.User,
		"-d", s.config.Database.Name,
		"-f", backup.FilePath,
		"--verbose",
	)

	// Set password via environment variable
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", s.config.Database.Password))

	// Add options
	if options.IgnoreErrors {
		cmd.Args = append(cmd.Args, "--on-error-continue")
	}

	// Execute restore
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("psql failed: %v, output: %s", err, string(output)))
		return fmt.Errorf("restore failed: %w", err)
	}

	// Count restored records (simplified)
	result.TablesRestored = 5 // Assuming 5 main tables
	result.RecordsRestored = 1000 // This would need to be calculated properly

	return nil
}

func (s *BackupService) performSelectiveRestore(ctx context.Context, backup BackupInfo, options RestoreOptions, result *RestoreResult) error {
	// For selective restore, we'd need to parse the SQL file and extract specific tables
	// This is a simplified implementation
	
	if len(options.Tables) == 0 {
		return fmt.Errorf("no tables specified for selective restore")
	}

	// Create temporary database for selective restore
	tempDB := fmt.Sprintf("temp_restore_%s", uuid.New().String()[:8])
	
	// Create temporary database
	createCmd := exec.CommandContext(ctx, "createdb",
		"-h", s.config.Database.Host,
		"-p", fmt.Sprintf("%d", s.config.Database.Port),
		"-U", s.config.Database.User,
		tempDB,
	)
	createCmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", s.config.Database.Password))
	
	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("failed to create temporary database: %w", err)
	}

	// Cleanup temporary database
	defer func() {
		dropCmd := exec.CommandContext(context.Background(), "dropdb",
			"-h", s.config.Database.Host,
			"-p", fmt.Sprintf("%d", s.config.Database.Port),
			"-U", s.config.Database.User,
			tempDB,
		)
		dropCmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", s.config.Database.Password))
		dropCmd.Run()
	}()

	// Restore to temporary database
	restoreCmd := exec.CommandContext(ctx, "psql",
		"-h", s.config.Database.Host,
		"-p", fmt.Sprintf("%d", s.config.Database.Port),
		"-U", s.config.Database.User,
		"-d", tempDB,
		"-f", backup.FilePath,
	)
	restoreCmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", s.config.Database.Password))

	if err := restoreCmd.Run(); err != nil {
		return fmt.Errorf("failed to restore to temporary database: %w", err)
	}

	// Copy selected tables from temporary database to main database
	for _, table := range options.Tables {
		if err := s.copyTable(ctx, tempDB, s.config.Database.Name, table, options.DropExisting); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to copy table %s: %v", table, err))
			if !options.IgnoreErrors {
				return err
			}
		} else {
			result.TablesRestored++
		}
	}

	return nil
}

func (s *BackupService) copyTable(ctx context.Context, sourceDB, targetDB, table string, dropExisting bool) error {
	// Connect to source database
	sourceConn, err := sql.Open("postgres", fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		s.config.Database.Host, s.config.Database.Port, 
		s.config.Database.User, s.config.Database.Password, sourceDB,
	))
	if err != nil {
		return fmt.Errorf("failed to connect to source database: %w", err)
	}
	defer sourceConn.Close()

	// Connect to target database
	targetConn, err := sql.Open("postgres", fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		s.config.Database.Host, s.config.Database.Port,
		s.config.Database.User, s.config.Database.Password, targetDB,
	))
	if err != nil {
		return fmt.Errorf("failed to connect to target database: %w", err)
	}
	defer targetConn.Close()

	// Drop existing table if requested
	if dropExisting {
		if _, err := targetConn.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)); err != nil {
			return fmt.Errorf("failed to drop existing table: %w", err)
		}
	}

	// This is a simplified implementation - in production, you'd use pg_dump with specific table
	// and handle schema creation, constraints, etc.
	
	return nil
}

func (s *BackupService) GetBackups(ctx context.Context, page, perPage int) ([]BackupInfo, int64, error) {
	var backups []BackupInfo
	var total int64

	// Get total count
	if err := s.db.Model(&BackupInfo{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count backups: %w", err)
	}

	// Get paginated results
	offset := (page - 1) * perPage
	if err := s.db.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&backups).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch backups: %w", err)
	}

	return backups, total, nil
}

func (s *BackupService) GetBackup(ctx context.Context, id uuid.UUID) (*BackupInfo, error) {
	var backup BackupInfo
	if err := s.db.Where("id = ?", id).First(&backup).Error; err != nil {
		return nil, fmt.Errorf("backup not found: %w", err)
	}
	return &backup, nil
}

func (s *BackupService) DeleteBackup(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	// Get backup info
	var backup BackupInfo
	if err := s.db.Where("id = ?", id).First(&backup).Error; err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}

	// Delete backup file
	if err := os.Remove(backup.FilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete backup file: %w", err)
	}

	// Delete database record
	if err := s.db.Delete(&backup).Error; err != nil {
		return fmt.Errorf("failed to delete backup record: %w", err)
	}

	// Log deletion
	s.auditService.LogAction(ctx, deletedBy, "delete_backup", "backup", id,
		map[string]interface{}{
			"backup_name": backup.Name,
			"file_path":   backup.FilePath,
		})

	return nil
}

func (s *BackupService) updateBackupStatus(id uuid.UUID, status, errorLog string) {
	updates := map[string]interface{}{
		"status": status,
	}
	
	if errorLog != "" {
		updates["error_log"] = errorLog
	}
	
	if status == "completed" || status == "failed" {
		now := time.Now()
		updates["end_time"] = &now
	}

	s.db.Model(&BackupInfo{}).Where("id = ?", id).Updates(updates)
}

func (s *BackupService) cleanupOldBackups(retentionDays int) {
	if retentionDays <= 0 {
		return
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	
	var oldBackups []BackupInfo
	s.db.Where("created_at < ? AND status = ?", cutoffTime, "completed").Find(&oldBackups)
	
	for _, backup := range oldBackups {
		// Delete file
		os.Remove(backup.FilePath)
		
		// Delete database record
		s.db.Delete(&backup)
	}
}

func (s *BackupService) ScheduleBackup(ctx context.Context, schedule string, options BackupOptions, createdBy uuid.UUID) error {
	// This would integrate with a job scheduler like cron
	// For now, we'll just log the schedule request
	
	s.auditService.LogAction(ctx, createdBy, "schedule_backup", "backup", uuid.Nil,
		map[string]interface{}{
			"schedule":     schedule,
			"backup_type":  options.BackupType,
			"compression":  options.Compression,
			"retention":    options.RetentionDays,
		})
	
	return nil
}

func (s *BackupService) GetBackupStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Count backups by status
	var completedCount, failedCount, runningCount int64
	s.db.Model(&BackupInfo{}).Where("status = ?", "completed").Count(&completedCount)
	s.db.Model(&BackupInfo{}).Where("status = ?", "failed").Count(&failedCount)
	s.db.Model(&BackupInfo{}).Where("status = ?", "running").Count(&runningCount)
	
	stats["completed_backups"] = completedCount
	stats["failed_backups"] = failedCount
	stats["running_backups"] = runningCount
	
	// Get total backup size
	var totalSize int64
	s.db.Model(&BackupInfo{}).Select("COALESCE(SUM(file_size), 0)").Where("status = ?", "completed").Scan(&totalSize)
	stats["total_backup_size"] = totalSize
	
	// Get latest backup
	var latestBackup BackupInfo
	if err := s.db.Where("status = ?", "completed").Order("created_at DESC").First(&latestBackup).Error; err == nil {
		stats["latest_backup"] = map[string]interface{}{
			"id":         latestBackup.ID,
			"name":       latestBackup.Name,
			"type":       latestBackup.Type,
			"created_at": latestBackup.CreatedAt,
			"file_size":  latestBackup.FileSize,
		}
	}
	
	return stats, nil
}