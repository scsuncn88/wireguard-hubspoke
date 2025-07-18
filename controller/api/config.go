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

type ConfigHandler struct {
	configService *services.ConfigService
	authService   *services.AuthService
}

func NewConfigHandler(configService *services.ConfigService, authService *services.AuthService) *ConfigHandler {
	return &ConfigHandler{
		configService: configService,
		authService:   authService,
	}
}

// ExportConfiguration godoc
// @Summary Export system configuration
// @Description Export complete system configuration including nodes, policies, and users (admin only)
// @Tags config
// @Accept json
// @Produce json
// @Param format query string false "Export format (json/yaml)" default(json)
// @Success 200 {object} types.APIResponse{data=string} "Base64 encoded configuration data"
// @Failure 403 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Router /config/export [get]
func (h *ConfigHandler) ExportConfiguration(c *gin.Context) {
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

	format := c.DefaultQuery("format", "json")
	if format != "json" && format != "yaml" {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid format. Supported formats: json, yaml",
		})
		return
	}

	data, err := h.configService.ExportConfiguration(c.Request.Context(), user.ID, format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Set appropriate headers for download
	filename := "wg-sdwan-config." + format
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/octet-stream")
	c.Data(http.StatusOK, "application/octet-stream", data)
}

// ImportConfiguration godoc
// @Summary Import system configuration
// @Description Import system configuration from uploaded file (admin only)
// @Tags config
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Configuration file (JSON or YAML)"
// @Param format formData string false "File format (json/yaml)" default(json)
// @Param overwrite_existing formData boolean false "Overwrite existing records" default(false)
// @Param skip_users formData boolean false "Skip importing users" default(false)
// @Param skip_nodes formData boolean false "Skip importing nodes" default(false)
// @Param skip_policies formData boolean false "Skip importing policies" default(false)
// @Success 200 {object} types.APIResponse{data=services.ImportResult}
// @Failure 400 {object} types.APIResponse
// @Failure 403 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Router /config/import [post]
func (h *ConfigHandler) ImportConfiguration(c *gin.Context) {
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

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "No file uploaded",
		})
		return
	}

	// Open and read file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Failed to open uploaded file",
		})
		return
	}
	defer src.Close()

	// Read file content
	data := make([]byte, file.Size)
	if _, err := src.Read(data); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Failed to read file content",
		})
		return
	}

	// Parse options
	format := c.DefaultPostForm("format", "json")
	overwriteExisting, _ := strconv.ParseBool(c.DefaultPostForm("overwrite_existing", "false"))
	skipUsers, _ := strconv.ParseBool(c.DefaultPostForm("skip_users", "false"))
	skipNodes, _ := strconv.ParseBool(c.DefaultPostForm("skip_nodes", "false"))
	skipPolicies, _ := strconv.ParseBool(c.DefaultPostForm("skip_policies", "false"))

	options := services.ImportOptions{
		OverwriteExisting: overwriteExisting,
		SkipUsers:         skipUsers,
		SkipNodes:         skipNodes,
		SkipPolicies:      skipPolicies,
		ImportedBy:        user.ID,
	}

	// Import configuration
	result, err := h.configService.ImportConfiguration(c.Request.Context(), data, format, options)
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
			Message: "Configuration imported successfully",
		})
	} else {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Data:    result,
			Error:   "Configuration import completed with errors",
		})
	}
}

// ValidateConfiguration godoc
// @Summary Validate configuration file
// @Description Validate configuration file before import (admin only)
// @Tags config
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Configuration file (JSON or YAML)"
// @Param format formData string false "File format (json/yaml)" default(json)
// @Success 200 {object} types.APIResponse{data=[]string} "Validation warnings"
// @Failure 400 {object} types.APIResponse
// @Failure 403 {object} types.APIResponse
// @Router /config/validate [post]
func (h *ConfigHandler) ValidateConfiguration(c *gin.Context) {
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

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "No file uploaded",
		})
		return
	}

	// Open and read file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Failed to open uploaded file",
		})
		return
	}
	defer src.Close()

	// Read file content
	data := make([]byte, file.Size)
	if _, err := src.Read(data); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Failed to read file content",
		})
		return
	}

	format := c.DefaultPostForm("format", "json")
	
	warnings, err := h.configService.ValidateConfiguration(c.Request.Context(), data, format)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    warnings,
		Message: "Configuration validated successfully",
	})
}

// GetConfigurationSummary godoc
// @Summary Get configuration summary
// @Description Get summary of current system configuration (admin only)
// @Tags config
// @Accept json
// @Produce json
// @Success 200 {object} types.APIResponse{data=map[string]interface{}}
// @Failure 403 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Router /config/summary [get]
func (h *ConfigHandler) GetConfigurationSummary(c *gin.Context) {
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

	summary, err := h.configService.GetConfigurationSummary(c.Request.Context())
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

// GenerateBackup godoc
// @Summary Generate system backup
// @Description Generate a complete system backup for disaster recovery (admin only)
// @Tags config
// @Accept json
// @Produce json
// @Param format query string false "Backup format (json/yaml)" default(json)
// @Success 200 {object} types.APIResponse{data=string} "Base64 encoded backup data"
// @Failure 403 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Router /config/backup [get]
func (h *ConfigHandler) GenerateBackup(c *gin.Context) {
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

	format := c.DefaultQuery("format", "json")
	if format != "json" && format != "yaml" {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid format. Supported formats: json, yaml",
		})
		return
	}

	data, err := h.configService.ExportConfiguration(c.Request.Context(), user.ID, format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Set appropriate headers for download
	filename := "wg-sdwan-backup-" + uuid.New().String()[:8] + "." + format
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/octet-stream")
	c.Data(http.StatusOK, "application/octet-stream", data)
}