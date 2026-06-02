package service

import "testing"

func TestNormalizeReinstallDiskSizeGB(t *testing.T) {
	tests := []struct {
		name      string
		requested int
		current   int
		min       int
		want      int
	}{
		{name: "默认沿用当前系统盘", requested: 0, current: 80, min: 40, want: 80},
		{name: "当前系统盘小于模板最小值时自动抬升", requested: 0, current: 20, min: 40, want: 40},
		{name: "显式指定小于模板最小值时后端兜底抬升", requested: 30, current: 80, min: 40, want: 40},
		{name: "允许缩小到模板最小值以上", requested: 60, current: 120, min: 40, want: 60},
		{name: "当前系统盘未知时回退模板最小值", requested: 0, current: 0, min: 50, want: 50},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeReinstallDiskSizeGB(tc.requested, tc.current, tc.min); got != tc.want {
				t.Fatalf("NormalizeReinstallDiskSizeGB() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestIsReinstallBootFamilyCompatible(t *testing.T) {
	tests := []struct {
		name         string
		currentBoot  string
		templateBoot string
		want         bool
	}{
		{name: "bios 与 bios 兼容", currentBoot: VMBootTypeBIOS, templateBoot: VMBootTypeBIOS, want: true},
		{name: "uefi 与 secure uefi 兼容", currentBoot: VMBootTypeUEFI, templateBoot: VMBootTypeUEFISecure, want: true},
		{name: "bios 与 uefi 不兼容", currentBoot: VMBootTypeBIOS, templateBoot: VMBootTypeUEFI, want: false},
		{name: "未知引导方式默认放行", currentBoot: "", templateBoot: VMBootTypeUEFI, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsReinstallBootFamilyCompatible(tc.currentBoot, tc.templateBoot); got != tc.want {
				t.Fatalf("IsReinstallBootFamilyCompatible() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsReinstallTaskForVM(t *testing.T) {
	raw := `{"name":"demo-vm","template":"ubuntu","disk_size":80}`
	if !isReinstallTaskForVM(raw, "demo-vm") {
		t.Fatalf("期望识别为同一台虚拟机的重装任务")
	}
	if isReinstallTaskForVM(raw, "other-vm") {
		t.Fatalf("不应把其他虚拟机识别为当前重装任务")
	}
	if isReinstallTaskForVM(`{"invalid":`, "demo-vm") {
		t.Fatalf("非法 JSON 不应命中重装任务")
	}
}
