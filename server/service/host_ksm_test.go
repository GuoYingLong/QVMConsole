package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withTempKSMPaths(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	oldBasePath := hostKSMBasePath
	oldConfigPath := hostKSMConfigPath
	oldUnitPath := hostKSMUnitPath
	hostKSMBasePath = base
	hostKSMConfigPath = filepath.Join(t.TempDir(), "ksm.env")
	hostKSMUnitPath = filepath.Join(t.TempDir(), "kvm-console-ksm.service")
	t.Cleanup(func() {
		hostKSMBasePath = oldBasePath
		hostKSMConfigPath = oldConfigPath
		hostKSMUnitPath = oldUnitPath
	})
	return base
}

func writeKSMTestFile(t *testing.T, base, name, value string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(base, name), []byte(value), 0644); err != nil {
		t.Fatalf("写入测试 KSM 文件失败: %v", err)
	}
}

func readKSMTestFile(t *testing.T, base, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(base, name))
	if err != nil {
		t.Fatalf("读取测试 KSM 文件失败: %v", err)
	}
	return strings.TrimSpace(string(content))
}

func TestDetectHostKSMProfile(t *testing.T) {
	profile, ok := findHostKSMProfile("balanced")
	if !ok {
		t.Fatal("未找到 balanced 挡位")
	}
	config := hostKSMProfileToConfig(profile)
	if got := detectHostKSMProfile(config); got != "balanced" {
		t.Fatalf("期望识别 balanced，实际为 %s", got)
	}
}

func TestApplyHostKSMRuntimeWritesProfileValues(t *testing.T) {
	base := withTempKSMPaths(t)
	for _, name := range []string{"run", "pages_to_scan", "sleep_millisecs", "merge_across_nodes", "use_zero_pages", "smart_scan"} {
		writeKSMTestFile(t, base, name, "0")
	}

	profile, _ := findHostKSMProfile("aggressive")
	if err := applyHostKSMRuntime(profile); err != nil {
		t.Fatalf("应用 KSM 运行时配置失败: %v", err)
	}

	if got := readKSMTestFile(t, base, "run"); got != "1" {
		t.Fatalf("run = %s, want 1", got)
	}
	if got := readKSMTestFile(t, base, "pages_to_scan"); got != "2000" {
		t.Fatalf("pages_to_scan = %s, want 2000", got)
	}
	if got := readKSMTestFile(t, base, "sleep_millisecs"); got != "20" {
		t.Fatalf("sleep_millisecs = %s, want 20", got)
	}
	if got := readKSMTestFile(t, base, "use_zero_pages"); got != "1" {
		t.Fatalf("use_zero_pages = %s, want 1", got)
	}
}

func TestGetHostKSMStatusReadsPersistentProfileAndMetrics(t *testing.T) {
	base := withTempKSMPaths(t)
	writeKSMTestFile(t, base, "run", "1")
	writeKSMTestFile(t, base, "pages_to_scan", "500")
	writeKSMTestFile(t, base, "sleep_millisecs", "50")
	writeKSMTestFile(t, base, "merge_across_nodes", "1")
	writeKSMTestFile(t, base, "use_zero_pages", "1")
	writeKSMTestFile(t, base, "smart_scan", "1")
	writeKSMTestFile(t, base, "pages_sharing", "1234")

	profile, _ := findHostKSMProfile("balanced")
	if err := os.MkdirAll(filepath.Dir(hostKSMConfigPath), 0755); err != nil {
		t.Fatalf("创建测试配置目录失败: %v", err)
	}
	if err := os.WriteFile(hostKSMConfigPath, []byte(buildHostKSMEnv(profile)), 0644); err != nil {
		t.Fatalf("写入测试持久配置失败: %v", err)
	}

	status := GetHostKSMStatus()
	if !status.Supported || !status.Enabled {
		t.Fatalf("期望 KSM 受支持且已启用: %+v", status)
	}
	if status.CurrentProfile != "balanced" {
		t.Fatalf("current_profile = %s, want balanced", status.CurrentProfile)
	}
	if status.PersistentProfile != "balanced" || !status.PersistentConfigured {
		t.Fatalf("持久配置读取错误: %+v", status)
	}
	if status.Metrics.PagesSharing == nil || *status.Metrics.PagesSharing != 1234 {
		t.Fatalf("pages_sharing 统计读取错误: %+v", status.Metrics.PagesSharing)
	}
}
