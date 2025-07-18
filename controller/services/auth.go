package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/wg-hubspoke/wg-hubspoke/common/types"
	"github.com/wg-hubspoke/wg-hubspoke/controller/models"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrUserExists       = errors.New("user already exists")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrInvalidToken     = errors.New("invalid token")
	ErrTokenExpired     = errors.New("token expired")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrInsufficientRole = errors.New("insufficient role")
)

type AuthService struct {
	db        *gorm.DB
	config    *types.Config
	auditSvc  *AuditService
}

type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Role     string    `json:"role"`
	jwt.RegisteredClaims
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token     string          `json:"token"`
	ExpiresAt time.Time       `json:"expires_at"`
	User      models.User     `json:"user"`
}

type CreateUserRequest struct {
	Username string           `json:"username" binding:"required"`
	Email    string           `json:"email" binding:"required,email"`
	Password string           `json:"password" binding:"required,min=8"`
	Role     models.UserRole  `json:"role"`
}

func NewAuthService(db *gorm.DB, config *types.Config, auditSvc *AuditService) *AuthService {
	return &AuthService{
		db:       db,
		config:   config,
		auditSvc: auditSvc,
	}
}

func (s *AuthService) Login(ctx context.Context, req LoginRequest, clientIP, userAgent string) (*LoginResponse, error) {
	var user models.User
	if err := s.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if !user.IsActive {
		return nil, ErrUnauthorized
	}

	if !user.CheckPassword(req.Password) {
		// Log failed login attempt
		s.auditSvc.LogAction(ctx, &user.ID, models.AuditActionLogin, "user", &user.ID, 
			fmt.Sprintf("Failed login attempt for user %s", user.Username), clientIP, userAgent)
		return nil, ErrInvalidPassword
	}

	// Generate JWT token
	token, expiresAt, err := s.generateToken(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Update last login time
	now := time.Now()
	user.LastLogin = &now
	if err := s.db.Save(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to update last login: %w", err)
	}

	// Log successful login
	s.auditSvc.LogAction(ctx, &user.ID, models.AuditActionLogin, "user", &user.ID, 
		fmt.Sprintf("User %s logged in successfully", user.Username), clientIP, userAgent)

	return &LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	}, nil
}

func (s *AuthService) CreateUser(ctx context.Context, req CreateUserRequest, createdBy *uuid.UUID) (*models.User, error) {
	// Check if user already exists
	var existingUser models.User
	if err := s.db.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error; err == nil {
		return nil, ErrUserExists
	}

	// Create new user
	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Role:     req.Role,
		IsActive: true,
	}

	if err := user.SetPassword(req.Password); err != nil {
		return nil, fmt.Errorf("failed to set password: %w", err)
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Log user creation
	s.auditSvc.LogAction(ctx, createdBy, models.AuditActionCreate, "user", &user.ID, 
		fmt.Sprintf("User %s created", user.Username), "", "")

	return user, nil
}

func (s *AuthService) GetUsers(ctx context.Context, page, perPage int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	query := s.db.Model(&models.User{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	offset := (page - 1) * perPage
	if err := query.Offset(offset).Limit(perPage).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get users: %w", err)
	}

	return users, total, nil
}

func (s *AuthService) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := s.db.Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (s *AuthService) UpdateUser(ctx context.Context, id uuid.UUID, updates map[string]interface{}, updatedBy *uuid.UUID) (*models.User, error) {
	var user models.User
	if err := s.db.Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if err := s.db.Model(&user).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Log user update
	s.auditSvc.LogAction(ctx, updatedBy, models.AuditActionUpdate, "user", &user.ID, 
		fmt.Sprintf("User %s updated", user.Username), "", "")

	return &user, nil
}

func (s *AuthService) DeleteUser(ctx context.Context, id uuid.UUID, deletedBy *uuid.UUID) error {
	var user models.User
	if err := s.db.Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	if err := s.db.Delete(&user).Error; err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Log user deletion
	s.auditSvc.LogAction(ctx, deletedBy, models.AuditActionDelete, "user", &user.ID, 
		fmt.Sprintf("User %s deleted", user.Username), "", "")

	return nil
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.Auth.JWTSecret), nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func (s *AuthService) RequireRole(userRole models.UserRole, requiredRole models.UserRole) error {
	if userRole == models.UserRoleAdmin {
		return nil // Admin can access everything
	}

	if requiredRole == models.UserRoleAdmin && userRole != models.UserRoleAdmin {
		return ErrInsufficientRole
	}

	return nil
}

func (s *AuthService) generateToken(user *models.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(s.config.Auth.JWTExpiration)

	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     string(user.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "wg-sdwan-controller",
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.Auth.JWTSecret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string, changedBy *uuid.UUID) error {
	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	if !user.CheckPassword(currentPassword) {
		return ErrInvalidPassword
	}

	if err := user.SetPassword(newPassword); err != nil {
		return fmt.Errorf("failed to set new password: %w", err)
	}

	if err := s.db.Save(&user).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Log password change
	s.auditSvc.LogAction(ctx, changedBy, models.AuditActionUpdate, "user", &user.ID, 
		fmt.Sprintf("Password changed for user %s", user.Username), "", "")

	return nil
}

func (s *AuthService) InitializeDefaultAdmin() error {
	var count int64
	if err := s.db.Model(&models.User{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	if count == 0 {
		admin := &models.User{
			Username: "admin",
			Email:    "admin@example.com",
			Role:     models.UserRoleAdmin,
			IsActive: true,
		}

		if err := admin.SetPassword("admin123"); err != nil {
			return fmt.Errorf("failed to set admin password: %w", err)
		}

		if err := s.db.Create(admin).Error; err != nil {
			return fmt.Errorf("failed to create admin user: %w", err)
		}
	}

	return nil
}