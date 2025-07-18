package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wg-hubspoke/wg-hubspoke/common/types"
	"github.com/wg-hubspoke/wg-hubspoke/controller/models"
	"github.com/wg-hubspoke/wg-hubspoke/controller/services"
)

type AuditHandler struct {
	auditService *services.AuditService
	authService  *services.AuthService
}

func NewAuditHandler(auditService *services.AuditService, authService *services.AuthService) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
		authService:  authService,
	}
}

// GetAuditLogs godoc
// @Summary Get audit logs
// @Description Get paginated audit logs with optional filters (admin only)
// @Tags audit
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(10)
// @Param user_id query string false "Filter by user ID"
// @Param action query string false "Filter by action"
// @Param resource query string false "Filter by resource"
// @Param resource_id query string false "Filter by resource ID"
// @Param start_time query string false "Start time (RFC3339)"
// @Param end_time query string false "End time (RFC3339)"
// @Success 200 {object} types.PaginatedResponse{data=[]models.AuditLog}
// @Failure 403 {object} types.APIResponse
// @Router /audit/logs [get]
func (h *AuditHandler) GetAuditLogs(c *gin.Context) {
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

	filters := make(map[string]interface{})

	if userID := c.Query("user_id"); userID != "" {
		if id, err := uuid.Parse(userID); err == nil {
			filters["user_id"] = id
		}
	}

	if action := c.Query("action"); action != "" {
		filters["action"] = action
	}

	if resource := c.Query("resource"); resource != "" {
		filters["resource"] = resource
	}

	if resourceID := c.Query("resource_id"); resourceID != "" {
		if id, err := uuid.Parse(resourceID); err == nil {
			filters["resource_id"] = id
		}
	}

	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filters["start_time"] = t
		}
	}

	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filters["end_time"] = t
		}
	}

	logs, total, err := h.auditService.GetAuditLogs(c.Request.Context(), page, perPage, filters)
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
			Data:    logs,
		},
		Pagination: types.PaginationInfo{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// GetAuditLog godoc
// @Summary Get audit log by ID
// @Description Get specific audit log by ID (admin only)
// @Tags audit
// @Accept json
// @Produce json
// @Param id path string true "Audit log ID"
// @Success 200 {object} types.APIResponse{data=models.AuditLog}
// @Failure 404 {object} types.APIResponse
// @Failure 403 {object} types.APIResponse
// @Router /audit/logs/{id} [get]
func (h *AuditHandler) GetAuditLog(c *gin.Context) {
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
			Error:   "Invalid audit log ID format",
		})
		return
	}

	log, err := h.auditService.GetAuditLog(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    log,
	})
}

// GetUserActivity godoc
// @Summary Get user activity
// @Description Get audit logs for a specific user (admin only)
// @Tags audit
// @Accept json
// @Produce json
// @Param user_id path string true "User ID"
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(10)
// @Success 200 {object} types.PaginatedResponse{data=[]models.AuditLog}
// @Failure 403 {object} types.APIResponse
// @Router /audit/users/{user_id}/activity [get]
func (h *AuditHandler) GetUserActivity(c *gin.Context) {
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

	userIDStr := c.Param("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid user ID format",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))

	logs, total, err := h.auditService.GetUserActivity(c.Request.Context(), userID, page, perPage)
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
			Data:    logs,
		},
		Pagination: types.PaginationInfo{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// GetResourceActivity godoc
// @Summary Get resource activity
// @Description Get audit logs for a specific resource (admin only)
// @Tags audit
// @Accept json
// @Produce json
// @Param resource path string true "Resource type"
// @Param resource_id path string true "Resource ID"
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(10)
// @Success 200 {object} types.PaginatedResponse{data=[]models.AuditLog}
// @Failure 403 {object} types.APIResponse
// @Router /audit/resources/{resource}/{resource_id}/activity [get]
func (h *AuditHandler) GetResourceActivity(c *gin.Context) {
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

	resource := c.Param("resource")
	resourceIDStr := c.Param("resource_id")
	resourceID, err := uuid.Parse(resourceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid resource ID format",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))

	logs, total, err := h.auditService.GetResourceActivity(c.Request.Context(), resource, resourceID, page, perPage)
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
			Data:    logs,
		},
		Pagination: types.PaginationInfo{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// GetActivitySummary godoc
// @Summary Get activity summary
// @Description Get activity summary for a time period (admin only)
// @Tags audit
// @Accept json
// @Produce json
// @Param start_time query string true "Start time (RFC3339)"
// @Param end_time query string true "End time (RFC3339)"
// @Success 200 {object} types.APIResponse{data=map[string]interface{}}
// @Failure 403 {object} types.APIResponse
// @Router /audit/summary [get]
func (h *AuditHandler) GetActivitySummary(c *gin.Context) {
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

	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	if startTimeStr == "" || endTimeStr == "" {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "start_time and end_time are required",
		})
		return
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid start_time format",
		})
		return
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid end_time format",
		})
		return
	}

	summary, err := h.auditService.GetActivitySummary(c.Request.Context(), startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    summary,
	})
}