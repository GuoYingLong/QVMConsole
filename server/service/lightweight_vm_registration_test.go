package service

import (
	"fmt"
	"path/filepath"
	"testing"

	"kvm_console/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupLightweightRegistrationQuotaTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	oldDB := model.DB
	t.Cleanup(func() {
		model.DB = oldDB
	})
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "lightweight-registration-quota-test.db")), &gorm.Config{})
	if err != nil {
		t.Skipf("当前环境无法打开 SQLite 测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.VPCSwitch{}, &model.LightweightVMRegistration{}, &model.LightweightVMQuota{}, &model.LightweightVMTrafficMonthly{}, &model.VmStatsRecord{}); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	model.DB = db
	if err := db.Create(&model.VPCSwitch{ID: 7, Username: "admin", Name: "light-vpc", CIDR: "10.7.0.0/24", BridgeMode: BridgeModeNAT}).Error; err != nil {
		t.Fatalf("写入测试 VPC 失败: %v", err)
	}
	if err := db.Create(&model.User{Username: "alice", Role: "user", CloudType: CloudTypeLightweight, DedicatedVPCSwitchID: 7}).Error; err != nil {
		t.Fatalf("写入测试用户失败: %v", err)
	}
	return db
}

func TestUpdateLightweightVMQuotaByVMNameUpdatesPendingRegistration(t *testing.T) {
	db := setupLightweightRegistrationQuotaTestDB(t)
	if err := db.Create(&model.LightweightVMRegistration{
		Username:          "alice",
		VMName:            "lightvm",
		Template:          "ubuntu",
		VCPU:              2,
		RAM:               2,
		SwitchID:          7,
		TrafficDownGB:     10,
		TrafficUpGB:       5,
		BandwidthDownMbps: 20,
		BandwidthUpMbps:   10,
		MaxPortForwards:   3,
		Status:            LightweightVMRegistrationStatusPending,
	}).Error; err != nil {
		t.Fatalf("写入注册记录失败: %v", err)
	}

	quota, reg, err := UpdateLightweightVMQuotaByVMName("alice", LightweightVMQuotaRequest{
		VMName:            "lightvm",
		TrafficDownGB:     30,
		TrafficUpGB:       12,
		BandwidthDownMbps: 50,
		BandwidthUpMbps:   25,
		MaxPortForwards:   8,
		MaxRuntimeHours:   24,
	})
	if err != nil {
		t.Fatalf("更新待确认 VM 配额失败: %v", err)
	}
	if quota != nil {
		t.Fatalf("待确认 VM 不应返回运行配额: %+v", quota)
	}
	if reg == nil || reg.TrafficDownGB != 30 || reg.TrafficUpGB != 12 || reg.BandwidthDownMbps != 50 || reg.BandwidthUpMbps != 25 || reg.MaxPortForwards != 8 || reg.MaxRuntimeHours != 24 {
		t.Fatalf("注册记录配额未更新: %+v", reg)
	}
}

func TestUpdateLightweightVMQuotaByVMNameUpdatesActiveQuota(t *testing.T) {
	db := setupLightweightRegistrationQuotaTestDB(t)
	if err := db.Create(&model.LightweightVMQuota{
		Username:          "alice",
		VMName:            "activevm",
		TrafficDownGB:     10,
		TrafficUpGB:       5,
		BandwidthDownMbps: 20,
		BandwidthUpMbps:   10,
		MaxPortForwards:   3,
		MaxRuntimeHours:   12,
	}).Error; err != nil {
		t.Fatalf("写入运行配额失败: %v", err)
	}

	quota, reg, err := UpdateLightweightVMQuotaByVMName("alice", LightweightVMQuotaRequest{
		VMName:            "activevm",
		TrafficDownGB:     60,
		TrafficUpGB:       40,
		BandwidthDownMbps: 100,
		BandwidthUpMbps:   80,
		MaxPortForwards:   12,
		MaxRuntimeHours:   48,
	})
	if err != nil {
		t.Fatalf("更新已开通 VM 配额失败: %v", err)
	}
	if reg != nil {
		t.Fatalf("无注册记录的运行 VM 不应返回注册视图: %+v", reg)
	}
	if quota == nil || quota.TrafficDownGB != 60 || quota.TrafficUpGB != 40 || quota.BandwidthDownMbps != 100 || quota.BandwidthUpMbps != 80 || quota.MaxPortForwards != 12 || quota.MaxRuntimeHours != 48 {
		t.Fatalf("运行配额未更新: %+v", quota)
	}
}

func TestRemoveLightweightVMRegistrationByVMNameCleansActiveRecords(t *testing.T) {
	db := setupLightweightRegistrationQuotaTestDB(t)
	if err := db.Create(&model.LightweightVMRegistration{
		Username: "alice",
		VMName:   "activevm",
		Template: "ubuntu",
		VCPU:     2,
		RAM:      2,
		SwitchID: 7,
		Status:   LightweightVMRegistrationStatusActive,
	}).Error; err != nil {
		t.Fatalf("写入注册记录失败: %v", err)
	}
	if err := db.Create(&model.LightweightVMQuota{Username: "alice", VMName: "activevm"}).Error; err != nil {
		t.Fatalf("写入运行配额失败: %v", err)
	}
	if err := db.Create(&model.LightweightVMTrafficMonthly{Username: "alice", VMName: "activevm", Month: currentTrafficMonth()}).Error; err != nil {
		t.Fatalf("写入月流量记录失败: %v", err)
	}

	if err := RemoveLightweightVMRegistrationByVMName("alice", "activevm"); err != nil {
		t.Fatalf("移除已开通 VM 失败: %v", err)
	}

	var regCount, quotaCount, trafficCount int64
	db.Model(&model.LightweightVMRegistration{}).Where("username = ? AND vm_name = ?", "alice", "activevm").Count(&regCount)
	db.Model(&model.LightweightVMQuota{}).Where("username = ? AND vm_name = ?", "alice", "activevm").Count(&quotaCount)
	db.Model(&model.LightweightVMTrafficMonthly{}).Where("username = ? AND vm_name = ?", "alice", "activevm").Count(&trafficCount)
	if regCount != 0 || quotaCount != 0 || trafficCount != 0 {
		t.Fatalf("移除后应清理注册/配额/月流量记录，got reg=%d quota=%d traffic=%d", regCount, quotaCount, trafficCount)
	}
}

func TestRemoveLightweightVMRegistrationByVMNameRejectsProvisioning(t *testing.T) {
	db := setupLightweightRegistrationQuotaTestDB(t)
	if err := db.Create(&model.LightweightVMRegistration{
		Username: "alice",
		VMName:   "bootingvm",
		Template: "ubuntu",
		VCPU:     2,
		RAM:      2,
		SwitchID: 7,
		Status:   LightweightVMRegistrationStatusProvisioning,
	}).Error; err != nil {
		t.Fatalf("写入注册记录失败: %v", err)
	}

	if err := RemoveLightweightVMRegistrationByVMName("alice", "bootingvm"); err == nil {
		t.Fatalf("开通中的 VM 不应允许移除")
	}
}

func TestIsVMAlreadyExistsError(t *testing.T) {
	if !isVMAlreadyExistsError(fmt.Errorf("虚拟机 'linuxtestuser' 已存在")) {
		t.Fatalf("应识别中文虚拟机已存在错误")
	}
	if !isVMAlreadyExistsError(fmt.Errorf("domain already exists")) {
		t.Fatalf("应识别英文虚拟机已存在错误")
	}
	if isVMAlreadyExistsError(fmt.Errorf("未获取到虚拟机 IP")) {
		t.Fatalf("不应把普通初始化错误识别成已存在错误")
	}
}
