package service

import (
	"log"
	"strings"
	"sync"
	"time"

	"kvm_console/model"
	"kvm_console/utils"
)

// ==================== 资源采集缓存 ====================
// 后台协程定时采集运行中VM的资源数据，缓存在内存中供列表接口快速读取，
// 同时定时持久化到数据库供历史查询。

// statsCache 内存缓存：VM名称 -> 最新资源数据
var statsCache = struct {
	sync.RWMutex
	data map[string]*VmStats
}{data: make(map[string]*VmStats)}

// hostStatsCache 宿主机最新资源数据缓存
var hostStatsCache = struct {
	sync.RWMutex
	data *HostStats
}{}

// StartStatsCollector 启动后台资源采集协程
// 每 10 秒采集一次运行中VM的资源数据（更新缓存）
// 每 60 秒将缓存快照持久化到数据库
func StartStatsCollector() {
	InitializeVMRuntimeTracker()
	InitializeUserRuntimeQuotaTracker()
	InitializeLightweightRuntimeQuotaTracker()

	go func() {
		collectTicker := time.NewTicker(10 * time.Second)
		persistTicker := time.NewTicker(60 * time.Second)
		defer collectTicker.Stop()
		defer persistTicker.Stop()

		log.Println("资源采集器已启动（采集间隔: 10s, 持久化间隔: 60s）")

		for {
			select {
			case <-collectTicker.C:
				collectHostStats()
				observedAt := time.Now()
				activeVMs, err := getRuntimeActiveVMSetFromHost()
				if err != nil {
					log.Printf("[运行时长] 获取宿主机运行中虚拟机列表失败: %v", err)
				} else {
					SyncAllUserRuntimeQuotaStatesWithActiveVMs(activeVMs, observedAt)
					syncAllLightweightVMRuntimeQuotaStatesWithActiveVMs(activeVMs, observedAt)
				}
				if !IsMaintenanceModeEnabled() {
					collectAllVMStats()
				}
			case <-persistTicker.C:
				persistStatsToDB()
				persistHostStatsToDB()
			}
		}
	}()

	// 启动流量配额检查定时器（每 60 秒检查 + 凌晨重置）
	StartTrafficQuotaChecker()
}

// collectHostStats 采集宿主机资源数据
func collectHostStats() {
	stats, err := GetHostStats()
	if err == nil {
		hostStatsCache.Lock()
		hostStatsCache.data = stats
		hostStatsCache.Unlock()
	}
}

// collectAllVMStats 批量采集所有运行中VM的资源
func collectAllVMStats() {
	SyncVMRuntimeStatesFromHost(time.Now())

	// 获取运行中的VM列表
	result := utils.ExecShell("virsh list --name --state-running 2>/dev/null | grep -v '^$'")
	if result.Error != nil {
		return
	}

	names := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	runningSet := make(map[string]bool)

	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		runningSet[name] = true

		// 采集资源（GetVMStats 内部有 1s sleep 用于 CPU 采样）
		stats, err := GetVMStats(name)
		if err != nil {
			continue
		}

		statsCache.Lock()
		statsCache.data[name] = stats
		statsCache.Unlock()
	}

	// 清理已关机的VM缓存
	statsCache.Lock()
	for name := range statsCache.data {
		if !runningSet[name] {
			delete(statsCache.data, name)
		}
	}
	statsCache.Unlock()
}

// persistStatsToDB 将当前缓存数据批量写入数据库
func persistStatsToDB() {
	statsCache.RLock()
	defer statsCache.RUnlock()

	now := time.Now()
	for vmName, stats := range statsCache.data {
		record := model.VmStatsRecord{
			VMName:      vmName,
			CPUPercent:  stats.CPUPercent,
			MemUsed:     stats.MemUsed,
			MemTotal:    stats.MemTotal,
			NetRxBytes:  stats.NetRxBytes,
			NetTxBytes:  stats.NetTxBytes,
			DiskRdBytes: stats.DiskRdBytes,
			DiskWrBytes: stats.DiskWrBytes,
			RecordedAt:  now,
		}
		if err := model.DB.Create(&record).Error; err != nil {
			log.Printf("持久化资源记录失败 [%s]: %v", vmName, err)
		}
	}
}

// persistHostStatsToDB 将当前宿主机缓存数据持久化到数据库
func persistHostStatsToDB() {
	hostStatsCache.RLock()
	stats := hostStatsCache.data
	hostStatsCache.RUnlock()

	if stats == nil {
		return
	}

	record := model.HostStatsRecord{
		CPUPercent:  stats.CPUPercent,
		MemUsed:     stats.MemUsed,
		MemTotal:    stats.MemTotal,
		NetRxBytes:  stats.NetRxBytes,
		NetTxBytes:  stats.NetTxBytes,
		DiskRdBytes: stats.DiskRdBytes,
		DiskWrBytes: stats.DiskWrBytes,
		RecordedAt:  time.Now(),
	}
	if err := model.DB.Create(&record).Error; err != nil {
		log.Printf("持久化宿主机资源记录失败: %v", err)
	}
}

// GetCachedStats 从缓存获取指定VM的最新资源数据（列表展示用）
func GetCachedStats(name string) *VmStats {
	statsCache.RLock()
	defer statsCache.RUnlock()
	return statsCache.data[name]
}

// GetAllCachedStats 获取全部缓存的资源数据
func GetAllCachedStats() map[string]*VmStats {
	statsCache.RLock()
	defer statsCache.RUnlock()

	copy := make(map[string]*VmStats, len(statsCache.data))
	for k, v := range statsCache.data {
		copy[k] = v
	}
	return copy
}

// DeleteVMStatsRecords 删除指定VM的所有历史资源记录
func DeleteVMStatsRecords(name string) {
	result := model.DB.Where("vm_name = ?", name).Delete(&model.VmStatsRecord{})
	if result.Error != nil {
		log.Printf("清理资源历史记录失败 [%s]: %v", name, result.Error)
	} else if result.RowsAffected > 0 {
		log.Printf("已清理 %s 的 %d 条资源历史记录", name, result.RowsAffected)
	}

	// 同时清理缓存
	statsCache.Lock()
	delete(statsCache.data, name)
	statsCache.Unlock()
}

// QueryVMStatsHistory 按日期范围查询VM的资源历史记录
func QueryVMStatsHistory(name string, start, end time.Time) ([]model.VmStatsRecord, error) {
	var records []model.VmStatsRecord
	err := model.DB.Where("vm_name = ? AND recorded_at >= ? AND recorded_at <= ?", name, start, end).
		Order("recorded_at ASC").
		Find(&records).Error
	return records, err
}

// QueryHostStatsHistory 按日期范围查询宿主机的资源历史记录
func QueryHostStatsHistory(start, end time.Time) ([]model.HostStatsRecord, error) {
	var records []model.HostStatsRecord
	err := model.DB.Where("recorded_at >= ? AND recorded_at <= ?", start, end).
		Order("recorded_at ASC").
		Find(&records).Error
	return records, err
}
