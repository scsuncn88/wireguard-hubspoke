package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wg-hubspoke/wg-hubspoke/common/types"
	"github.com/wg-hubspoke/wg-hubspoke/controller/api"
	"github.com/wg-hubspoke/wg-hubspoke/controller/models"
	"github.com/wg-hubspoke/wg-hubspoke/controller/services"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	version    = "dev"
	buildTime  = "unknown"
	commitHash = "unknown"
)

func main() {
	// Load environment variables
	if err := loadEnv(); err != nil {
		log.Fatalf("Failed to load environment: %v", err)
	}

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := initDatabase(config)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize services
	nodeService := services.NewNodeService(db, config)
	healthService := services.NewHealthService(db, version)
	authService := services.NewAuthService(db, config)
	auditService := services.NewAuditService(db)
	monitoringService := services.NewMonitoringService(db)
	haService := services.NewHAService(db, config)
	configService := services.NewConfigService(db, auditService)
	backupService := services.NewBackupService(db, config, auditService)
	securityService := services.NewSecurityService(db, config, auditService)

	// Initialize handlers
	nodesHandler := api.NewNodesHandler(nodeService)
	healthHandler := api.NewHealthHandler(healthService, version)
	authHandler := api.NewAuthHandler(authService, auditService)
	auditHandler := api.NewAuditHandler(auditService, authService)
	monitoringHandler := api.NewMonitoringHandler(monitoringService)
	haHandler := api.NewHAHandler(haService)
	configHandler := api.NewConfigHandler(configService, authService)
	backupHandler := api.NewBackupHandler(backupService, authService)
	securityHandler := api.NewSecurityHandler(securityService, authService)

	// Setup router
	router := setupRouter(nodesHandler, healthHandler, authHandler, auditHandler, monitoringHandler, haHandler, configHandler, backupHandler, securityHandler, authService, auditService)

	// Start HA service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := haService.Start(ctx); err != nil {
		log.Fatalf("Failed to start HA service: %v", err)
	}

	// Start security cleanup tasks
	go securityService.StartCleanupTasks(ctx)

	// Create server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port),
		Handler:      router,
		ReadTimeout:  config.Server.ReadTimeout,
		WriteTimeout: config.Server.WriteTimeout,
	}

	// Start server
	go func() {
		log.Printf("Starting server on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Stop HA service
	if err := haService.Stop(); err != nil {
		log.Printf("Error stopping HA service: %v", err)
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func loadEnv() error {
	// In a real implementation, this would use python-dotenv or similar
	// For now, we'll assume environment variables are already set
	return nil
}

func loadConfig() (*types.Config, error) {
	// Load configuration from environment variables
	config := &types.Config{
		Server: types.ServerConfig{
			Host:         getEnv("CONTROLLER_HOST", "0.0.0.0"),
			Port:         getEnvInt("CONTROLLER_PORT", 8080),
			ReadTimeout:  time.Duration(getEnvInt("READ_TIMEOUT", 10)) * time.Second,
			WriteTimeout: time.Duration(getEnvInt("WRITE_TIMEOUT", 10)) * time.Second,
		},
		Database: types.DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			Name:     getEnv("DB_NAME", "wireguard_sdwan"),
			User:     getEnv("DB_USER", "wg_admin"),
			Password: getEnv("DB_PASSWORD", "password"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		WG: types.WGConfig{
			Interface:           getEnv("WG_INTERFACE", "wg0"),
			Subnet:              getEnv("WG_SUBNET", "10.100.0.0/16"),
			PortRangeStart:      getEnvInt("WG_PORT_RANGE_START", 51820),
			PortRangeEnd:        getEnvInt("WG_PORT_RANGE_END", 51870),
			PersistentKeepalive: getEnvInt("WG_PERSISTENT_KEEPALIVE", 25),
			MTU:                 getEnvInt("WG_MTU", 1420),
			ConfigPath:          getEnv("WG_CONFIG_PATH", "/etc/wireguard/"),
		},
		Log: types.LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		JWT: types.JWTConfig{
			Secret:    getEnv("JWT_SECRET", "your-secret-key"),
			ExpiresIn: time.Duration(getEnvInt("JWT_EXPIRES_IN", 24)) * time.Hour,
		},
		HA: types.HAConfig{
			Enabled:           getEnvBool("HA_ENABLED", false),
			NodeID:            getEnv("HA_NODE_ID", ""),
			ClusterID:         getEnv("HA_CLUSTER_ID", "default"),
			PeerNodes:         getEnvStringSlice("HA_PEER_NODES", []string{}),
			HeartbeatInterval: time.Duration(getEnvInt("HA_HEARTBEAT_INTERVAL", 30)) * time.Second,
			ElectionTimeout:   time.Duration(getEnvInt("HA_ELECTION_TIMEOUT", 60)) * time.Second,
		},
	}

	return config, nil
}

func initDatabase(config *types.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=UTC",
		config.Database.Host,
		config.Database.User,
		config.Database.Password,
		config.Database.Name,
		config.Database.Port,
		config.Database.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(
		&models.Node{},
		&models.Topology{},
		&models.Policy{},
		&models.User{},
		&models.AuditLog{},
		&services.BackupInfo{},
		&services.SecurityEvent{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

func setupRouter(nodesHandler *api.NodesHandler, healthHandler *api.HealthHandler, authHandler *api.AuthHandler, auditHandler *api.AuditHandler, monitoringHandler *api.MonitoringHandler, haHandler *api.HAHandler, configHandler *api.ConfigHandler, backupHandler *api.BackupHandler, securityHandler *api.SecurityHandler, authService *services.AuthService, auditService *services.AuditService) *gin.Engine {
	router := gin.Default()

	// Add security middleware
	router.Use(securityHandler.SecurityMiddleware())

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Add audit middleware
	router.Use(func(c *gin.Context) {
		// Skip audit for health checks and internal endpoints
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/ready" || c.Request.URL.Path == "/live" {
			c.Next()
			return
		}

		// Log the request
		start := time.Now()
		c.Next()
		
		// Log after processing
		latency := time.Since(start)
		auditService.LogRequest(c.Request.Context(), c.Request.Method, c.Request.URL.Path, c.Writer.Status(), latency, c.ClientIP())
	})

	// Health endpoints
	router.GET("/health", healthHandler.HealthCheck)
	router.GET("/ready", healthHandler.ReadinessCheck)
	router.GET("/live", healthHandler.LivenessCheck)

	// Prometheus metrics endpoint
	router.GET("/metrics", monitoringHandler.GetPrometheusMetrics)

	// Authentication endpoints
	auth := router.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/register", authHandler.Register)
		auth.POST("/logout", authHandler.Logout)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.POST("/change-password", authHandler.ChangePassword)
	}

	// HA endpoints
	ha := router.Group("/ha")
	{
		ha.GET("/status", haHandler.GetClusterStatus)
		ha.GET("/health", haHandler.GetHealthStatus)
		ha.POST("/election", haHandler.HandleVoteRequest)
		ha.POST("/leader", haHandler.HandleLeaderAnnouncement)
		ha.POST("/sync", haHandler.SyncConfiguration)
	}

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Authentication middleware for API routes
		v1.Use(authService.AuthMiddleware())

		// Node management
		nodes := v1.Group("/nodes")
		{
			nodes.POST("", nodesHandler.RegisterNode)
			nodes.GET("", nodesHandler.GetNodes)
			nodes.GET("/:id", nodesHandler.GetNode)
			nodes.PUT("/:id", nodesHandler.UpdateNode)
			nodes.DELETE("/:id", nodesHandler.DeleteNode)
			nodes.GET("/:id/config", nodesHandler.GetNodeConfig)
		}

		// User management
		users := v1.Group("/users")
		{
			users.POST("", authHandler.CreateUser)
			users.GET("", authHandler.GetUsers)
			users.GET("/:id", authHandler.GetUser)
			users.PUT("/:id", authHandler.UpdateUser)
			users.DELETE("/:id", authHandler.DeleteUser)
		}

		// Audit logs
		audit := v1.Group("/audit")
		{
			audit.GET("/logs", auditHandler.GetAuditLogs)
			audit.GET("/logs/:id", auditHandler.GetAuditLog)
			audit.GET("/users/:user_id/activity", auditHandler.GetUserActivity)
			audit.GET("/resources/:resource/:resource_id/activity", auditHandler.GetResourceActivity)
			audit.GET("/summary", auditHandler.GetActivitySummary)
		}

		// Monitoring
		monitoring := v1.Group("/monitoring")
		{
			monitoring.POST("/nodes/:node_id/metrics", monitoringHandler.UpdateNodeMetrics)
			monitoring.GET("/nodes/:node_id/metrics", monitoringHandler.GetNodeMetrics)
			monitoring.GET("/nodes/metrics", monitoringHandler.GetAllNodeMetrics)
			monitoring.GET("/nodes/:node_id/health", monitoringHandler.GetNodeHealth)
			monitoring.GET("/nodes/:node_id/history", monitoringHandler.GetMetricsHistory)
			monitoring.GET("/system/metrics", monitoringHandler.GetSystemMetrics)
			monitoring.GET("/topology/health", monitoringHandler.GetTopologyHealth)
			monitoring.GET("/report", monitoringHandler.GenerateReport)
		}

		// Configuration management
		config := v1.Group("/config")
		{
			config.GET("/export", configHandler.ExportConfiguration)
			config.POST("/import", configHandler.ImportConfiguration)
			config.POST("/validate", configHandler.ValidateConfiguration)
			config.GET("/summary", configHandler.GetConfigurationSummary)
			config.GET("/backup", configHandler.GenerateBackup)
		}

		// Backup management
		backup := v1.Group("/backup")
		{
			backup.POST("/create", backupHandler.CreateBackup)
			backup.GET("", backupHandler.GetBackups)
			backup.GET("/:id", backupHandler.GetBackup)
			backup.POST("/restore", backupHandler.RestoreBackup)
			backup.DELETE("/:id", backupHandler.DeleteBackup)
			backup.POST("/schedule", backupHandler.ScheduleBackup)
			backup.GET("/stats", backupHandler.GetBackupStats)
		}

		// Security management
		security := v1.Group("/security")
		{
			security.GET("/report", securityHandler.GetSecurityReport)
			security.GET("/policies", securityHandler.GetSecurityPolicies)
			security.PUT("/policies", securityHandler.UpdateSecurityPolicies)
			security.POST("/validate-password", securityHandler.ValidatePassword)
			security.POST("/generate-token", securityHandler.GenerateSecureToken)
			security.GET("/events", securityHandler.GetSecurityEvents)
			security.POST("/whitelist", securityHandler.AddAllowedIP)
			security.GET("/blocked-ips", securityHandler.GetBlockedIPs)
		}
	}

	return router
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Split by comma for multiple values
		return strings.Split(value, ",")
	}
	return defaultValue
}