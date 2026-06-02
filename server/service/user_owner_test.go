package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kvm_console/config"
	"kvm_console/model"
)

func setupUserOwnerTestEnv(t *testing.T) string {
	t.Helper()

	oldDB := model.DB
	oldConfig := config.GlobalConfig
	vmAccessDir := filepath.Join(t.TempDir(), "vm-access")

	t.Cleanup(func() {
		model.DB = oldDB
		config.GlobalConfig = oldConfig
	})

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "user-owner-test.db")), &gorm.Config{})
	if err != nil {
		t.Skipf("当前环境无法打开 SQLite 测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}

	model.DB = db
	config.GlobalConfig = &config.Config{
		VMAccessDir: vmAccessDir,
	}

	if err := os.MkdirAll(vmAccessDir, 0755); err != nil {
		t.Fatalf("创建测试 VM 访问目录失败: %v", err)
	}

	return vmAccessDir
}

func TestFindVMOwnerIgnoresStaleVMAccessUser(t *testing.T) {
	vmAccessDir := setupUserOwnerTestEnv(t)

	if err := model.DB.Create(&model.User{Username: "test", Role: "user", Status: "active"}).Error; err != nil {
		t.Fatalf("写入测试用户失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vmAccessDir, "ghost"), []byte("demo\n"), 0644); err != nil {
		t.Fatalf("写入幽灵用户映射失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vmAccessDir, "test"), []byte("demo\n"), 0644); err != nil {
		t.Fatalf("写入真实用户映射失败: %v", err)
	}

	owner := FindVMOwner("demo")
	if owner != "test" {
		t.Fatalf("期望忽略无效用户并返回真实归属 test，实际为 %q", owner)
	}
}

func TestAssignVMsToUserWithQuotasRejectsMissingUserBeforeWritingAccessFile(t *testing.T) {
	vmAccessDir := setupUserOwnerTestEnv(t)

	err := AssignVMsToUserWithQuotas("ghost", []string{"demo"}, nil)
	if err == nil || !strings.Contains(err.Error(), "用户不存在") {
		t.Fatalf("期望返回用户不存在错误，实际为: %v", err)
	}

	if _, statErr := os.Stat(filepath.Join(vmAccessDir, "ghost")); !os.IsNotExist(statErr) {
		t.Fatalf("缺失用户不应写入 VM 访问文件，实际错误: %v", statErr)
	}
}
