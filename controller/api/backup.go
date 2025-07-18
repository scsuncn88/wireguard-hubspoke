package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wg-hubspoke/wg-hubspoke/common/types"
	"github.com/wg-hubspoke/wg-hubspoke/controller/models"
	"github.com/wg-hubspoke/wg-hubspoke/controller/services"
)

type BackupHandler struct {
	backupService *services.BackupService
	authService   *services.AuthService
}

func NewBackupHandler(backupService *services.BackupService, authService *services.AuthService) *BackupHandler {
	return &BackupHandler{
		backupService: backupService,
		authService:   authService,
	}
}

// CreateBackup godoc
// @Summary Create database backup
// @Description Create a new database backup with specified options (admin only)
// @Tags backup
// @Accept json
// @Produce json
// @Param backup body services.BackupOptions true "Backup options"
// @Success 200 {object} types.APIResponse{data=services.BackupInfo}
// @Failure 400 {object} types.APIResponse
// @Failure 403 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Router /backup/create [post]
func (h *BackupHandler) CreateBackup(c *gin.Context) {
	currentUser, exists := c.Get("current_user")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	user := currentUser.(*models.User)
	if err := h.authService.RequireRole(user.Role, models.UserRoleAdmin); err != nil {
		c.JSON(http.StatusForbidden, types.APIResponse{
			Success: false,
			Error:   "Admin access required",
		})
		return
	}

	var options services.BackupOptions
	if err := c.ShouldBindJSON(&options); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Set default values
	if options.BackupType == "" {
		options.BackupType = "full"
	}
	if options.RetentionDays == 0 {
		options.RetentionDays = 30
	}

	backup, err := h.backupService.CreateBackup(c.Request.Context(), options, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    backup,
		Message: "Backup created successfully",
	})
}

// GetBackups godoc
// @Summary Get backup list
// @Description Get paginated list of database backups (admin only)
// @Tags backup
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(10)
// @Success 200 {object} types.PaginatedResponse{data=[]services.BackupInfo}
// @Failure 403 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Router /backup [get]
func (h *BackupHandler) GetBackups(c *gin.Context) {
	currentUser, exists := c.Get("current_user")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	user := currentUser.(*models.User)
	if err := h.authService.RequireRole(user.Role, models.UserRoleAdmin); err != nil {
		c.JSON(http.StatusForbidden, types.APIResponse{
			Success: false,
			Error:   "Admin access required",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))

	backups, total, err := h.backupService.GetBackups(c.Request.Context(), page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	totalPages := int((total + int64(perPage) - 1) / int64(perPage))

	c.JSON(http.StatusOK, types.PaginatedResponse{
		APIResponse: types.APIResponse{
			Success: true,
			Data:    backups,
		},
		Pagination: types.PaginationInfo{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// GetBackup godoc
// @Summary Get backup details
// @Description Get details of a specific backup (admin only)
// @Tags backup
// @Accept json
// @Produce json
// @Param id path string true "Backup ID"
// @Success 200 {object} types.APIResponse{data=services.BackupInfo}
// @Failure 400 {object} types.APIResponse
// @Failure 403 {object} types.APIResponse
// @Failure 404 {object} types.APIResponse
// @Router /backup/{id} [get]
func (h *BackupHandler) GetBackup(c *gin.Context) {
	currentUser, exists := c.Get("current_user")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	user := currentUser.(*models.User)
	if err := h.authService.RequireRole(user.Role, models.UserRoleAdmin); err != nil {
		c.JSON(http.StatusForbidden, types.APIResponse{
			Success: false,
			Error:   "Admin access required",
		})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid backup ID format",
		})
		return
	}

	backup, err := h.backupService.GetBackup(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    backup,
	})
}

// RestoreBackup godoc
// @Summary Restore database backup
// @Description Restore database from a backup (admin only)
// @Tags backup
// @Accept json
// @Produce json
// @Param restore body services.RestoreOptions true "Restore options"
// @Success 200 {object} types.APIResponse{data=services.RestoreResult}
// @Failure 400 {object} types.APIResponse
// @Failure 403 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Router /backup/restore [post]
func (h *BackupHandler) RestoreBackup(c *gin.Context) {
	currentUser, exists := c.Get("current_user")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	user := currentUser.(*models.User)
	if err := h.authService.RequireRole(user.Role, models.UserRoleAdmin); err != nil {
		c.JSON(http.StatusForbidden, types.APIResponse{
			Success: false,
			Error:   "Admin access required",
		})
		return
	}

	var options services.RestoreOptions
	if err := c.ShouldBindJSON(&options); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Set default values
	if options.RestoreType == "" {
		options.RestoreType = "full"
	}

	result, err := h.backupService.RestoreBackup(c.Request.Context(), options, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if result.Success {
		c.JSON(http.StatusOK, types.APIResponse{
			Success: true,
			Data:    result,
			Message: "Backup restored successfully",
		})
	} else {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Data:    result,
			Error:   "Backup restore completed with errors",
		})
	}
}

// DeleteBackup godoc
// @Summary Delete backup
// @Description Delete a backup and its associated file (admin only)
// @Tags backup
// @Accept json
// @Produce json
// @Param id path string true "Backup ID"
// @Success 200 {object} types.APIResponse
// @Failure 400 {object} types.APIResponse
// @Failure 403 {object} types.APIResponse
// @Failure 404 {object} types.APIResponse
// @Router /backup/{id} [delete]
func (h *BackupHandler) DeleteBackup(c *gin.Context) {
	currentUser, exists := c.Get("current_user")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	user := currentUser.(*models.User)
	if err := h.authService.RequireRole(user.Role, models.UserRoleAdmin); err != nil {
		c.JSON(http.StatusForbidden, types.APIResponse{
			Success: false,
			Error:   "Admin access required",
		})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid backup ID format",
		})
		return
	}

	err = h.backupService.DeleteBackup(c.Request.Context(), id, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Message: "Backup deleted successfully",
	})
}

// ScheduleBackup godoc
// @Summary Schedule automatic backup
// @Description Schedule automatic backup with cron expression (admin only)
// @Tags backup
// @Accept json
// @Produce json
// @Param schedule body map[string]interface{} true "Schedule options"
// @Success 200 {object} types.APIResponse
// @Failure 400 {object} types.APIResponse
// @Failure 403 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Router /backup/schedule [post]
func (h *BackupHandler) ScheduleBackup(c *gin.Context) {
	currentUser, exists := c.Get("current_user")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	user := currentUser.(*models.User)
	if err := h.authService.RequireRole(user.Role, models.UserRoleAdmin); err != nil {
		c.JSON(http.StatusForbidden, types.APIResponse{
			Success: false,
			Error:   "Admin access required",
		})
		return
	}

	var request struct {
		Schedule string                    `json:"schedule"`
		Options  services.BackupOptions   `json:"options"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	err := h.backupService.ScheduleBackup(c.Request.Context(), request.Schedule, request.Options, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Message: "Backup scheduled successfully",
	})
}

// GetBackupStats godoc
// @Summary Get backup statistics
// @Description Get backup system statistics (admin only)
// @Tags backup
// @Accept json
// @Produce json
// @Success 200 {object} types.APIResponse{data=map[string]interface{}}
// @Failure 403 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Router /backup/stats [get]
func (h *BackupHandler) GetBackupStats(c *gin.Context) {
	currentUser, exists := c.Get("current_user")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	user := currentUser.(*models.User)
	if err := h.authService.RequireRole(user.Role, models.UserRoleAdmin); err != nil {
		c.JSON(http.StatusForbidden, types.APIResponse{
			Success: false,
			Error:   "Admin access required",
		})
		return
	}

	stats, err := h.backupService.GetBackupStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    stats,
	})
}