package service

import (
	"path/filepath"
	"testing"
	"time"

	"kvm_console/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBuildUserRuntimeQuotaSnapshot(t *testing.T) {
	now := time.Now()
	lastObserved := now.Add(-30 * time.Minute)
	user := &model.User{
		MaxRuntimeHours:       3,
		UsedRuntimeSeconds:    3600,
		RuntimeActiveVMCount:  2,
		RuntimeLastObservedAt: &lastObserved,
	}

	snapshot := BuildUserRuntimeQuotaSnapshot(user, now)
	if snapshot.UsedSeconds != 7200 {
		t.Fatalf("累计运行时长错误，got=%d want=%d", snapshot.UsedSeconds, 7200)
	}
	if snapshot.RemainingSeconds != 3600 {
		t.Fatalf("剩余运行时长错误，got=%d want=%d", snapshot.RemainingSeconds, 3600)
	}
	if snapshot.QuotaReached {
		t.Fatal("未达到配额上限时不应标记为已耗尽")
	}
}

func TestCheckRuntimeQuotaAvailableForUser(t *testing.T) {
	now := time.Now()
	user := &model.User{
		Username:              "alice",
		Role:                  "user",
		MaxRuntimeHours:       1,
		UsedRuntimeSeconds:    int64(time.Hour / time.Second),
		RuntimeLastObservedAt: &now,
	}

	if err := CheckRuntimeQuotaAvailableForUser(user, now); err == nil {
		t.Fatal("配额已耗尽时应返回错误")
	}
}

func TestPersistUserRuntimeQuotaStateClearsWarningWhenQuotaRaised(t *testing.T) {
	oldDB := model.DB
	t.Cleanup(func() {
		model.DB = oldDB
	})

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "runtime-quota-test.db")), &gorm.Config{})
	if err != nil {
		t.Skipf("当前环境无法打开 SQLite 测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	model.DB = db

	now := time.Now()
	warnedAt := now.Add(-time.Hour)
	user := model.User{
		Username:             "alice",
		Role:                 "user",
		MaxRuntimeHours:      10,
		UsedRuntimeSeconds:   3600,
		RuntimeWarningSentAt: &warnedAt,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	snapshot, shouldWarn, err := persistUserRuntimeQuotaState(&user, 0, now)
	if err != nil {
		t.Fatalf("持久化运行时长状态失败: %v", err)
	}
	if shouldWarn {
		t.Fatal("剩余时间大于预警阈值时不应再次触发预警")
	}
	if snapshot.QuotaReached {
		t.Fatal("配额远未耗尽时不应标记为已耗尽")
	}

	var refreshed model.User
	if err := db.Where("id = ?", user.ID).First(&refreshed).Error; err != nil {
		t.Fatalf("读取测试用户失败: %v", err)
	}
	if refreshed.RuntimeWarningSentAt != nil {
		t.Fatal("剩余时间重新回到阈值以上后，应清空预警发送标记")
	}
}
