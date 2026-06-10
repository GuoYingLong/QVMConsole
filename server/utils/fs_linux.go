//go:build linux

package utils

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// ReadMemInfo 解析 /proc/meminfo，返回以 kB 为单位的 map
// key 为字段名（如 "MemTotal", "MemAvailable", "SwapTotal", "SwapFree"）
func ReadMemInfo() (map[string]int64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, fmt.Errorf("打开 /proc/meminfo 失败: %w", err)
	}
	defer f.Close()

	result := make(map[string]int64)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// 格式: "Key:    Value kB" 或 "Key:    Value"
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:colonIdx])
		valuePart := strings.TrimSpace(line[colonIdx+1:])

		// 去掉可能的 "kB" 后缀
		valuePart = strings.TrimSuffix(valuePart, " kB")
		valuePart = strings.TrimSpace(valuePart)

		val, err := strconv.ParseInt(valuePart, 10, 64)
		if err != nil {
			continue
		}
		result[key] = val
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 /proc/meminfo 失败: %w", err)
	}

	return result, nil
}

// GetDiskSpace 使用 syscall.Statfs 获取指定路径的磁盘空间信息
// 返回值均为 kB 单位
func GetDiskSpace(path string) (total, used, available int64, err error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, 0, fmt.Errorf("获取磁盘空间失败: %w", err)
	}

	// Bsize 可能很大，先转 int64 再运算避免溢出
	bsize := int64(stat.Bsize)
	total = (int64(stat.Blocks) * bsize) / 1024
	available = (int64(stat.Bavail) * bsize) / 1024
	used = total - available

	return total, used, available, nil
}
