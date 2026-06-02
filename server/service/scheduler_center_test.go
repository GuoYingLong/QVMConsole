package service

import (
	"path/filepath"
	"testing"
	"time"

	"kvm_console/config"
	"kvm_console/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSchedulerEventTestDB(t *testing.T) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "scheduler-event-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Skipf("当前环境无法打开 SQLite 测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.SchedulerEvent{}); err != nil {
		t.Skipf("当前环境无法迁移 SQLite 测试数据库: %v", err)
	}

	model.DB = db
	config.GlobalConfig = &config.Config{
		DynamicMemorySchedulerEnabled: true,
		SchedulerEventRetentionHours:  168,
	}
	registerDynamicMemorySchedulers()
}

func TestSchedulerEventLifecycleAndFilters(t *testing.T) {
	setupSchedulerEventTestDB(t)

	event, err := StartSchedulerEvent(SchedulerEventStartInput{
		SchedulerKey:   schedulerKeyDynamicMemoryBalloon,
		SchedulerName:  schedulerNameDynamicMemoryBalloon,
		SchedulerGroup: schedulerGroupDynamicMemory,
		VMName:         "vm-alpha",
		VMBackend:      memoryBackendBalloon,
		TriggerReason:  "可用内存比例低于阈值，触发扩容",
	})
	if err != nil {
		t.Fatalf("创建调度事件失败: %v", err)
	}
	if event.Status != SchedulerEventStatusRunning {
		t.Fatalf("期望初始状态为 running，实际为 %s", event.Status)
	}

	if err := FinishSchedulerEventSuccess(event, "已将当前内存从 2048MB 调整到 3072MB"); err != nil {
		t.Fatalf("更新调度事件成功状态失败: %v", err)
	}
	if event.Status != SchedulerEventStatusSuccess {
		t.Fatalf("期望成功状态为 success，实际为 %s", event.Status)
	}
	if event.FinishedAt == nil {
		t.Fatalf("期望成功事件写入完成时间")
	}

	failedEvent, err := StartSchedulerEvent(SchedulerEventStartInput{
		SchedulerKey:   schedulerKeyDynamicMemoryVirtioMem,
		SchedulerName:  schedulerNameDynamicMemoryVirtioMem,
		SchedulerGroup: schedulerGroupDynamicMemory,
		VMName:         "vm-beta",
		VMBackend:      memoryBackendVirtioMem,
		TriggerReason:  "来宾内存使用率超过 70.0%，触发扩容",
	})
	if err != nil {
		t.Fatalf("创建失败事件失败: %v", err)
	}
	if err := FinishSchedulerEventFailed(failedEvent, "宿主机可用内存不足"); err != nil {
		t.Fatalf("更新调度事件失败状态失败: %v", err)
	}

	list, total, err := model.ListSchedulerEvents(model.SchedulerEventFilter{
		Page:         1,
		PageSize:     20,
		SchedulerKey: schedulerKeyDynamicMemoryVirtioMem,
		Status:       SchedulerEventStatusFailed,
	})
	if err != nil {
		t.Fatalf("查询调度事件失败: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("期望筛选后仅返回 1 条失败事件，实际 total=%d len=%d", total, len(list))
	}
	if list[0].VMName != "vm-beta" {
		t.Fatalf("期望返回 vm-beta，实际为 %s", list[0].VMName)
	}
	if list[0].ErrorMessage != "宿主机可用内存不足" {
		t.Fatalf("期望保留失败原因，实际为 %s", list[0].ErrorMessage)
	}
}

func TestSchedulerLatestEventAndCleanup(t *testing.T) {
	setupSchedulerEventTestDB(t)

	oldEvent := &model.SchedulerEvent{
		SchedulerKey:   schedulerKeyDynamicMemoryBalloon,
		SchedulerName:  schedulerNameDynamicMemoryBalloon,
		SchedulerGroup: schedulerGroupDynamicMemory,
		VMName:         "vm-old",
		VMBackend:      memoryBackendBalloon,
		Status:         SchedulerEventStatusSuccess,
		TriggerReason:  "旧事件",
		StartedAt:      time.Now().Add(-200 * time.Hour),
		CreatedAt:      time.Now().Add(-200 * time.Hour),
		UpdatedAt:      time.Now().Add(-200 * time.Hour),
	}
	if err := model.CreateSchedulerEvent(oldEvent); err != nil {
		t.Fatalf("写入旧事件失败: %v", err)
	}

	newEvent, err := StartSchedulerEvent(SchedulerEventStartInput{
		SchedulerKey:   schedulerKeyDynamicMemoryBalloon,
		SchedulerName:  schedulerNameDynamicMemoryBalloon,
		SchedulerGroup: schedulerGroupDynamicMemory,
		VMName:         "vm-new",
		VMBackend:      memoryBackendBalloon,
		TriggerReason:  "新事件",
	})
	if err != nil {
		t.Fatalf("写入新事件失败: %v", err)
	}
	if err := FinishSchedulerEventSuccess(newEvent, "完成"); err != nil {
		t.Fatalf("更新新事件失败: %v", err)
	}

	schedulers, err := ListSchedulers()
	if err != nil {
		t.Fatalf("获取调度器概览失败: %v", err)
	}
	if len(schedulers) == 0 {
		t.Fatalf("期望返回已注册调度器")
	}

	runSchedulerEventCleanupOnce()

	list, total, err := model.ListSchedulerEvents(model.SchedulerEventFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("清理后查询事件失败: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("期望清理后仅保留 1 条事件，实际 total=%d len=%d", total, len(list))
	}
	if list[0].VMName != "vm-new" {
		t.Fatalf("期望保留最新事件，实际为 %s", list[0].VMName)
	}
}

func TestDynamicMemorySchedulerReasonBuilders(t *testing.T) {
	if got := buildBalloonExpandReason(0.10, 0.15); got != "可用内存比例 10.0% 低于增长阈值 15.0%，触发扩容" {
		t.Fatalf("气球扩容原因不符合预期: %s", got)
	}
	if got := buildBalloonReclaimReason(0.42, 0.35); got != "空闲内存比例 42.0% 高于回收阈值 35.0%，触发回收" {
		t.Fatalf("气球回收原因不符合预期: %s", got)
	}
	if got := buildVirtioMemExpandReason(0.81); got != "来宾内存使用率 81.0% 超过 70.0%，触发扩容" {
		t.Fatalf("弹性内存扩容原因不符合预期: %s", got)
	}
	if got := buildVirtioMemReclaimReason(0.35); got != "来宾内存使用率 35.0% 低于 50.0%，触发缩容" {
		t.Fatalf("弹性内存缩容原因不符合预期: %s", got)
	}
}
