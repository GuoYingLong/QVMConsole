package model

import (
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestVMCacheMigrationCreatesTableAndDefaults(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "vm-cache-model.db")), &gorm.Config{})
	if err != nil {
		t.Skipf("当前环境无法打开 SQLite 测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&VMCache{}); err != nil {
		t.Fatalf("迁移 VMCache 失败: %v", err)
	}
	if !db.Migrator().HasTable(&VMCache{}) {
		t.Fatal("VMCache 表未创建")
	}
	if !db.Migrator().HasIndex(&VMCache{}, "idx_vm_caches_owner_username") {
		t.Fatal("VMCache owner_username 索引未创建")
	}

	if err := db.Exec("INSERT INTO vm_caches (name, owner_username) VALUES (?, ?)", "vm-1", "alice").Error; err != nil {
		t.Fatalf("插入测试缓存失败: %v", err)
	}

	var record VMCache
	if err := db.Where("name = ?", "vm-1").First(&record).Error; err != nil {
		t.Fatalf("读取测试缓存失败: %v", err)
	}
	if !record.Present {
		t.Fatal("VMCache.present 默认值应为 true")
	}

	if err := db.Create(&VMCache{Name: "vm-1", OwnerUsername: "bob", Present: true}).Error; err == nil {
		t.Fatal("VMCache.name 应保持唯一")
	}
}
