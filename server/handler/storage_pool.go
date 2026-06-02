package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kvm_console/model"
	"kvm_console/service"
	"kvm_console/taskqueue"
)

// GetStoragePoolList 获取宿主机硬盘存储池列表
func GetStoragePoolList(c *gin.Context) {
	pools, err := service.ListStoragePools()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取存储池列表失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "data": pools})
}

// GetStoragePoolDetail 获取单个宿主机硬盘详情
func GetStoragePoolDetail(c *gin.Context) {
	id := c.Param("id")
	pool, err := service.GetStoragePool(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "data": pool})
}

// GetAllISOs 获取全局 ISO（聚合）
func GetAllISOs(c *gin.Context) {
	isos, err := service.GetAllISOs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取 ISO 列表失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "data": isos})
}

// UpdateStoragePoolConfig 更新显示名称和启用状态
func UpdateStoragePoolConfig(c *gin.Context) {
	id := c.Param("id")
	var req service.UpdateHostStoragePoolConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := service.UpdateHostStoragePoolConfig(id, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "存储池配置已更新"})
}

// SetDefaultStoragePool 设置默认虚拟机存储位置
func SetDefaultStoragePool(c *gin.Context) {
	id := c.Param("id")
	if err := service.SetDefaultHostStoragePool(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "已设为默认存储位置"})
}

// FormatMountStoragePool 提交格式化并挂载任务
func FormatMountStoragePool(c *gin.Context) {
	if !requireHighRiskVerification(c, "format_storage_pool") {
		return
	}
	id := c.Param("id")
	username, _ := c.Get("username")
	usernameStr, _ := username.(string)
	task, err := taskqueue.SubmitWithStruct(model.TaskTypeStorageFormat, gin.H{"id": id}, usernameStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "提交格式化任务失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "格式化并挂载任务已提交",
		"data":    gin.H{"task_id": task.ID},
	})
}

// GetVMStorageTargets 获取创建虚拟机可选存储位置
func GetVMStorageTargets(c *gin.Context) {
	role, _ := c.Get("role")
	targets, err := service.ListVMStorageTargets(role == "admin")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取虚拟机存储位置失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "data": targets})
}
