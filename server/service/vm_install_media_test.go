package service

import (
	"strings"
	"testing"
)

// 测试 ISO 列表规范化，保证兼容旧字段并去重。
func TestNormalizeInstallISOSelection(t *testing.T) {
	primary, extra := NormalizeInstallISOSelection(
		" /iso/setup.iso ",
		[]string{"", "/iso/setup.iso", "/iso/virtio.iso", " /iso/tools.iso "},
	)

	if primary != "/iso/setup.iso" {
		t.Fatalf("primary=%q, want %q", primary, "/iso/setup.iso")
	}
	if len(extra) != 2 {
		t.Fatalf("len(extra)=%d, want 2", len(extra))
	}
	if extra[0] != "/iso/virtio.iso" || extra[1] != "/iso/tools.iso" {
		t.Fatalf("extra=%v, want [/iso/virtio.iso /iso/tools.iso]", extra)
	}
}

// 测试附加 ISO 时会沿用主光驱总线并分配新的设备名。
func TestApplyAdditionalCDROMsToDomainXML(t *testing.T) {
	xmlContent := `<domain type='kvm'>
  <name>demo</name>
  <devices>
    <disk type='file' device='disk'>
      <source file='/var/lib/libvirt/images/demo.qcow2'/>
      <target dev='vda' bus='virtio'/>
    </disk>
    <disk type='file' device='cdrom'>
      <source file='/iso/setup.iso'/>
      <target dev='sda' bus='sata'/>
      <readonly/>
    </disk>
  </devices>
</domain>`

	updated, err := ApplyAdditionalCDROMsToDomainXML(xmlContent, []string{"/iso/virtio.iso", "/iso/tools.iso"})
	if err != nil {
		t.Fatalf("ApplyAdditionalCDROMsToDomainXML returned error: %v", err)
	}

	cdromCount := strings.Count(updated, "device='cdrom'") + strings.Count(updated, `device="cdrom"`)
	if cdromCount != 3 {
		t.Fatalf("updated cdrom count mismatch: %s", updated)
	}
	if !strings.Contains(updated, `source file="/iso/virtio.iso"`) {
		t.Fatalf("missing virtio iso xml: %s", updated)
	}
	if !strings.Contains(updated, `target dev="sdb" bus="sata"`) {
		t.Fatalf("missing sdb target: %s", updated)
	}
	if !strings.Contains(updated, `target dev="sdc" bus="sata"`) {
		t.Fatalf("missing sdc target: %s", updated)
	}
}
