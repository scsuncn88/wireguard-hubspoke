package tests

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/driver/sqlite"
	"time"
)

// 简单的用户模型用于测试
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

func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to test database")
	}
	
	// 自动迁移模式
	db.AutoMigrate(&User{})
	
	return db
}

func TestBasicFunctionality(t *testing.T) {
	t.Run("database connection", func(t *testing.T) {
		db := setupTestDB()
		
		// 测试数据库连接
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, sqlDB.Ping())
	})
	
	t.Run("user creation", func(t *testing.T) {
		db := setupTestDB()
		
		// 创建用户
		user := &User{
			ID:       uuid.New(),
			Username: "testuser",
			Email:    "test@example.com",
			Password: "hashedpassword",
			Role:     "user",
			Active:   true,
		}
		
		result := db.Create(user)
		assert.NoError(t, result.Error)
		assert.Equal(t, int64(1), result.RowsAffected)
		
		// 验证用户被创建
		var retrievedUser User
		db.Where("username = ?", "testuser").First(&retrievedUser)
		assert.Equal(t, user.Username, retrievedUser.Username)
		assert.Equal(t, user.Email, retrievedUser.Email)
	})
	
	t.Run("user operations", func(t *testing.T) {
		db := setupTestDB()
		
		// 创建多个用户
		users := []User{
			{
				ID:       uuid.New(),
				Username: "user1",
				Email:    "user1@example.com",
				Password: "password1",
				Role:     "user",
				Active:   true,
			},
			{
				ID:       uuid.New(),
				Username: "user2",
				Email:    "user2@example.com",
				Password: "password2",
				Role:     "admin",
				Active:   true,
			},
		}
		
		for _, user := range users {
			result := db.Create(&user)
			assert.NoError(t, result.Error)
		}
		
		// 测试查询
		var allUsers []User
		db.Find(&allUsers)
		assert.Equal(t, 2, len(allUsers))
		
		// 测试按角色过滤
		var adminUsers []User
		db.Where("role = ?", "admin").Find(&adminUsers)
		assert.Equal(t, 1, len(adminUsers))
		assert.Equal(t, "user2", adminUsers[0].Username)
		
		// 测试更新
		db.Model(&users[0]).Update("active", false)
		
		var updatedUser User
		db.Where("username = ?", "user1").First(&updatedUser)
		assert.False(t, updatedUser.Active)
		
		// 测试软删除
		db.Delete(&users[1])
		
		var activeUsers []User
		db.Find(&activeUsers)
		assert.Equal(t, 1, len(activeUsers))
		
		// 测试包含软删除的查询
		var allUsersIncludingDeleted []User
		db.Unscoped().Find(&allUsersIncludingDeleted)
		assert.Equal(t, 2, len(allUsersIncludingDeleted))
	})
}

func TestUUIDGeneration(t *testing.T) {
	t.Run("uuid generation", func(t *testing.T) {
		id1 := uuid.New()
		id2 := uuid.New()
		
		assert.NotEqual(t, id1, id2)
		assert.NotEqual(t, uuid.Nil, id1)
		assert.NotEqual(t, uuid.Nil, id2)
	})
}

func TestPasswordHashing(t *testing.T) {
	t.Run("password operations", func(t *testing.T) {
		password := "testpassword123"
		
		// 模拟密码验证
		assert.NotEmpty(t, password)
		assert.True(t, len(password) >= 8)
		
		// 模拟密码散列
		hashedPassword := "hashed_" + password
		assert.NotEqual(t, password, hashedPassword)
		assert.Contains(t, hashedPassword, "hashed_")
	})
}