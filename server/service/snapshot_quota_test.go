package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"kvm_console/config"
	"kvm_console/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSnapshotQuotaTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	oldDB := model.DB
	oldConfig := config.GlobalConfig
	t.Cleanup(func() {
		model.DB = oldDB
		config.GlobalConfig = oldConfig
	})

	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("当前环境缺少 bash，跳过快照配额测试")
	}

	vmAccessDir := filepath.Join(t.TempDir(), "vm-access")
	if err := os.MkdirAll(vmAccessDir, 0755); err != nil {
		t.Fatalf("创建测试 VM 访问目录失败: %v", err)
	}
	config.GlobalConfig = &config.Config{VMAccessDir: vmAccessDir}

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "snapshot-quota-test.db")), &gorm.Config{})
	if err != nil {
		t.Skipf("当前环境无法打开 SQLite 测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.LightweightVMQuota{}); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	model.DB = db
	return db
}

func installFakeVirshForSnapshotQuotaTest(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	var filePath, content string
	if runtime.GOOS == "windows" {
		filePath = filepath.Join(dir, "virsh.cmd")
		content = "@echo off\r\n" +
			"if \"%1\"==\"snapshot-list\" (\r\n" +
			"  if \"%2\"==\"elastic-a\" (\r\n" +
			"    echo snap-1\r\n" +
			"    echo snap-2\r\n" +
			"    exit /b 0\r\n" +
			"  )\r\n" +
			"  if \"%2\"==\"elastic-b\" exit /b 0\r\n" +
			"  if \"%2\"==\"light-vm\" (\r\n" +
			"    echo snap-1\r\n" +
			"    echo snap-2\r\n" +
			"    exit /b 0\r\n" +
			"  )\r\n" +
			"  if \"%2\"==\"unlimited-vm\" (\r\n" +
			"    echo snap-1\r\n" +
			"    echo snap-2\r\n" +
			"    echo snap-3\r\n" +
			"    exit /b 0\r\n" +
			"  )\r\n" +
			")\r\n" +
			"exit /b 1\r\n"
	} else {
		filePath = filepath.Join(dir, "virsh")
		content = "#!/bin/sh\n" +
			"if [ \"$1\" = \"snapshot-list\" ]; then\n" +
			"  case \"$2\" in\n" +
			"    elastic-a)\n" +
			"      printf 'snap-1\\nsnap-2\\n'\n" +
			"      exit 0\n" +
			"      ;;\n" +
			"    elastic-b)\n" +
			"      exit 0\n" +
			"      ;;\n" +
			"    light-vm)\n" +
			"      printf 'snap-1\\nsnap-2\\n'\n" +
			"      exit 0\n" +
			"      ;;\n" +
			"    unlimited-vm)\n" +
			"      printf 'snap-1\\nsnap-2\\nsnap-3\\n'\n" +
			"      exit 0\n" +
			"      ;;\n" +
			"  esac\n" +
			"fi\n" +
			"exit 1\n"
	}
	if err := os.WriteFile(filePath, []byte(content), 0755); err != nil {
		t.Fatalf("写入 fake virsh 失败: %v", err)
	}
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", dir+string(os.PathListSeparator)+oldPath); err != nil {
		t.Fatalf("更新 PATH 失败: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Setenv("PATH", oldPath)
	})
}

func TestCheckUserSnapshotQuota(t *testing.T) {
	db := setupSnapshotQuotaTestDB(t)
	installFakeVirshForSnapshotQuotaTest(t)

	if err := db.Create(&model.User{Username: "alice", Role: "user", MaxSnapshots: 3}).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	accessFile := filepath.Join(config.GlobalConfig.VMAccessDir, "alice")
	if err := os.WriteFile(accessFile, []byte("elastic-a\nelastic-b\n"), 0644); err != nil {
		t.Fatalf("写入用户 VM 访问列表失败: %v", err)
	}

	if err := CheckUserSnapshotQuota("alice", 1); err != nil {
		t.Fatalf("剩余快照配额足够时不应报错: %v", err)
	}
	if err := CheckUserSnapshotQuota("alice", 2); err == nil {
		t.Fatal("超过弹性云快照配额时应返回错误")
	}
}

func TestCheckLightweightVMSnapshotQuota(t *testing.T) {
	db := setupSnapshotQuotaTestDB(t)
	installFakeVirshForSnapshotQuotaTest(t)

	records := []model.LightweightVMQuota{
		{Username: "bob", VMName: "light-vm", MaxSnapshots: 2},
		{Username: "bob", VMName: "unlimited-vm", MaxSnapshots: 0},
	}
	for _, item := range records {
		if err := db.Create(&item).Error; err != nil {
			t.Fatalf("创建轻量云快照配额失败: %v", err)
		}
	}

	if err := CheckLightweightVMSnapshotQuota("bob", "light-vm", 1); err == nil {
		t.Fatal("超过轻量云单 VM 快照配额时应返回错误")
	}
	if err := CheckLightweightVMSnapshotQuota("bob", "unlimited-vm", 10); err != nil {
		t.Fatalf("单 VM 快照不限额时不应报错: %v", err)
	}
}
