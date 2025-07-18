package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wg-hubspoke/wg-hubspoke/common/types"
	"github.com/wg-hubspoke/wg-hubspoke/controller/models"
	"gorm.io/gorm"
)

type SecurityService struct {
	db                 *gorm.DB
	config             *types.Config
	auditService       *AuditService
	failedAttempts     map[string]*LoginAttempts
	rateLimiter        map[string]*RateLimitInfo
	mutex              sync.RWMutex
	blockedIPs         map[string]time.Time
	allowedCIDRs       []*net.IPNet
	sessionTokens      map[string]*SessionInfo
	securityPolicies   *SecurityPolicies
}

type LoginAttempts struct {
	Count     int
	LastAttempt time.Time
	BlockedUntil time.Time
}

type RateLimitInfo struct {
	Count     int
	ResetTime time.Time
}

type SessionInfo struct {
	UserID    uuid.UUID
	IP        string
	UserAgent string
	CreatedAt time.Time
	LastUsed  time.Time
	ExpiresAt time.Time
}

type SecurityPolicies struct {
	MaxLoginAttempts    int           `json:"max_login_attempts"`
	LoginLockoutTime    time.Duration `json:"login_lockout_time"`
	SessionTimeout      time.Duration `json:"session_timeout"`
	MaxSessions         int           `json:"max_sessions"`
	PasswordMinLength   int           `json:"password_min_length"`
	PasswordComplexity  bool          `json:"password_complexity"`
	RequireMFA          bool          `json:"require_mfa"`
	IPWhitelistOnly     bool          `json:"ip_whitelist_only"`
	RateLimitRequests   int           `json:"rate_limit_requests"`
	RateLimitWindow     time.Duration `json:"rate_limit_window"`
	EnableCSRFProtection bool         `json:"enable_csrf_protection"`
	EnableHTTPS         bool          `json:"enable_https"`
	HSTSMaxAge          int           `json:"hsts_max_age"`
}

type SecurityEvent struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	EventType   string    `json:"event_type" gorm:"not null"`
	Severity    string    `json:"severity" gorm:"not null"`
	IP          string    `json:"ip"`
	UserAgent   string    `json:"user_agent"`
	UserID      *uuid.UUID `json:"user_id" gorm:"type:uuid"`
	Description string    `json:"description"`
	Metadata    string    `json:"metadata"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type SecurityReport struct {
	Period              string                 `json:"period"`
	TotalEvents         int64                  `json:"total_events"`
	CriticalEvents      int64                  `json:"critical_events"`
	FailedLogins        int64                  `json:"failed_logins"`
	BlockedIPs          int64                  `json:"blocked_ips"`
	SuspiciousActivity  int64                  `json:"suspicious_activity"`
	EventsByType        map[string]int64       `json:"events_by_type"`
	TopAttackingIPs     []string               `json:"top_attacking_ips"`
	RecentEvents        []SecurityEvent        `json:"recent_events"`
	Recommendations     []string               `json:"recommendations"`
	GeneratedAt         time.Time              `json:"generated_at"`
}

func NewSecurityService(db *gorm.DB, config *types.Config, auditService *AuditService) *SecurityService {
	return &SecurityService{
		db:             db,
		config:         config,
		auditService:   auditService,
		failedAttempts: make(map[string]*LoginAttempts),
		rateLimiter:    make(map[string]*RateLimitInfo),
		blockedIPs:     make(map[string]time.Time),
		allowedCIDRs:   []*net.IPNet{},
		sessionTokens:  make(map[string]*SessionInfo),
		securityPolicies: &SecurityPolicies{
			MaxLoginAttempts:    5,
			LoginLockoutTime:    15 * time.Minute,
			SessionTimeout:      24 * time.Hour,
			MaxSessions:         5,
			PasswordMinLength:   8,
			PasswordComplexity:  true,
			RequireMFA:          false,
			IPWhitelistOnly:     false,
			RateLimitRequests:   100,
			RateLimitWindow:     time.Minute,
			EnableCSRFProtection: true,
			EnableHTTPS:         true,
			HSTSMaxAge:          31536000, // 1 year
		},
	}
}

func (s *SecurityService) RecordFailedLogin(ctx context.Context, ip, userAgent string, userID *uuid.UUID) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Record failed attempt
	if attempts, exists := s.failedAttempts[ip]; exists {
		attempts.Count++
		attempts.LastAttempt = time.Now()
		
		if attempts.Count >= s.securityPolicies.MaxLoginAttempts {
			attempts.BlockedUntil = time.Now().Add(s.securityPolicies.LoginLockoutTime)
			s.blockedIPs[ip] = attempts.BlockedUntil
		}
	} else {
		s.failedAttempts[ip] = &LoginAttempts{
			Count:       1,
			LastAttempt: time.Now(),
		}
	}

	// Log security event
	s.logSecurityEvent(ctx, "failed_login", "warning", ip, userAgent, userID, 
		fmt.Sprintf("Failed login attempt from %s", ip), map[string]interface{}{
			"attempt_count": s.failedAttempts[ip].Count,
		})
}

func (s *SecurityService) RecordSuccessfulLogin(ctx context.Context, ip, userAgent string, userID uuid.UUID) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Reset failed attempts for this IP
	delete(s.failedAttempts, ip)
	delete(s.blockedIPs, ip)

	// Log security event
	s.logSecurityEvent(ctx, "successful_login", "info", ip, userAgent, &userID,
		fmt.Sprintf("Successful login from %s", ip), nil)
}

func (s *SecurityService) IsIPBlocked(ip string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if blockedUntil, exists := s.blockedIPs[ip]; exists {
		if time.Now().Before(blockedUntil) {
			return true
		}
		// Cleanup expired blocks
		delete(s.blockedIPs, ip)
	}

	return false
}

func (s *SecurityService) IsIPAllowed(ip string) bool {
	if !s.securityPolicies.IPWhitelistOnly {
		return true
	}

	clientIP := net.ParseIP(ip)
	if clientIP == nil {
		return false
	}

	for _, cidr := range s.allowedCIDRs {
		if cidr.Contains(clientIP) {
			return true
		}
	}

	return false
}

func (s *SecurityService) CheckRateLimit(ip string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	
	if rateInfo, exists := s.rateLimiter[ip]; exists {
		if now.After(rateInfo.ResetTime) {
			// Reset rate limit window
			rateInfo.Count = 1
			rateInfo.ResetTime = now.Add(s.securityPolicies.RateLimitWindow)
		} else {
			rateInfo.Count++
			if rateInfo.Count > s.securityPolicies.RateLimitRequests {
				return false
			}
		}
	} else {
		s.rateLimiter[ip] = &RateLimitInfo{
			Count:     1,
			ResetTime: now.Add(s.securityPolicies.RateLimitWindow),
		}
	}

	return true
}

func (s *SecurityService) ValidatePassword(password string) []string {
	var errors []string

	if len(password) < s.securityPolicies.PasswordMinLength {
		errors = append(errors, fmt.Sprintf("Password must be at least %d characters long", s.securityPolicies.PasswordMinLength))
	}

	if s.securityPolicies.PasswordComplexity {
		if !regexp.MustCompile(`[a-z]`).MatchString(password) {
			errors = append(errors, "Password must contain at least one lowercase letter")
		}
		if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
			errors = append(errors, "Password must contain at least one uppercase letter")
		}
		if !regexp.MustCompile(`[0-9]`).MatchString(password) {
			errors = append(errors, "Password must contain at least one number")
		}
		if !regexp.MustCompile(`[^a-zA-Z0-9]`).MatchString(password) {
			errors = append(errors, "Password must contain at least one special character")
		}
	}

	// Check for common weak passwords
	commonPasswords := []string{
		"password", "123456", "password123", "admin", "qwerty",
		"letmein", "welcome", "monkey", "dragon", "master",
	}
	
	for _, common := range commonPasswords {
		if strings.ToLower(password) == common {
			errors = append(errors, "Password is too common")
			break
		}
	}

	return errors
}

func (s *SecurityService) GenerateSecureToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

func (s *SecurityService) GenerateCSRFToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (s *SecurityService) ValidateCSRFToken(token, sessionToken string) bool {
	// Simple CSRF validation - in production, use more sophisticated methods
	if token == "" || sessionToken == "" {
		return false
	}

	// Generate expected token based on session
	h := sha256.New()
	h.Write([]byte(sessionToken))
	expected := hex.EncodeToString(h.Sum(nil))

	return token == expected
}

func (s *SecurityService) CreateSession(userID uuid.UUID, ip, userAgent string) (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check for existing sessions for this user
	userSessions := 0
	for _, session := range s.sessionTokens {
		if session.UserID == userID {
			userSessions++
		}
	}

	if userSessions >= s.securityPolicies.MaxSessions {
		return "", fmt.Errorf("maximum number of sessions reached")
	}

	// Generate session token
	token := s.GenerateSecureToken()
	
	// Create session
	session := &SessionInfo{
		UserID:    userID,
		IP:        ip,
		UserAgent: userAgent,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		ExpiresAt: time.Now().Add(s.securityPolicies.SessionTimeout),
	}

	s.sessionTokens[token] = session
	return token, nil
}

func (s *SecurityService) ValidateSession(token string) (*SessionInfo, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	session, exists := s.sessionTokens[token]
	if !exists {
		return nil, false
	}

	if time.Now().After(session.ExpiresAt) {
		delete(s.sessionTokens, token)
		return nil, false
	}

	// Update last used time
	session.LastUsed = time.Now()
	return session, true
}

func (s *SecurityService) RevokeSession(token string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.sessionTokens, token)
}

func (s *SecurityService) GetSecurityMiddleware() func(c *http.Request) bool {
	return func(r *http.Request) bool {
		ip := s.getClientIP(r)
		
		// Check if IP is blocked
		if s.IsIPBlocked(ip) {
			return false
		}

		// Check if IP is allowed
		if !s.IsIPAllowed(ip) {
			return false
		}

		// Check rate limiting
		if !s.CheckRateLimit(ip) {
			return false
		}

		return true
	}
}

func (s *SecurityService) GetSecurityHeaders() map[string]string {
	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"Content-Security-Policy": "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'",
	}

	if s.securityPolicies.EnableHTTPS {
		headers["Strict-Transport-Security"] = fmt.Sprintf("max-age=%d; includeSubDomains", s.securityPolicies.HSTSMaxAge)
	}

	return headers
}

func (s *SecurityService) ScanForVulnerabilities(ctx context.Context) (*SecurityReport, error) {
	report := &SecurityReport{
		Period:         "24h",
		EventsByType:   make(map[string]int64),
		GeneratedAt:    time.Now(),
		Recommendations: []string{},
	}

	// Get security events from last 24 hours
	var events []SecurityEvent
	since := time.Now().Add(-24 * time.Hour)
	
	if err := s.db.Where("created_at > ?", since).Find(&events).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch security events: %w", err)
	}

	// Analyze events
	report.TotalEvents = int64(len(events))
	
	for _, event := range events {
		report.EventsByType[event.EventType]++
		
		if event.Severity == "critical" {
			report.CriticalEvents++
		}
		
		if event.EventType == "failed_login" {
			report.FailedLogins++
		}
		
		if event.EventType == "suspicious_activity" {
			report.SuspiciousActivity++
		}
	}

	// Count blocked IPs
	s.mutex.RLock()
	report.BlockedIPs = int64(len(s.blockedIPs))
	s.mutex.RUnlock()

	// Get top attacking IPs
	ipCounts := make(map[string]int)
	for _, event := range events {
		if event.EventType == "failed_login" && event.IP != "" {
			ipCounts[event.IP]++
		}
	}

	// Sort and get top 10
	// This is simplified - in production, use proper sorting
	for ip, count := range ipCounts {
		if count > 3 {
			report.TopAttackingIPs = append(report.TopAttackingIPs, ip)
		}
	}

	// Recent events (last 10)
	if len(events) > 10 {
		report.RecentEvents = events[:10]
	} else {
		report.RecentEvents = events
	}

	// Generate recommendations
	if report.FailedLogins > 100 {
		report.Recommendations = append(report.Recommendations, "High number of failed logins detected. Consider implementing additional IP blocking.")
	}
	
	if report.CriticalEvents > 0 {
		report.Recommendations = append(report.Recommendations, "Critical security events detected. Review logs immediately.")
	}
	
	if !s.securityPolicies.EnableHTTPS {
		report.Recommendations = append(report.Recommendations, "HTTPS is not enabled. Enable HTTPS for better security.")
	}

	return report, nil
}

func (s *SecurityService) UpdateSecurityPolicies(ctx context.Context, policies *SecurityPolicies, updatedBy uuid.UUID) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	oldPolicies := *s.securityPolicies
	s.securityPolicies = policies

	// Log policy change
	s.auditService.LogAction(ctx, updatedBy, "update_security_policies", "security_policies", uuid.Nil,
		map[string]interface{}{
			"old_policies": oldPolicies,
			"new_policies": policies,
		})

	return nil
}

func (s *SecurityService) GetSecurityPolicies() *SecurityPolicies {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Return a copy
	policies := *s.securityPolicies
	return &policies
}

func (s *SecurityService) AddAllowedCIDR(cidr string) error {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR: %w", err)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.allowedCIDRs = append(s.allowedCIDRs, ipNet)
	return nil
}

func (s *SecurityService) ConfigureTLS() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		PreferServerCipherSuites: true,
		InsecureSkipVerify:      false,
	}
}

func (s *SecurityService) ValidateCertificate(cert *x509.Certificate) error {
	// Basic certificate validation
	if cert.NotAfter.Before(time.Now()) {
		return fmt.Errorf("certificate has expired")
	}

	if cert.NotBefore.After(time.Now()) {
		return fmt.Errorf("certificate is not yet valid")
	}

	// Check for weak key sizes
	if cert.PublicKey != nil {
		// This is simplified - in production, check specific key types
		if strings.Contains(cert.SignatureAlgorithm.String(), "SHA1") {
			return fmt.Errorf("certificate uses weak SHA1 signature")
		}
	}

	return nil
}

func (s *SecurityService) logSecurityEvent(ctx context.Context, eventType, severity, ip, userAgent string, userID *uuid.UUID, description string, metadata map[string]interface{}) {
	event := SecurityEvent{
		EventType:   eventType,
		Severity:    severity,
		IP:          ip,
		UserAgent:   userAgent,
		UserID:      userID,
		Description: description,
	}

	// Serialize metadata
	if metadata != nil {
		// This is simplified - in production, use proper JSON serialization
		event.Metadata = fmt.Sprintf("%v", metadata)
	}

	s.db.Create(&event)
}

func (s *SecurityService) getClientIP(r *http.Request) string {
	// Check for forwarded headers
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to remote address
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

func (s *SecurityService) CleanupExpiredSessions() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	for token, session := range s.sessionTokens {
		if now.After(session.ExpiresAt) {
			delete(s.sessionTokens, token)
		}
	}
}

func (s *SecurityService) CleanupExpiredBlocks() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	for ip, blockedUntil := range s.blockedIPs {
		if now.After(blockedUntil) {
			delete(s.blockedIPs, ip)
			delete(s.failedAttempts, ip)
		}
	}
}

func (s *SecurityService) StartCleanupTasks(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.CleanupExpiredSessions()
			s.CleanupExpiredBlocks()
		}
	}
}