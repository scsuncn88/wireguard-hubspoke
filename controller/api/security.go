package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wg-hubspoke/wg-hubspoke/common/types"
	"github.com/wg-hubspoke/wg-hubspoke/controller/models"
	"github.com/wg-hubspoke/wg-hubspoke/controller/services"
)

type SecurityHandler struct {
	securityService *services.SecurityService
	authService     *services.AuthService
}

func NewSecurityHandler(securityService *services.SecurityService, authService *services.AuthService) *SecurityHandler {
	return &SecurityHandler{
		securityService: securityService,
		authService:     authService,
	}
}

// GetSecurityReport godoc
// @Summary Get security report
// @Description Get comprehensive security report and vulnerability scan (admin only)
// @Tags security
// @Accept json
// @Produce json
// @Success 200 {object} types.APIResponse{data=services.SecurityReport}
// @Failure 403 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Router /security/report [get]
func (h *SecurityHandler) GetSecurityReport(c *gin.Context) {
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

	report, err := h.securityService.ScanForVulnerabilities(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    report,
	})
}

// GetSecurityPolicies godoc
// @Summary Get security policies
// @Description Get current security policies configuration (admin only)
// @Tags security
// @Accept json
// @Produce json
// @Success 200 {object} types.APIResponse{data=services.SecurityPolicies}
// @Failure 403 {object} types.APIResponse
// @Router /security/policies [get]
func (h *SecurityHandler) GetSecurityPolicies(c *gin.Context) {
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

	policies := h.securityService.GetSecurityPolicies()
	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    policies,
	})
}

// UpdateSecurityPolicies godoc
// @Summary Update security policies
// @Description Update security policies configuration (admin only)
// @Tags security
// @Accept json
// @Produce json
// @Param policies body services.SecurityPolicies true "Security policies"
// @Success 200 {object} types.APIResponse
// @Failure 400 {object} types.APIResponse
// @Failure 403 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Router /security/policies [put]
func (h *SecurityHandler) UpdateSecurityPolicies(c *gin.Context) {
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

	var policies services.SecurityPolicies
	if err := c.ShouldBindJSON(&policies); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	err := h.securityService.UpdateSecurityPolicies(c.Request.Context(), &policies, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Message: "Security policies updated successfully",
	})
}

// ValidatePassword godoc
// @Summary Validate password strength
// @Description Validate password against security policies
// @Tags security
// @Accept json
// @Produce json
// @Param password body map[string]string true "Password to validate"
// @Success 200 {object} types.APIResponse{data=[]string} "Validation errors"
// @Failure 400 {object} types.APIResponse
// @Router /security/validate-password [post]
func (h *SecurityHandler) ValidatePassword(c *gin.Context) {
	var request struct {
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	errors := h.securityService.ValidatePassword(request.Password)
	
	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    errors,
		Message: "Password validation completed",
	})
}

// GenerateSecureToken godoc
// @Summary Generate secure token
// @Description Generate a cryptographically secure token (admin only)
// @Tags security
// @Accept json
// @Produce json
// @Success 200 {object} types.APIResponse{data=string} "Generated token"
// @Failure 403 {object} types.APIResponse
// @Router /security/generate-token [post]
func (h *SecurityHandler) GenerateSecureToken(c *gin.Context) {
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

	token := h.securityService.GenerateSecureToken()
	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    token,
		Message: "Secure token generated",
	})
}

// GetSecurityEvents godoc
// @Summary Get security events
// @Description Get paginated security events (admin only)
// @Tags security
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(10)
// @Param event_type query string false "Filter by event type"
// @Param severity query string false "Filter by severity"
// @Success 200 {object} types.PaginatedResponse{data=[]services.SecurityEvent}
// @Failure 403 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Router /security/events [get]
func (h *SecurityHandler) GetSecurityEvents(c *gin.Context) {
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
	eventType := c.Query("event_type")
	severity := c.Query("severity")

	// This would need to be implemented in SecurityService
	// For now, return a placeholder response
	c.JSON(http.StatusOK, types.PaginatedResponse{
		APIResponse: types.APIResponse{
			Success: true,
			Data:    []services.SecurityEvent{},
		},
		Pagination: types.PaginationInfo{
			Page:       page,
			PerPage:    perPage,
			Total:      0,
			TotalPages: 0,
		},
	})
}

// AddAllowedIP godoc
// @Summary Add allowed IP/CIDR
// @Description Add IP address or CIDR to whitelist (admin only)
// @Tags security
// @Accept json
// @Produce json
// @Param ip body map[string]string true "IP or CIDR to allow"
// @Success 200 {object} types.APIResponse
// @Failure 400 {object} types.APIResponse
// @Failure 403 {object} types.APIResponse
// @Router /security/whitelist [post]
func (h *SecurityHandler) AddAllowedIP(c *gin.Context) {
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
		CIDR string `json:"cidr"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	err := h.securityService.AddAllowedCIDR(request.CIDR)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Message: "IP/CIDR added to whitelist successfully",
	})
}

// GetBlockedIPs godoc
// @Summary Get blocked IPs
// @Description Get list of currently blocked IP addresses (admin only)
// @Tags security
// @Accept json
// @Produce json
// @Success 200 {object} types.APIResponse{data=map[string]interface{}}
// @Failure 403 {object} types.APIResponse
// @Router /security/blocked-ips [get]
func (h *SecurityHandler) GetBlockedIPs(c *gin.Context) {
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

	// This would need to be implemented in SecurityService
	// For now, return a placeholder response
	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"blocked_ips": []string{},
			"total":       0,
		},
	})
}

// SecurityMiddleware provides security middleware for Gin
func (h *SecurityHandler) SecurityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set security headers
		headers := h.securityService.GetSecurityHeaders()
		for key, value := range headers {
			c.Header(key, value)
		}

		// Check if IP is blocked
		clientIP := c.ClientIP()
		if h.securityService.IsIPBlocked(clientIP) {
			c.JSON(http.StatusTooManyRequests, types.APIResponse{
				Success: false,
				Error:   "IP temporarily blocked due to suspicious activity",
			})
			c.Abort()
			return
		}

		// Check if IP is allowed
		if !h.securityService.IsIPAllowed(clientIP) {
			c.JSON(http.StatusForbidden, types.APIResponse{
				Success: false,
				Error:   "IP not in whitelist",
			})
			c.Abort()
			return
		}

		// Check rate limiting
		if !h.securityService.CheckRateLimit(clientIP) {
			c.JSON(http.StatusTooManyRequests, types.APIResponse{
				Success: false,
				Error:   "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// CSRFMiddleware provides CSRF protection
func (h *SecurityHandler) CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip CSRF for GET requests
		if c.Request.Method == "GET" {
			c.Next()
			return
		}

		// Skip CSRF for certain endpoints
		if c.Request.URL.Path == "/auth/login" {
			c.Next()
			return
		}

		// Check CSRF token
		token := c.GetHeader("X-CSRF-Token")
		sessionToken := c.GetHeader("Authorization")

		if !h.securityService.ValidateCSRFToken(token, sessionToken) {
			c.JSON(http.StatusForbidden, types.APIResponse{
				Success: false,
				Error:   "Invalid CSRF token",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}