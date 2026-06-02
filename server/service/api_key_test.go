package service

import (
	"path/filepath"
	"testing"

	"kvm_console/config"
	"kvm_console/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAPIKeyTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	oldDB := model.DB
	oldConfig := config.GlobalConfig
	t.Cleanup(func() {
		model.DB = oldDB
		config.GlobalConfig = oldConfig
	})
	config.GlobalConfig = &config.Config{SecuritySecret: "api-key-test-secret"}

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "api-key-test.db")), &gorm.Config{})
	if err != nil {
		t.Skipf("当前环境无法打开 SQLite 测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserAPIKey{}); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	model.DB = db
	return db
}

func TestRotateAndAuthenticateUserAPIKey(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	user := model.User{Username: "alice", Role: "user", Status: UserStatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	generated, err := RotateUserAPIKey(user.ID)
	if err != nil {
		t.Fatalf("生成 API Key 失败: %v", err)
	}
	if generated.APIKeyID == "" || generated.APIKey == "" || !generated.Enabled {
		t.Fatalf("生成结果不完整: %#v", generated)
	}

	authed, err := AuthenticateAPIKey(generated.APIKeyID, generated.APIKey)
	if err != nil {
		t.Fatalf("API Key 应认证成功: %v", err)
	}
	if authed.Username != user.Username {
		t.Fatalf("认证用户不正确，got=%s", authed.Username)
	}
}

func TestRotateUserAPIKeyInvalidatesOldKey(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	user := model.User{Username: "alice", Role: "user", Status: UserStatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	first, err := RotateUserAPIKey(user.ID)
	if err != nil {
		t.Fatalf("第一次生成 API Key 失败: %v", err)
	}
	second, err := RotateUserAPIKey(user.ID)
	if err != nil {
		t.Fatalf("第二次生成 API Key 失败: %v", err)
	}
	if _, err := AuthenticateAPIKey(first.APIKeyID, first.APIKey); err == nil {
		t.Fatal("旧 API Key 应在重新生成后失效")
	}
	if _, err := AuthenticateAPIKey(second.APIKeyID, second.APIKey); err != nil {
		t.Fatalf("新 API Key 应认证成功: %v", err)
	}
}

func TestRevokeUserAPIKeyRejectsAuthentication(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	user := model.User{Username: "alice", Role: "user", Status: UserStatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	generated, err := RotateUserAPIKey(user.ID)
	if err != nil {
		t.Fatalf("生成 API Key 失败: %v", err)
	}
	if err := RevokeUserAPIKey(user.ID); err != nil {
		t.Fatalf("撤销 API Key 失败: %v", err)
	}
	if _, err := AuthenticateAPIKey(generated.APIKeyID, generated.APIKey); err == nil {
		t.Fatal("撤销后的 API Key 应认证失败")
	}
}
