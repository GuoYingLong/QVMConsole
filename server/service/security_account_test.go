package service

import (
	"path/filepath"
	"testing"
	"time"

	"kvm_console/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSecurityAccountTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	oldDB := model.DB
	t.Cleanup(func() {
		model.DB = oldDB
	})

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "security-account-test.db")), &gorm.Config{})
	if err != nil {
		t.Skipf("当前环境无法打开 SQLite 测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.AuthActionToken{}); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	model.DB = db
	return db
}

func TestCreatePendingInvitedUserAllowsDuplicateEmail(t *testing.T) {
	setupSecurityAccountTestDB(t)

	first, _, err := CreatePendingInvitedUser("alice", "shared@example.com", "user", "elastic", 0, 0, 0, 0, 0, 0, 0, false, 0, 0, 0, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("创建第一个待激活用户失败: %v", err)
	}
	second, _, err := CreatePendingInvitedUser("bob", "shared@example.com", "user", "elastic", 0, 0, 0, 0, 0, 0, 0, false, 0, 0, 0, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("相同邮箱应允许绑定多个用户: %v", err)
	}
	if first.ID == second.ID {
		t.Fatal("不同用户应拥有不同 ID")
	}
}

func TestBindUserEmailAllowsSharedEmail(t *testing.T) {
	db := setupSecurityAccountTestDB(t)

	now := time.Now()
	users := []model.User{
		{Username: "alice", Email: "alice@example.com", Role: "user", Status: UserStatusActive, EmailVerifiedAt: &now},
		{Username: "bob", Email: "bob@example.com", Role: "user", Status: UserStatusActive, EmailVerifiedAt: &now},
	}
	for _, user := range users {
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("创建测试用户失败: %v", err)
		}
	}

	var bob model.User
	if err := db.Where("username = ?", "bob").First(&bob).Error; err != nil {
		t.Fatalf("读取 bob 失败: %v", err)
	}
	if err := BindUserEmail(bob.ID, "alice@example.com", now); err != nil {
		t.Fatalf("相同邮箱应允许重复绑定: %v", err)
	}

	var refreshed model.User
	if err := db.First(&refreshed, bob.ID).Error; err != nil {
		t.Fatalf("读取更新后的用户失败: %v", err)
	}
	if refreshed.Email != "alice@example.com" {
		t.Fatalf("邮箱更新失败，got=%s", refreshed.Email)
	}
}

func TestListPasswordResetAccountsByEmailOnlyReturnsActiveUsers(t *testing.T) {
	db := setupSecurityAccountTestDB(t)

	users := []model.User{
		{Username: "charlie", Email: "shared@example.com", Role: "user", Status: UserStatusActive},
		{Username: "alice", Email: "shared@example.com", Role: "admin", Status: UserStatusActive},
		{Username: "disabled", Email: "shared@example.com", Role: "user", Status: UserStatusDisabled},
		{Username: "pending", Email: "shared@example.com", Role: "user", Status: UserStatusPendingInvite},
	}
	for _, user := range users {
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("创建测试用户失败: %v", err)
		}
	}

	accounts, err := ListPasswordResetAccountsByEmail("shared@example.com")
	if err != nil {
		t.Fatalf("列出找回密码账号失败: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("仅应返回 2 个已激活账号，got=%d", len(accounts))
	}
	if accounts[0].Username != "alice" || accounts[1].Username != "charlie" {
		t.Fatalf("账号列表应按用户名排序，got=%+v", accounts)
	}
}

func TestFindPasswordResetUserRequiresExactEmailAndUsername(t *testing.T) {
	db := setupSecurityAccountTestDB(t)

	user := model.User{
		Username: "alice",
		Email:    "shared@example.com",
		Role:     "user",
		Status:   UserStatusActive,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	found, err := FindPasswordResetUser("shared@example.com", "alice")
	if err != nil {
		t.Fatalf("应能按邮箱和用户名命中用户: %v", err)
	}
	if found.ID != user.ID {
		t.Fatalf("命中用户错误，got=%d want=%d", found.ID, user.ID)
	}

	if _, err := FindPasswordResetUser("shared@example.com", "bob"); err == nil {
		t.Fatal("用户名不匹配时应返回错误")
	}
}

func TestCreatePasswordResetTokenRejectsAmbiguousEmail(t *testing.T) {
	db := setupSecurityAccountTestDB(t)

	users := []model.User{
		{Username: "alice", Email: "shared@example.com", Role: "user", Status: UserStatusActive},
		{Username: "bob", Email: "shared@example.com", Role: "user", Status: UserStatusActive},
	}
	for _, user := range users {
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("创建测试用户失败: %v", err)
		}
	}

	if _, _, err := CreatePasswordResetToken("shared@example.com"); err == nil {
		t.Fatal("同邮箱多账号时，旧找回接口应拒绝直接生成重置链接")
	}
}
