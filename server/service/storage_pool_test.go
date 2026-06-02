package service

import (
	"encoding/json"
	"testing"

	"kvm_console/config"
	"kvm_console/model"
)

func TestFlexibleMountpointsUnmarshal(t *testing.T) {
	var out struct {
		Mountpoints flexibleMountpoints `json:"mountpoints"`
	}
	if err := json.Unmarshal([]byte(`{"mountpoints":["/data"]}`), &out); err != nil {
		t.Fatalf("解析数组挂载点失败: %v", err)
	}
	if len(out.Mountpoints) != 1 || out.Mountpoints[0] != "/data" {
		t.Fatalf("数组挂载点结果不正确: %#v", out.Mountpoints)
	}
	if err := json.Unmarshal([]byte(`{"mountpoints":"/data2"}`), &out); err != nil {
		t.Fatalf("解析字符串挂载点失败: %v", err)
	}
	if len(out.Mountpoints) != 1 || out.Mountpoints[0] != "/data2" {
		t.Fatalf("字符串挂载点结果不正确: %#v", out.Mountpoints)
	}
}

func TestBuildStoragePoolTreeMergesConfigAndDetectsSafety(t *testing.T) {
	devices := []lsblkDevice{
		{
			Name: "sda", KName: "sda", Path: "/dev/sda", Type: "disk", Size: 1000,
			Children: []lsblkDevice{
				{Name: "sda1", KName: "sda1", Path: "/dev/sda1", Type: "part", Size: 1000, FSType: "ext4", Mountpoints: flexibleMountpoints{"/"}},
			},
		},
		{Name: "sdb", KName: "sdb", Path: "/dev/sdb", Type: "disk", Size: 2000},
		{Name: "sdc", KName: "sdc", Path: "/dev/sdc", Type: "disk", Size: 3000, FSType: "ext4", Mountpoints: flexibleMountpoints{"/mnt/data"}},
		{Name: "sr0", KName: "sr0", Path: "/dev/sr0", Type: "rom", Size: 10, Readonly: true, Removable: true},
	}
	configs := map[string]model.HostStoragePool{
		"sdc": {
			DeviceID:    "sdc",
			DisplayName: "数据盘",
			Enabled:     true,
			IsDefault:   true,
			MountPath:   "/mnt/data",
		},
	}
	dfUsage := map[string]mountUsage{
		"/mnt/data": {Target: "/mnt/data", Size: 3000, Used: 1000, Available: 2000},
		"/":         {Target: "/", Size: 1000, Used: 500, Available: 500},
	}

	pools := buildStoragePoolTree(devices, nil, dfUsage, nil, configs)
	rootDisk := findStoragePoolByID(pools, "sda")
	if rootDisk == nil || !rootDisk.SystemDisk || rootDisk.CanFormat {
		t.Fatalf("系统盘安全状态不正确: %#v", rootDisk)
	}
	emptyDisk := findStoragePoolByID(pools, "sdb")
	if emptyDisk == nil || !emptyDisk.CanFormat || emptyDisk.CanUseForVM {
		t.Fatalf("空盘状态不正确: %#v", emptyDisk)
	}
	dataDisk := findStoragePoolByID(pools, "sdc")
	if dataDisk == nil {
		t.Fatal("未找到数据盘")
	}
	if !dataDisk.Enabled || !dataDisk.IsDefault || !dataDisk.CanUseForVM || dataDisk.DisplayName != "数据盘" {
		t.Fatalf("数据盘配置合并不正确: %#v", dataDisk)
	}
	if dataDisk.Available != 2000 || dataDisk.UsePercent != 33 {
		t.Fatalf("数据盘容量统计不正确: %#v", dataDisk)
	}
	rom := findStoragePoolByID(pools, "sr0")
	if rom == nil || rom.CanFormat || rom.CanUseForVM {
		t.Fatalf("光驱状态不正确: %#v", rom)
	}
}

func TestNormalizeStorageDeviceID(t *testing.T) {
	cases := map[string]string{
		"/dev/sdb":                             "sdb",
		"/dev/disk/by-id/ata-QEMU_HARDDISK_01": "ata-QEMU_HARDDISK_01",
		"uuid with space":                      "uuid-with-space",
	}
	for input, want := range cases {
		if got := normalizeStorageDeviceID(input); got != want {
			t.Fatalf("normalizeStorageDeviceID(%q)=%q，期望 %q", input, got, want)
		}
	}
}

func TestConfiguredISODir(t *testing.T) {
	previous := config.GlobalConfig
	t.Cleanup(func() {
		config.GlobalConfig = previous
	})

	config.GlobalConfig = nil
	if got := configuredISODir(); got != config.DefaultISODir {
		t.Fatalf("configuredISODir()=%q，期望默认值 %q", got, config.DefaultISODir)
	}

	config.GlobalConfig = &config.Config{ISODir: "  /data/iso  "}
	if got := configuredISODir(); got != "/data/iso" {
		t.Fatalf("configuredISODir()=%q，期望 /data/iso", got)
	}

	config.GlobalConfig = &config.Config{ISODir: "   "}
	if got := configuredISODir(); got != config.DefaultISODir {
		t.Fatalf("空白 ISO 目录应回退默认值，得到 %q", got)
	}
}
