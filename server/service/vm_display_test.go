package service

import (
	"strings"
	"testing"
)

func TestResolveVMVideoModel(t *testing.T) {
	tests := []struct {
		name       string
		videoModel string
		osType     string
		expected   string
	}{
		{name: "保留有效值", videoModel: "vmvga", osType: "windows", expected: "vmvga"},
		{name: "Windows 默认 VGA", videoModel: "", osType: "windows", expected: "vga"},
		{name: "Linux 默认 VirtIO", videoModel: "", osType: "linux", expected: "virtio"},
		{name: "非法值回退默认", videoModel: "unknown", osType: "windows", expected: "vga"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveVMVideoModel(tt.videoModel, tt.osType); got != tt.expected {
				t.Fatalf("期望 %s，实际 %s", tt.expected, got)
			}
		})
	}
}

func TestParseVMVideoModelFromDomainXML(t *testing.T) {
	xmlSamples := []string{
		`<domain><devices><video><model type='vmvga'/></video></devices></domain>`,
		`<domain><devices><video><model type="virtio" heads="1"/></video></devices></domain>`,
	}

	expected := []string{"vmvga", "virtio"}
	for i, sample := range xmlSamples {
		if got := ParseVMVideoModelFromDomainXML(sample); got != expected[i] {
			t.Fatalf("样例 %d 期望 %s，实际 %s", i, expected[i], got)
		}
	}
}

func TestApplyVMVideoModelToDomainXML(t *testing.T) {
	xmlContent := `<domain type='kvm'>
  <devices>
    <video>
      <model type='virtio' heads='1' primary='yes'/>
    </video>
  </devices>
</domain>`

	updated := ApplyVMVideoModelToDomainXML(xmlContent, "vga", "windows")
	if !strings.Contains(updated, "<model type='vga'/>") {
		t.Fatalf("更新后未写入 VGA 模型:\n%s", updated)
	}
	if strings.Contains(updated, "virtio") {
		t.Fatalf("更新后仍保留旧的 virtio 模型:\n%s", updated)
	}
}

func TestApplyVMVideoModelToDomainXMLInsertDefault(t *testing.T) {
	xmlContent := `<domain type='kvm'>
  <devices>
    <graphics type='vnc'/>
  </devices>
</domain>`

	updated := ApplyVMVideoModelToDomainXML(xmlContent, "", "windows")
	if !strings.Contains(updated, "<video>") || !strings.Contains(updated, "<model type='vga'/>") {
		t.Fatalf("缺少默认 VGA 视频节点:\n%s", updated)
	}
}

func TestApplyWindowsGuestOptimizationsToDomainXML(t *testing.T) {
	xmlContent := `<domain type='kvm'>
  <features>
    <acpi/>
    <hyperv mode='custom'>
      <relaxed state='on'/>
    </hyperv>
  </features>
</domain>`

	updated := ApplyWindowsGuestOptimizationsToDomainXML(xmlContent)
	expectedFragments := []string{
		"<vpindex state='on'/>",
		"<synic state='on'/>",
		"<stimer state='on'>",
		"<timer name='hypervclock' present='yes'/>",
		"<ipi state='on'/>",
	}
	for _, fragment := range expectedFragments {
		if !strings.Contains(updated, fragment) {
			t.Fatalf("更新后的 Hyper-V 配置缺少片段 %q:\n%s", fragment, updated)
		}
	}
}

func TestApplyWindowsGuestOptimizationsExpandsSelfClosingClock(t *testing.T) {
	xmlContent := `<domain type='kvm'>
  <features>
    <acpi/>
  </features>
  <clock offset='localtime'/>
  <devices/>
</domain>`

	updated := ApplyWindowsGuestOptimizationsToDomainXML(xmlContent)
	if strings.Contains(updated, "<clock offset='localtime'/>") {
		t.Fatalf("自闭合 clock 未展开:\n%s", updated)
	}
	if !strings.Contains(updated, "<timer name='hypervclock' present='yes'/>") {
		t.Fatalf("缺少 hypervclock timer:\n%s", updated)
	}
}

func TestApplyWindowsGuestOptimizationsDoesNotDuplicateHyperVClock(t *testing.T) {
	xmlContent := `<domain type='kvm'>
  <features>
    <acpi/>
  </features>
  <clock offset='localtime'>
    <timer name='hypervclock' present='yes'/>
  </clock>
  <devices/>
</domain>`

	updated := ApplyWindowsGuestOptimizationsToDomainXML(xmlContent)
	if count := strings.Count(updated, "name='hypervclock'"); count != 1 {
		t.Fatalf("hypervclock timer 不应重复，实际数量 %d:\n%s", count, updated)
	}
}
