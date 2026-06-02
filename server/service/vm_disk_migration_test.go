package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildUniqueDiskMigrationTargetPathUsesOriginalNameWhenFree(t *testing.T) {
	dir := t.TempDir()
	path, err := buildUniqueDiskMigrationTargetPath(dir, "/var/lib/libvirt/images/demo-vda.qcow2", time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("生成目标路径失败: %v", err)
	}
	want := filepath.Join(dir, "demo-vda.qcow2")
	if path != want {
		t.Fatalf("目标路径不符合预期: got %s want %s", path, want)
	}
}

func TestBuildUniqueDiskMigrationTargetPathAddsSuffixWhenConflict(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "demo-vda.qcow2")
	if err := os.WriteFile(existing, []byte("exists"), 0600); err != nil {
		t.Fatalf("创建冲突文件失败: %v", err)
	}
	now := time.Date(2026, 5, 1, 12, 3, 4, 0, time.UTC)
	path, err := buildUniqueDiskMigrationTargetPath(dir, "/var/lib/libvirt/images/demo-vda.qcow2", now)
	if err != nil {
		t.Fatalf("生成冲突目标路径失败: %v", err)
	}
	want := filepath.Join(dir, "demo-vda_migrated_20260501120304.qcow2")
	if path != want {
		t.Fatalf("冲突目标路径不符合预期: got %s want %s", path, want)
	}
}

func TestSameCleanPathNormalizesPath(t *testing.T) {
	if !sameCleanPath("/mnt/data/vm-disks/../vm-disks/demo.qcow2", "/mnt/data/vm-disks/demo.qcow2") {
		t.Fatalf("应当识别为同一路径")
	}
}
