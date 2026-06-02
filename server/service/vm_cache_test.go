package service

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"kvm_console/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupVMCacheServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	oldDB := model.DB
	oldNow := vmCacheNow
	oldListNames := vmCacheListNamesFromHost
	oldBuild := vmCacheBuildRecordFromHost
	oldRunFullSync := vmCacheRunFullSync
	oldCooldown := vmCacheRefreshCooldown

	t.Cleanup(func() {
		model.DB = oldDB
		vmCacheNow = oldNow
		vmCacheListNamesFromHost = oldListNames
		vmCacheBuildRecordFromHost = oldBuild
		vmCacheRunFullSync = oldRunFullSync
		vmCacheRefreshCooldown = oldCooldown
		resetVMCacheRefreshState()
	})

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "vm-cache-service.db")), &gorm.Config{})
	if err != nil {
		t.Skipf("当前环境无法打开 SQLite 测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.VMCache{}); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	model.DB = db
	resetVMCacheRefreshState()
	return db
}

func resetVMCacheRefreshState() {
	vmCacheRefreshState.Lock()
	vmCacheRefreshState.inProgress = false
	vmCacheRefreshState.lastTriggered = time.Time{}
	vmCacheRefreshState.Unlock()
}

func TestSyncVMCacheFromHostUpsertsAndMarksMissing(t *testing.T) {
	db := setupVMCacheServiceTestDB(t)
	fixedNow := time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)
	vmCacheNow = func() time.Time { return fixedNow }

	records := map[string]model.VMCache{
		"vm-a": {
			Name:          "vm-a",
			OwnerUsername: "alice",
			Status:        "running",
			VCPU:          2,
			MemoryMB:      2048,
			MaxMemoryMB:   4096,
			Present:       true,
			LastSyncedAt:  fixedNow,
		},
		"vm-b": {
			Name:          "vm-b",
			OwnerUsername: "bob",
			Status:        "shut off",
			VCPU:          4,
			MemoryMB:      4096,
			MaxMemoryMB:   4096,
			Present:       true,
			LastSyncedAt:  fixedNow,
		},
	}
	vmCacheListNamesFromHost = func() ([]string, error) {
		return []string{"vm-a", "vm-b"}, nil
	}
	vmCacheBuildRecordFromHost = func(name string, syncedAt time.Time) (model.VMCache, error) {
		record := records[name]
		record.LastSyncedAt = syncedAt
		return record, nil
	}

	if err := SyncVMCacheFromHost(); err != nil {
		t.Fatalf("首次同步失败: %v", err)
	}

	var count int64
	if err := db.Model(&model.VMCache{}).Where("present = ?", true).Count(&count).Error; err != nil {
		t.Fatalf("统计同步结果失败: %v", err)
	}
	if count != 2 {
		t.Fatalf("首次同步后可见 VM 数量不正确，got=%d", count)
	}

	records["vm-a"] = model.VMCache{
		Name:          "vm-a",
		OwnerUsername: "alice",
		Status:        "running",
		VCPU:          8,
		MemoryMB:      8192,
		MaxMemoryMB:   8192,
		Present:       true,
		LastSyncedAt:  fixedNow.Add(time.Minute),
	}
	vmCacheListNamesFromHost = func() ([]string, error) {
		return []string{"vm-a"}, nil
	}
	vmCacheNow = func() time.Time { return fixedNow.Add(time.Minute) }

	if err := SyncVMCacheFromHost(); err != nil {
		t.Fatalf("第二次同步失败: %v", err)
	}

	var vmA model.VMCache
	if err := db.Where("name = ?", "vm-a").First(&vmA).Error; err != nil {
		t.Fatalf("读取 vm-a 缓存失败: %v", err)
	}
	if vmA.VCPU != 8 || vmA.MemoryMB != 8192 {
		t.Fatalf("vm-a 缓存未正确更新: %+v", vmA)
	}

	var vmB model.VMCache
	if err := db.Where("name = ?", "vm-b").First(&vmB).Error; err != nil {
		t.Fatalf("读取 vm-b 缓存失败: %v", err)
	}
	if vmB.Present {
		t.Fatal("宿主机已不存在的 vm-b 应被标记为失效")
	}
}

func TestSyncVMCacheFromHostFailureKeepsExistingRecords(t *testing.T) {
	db := setupVMCacheServiceTestDB(t)
	existing := model.VMCache{
		Name:          "vm-old",
		OwnerUsername: "alice",
		Status:        "running",
		Present:       true,
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("创建初始缓存失败: %v", err)
	}

	vmCacheListNamesFromHost = func() ([]string, error) {
		return nil, errors.New("virsh 不可用")
	}

	if err := SyncVMCacheFromHost(); err == nil {
		t.Fatal("宿主机同步失败时应返回错误")
	}

	var record model.VMCache
	if err := db.Where("name = ?", "vm-old").First(&record).Error; err != nil {
		t.Fatalf("读取旧缓存失败: %v", err)
	}
	if !record.Present {
		t.Fatal("宿主机同步失败时不应误标记旧缓存失效")
	}
}

func TestRefreshVMCacheByNameMarksMissingWhenHostVMDisappears(t *testing.T) {
	db := setupVMCacheServiceTestDB(t)
	if err := db.Create(&model.VMCache{
		Name:          "vm-gone",
		OwnerUsername: "alice",
		Status:        "running",
		Present:       true,
	}).Error; err != nil {
		t.Fatalf("创建初始缓存失败: %v", err)
	}

	vmCacheBuildRecordFromHost = func(name string, syncedAt time.Time) (model.VMCache, error) {
		return model.VMCache{}, errVMCacheSourceMissing
	}

	if err := RefreshVMCacheByName("vm-gone"); err != nil {
		t.Fatalf("刷新缓存失败: %v", err)
	}

	var record model.VMCache
	if err := db.Where("name = ?", "vm-gone").First(&record).Error; err != nil {
		t.Fatalf("读取刷新结果失败: %v", err)
	}
	if record.Present {
		t.Fatal("宿主机 VM 消失后缓存应被标记失效")
	}
}

func TestListCachedVMsByOwnerOnlyReturnsVisibleVMs(t *testing.T) {
	db := setupVMCacheServiceTestDB(t)
	seed := []model.VMCache{
		{Name: "vm-a", OwnerUsername: "alice", Status: "running", Present: true},
		{Name: "vm-b", OwnerUsername: "alice", Status: "shut off", Present: false},
		{Name: "vm-c", OwnerUsername: "bob", Status: "running", Present: true},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("写入测试缓存失败: %v", err)
	}

	vms, err := ListCachedVMsByOwner("alice")
	if err != nil {
		t.Fatalf("读取用户缓存失败: %v", err)
	}
	if len(vms) != 1 || vms[0].Name != "vm-a" {
		t.Fatalf("归属过滤结果不正确: %+v", vms)
	}
}

func TestTriggerAdminVMCacheRefreshIfNeededHonorsCooldownAndMutex(t *testing.T) {
	setupVMCacheServiceTestDB(t)
	vmCacheRefreshCooldown = 10 * time.Second

	currentTime := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	vmCacheNow = func() time.Time { return currentTime }

	started := make(chan struct{}, 4)
	release := make(chan struct{})
	vmCacheRunFullSync = func() error {
		started <- struct{}{}
		<-release
		return nil
	}

	TriggerAdminVMCacheRefreshIfNeeded()
	TriggerAdminVMCacheRefreshIfNeeded()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("首次触发未启动后台刷新")
	}
	select {
	case <-started:
		t.Fatal("并发刷新未被互斥控制")
	case <-time.After(200 * time.Millisecond):
	}

	close(release)
	time.Sleep(200 * time.Millisecond)

	currentTime = currentTime.Add(5 * time.Second)
	TriggerAdminVMCacheRefreshIfNeeded()
	select {
	case <-started:
		t.Fatal("冷却期内不应再次触发刷新")
	case <-time.After(200 * time.Millisecond):
	}

	currentTime = currentTime.Add(6 * time.Second)
	release = make(chan struct{})
	vmCacheRunFullSync = func() error {
		started <- struct{}{}
		close(release)
		return nil
	}
	TriggerAdminVMCacheRefreshIfNeeded()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("冷却期结束后应允许再次触发刷新")
	}
}
