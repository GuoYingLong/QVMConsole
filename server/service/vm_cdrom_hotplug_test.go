package service

import "testing"

// 测试运行中新增光驱时会强制选择支持热插的 scsi 总线。
func TestSelectNewCDROMBusRunningUsesSCSI(t *testing.T) {
	bus := selectNewCDROMBus("running", []DiskInfo{
		{Device: "sda", DeviceType: "cdrom", Bus: "sata"},
	})
	if bus != "scsi" {
		t.Fatalf("bus=%q, want %q", bus, "scsi")
	}
}

// 测试关机新增光驱时优先沿用现有光驱总线。
func TestSelectNewCDROMBusStoppedReuseExistingCDROMBus(t *testing.T) {
	bus := selectNewCDROMBus("shut off", []DiskInfo{
		{Device: "vda", DeviceType: "disk", Bus: "virtio"},
		{Device: "hda", DeviceType: "cdrom", Bus: "ide"},
	})
	if bus != "ide" {
		t.Fatalf("bus=%q, want %q", bus, "ide")
	}
}

// 测试未找到历史光驱时默认走 sata，保持原有离线添加行为。
func TestSelectNewCDROMBusStoppedDefaultsToSATA(t *testing.T) {
	bus := selectNewCDROMBus("shut off", []DiskInfo{
		{Device: "vda", DeviceType: "disk", Bus: "virtio"},
	})
	if bus != "sata" {
		t.Fatalf("bus=%q, want %q", bus, "sata")
	}
}

// 测试 SCSI 控制器检测兼容单双引号 XML。
func TestHasSCSIController(t *testing.T) {
	cases := []struct {
		name string
		xml  string
		want bool
	}{
		{name: "single quote", xml: "<controller type='scsi' index='0'/>", want: true},
		{name: "double quote", xml: `<controller type="scsi" index="0"/>`, want: true},
		{name: "missing", xml: "<controller type='sata' index='0'/>", want: false},
	}

	for _, tc := range cases {
		if got := hasSCSIController(tc.xml); got != tc.want {
			t.Fatalf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

// 测试能从 info pci 输出中找出空闲的 pcie-root-port 下游总线。
func TestParseFreePCIERootPortBuses(t *testing.T) {
	infoPCI := `  Bus  0, device   2, function 0:
    PCI bridge: PCI device 1b36:000c
      secondary bus 1.
  Bus  0, device   2, function 1:
    PCI bridge: PCI device 1b36:000c
      secondary bus 2.
  Bus  2, device   0, function 0:
    USB controller: PCI device 1b36:000d
  Bus  0, device   2, function 2:
    PCI bridge: PCI device 1b36:000c
      secondary bus 6.`

	buses := parseFreePCIERootPortBuses(infoPCI)
	if len(buses) != 2 {
		t.Fatalf("len(buses)=%d, want 2", len(buses))
	}
	if buses[0] != 1 || buses[1] != 6 {
		t.Fatalf("buses=%v, want [1 6]", buses)
	}
}

// 测试当所有 root-port 下游总线都已被占用时返回空结果。
func TestParseFreePCIERootPortBusesNoFreeSlot(t *testing.T) {
	infoPCI := `  Bus  0, device   2, function 0:
    PCI bridge: PCI device 1b36:000c
      secondary bus 1.
  Bus  1, device   0, function 0:
    Ethernet controller: PCI device 1af4:1041`

	buses := parseFreePCIERootPortBuses(infoPCI)
	if len(buses) != 0 {
		t.Fatalf("buses=%v, want []", buses)
	}
}
