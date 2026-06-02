package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withTempZRAMPaths(t *testing.T) {
	t.Helper()
	oldConfigPath := hostZRAMConfigPath
	oldUnitPath := hostZRAMUnitPath
	oldPanelUnitPaths := hostZRAMPanelUnitPaths
	oldExecutableFallbacks := hostZRAMExecutableFallbacks
	hostZRAMConfigPath = filepath.Join(t.TempDir(), "zram.env")
	hostZRAMUnitPath = filepath.Join(t.TempDir(), "kvm-console-zram.service")
	t.Cleanup(func() {
		hostZRAMConfigPath = oldConfigPath
		hostZRAMUnitPath = oldUnitPath
		hostZRAMPanelUnitPaths = oldPanelUnitPaths
		hostZRAMExecutableFallbacks = oldExecutableFallbacks
	})
}

func TestBuildHostZRAMEnvWritesProfileValues(t *testing.T) {
	profile, ok := findHostZRAMProfile("balanced")
	if !ok {
		t.Fatal("未找到 balanced 挡位")
	}

	env := buildHostZRAMEnv(profile)
	for _, want := range []string{
		"ZRAM_PROFILE=balanced",
		"ZRAM_SIZE_PERCENT=20",
		"ZRAM_MAX_SIZE_MB=32768",
		"ZRAM_ALGORITHM=lz4",
		"ZRAM_PRIORITY=80",
		"ZRAM_LABEL=kvm-zram",
	} {
		if !strings.Contains(env, want) {
			t.Fatalf("zRAM 配置缺少 %s:\n%s", want, env)
		}
	}
}

func TestReadHostZRAMPersistentConfig(t *testing.T) {
	withTempZRAMPaths(t)
	profile, _ := findHostZRAMProfile("aggressive")
	if err := os.MkdirAll(filepath.Dir(hostZRAMConfigPath), 0755); err != nil {
		t.Fatalf("创建测试 zRAM 配置目录失败: %v", err)
	}
	if err := os.WriteFile(hostZRAMConfigPath, []byte(buildHostZRAMEnv(profile)), 0644); err != nil {
		t.Fatalf("写入测试 zRAM 配置失败: %v", err)
	}

	profileKey, config, configured := readHostZRAMPersistentConfig()
	if !configured || profileKey != "aggressive" || config == nil {
		t.Fatalf("持久配置读取错误: profile=%s configured=%v config=%+v", profileKey, configured, config)
	}
	if config.SizePercent != 35 || config.MaxSizeMB != 65536 || config.Priority != 100 {
		t.Fatalf("持久配置参数读取错误: %+v", config)
	}
}

func TestDetectHostZRAMProfileOff(t *testing.T) {
	if got := detectHostZRAMProfile(HostZRAMRuntimeConfig{}); got != "off" {
		t.Fatalf("空运行配置应识别为 off，实际为 %s", got)
	}
}

func TestBuildHostZRAMUnitPrefersPanelServiceExecutable(t *testing.T) {
	withTempZRAMPaths(t)
	t.Setenv("KVM_CONSOLE_BINARY", "")
	tempDir := t.TempDir()
	execPath := filepath.Join(tempDir, "kvm-console")
	if err := os.WriteFile(execPath, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("写入测试可执行文件失败: %v", err)
	}
	unitPath := filepath.Join(tempDir, "kvm-console.service")
	if err := os.WriteFile(unitPath, []byte("[Service]\nExecStart="+execPath+"\n"), 0644); err != nil {
		t.Fatalf("写入测试 unit 失败: %v", err)
	}
	hostZRAMPanelUnitPaths = []string{unitPath}
	hostZRAMExecutableFallbacks = []string{filepath.Join(tempDir, "fallback")}

	unit := buildHostZRAMUnit()
	if !strings.Contains(unit, "ExecStart="+execPath+" host-zram-apply") {
		t.Fatalf("zRAM unit 应使用面板主服务路径:\n%s", unit)
	}
}

func TestParseSystemdExecStartPath(t *testing.T) {
	content := `[Unit]
Description=test

[Service]
ExecStart=/opt/kvm-console/kvm-console --flag
`
	if got := parseSystemdExecStartPath(content); got != "/opt/kvm-console/kvm-console" {
		t.Fatalf("解析 ExecStart 错误: %s", got)
	}
}
