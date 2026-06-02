package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"kvm_console/config"
	"kvm_console/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAggregateUserDailyTrafficSkipsVPCBoundVMs(t *testing.T) {
	oldDB := model.DB
	oldConfig := config.GlobalConfig
	t.Cleanup(func() {
		model.DB = oldDB
		config.GlobalConfig = oldConfig
	})

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "traffic-quota-test.db")), &gorm.Config{})
	if err != nil {
		t.Skipf("当前环境无法打开 SQLite 测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.VmStatsRecord{}, &model.VPCVMBinding{}); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	model.DB = db

	accessDir := t.TempDir()
	config.GlobalConfig = &config.Config{VMAccessDir: accessDir}
	if err := os.WriteFile(filepath.Join(accessDir, "alice"), []byte("legacy-vm\nvpc-vm\n"), 0644); err != nil {
		t.Fatalf("写入测试 VM 访问列表失败: %v", err)
	}

	now := time.Now()
	records := []model.VmStatsRecord{
		{VMName: "legacy-vm", NetRxBytes: 100, NetTxBytes: 200, RecordedAt: now.Add(-2 * time.Hour)},
		{VMName: "legacy-vm", NetRxBytes: 600, NetTxBytes: 900, RecordedAt: now.Add(-time.Hour)},
		{VMName: "vpc-vm", NetRxBytes: 1000, NetTxBytes: 2000, RecordedAt: now.Add(-2 * time.Hour)},
		{VMName: "vpc-vm", NetRxBytes: 9000, NetTxBytes: 12000, RecordedAt: now.Add(-time.Hour)},
	}
	for _, record := range records {
		if err := db.Create(&record).Error; err != nil {
			t.Fatalf("写入测试流量记录失败: %v", err)
		}
	}
	if err := db.Create(&model.VPCVMBinding{
		VMName:          "vpc-vm",
		Username:        "alice",
		SwitchID:        1,
		SecurityGroupID: 1,
	}).Error; err != nil {
		t.Fatalf("写入测试 VPC 绑定失败: %v", err)
	}

	down, up := AggregateUserDailyTraffic("alice")
	if down != 500 || up != 700 {
		t.Fatalf("用户级流量应只统计非 VPC VM，got down=%d up=%d", down, up)
	}
}
