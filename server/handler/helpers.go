package handler

// helpers.go 存放跨 handler 使用的公共工具函数，供同一 package 内各 handler 文件共享。

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"kvm_console/service"
	vm_memory "kvm_console/service/vm/memory"
)

// respondVMListError 统一处理虚拟机列表查询失败的响应（区分 libvirt 不可用与其他错误）
func respondVMListError(c *gin.Context, err error) {
	if service.IsLibvirtUnavailableError(err) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    http.StatusServiceUnavailable,
			"message": "libvirt 服务未启动或未就绪，当前无法获取虚拟机列表",
		})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{
		"code":    500,
		"message": "获取虚拟机列表失败: " + err.Error(),
	})
}

// parseBoolQuery 解析布尔型查询参数，支持 1/true/yes/on（不区分大小写）
func parseBoolQuery(c *gin.Context, key string) bool {
	switch strings.ToLower(strings.TrimSpace(c.Query(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// buildVMListOptions 从查询参数构建 VM 列表选项
func buildVMListOptions(c *gin.Context) service.VMListOptions {
	return service.VMListOptions{
		IncludeResourceUsage: parseBoolQuery(c, "include_resource_usage"),
		IncludeIP:            parseBoolQuery(c, "include_ip"),
	}
}

// sanitizeUserMemoryDynamicRequest 对非管理员用户提交的动态内存请求进行安全校验与默认值填充
func sanitizeUserMemoryDynamicRequest(req *vm_memory.VMMemoryDynamicRequest, baseMemoryGB int) *vm_memory.VMMemoryDynamicRequest {
	if req == nil || req.DynamicEnabled == nil {
		return nil
	}
	if baseMemoryGB <= 0 {
		baseMemoryGB = 1
	}
	enabled := *req.DynamicEnabled
	backend := req.MemoryBackend
	if backend != "virtio_mem" {
		backend = "balloon"
	}
	if !enabled {
		return &vm_memory.VMMemoryDynamicRequest{
			DynamicEnabled: &enabled,
			MemoryBackend:  backend,
			MemoryInitial:  baseMemoryGB,
		}
	}
	memoryMin := max(1, baseMemoryGB/2)
	memoryMax := max(baseMemoryGB, (baseMemoryGB*13+9)/10)
	memoryInitial := baseMemoryGB
	autoBalloon := true
	if backend == "virtio_mem" {
		memoryInitial = memoryMin
		autoBalloon = false
	}
	memoryCurrent := 0
	if req.MemoryCurrent > 0 {
		memoryCurrent = req.MemoryCurrent
		if memoryCurrent < memoryInitial {
			memoryCurrent = memoryInitial
		}
		if memoryCurrent > memoryMax {
			memoryCurrent = memoryMax
		}
	}
	return &vm_memory.VMMemoryDynamicRequest{
		DynamicEnabled: &enabled,
		MemoryBackend:  backend,
		MemoryInitial:  memoryInitial,
		MemoryMin:      memoryMin,
		MemoryMax:      memoryMax,
		AutoBalloon:    &autoBalloon,
		MemoryCurrent:  memoryCurrent,
	}
}
