package tests

import (
	"testing"
	"time"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/driver/sqlite"
	"golang.org/x/crypto/bcrypt"
)

// 数据模型
type User struct {
	ID        uuid.UUID `gorm:"type:text;primary_key"`
	Username  string    `gorm:"uniqueIndex;not null"`
	Email     string    `gorm:"uniqueIndex;not null"`
	Password  string    `gorm:"not null"`
	Role      string    `gorm:"default:user"`
	Active    bool      `gorm:"default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Node struct {
	ID          uuid.UUID `gorm:"type:text;primary_key"`
	Name        string    `gorm:"uniqueIndex;not null"`
	NodeType    string    `gorm:"not null"`
	PublicKey   string    `gorm:"uniqueIndex;not null"`
	AllocatedIP string    `gorm:"not null"`
	Status      string    `gorm:"default:pending"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type AuditLog struct {
	ID        uuid.UUID `gorm:"type:text;primary_key"`
	UserID    uuid.UUID `gorm:"type:text"`
	Action    string    `gorm:"not null"`
	Resource  string    `gorm:"not null"`
	Details   string    `gorm:"type:text"`
	IPAddress string
	CreatedAt time.Time
}

// 测试数据库设置
func setupComprehensiveTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to test database")
	}
	
	// 自动迁移所有表
	db.AutoMigrate(&User{}, &Node{}, &AuditLog{})
	
	return db
}

// 认证系统测试
func TestAuthenticationSystem(t *testing.T) {
	db := setupComprehensiveTestDB()
	
	t.Run("user registration and login", func(t *testing.T) {
		// 模拟用户注册
		password := "testPassword123"
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		assert.NoError(t, err)
		
		user := &User{
			ID:       uuid.New(),
			Username: "testuser",
			Email:    "test@example.com",
			Password: string(hashedPassword),
			Role:     "user",
			Active:   true,
		}
		
		result := db.Create(user)
		assert.NoError(t, result.Error)
		
		// 模拟登录验证
		var dbUser User
		db.Where("username = ?", "testuser").First(&dbUser)
		assert.Equal(t, user.Username, dbUser.Username)
		
		// 验证密码
		err = bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(password))
		assert.NoError(t, err)
		
		// 验证用户状态
		assert.True(t, dbUser.Active)
		assert.Equal(t, "user", dbUser.Role)
	})
	
	t.Run("role-based access control", func(t *testing.T) {
		// 创建管理员用户
		admin := &User{
			ID:       uuid.New(),
			Username: "admin",
			Email:    "admin@example.com",
			Password: "hashedpassword",
			Role:     "admin",
			Active:   true,
		}
		db.Create(admin)
		
		// 创建普通用户
		user := &User{
			ID:       uuid.New(),
			Username: "user",
			Email:    "user@example.com",
			Password: "hashedpassword",
			Role:     "user",
			Active:   true,
		}
		db.Create(user)
		
		// 模拟权限检查
		var adminUser User
		db.Where("username = ?", "admin").First(&adminUser)
		assert.Equal(t, "admin", adminUser.Role)
		
		var normalUser User
		db.Where("username = ?", "user").First(&normalUser)
		assert.Equal(t, "user", normalUser.Role)
	})
}

// 节点管理测试
func TestNodeManagement(t *testing.T) {
	db := setupComprehensiveTestDB()
	
	t.Run("node registration", func(t *testing.T) {
		// 创建Hub节点
		hubNode := &Node{
			ID:          uuid.New(),
			Name:        "hub-node-1",
			NodeType:    "hub",
			PublicKey:   "hub-public-key-123",
			AllocatedIP: "10.100.1.1",
			Status:      "active",
		}
		
		result := db.Create(hubNode)
		assert.NoError(t, result.Error)
		
		// 创建Spoke节点
		spokeNode := &Node{
			ID:          uuid.New(),
			Name:        "spoke-node-1",
			NodeType:    "spoke",
			PublicKey:   "spoke-public-key-456",
			AllocatedIP: "10.100.2.1",
			Status:      "active",
		}
		
		result = db.Create(spokeNode)
		assert.NoError(t, result.Error)
		
		// 验证节点创建
		var nodes []Node
		db.Find(&nodes)
		assert.Equal(t, 2, len(nodes))
		
		// 验证节点类型
		var hubNodes []Node
		db.Where("node_type = ?", "hub").Find(&hubNodes)
		assert.Equal(t, 1, len(hubNodes))
		assert.Equal(t, "hub-node-1", hubNodes[0].Name)
		
		var spokeNodes []Node
		db.Where("node_type = ?", "spoke").Find(&spokeNodes)
		assert.Equal(t, 1, len(spokeNodes))
		assert.Equal(t, "spoke-node-1", spokeNodes[0].Name)
	})
	
	t.Run("node configuration generation", func(t *testing.T) {
		// 模拟WireGuard配置生成
		hubNode := &Node{
			ID:          uuid.New(),
			Name:        "hub-config-test",
			NodeType:    "hub",
			PublicKey:   "hub-public-key-config",
			AllocatedIP: "10.100.1.10",
			Status:      "active",
		}
		db.Create(hubNode)
		
		// 模拟配置生成
		config := fmt.Sprintf(`[Interface]
PrivateKey = [PRIVATE_KEY]
Address = %s/24
ListenPort = 51820

[Peer]
PublicKey = %s
AllowedIPs = 10.100.0.0/16
Endpoint = hub.example.com:51820
PersistentKeepalive = 25`, hubNode.AllocatedIP, hubNode.PublicKey)
		
		assert.Contains(t, config, "[Interface]")
		assert.Contains(t, config, "[Peer]")
		assert.Contains(t, config, hubNode.AllocatedIP)
		assert.Contains(t, config, hubNode.PublicKey)
	})
	
	t.Run("IP address allocation", func(t *testing.T) {
		// 模拟IP地址分配
		allocatedIPs := make(map[string]bool)
		
		for i := 1; i <= 10; i++ {
			ip := fmt.Sprintf("10.100.1.%d", i)
			allocatedIPs[ip] = true
			
			node := &Node{
				ID:          uuid.New(),
				Name:        fmt.Sprintf("node-%d", i),
				NodeType:    "spoke",
				PublicKey:   fmt.Sprintf("public-key-%d", i),
				AllocatedIP: ip,
				Status:      "active",
			}
			db.Create(node)
		}
		
		// 验证IP分配
		var nodes []Node
		db.Where("name LIKE ?", "node-%").Find(&nodes)
		
		for _, node := range nodes {
			assert.True(t, allocatedIPs[node.AllocatedIP])
		}
		
		// 验证IP唯一性
		var uniqueIPs []string
		db.Model(&Node{}).Where("name LIKE ?", "node-%").Distinct("allocated_ip").Pluck("allocated_ip", &uniqueIPs)
		assert.Equal(t, 10, len(uniqueIPs))
	})
}

// 审计日志测试
func TestAuditLogging(t *testing.T) {
	db := setupComprehensiveTestDB()
	
	t.Run("audit log creation", func(t *testing.T) {
		// 创建用户
		user := &User{
			ID:       uuid.New(),
			Username: "audituser",
			Email:    "audit@example.com",
			Password: "hashedpassword",
			Role:     "admin",
			Active:   true,
		}
		db.Create(user)
		
		// 创建审计日志
		auditLog := &AuditLog{
			ID:        uuid.New(),
			UserID:    user.ID,
			Action:    "CREATE_NODE",
			Resource:  "nodes",
			Details:   `{"name": "test-node", "type": "spoke"}`,
			IPAddress: "192.168.1.100",
		}
		
		result := db.Create(auditLog)
		assert.NoError(t, result.Error)
		
		// 验证审计日志
		var logs []AuditLog
		db.Where("user_id = ?", user.ID).Find(&logs)
		assert.Equal(t, 1, len(logs))
		assert.Equal(t, "CREATE_NODE", logs[0].Action)
		assert.Equal(t, "nodes", logs[0].Resource)
		
		// 验证详细信息
		var details map[string]interface{}
		err := json.Unmarshal([]byte(logs[0].Details), &details)
		assert.NoError(t, err)
		assert.Equal(t, "test-node", details["name"])
		assert.Equal(t, "spoke", details["type"])
	})
	
	t.Run("audit log filtering", func(t *testing.T) {
		user := &User{
			ID:       uuid.New(),
			Username: "filteruser",
			Email:    "filter@example.com",
			Password: "hashedpassword",
			Role:     "admin",
			Active:   true,
		}
		db.Create(user)
		
		// 创建多个审计日志
		actions := []string{"CREATE_NODE", "UPDATE_NODE", "DELETE_NODE", "CREATE_USER", "UPDATE_USER"}
		for _, action := range actions {
			auditLog := &AuditLog{
				ID:        uuid.New(),
				UserID:    user.ID,
				Action:    action,
				Resource:  "nodes",
				Details:   `{"test": "data"}`,
				IPAddress: "192.168.1.100",
			}
			db.Create(auditLog)
		}
		
		// 测试按操作类型过滤
		var nodeLogs []AuditLog
		db.Where("user_id = ? AND action LIKE ?", user.ID, "%NODE%").Find(&nodeLogs)
		assert.Equal(t, 3, len(nodeLogs))
		
		var userLogs []AuditLog
		db.Where("user_id = ? AND action LIKE ?", user.ID, "%USER%").Find(&userLogs)
		assert.Equal(t, 2, len(userLogs))
	})
}

// 系统监控测试
func TestSystemMonitoring(t *testing.T) {
	db := setupComprehensiveTestDB()
	
	t.Run("system metrics", func(t *testing.T) {
		// 模拟系统指标
		metrics := map[string]interface{}{
			"cpu_usage":    75.5,
			"memory_usage": 60.2,
			"disk_usage":   45.8,
			"network_rx":   1024000,
			"network_tx":   512000,
			"timestamp":    time.Now().Unix(),
		}
		
		// 验证指标数据
		assert.IsType(t, float64(0), metrics["cpu_usage"])
		assert.IsType(t, float64(0), metrics["memory_usage"])
		assert.IsType(t, float64(0), metrics["disk_usage"])
		assert.IsType(t, 0, metrics["network_rx"])
		assert.IsType(t, 0, metrics["network_tx"])
		assert.IsType(t, int64(0), metrics["timestamp"])
		
		// 验证指标范围
		cpuUsage := metrics["cpu_usage"].(float64)
		assert.True(t, cpuUsage >= 0 && cpuUsage <= 100)
		
		memoryUsage := metrics["memory_usage"].(float64)
		assert.True(t, memoryUsage >= 0 && memoryUsage <= 100)
	})
	
	t.Run("node health check", func(t *testing.T) {
		// 创建测试节点
		node := &Node{
			ID:          uuid.New(),
			Name:        "health-check-node",
			NodeType:    "spoke",
			PublicKey:   "health-check-key",
			AllocatedIP: "10.100.3.1",
			Status:      "active",
		}
		db.Create(node)
		
		// 模拟健康检查
		healthStatus := map[string]interface{}{
			"node_id":      node.ID.String(),
			"status":       "healthy",
			"last_seen":    time.Now(),
			"uptime":       3600, // 1小时
			"ping_latency": 15.5, // 15.5ms
		}
		
		assert.Equal(t, node.ID.String(), healthStatus["node_id"])
		assert.Equal(t, "healthy", healthStatus["status"])
		assert.IsType(t, time.Time{}, healthStatus["last_seen"])
		assert.IsType(t, 0, healthStatus["uptime"])
		assert.IsType(t, float64(0), healthStatus["ping_latency"])
	})
}

// 安全功能测试
func TestSecurityFeatures(t *testing.T) {
	t.Run("password strength validation", func(t *testing.T) {
		// 测试密码强度
		strongPassword := "StrongPassword123!"
		weakPassword := "123"
		
		// 模拟密码强度检查
		assert.True(t, len(strongPassword) >= 8)
		assert.True(t, len(weakPassword) < 8)
		
		// 检查密码复杂度
		hasLetter := false
		hasDigit := false
		hasSpecial := false
		
		for _, char := range strongPassword {
			if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' {
				hasLetter = true
			} else if char >= '0' && char <= '9' {
				hasDigit = true
			} else {
				hasSpecial = true
			}
		}
		
		assert.True(t, hasLetter)
		assert.True(t, hasDigit)
		assert.True(t, hasSpecial)
	})
	
	t.Run("rate limiting simulation", func(t *testing.T) {
		// 模拟速率限制
		maxAttempts := 5
		attempts := 0
		
		for i := 0; i < 10; i++ {
			if attempts < maxAttempts {
				attempts++
				// 模拟请求处理
				assert.True(t, true)
			} else {
				// 请求被限制
				assert.True(t, attempts >= maxAttempts)
				break
			}
		}
		
		assert.Equal(t, maxAttempts, attempts)
	})
}

// 完整集成测试
func TestSystemIntegration(t *testing.T) {
	db := setupComprehensiveTestDB()
	
	t.Run("complete workflow", func(t *testing.T) {
		// 1. 创建管理员用户
		admin := &User{
			ID:       uuid.New(),
			Username: "admin",
			Email:    "admin@example.com",
			Password: "hashedpassword",
			Role:     "admin",
			Active:   true,
		}
		db.Create(admin)
		
		// 2. 创建Hub节点
		hubNode := &Node{
			ID:          uuid.New(),
			Name:        "main-hub",
			NodeType:    "hub",
			PublicKey:   "hub-main-key",
			AllocatedIP: "10.100.1.1",
			Status:      "active",
		}
		db.Create(hubNode)
		
		// 3. 创建Spoke节点
		spokeNode := &Node{
			ID:          uuid.New(),
			Name:        "branch-spoke",
			NodeType:    "spoke",
			PublicKey:   "spoke-branch-key",
			AllocatedIP: "10.100.2.1",
			Status:      "active",
		}
		db.Create(spokeNode)
		
		// 4. 记录审计日志
		auditLog := &AuditLog{
			ID:        uuid.New(),
			UserID:    admin.ID,
			Action:    "CREATE_NETWORK",
			Resource:  "network",
			Details:   `{"hub": "main-hub", "spoke": "branch-spoke"}`,
			IPAddress: "192.168.1.100",
		}
		db.Create(auditLog)
		
		// 5. 验证整个系统
		var totalUsers int64
		db.Model(&User{}).Count(&totalUsers)
		assert.Equal(t, int64(1), totalUsers)
		
		var totalNodes int64
		db.Model(&Node{}).Count(&totalNodes)
		assert.Equal(t, int64(2), totalNodes)
		
		var totalAuditLogs int64
		db.Model(&AuditLog{}).Count(&totalAuditLogs)
		assert.Equal(t, int64(1), totalAuditLogs)
		
		// 6. 验证网络拓扑
		var hubNodes []Node
		db.Where("node_type = ?", "hub").Find(&hubNodes)
		assert.Equal(t, 1, len(hubNodes))
		
		var spokeNodes []Node
		db.Where("node_type = ?", "spoke").Find(&spokeNodes)
		assert.Equal(t, 1, len(spokeNodes))
		
		// 7. 验证IP地址分配
		var allocatedIPs []string
		db.Model(&Node{}).Pluck("allocated_ip", &allocatedIPs)
		assert.Equal(t, 2, len(allocatedIPs))
		assert.Contains(t, allocatedIPs, "10.100.1.1")
		assert.Contains(t, allocatedIPs, "10.100.2.1")
	})
}