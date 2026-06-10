package service

// maintenance_helper.go — 保留维护模式相关的内部辅助函数，
// 这些函数需要访问 service 根包的未导出状态（如 statsCache）。

func clearRuntimeCachesForMaintenance() {
	statsCache.Lock()
	statsCache.data = make(map[string]*VmStats)
	statsCache.Unlock()
}
