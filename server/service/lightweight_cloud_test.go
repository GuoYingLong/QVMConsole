package service

import (
	"path/filepath"
	"testing"
	"time"

	"kvm_console/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAggregateLightweightVMMonthlyTrafficSkipsNegativeDeltas(t *testing.T) {
	oldDB := model.DB
	t.Cleanup(func() {
		model.DB = oldDB
	})

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "lightweight-cloud-test.db")), &gorm.Config{})
	if err != nil {
		t.Skipf("当前环境无法打开 SQLite 测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.VmStatsRecord{}, &model.LightweightVMTrafficMonthly{}); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	model.DB = db

	now := time.Now()
	records := []model.VmStatsRecord{
		{VMName: "light-vm", NetRxBytes: 1000, NetTxBytes: 2000, RecordedAt: now.Add(-4 * time.Hour)},
		{VMName: "light-vm", NetRxBytes: 2500, NetTxBytes: 3000, RecordedAt: now.Add(-3 * time.Hour)},
		{VMName: "light-vm", NetRxBytes: 100, NetTxBytes: 100, RecordedAt: now.Add(-2 * time.Hour)},
		{VMName: "light-vm", NetRxBytes: 900, NetTxBytes: 700, RecordedAt: now.Add(-time.Hour)},
	}
	for _, record := range records {
		if err := db.Create(&record).Error; err != nil {
			t.Fatalf("写入测试流量记录失败: %v", err)
		}
	}

	down, up := AggregateLightweightVMMonthlyTraffic("light-vm")
	if down != 2300 || up != 1600 {
		t.Fatalf("轻量云 VM 流量应跳过重启归零负增量，got down=%d up=%d", down, up)
	}
}

func TestAggregateLightweightVMMonthlyTrafficAppliesOffset(t *testing.T) {
	oldDB := model.DB
	t.Cleanup(func() {
		model.DB = oldDB
	})

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "lightweight-cloud-offset-test.db")), &gorm.Config{})
	if err != nil {
		t.Skipf("当前环境无法打开 SQLite 测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.VmStatsRecord{}, &model.LightweightVMTrafficMonthly{}); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	model.DB = db

	now := time.Now()
	if err := db.Create(&model.VmStatsRecord{VMName: "light-vm", NetRxBytes: 100, NetTxBytes: 100, RecordedAt: now.Add(-2 * time.Hour)}).Error; err != nil {
		t.Fatalf("写入起始流量记录失败: %v", err)
	}
	if err := db.Create(&model.VmStatsRecord{VMName: "light-vm", NetRxBytes: 2100, NetTxBytes: 4100, RecordedAt: now.Add(-time.Hour)}).Error; err != nil {
		t.Fatalf("写入结束流量记录失败: %v", err)
	}
	if err := db.Create(&model.LightweightVMTrafficMonthly{
		VMName:     "light-vm",
		Username:   "alice",
		Month:      currentTrafficMonth(),
		OffsetDown: 500,
		OffsetUp:   1200,
	}).Error; err != nil {
		t.Fatalf("写入 offset 失败: %v", err)
	}

	down, up := AggregateLightweightVMMonthlyTraffic("light-vm")
	if down != 1500 || up != 2800 {
		t.Fatalf("轻量云 VM 流量应扣除 offset，got down=%d up=%d", down, up)
	}
}

func TestFillLightweightVMNICRuntimeUsesVMStatsRecords(t *testing.T) {
	oldDB := model.DB
	t.Cleanup(func() {
		model.DB = oldDB
	})

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "lightweight-cloud-runtime-test.db")), &gorm.Config{})
	if err != nil {
		t.Skipf("当前环境无法打开 SQLite 测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.VmStatsRecord{}); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	model.DB = db

	now := time.Now()
	records := []model.VmStatsRecord{
		{VMName: "light-vm", NetRxBytes: 1000, NetTxBytes: 2000, RecordedAt: now.Add(-10 * time.Second)},
		{VMName: "light-vm", NetRxBytes: 11240, NetTxBytes: 22500, RecordedAt: now},
	}
	for _, record := range records {
		if err := db.Create(&record).Error; err != nil {
			t.Fatalf("写入测试网卡记录失败: %v", err)
		}
	}

	quota := &model.LightweightVMQuota{VMName: "light-vm"}
	fillLightweightVMNICRuntime(quota)
	if quota.CurrentNetRxBytes != 11240 || quota.CurrentNetTxBytes != 22500 {
		t.Fatalf("应返回最新 VM 网卡累计值，got rx=%d tx=%d", quota.CurrentNetRxBytes, quota.CurrentNetTxBytes)
	}
	if quota.CurrentNetRxRate != "1.00 KB/s" || quota.CurrentNetTxRate != "2.00 KB/s" {
		t.Fatalf("应按 VM 网卡历史记录计算实时速率，got rx=%s tx=%s", quota.CurrentNetRxRate, quota.CurrentNetTxRate)
	}
}

func TestBuildLightweightVMRuntimeQuotaSnapshot(t *testing.T) {
	now := time.Now()
	lastObserved := now.Add(-45 * time.Minute)
	quota := &model.LightweightVMQuota{
		MaxRuntimeHours:       3,
		UsedRuntimeSeconds:    int64(time.Hour / time.Second),
		RuntimeIsActive:       true,
		RuntimeLastObservedAt: &lastObserved,
	}

	snapshot := BuildLightweightVMRuntimeQuotaSnapshot(quota, now)
	if snapshot.UsedSeconds != int64((time.Hour+45*time.Minute)/time.Second) {
		t.Fatalf("累计运行时长错误，got=%d", snapshot.UsedSeconds)
	}
	if snapshot.RemainingSeconds != int64((75*time.Minute)/time.Second) {
		t.Fatalf("剩余运行时长错误，got=%d", snapshot.RemainingSeconds)
	}
	if snapshot.QuotaReached {
		t.Fatal("未达到配额上限时不应标记为已耗尽")
	}
}

func TestCheckLightweightVMRuntimeQuotaAvailableForQuota(t *testing.T) {
	now := time.Now()
	quota := &model.LightweightVMQuota{
		VMName:                "light-vm",
		MaxRuntimeHours:       1,
		UsedRuntimeSeconds:    int64(time.Hour / time.Second),
		RuntimeLastObservedAt: &now,
	}

	if err := CheckLightweightVMRuntimeQuotaAvailableForQuota(quota, now); err == nil {
		t.Fatal("轻量云 VM 运行时长配额已耗尽时应返回错误")
	}
}

func TestCleanupLightweightVMResourcesDeletesRegistration(t *testing.T) {
	oldDB := model.DB
	t.Cleanup(func() {
		model.DB = oldDB
	})

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "lightweight-cleanup-test.db")), &gorm.Config{})
	if err != nil {
		t.Skipf("当前环境无法打开 SQLite 测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.LightweightVMRegistration{}, &model.LightweightVMQuota{}, &model.LightweightVMTrafficMonthly{}); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	model.DB = db

	if err := db.Create(&model.LightweightVMRegistration{
		Username: "alice",
		VMName:   "deletedvm",
		Template: "ubuntu",
		VCPU:     1,
		RAM:      1,
		Status:   LightweightVMRegistrationStatusActive,
	}).Error; err != nil {
		t.Fatalf("写入注册记录失败: %v", err)
	}
	if err := db.Create(&model.LightweightVMQuota{Username: "alice", VMName: "deletedvm"}).Error; err != nil {
		t.Fatalf("写入运行配额失败: %v", err)
	}
	if err := db.Create(&model.LightweightVMTrafficMonthly{Username: "alice", VMName: "deletedvm", Month: currentTrafficMonth()}).Error; err != nil {
		t.Fatalf("写入月流量记录失败: %v", err)
	}

	CleanupLightweightVMResources("deletedvm")

	var regCount, quotaCount, trafficCount int64
	db.Model(&model.LightweightVMRegistration{}).Where("vm_name = ?", "deletedvm").Count(&regCount)
	db.Model(&model.LightweightVMQuota{}).Where("vm_name = ?", "deletedvm").Count(&quotaCount)
	db.Model(&model.LightweightVMTrafficMonthly{}).Where("vm_name = ?", "deletedvm").Count(&trafficCount)
	if regCount != 0 || quotaCount != 0 || trafficCount != 0 {
		t.Fatalf("删除 VM 应清理轻量云注册/配额/流量记录，got reg=%d quota=%d traffic=%d", regCount, quotaCount, trafficCount)
	}
}
