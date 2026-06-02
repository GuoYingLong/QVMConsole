package service

import (
	"path/filepath"
	"testing"

	"kvm_console/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPortForwardQuotaTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "port-forward-quota-test.db")), &gorm.Config{})
	if err != nil {
		t.Skipf("当前环境无法打开 SQLite 测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	return db
}

func TestCheckUserPortForwardQuota(t *testing.T) {
	oldDB := model.DB
	t.Cleanup(func() {
		model.DB = oldDB
	})

	db := setupPortForwardQuotaTestDB(t)
	model.DB = db

	user := model.User{Username: "alice", Role: "user", EnablePortForward: true, MaxPortForwards: 1}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	if err := CheckUserPortForwardQuota("alice", 1); err != nil {
		t.Fatalf("剩余配额足够时不应报错: %v", err)
	}
	if err := CheckUserPortForwardQuota("alice", 2); err == nil {
		t.Fatal("超过端口转发配额时应返回错误")
	}
}

func TestCheckUserPortForwardQuotaSkipsAdminAndUnlimited(t *testing.T) {
	oldDB := model.DB
	t.Cleanup(func() {
		model.DB = oldDB
	})

	db := setupPortForwardQuotaTestDB(t)
	model.DB = db

	users := []model.User{
		{Username: "admin", Role: "admin", EnablePortForward: true, MaxPortForwards: 1},
		{Username: "bob", Role: "user", EnablePortForward: true, MaxPortForwards: 0},
	}
	for _, user := range users {
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("创建测试用户失败: %v", err)
		}
	}

	if err := CheckUserPortForwardQuota("admin", 100); err != nil {
		t.Fatalf("管理员不应受端口转发配额限制: %v", err)
	}
	if err := CheckUserPortForwardQuota("bob", 100); err != nil {
		t.Fatalf("不限额用户不应受端口转发配额限制: %v", err)
	}
}

func TestNormalizeEditablePortForwardProtocol(t *testing.T) {
	validCases := map[string]string{
		"":    "tcp",
		"tcp": "tcp",
		"UDP": "udp",
	}
	for input, expected := range validCases {
		got, err := normalizeEditablePortForwardProtocol(input)
		if err != nil {
			t.Fatalf("协议 %q 不应报错: %v", input, err)
		}
		if got != expected {
			t.Fatalf("协议 %q 规范化结果错误，got=%q want=%q", input, got, expected)
		}
	}

	if _, err := normalizeEditablePortForwardProtocol("both"); err == nil {
		t.Fatal("编辑端口转发时不应接受 both 协议")
	}
}

func TestCheckUserPortForwardFeatureEnabled(t *testing.T) {
	oldDB := model.DB
	t.Cleanup(func() {
		model.DB = oldDB
	})

	db := setupPortForwardQuotaTestDB(t)
	model.DB = db

	users := []model.User{
		{Username: "admin", Role: "admin", EnablePortForward: false},
		{Username: "enabled-user", Role: "user", EnablePortForward: true},
		{Username: "disabled-user", Role: "user", EnablePortForward: false},
	}
	for _, user := range users {
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("创建测试用户失败: %v", err)
		}
	}

	if err := CheckUserPortForwardFeatureEnabled("admin"); err != nil {
		t.Fatalf("管理员不应受端口转发功能开关限制: %v", err)
	}
	if err := CheckUserPortForwardFeatureEnabled("enabled-user"); err != nil {
		t.Fatalf("已开通端口转发的用户不应报错: %v", err)
	}
	if err := CheckUserPortForwardFeatureEnabled("disabled-user"); err == nil {
		t.Fatal("未开通端口转发的用户应返回错误")
	}
}
