package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wg-hubspoke/wg-hubspoke/common/types"
	"github.com/wg-hubspoke/wg-hubspoke/controller/models"
	"github.com/wg-hubspoke/wg-hubspoke/controller/services"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Login godoc
// @Summary User login
// @Description Authenticate user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body services.LoginRequest true "Login credentials"
// @Success 200 {object} types.APIResponse{data=services.LoginResponse}
// @Failure 400 {object} types.APIResponse
// @Failure 401 {object} types.APIResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	resp, err := h.authService.Login(c.Request.Context(), req, clientIP, userAgent)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err == services.ErrUserNotFound || err == services.ErrInvalidPassword {
			statusCode = http.StatusUnauthorized
		}

		c.JSON(statusCode, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    resp,
		Message: "Login successful",
	})
}

// CreateUser godoc
// @Summary Create new user
// @Description Create a new user (admin only)
// @Tags auth
// @Accept json
// @Produce json
// @Param user body services.CreateUserRequest true "User data"
// @Success 201 {object} types.APIResponse{data=models.User}
// @Failure 400 {object} types.APIResponse
// @Failure 403 {object} types.APIResponse
// @Failure 409 {object} types.APIResponse
// @Router /auth/users [post]
func (h *AuthHandler) CreateUser(c *gin.Context) {
	var req services.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Get current user from context
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
			Error:   "Insufficient permissions",
		})
		return
	}

	newUser, err := h.authService.CreateUser(c.Request.Context(), req, &user.ID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err == services.ErrUserExists {
			statusCode = http.StatusConflict
		}

		c.JSON(statusCode, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, types.APIResponse{
		Success: true,
		Data:    newUser,
		Message: "User created successfully",
	})
}

// GetUsers godoc
// @Summary List users
// @Description Get paginated list of users (admin only)
// @Tags auth
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(10)
// @Success 200 {object} types.PaginatedResponse{data=[]models.User}
// @Failure 403 {object} types.APIResponse
// @Router /auth/users [get]
func (h *AuthHandler) GetUsers(c *gin.Context) {
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
			Error:   "Insufficient permissions",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))

	users, total, err := h.authService.GetUsers(c.Request.Context(), page, perPage)
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
			Data:    users,
		},
		Pagination: types.PaginationInfo{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// GetUser godoc
// @Summary Get user by ID
// @Description Get user details by ID (admin only)
// @Tags auth
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} types.APIResponse{data=models.User}
// @Failure 404 {object} types.APIResponse
// @Failure 403 {object} types.APIResponse
// @Router /auth/users/{id} [get]
func (h *AuthHandler) GetUser(c *gin.Context) {
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
			Error:   "Insufficient permissions",
		})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid user ID format",
		})
		return
	}

	targetUser, err := h.authService.GetUser(c.Request.Context(), id)
	if err != nil {
		if err == services.ErrUserNotFound {
			c.JSON(http.StatusNotFound, types.APIResponse{
				Success: false,
				Error:   "User not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    targetUser,
	})
}

// UpdateUser godoc
// @Summary Update user
// @Description Update user details (admin only)
// @Tags auth
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param user body map[string]interface{} true "User updates"
// @Success 200 {object} types.APIResponse{data=models.User}
// @Failure 404 {object} types.APIResponse
// @Failure 403 {object} types.APIResponse
// @Router /auth/users/{id} [put]
func (h *AuthHandler) UpdateUser(c *gin.Context) {
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
			Error:   "Insufficient permissions",
		})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid user ID format",
		})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	updatedUser, err := h.authService.UpdateUser(c.Request.Context(), id, updates, &user.ID)
	if err != nil {
		if err == services.ErrUserNotFound {
			c.JSON(http.StatusNotFound, types.APIResponse{
				Success: false,
				Error:   "User not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    updatedUser,
		Message: "User updated successfully",
	})
}

// DeleteUser godoc
// @Summary Delete user
// @Description Delete user by ID (admin only)
// @Tags auth
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} types.APIResponse
// @Failure 404 {object} types.APIResponse
// @Failure 403 {object} types.APIResponse
// @Router /auth/users/{id} [delete]
func (h *AuthHandler) DeleteUser(c *gin.Context) {
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
			Error:   "Insufficient permissions",
		})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid user ID format",
		})
		return
	}

	err = h.authService.DeleteUser(c.Request.Context(), id, &user.ID)
	if err != nil {
		if err == services.ErrUserNotFound {
			c.JSON(http.StatusNotFound, types.APIResponse{
				Success: false,
				Error:   "User not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Message: "User deleted successfully",
	})
}

// GetCurrentUser godoc
// @Summary Get current user
// @Description Get current authenticated user's profile
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} types.APIResponse{data=models.User}
// @Failure 401 {object} types.APIResponse
// @Router /auth/me [get]
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	currentUser, exists := c.Get("current_user")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	user := currentUser.(*models.User)
	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    user,
	})
}

// ChangePassword godoc
// @Summary Change password
// @Description Change user's password
// @Tags auth
// @Accept json
// @Produce json
// @Param passwords body map[string]string true "Current and new passwords"
// @Success 200 {object} types.APIResponse
// @Failure 400 {object} types.APIResponse
// @Failure 401 {object} types.APIResponse
// @Router /auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	currentUser, exists := c.Get("current_user")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	user := currentUser.(*models.User)

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	err := h.authService.ChangePassword(c.Request.Context(), user.ID, req.CurrentPassword, req.NewPassword, &user.ID)
	if err != nil {
		if err == services.ErrInvalidPassword {
			c.JSON(http.StatusBadRequest, types.APIResponse{
				Success: false,
				Error:   "Current password is incorrect",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Message: "Password changed successfully",
	})
}

// AuthMiddleware - JWT authentication middleware
func (h *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, types.APIResponse{
				Success: false,
				Error:   "Authorization header required",
			})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, types.APIResponse{
				Success: false,
				Error:   "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		claims, err := h.authService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, types.APIResponse{
				Success: false,
				Error:   "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Get user from database
		user, err := h.authService.GetUser(c.Request.Context(), claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, types.APIResponse{
				Success: false,
				Error:   "User not found",
			})
			c.Abort()
			return
		}

		if !user.IsActive {
			c.JSON(http.StatusUnauthorized, types.APIResponse{
				Success: false,
				Error:   "User account is disabled",
			})
			c.Abort()
			return
		}

		c.Set("current_user", user)
		c.Set("user_claims", claims)
		c.Next()
	}
}

// AdminMiddleware - Admin role requirement middleware
func (h *AuthHandler) AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser, exists := c.Get("current_user")
		if !exists {
			c.JSON(http.StatusUnauthorized, types.APIResponse{
				Success: false,
				Error:   "Unauthorized",
			})
			c.Abort()
			return
		}

		user := currentUser.(*models.User)
		if err := h.authService.RequireRole(user.Role, models.UserRoleAdmin); err != nil {
			c.JSON(http.StatusForbidden, types.APIResponse{
				Success: false,
				Error:   "Admin access required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}