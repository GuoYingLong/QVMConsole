package pool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kvm_console/model"
	"kvm_console/service/snapshot"
	"kvm_console/utils"
)

// FormatAndMountStoragePool 格式化指定块设备并挂载为虚拟机存储位置。
func FormatAndMountStoragePool(ctx context.Context, id string, progress func(int, string)) error {
	pool, err := GetStoragePool(id)
	if err != nil {
		return err
	}
	if err := validateFormatTarget(*pool); err != nil {
		return err
	}
	mountPath := defaultStorageMountPath(id)
	devicePath := pool.DevicePath

	progress(10, "正在清理旧文件系统标记...")
	utils.ExecCommandContextWithTimeout(ctx, "wipefs", 2*time.Minute, "-a", devicePath)
	if ctx.Err() != nil {
		return ctx.Err()
	}

	progress(30, "正在格式化为 ext4...")
	mkfs := utils.ExecCommandContextWithTimeout(ctx, "mkfs.ext4", 10*time.Minute, "-F", devicePath)
	if mkfs.Error != nil {
		return fmt.Errorf("格式化硬盘失败: %s", mkfs.Stderr)
	}

	progress(55, "正在读取文件系统 UUID...")
	blkid := utils.ExecCommandContextWithTimeout(ctx, "blkid", 30*time.Second, "-s", "UUID", "-o", "value", devicePath)
	if blkid.Error != nil || strings.TrimSpace(blkid.Stdout) == "" {
		return fmt.Errorf("读取新文件系统 UUID 失败: %s", blkid.Stderr)
	}
	uuid := strings.TrimSpace(blkid.Stdout)

	progress(65, "正在写入开机自动挂载配置...")
	if err := os.MkdirAll(mountPath, 0755); err != nil {
		return fmt.Errorf("创建挂载目录失败: %w", err)
	}
	if err := ensureFstabEntry(uuid, mountPath); err != nil {
		return err
	}

	progress(75, "正在挂载硬盘...")
	mount := utils.ExecCommandContextWithTimeout(ctx, "mount", 2*time.Minute, mountPath)
	if mount.Error != nil {
		return fmt.Errorf("挂载硬盘失败: %s", mount.Stderr)
	}

	progress(85, "正在创建虚拟机磁盘目录...")
	vmDir := filepath.Join(mountPath, "vm-disks")
	if err := ensureVMStorageDir(vmDir); err != nil {
		return err
	}

	progress(92, "正在保存存储池配置...")
	displayName := pool.DisplayName
	if strings.TrimSpace(displayName) == "" {
		displayName = defaultStorageDisplayName(*pool)
	}
	cfg := model.HostStoragePool{DeviceID: id}
	if err := model.DB.Where("device_id = ?", id).Assign(map[string]interface{}{
		"display_name": displayName,
		"enabled":      true,
		"mount_path":   mountPath,
	}).FirstOrCreate(&cfg).Error; err != nil {
		return fmt.Errorf("保存存储池配置失败: %w", err)
	}

	progress(100, "硬盘已格式化并挂载")
	return nil
}

func ensureVMStorageDir(dir string) error {
	if strings.TrimSpace(dir) == "" {
		return fmt.Errorf("虚拟机磁盘目录为空")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建虚拟机磁盘目录失败: %w", err)
	}
	if err := snapshot.EnsureLibvirtStorageAppArmorAccessForPaths([]string{dir}); err != nil {
		return fmt.Errorf("配置 libvirt 自定义存储访问规则失败: %w", err)
	}
	utils.ExecCommand("chown", "libvirt-qemu:kvm", dir)
	return nil
}

func ensureFstabEntry(uuid, mountPath string) error {
	data, err := os.ReadFile("/etc/fstab")
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("读取 /etc/fstab 失败: %w", err)
	}
	line := fmt.Sprintf("UUID=%s %s ext4 defaults,nofail 0 2", uuid, mountPath)
	var lines []string
	found := false
	for _, existing := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(existing)
		if trimmed == "" {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) >= 2 && (fields[0] == "UUID="+uuid || fields[1] == mountPath) {
			lines = append(lines, line)
			found = true
			continue
		}
		lines = append(lines, existing)
	}
	if !found {
		lines = append(lines, line)
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile("/etc/fstab", []byte(content), 0644); err != nil {
		return fmt.Errorf("写入 /etc/fstab 失败: %w", err)
	}
	return nil
}
