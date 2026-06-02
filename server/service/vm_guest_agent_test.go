package service

import (
	"strings"
	"testing"
)

func TestParseVMGuestAgentConfigFromDomainXML(t *testing.T) {
	xmlContent := `<domain type='kvm'>
  <devices>
    <channel type='unix'>
      <source mode='bind'/>
      <target type='virtio' name='org.qemu.guest_agent.0'/>
    </channel>
  </devices>
</domain>`

	cfg := ParseVMGuestAgentConfigFromDomainXML(xmlContent)
	if cfg == nil || !cfg.Enabled {
		t.Fatalf("期望解析到已启用的 QEMU Guest Agent，实际: %#v", cfg)
	}
}

func TestApplyVMGuestAgentConfigToDomainXMLEnable(t *testing.T) {
	xmlContent := `<domain type='kvm'>
  <devices>
    <memballoon model='virtio'/>
  </devices>
</domain>`

	updated, err := ApplyVMGuestAgentConfigToDomainXML(xmlContent, &VMGuestAgentConfig{Enabled: true})
	if err != nil {
		t.Fatalf("ApplyVMGuestAgentConfigToDomainXML returned error: %v", err)
	}

	expected := []string{
		"<channel type='unix'>",
		"<target type='virtio' name='org.qemu.guest_agent.0'/>",
	}
	for _, fragment := range expected {
		if !strings.Contains(updated, fragment) {
			t.Fatalf("更新后的 XML 缺少片段 %q:\n%s", fragment, updated)
		}
	}
}

func TestApplyVMGuestAgentConfigToDomainXMLDisable(t *testing.T) {
	xmlContent := `<domain type='kvm'>
  <devices>
    <channel type='unix'>
      <source mode='bind'/>
      <target type='virtio' name='org.qemu.guest_agent.0'/>
    </channel>
    <memballoon model='virtio'/>
  </devices>
</domain>`

	updated, err := ApplyVMGuestAgentConfigToDomainXML(xmlContent, &VMGuestAgentConfig{Enabled: false})
	if err != nil {
		t.Fatalf("ApplyVMGuestAgentConfigToDomainXML returned error: %v", err)
	}

	if strings.Contains(updated, "org.qemu.guest_agent.0") {
		t.Fatalf("禁用后仍然存在 guest agent 通道:\n%s", updated)
	}
}
