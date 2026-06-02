package service

import (
	"strings"
	"testing"
)

func TestApplyWindowsCPUTopologyToDomainXMLExpandsSelfClosingCPU(t *testing.T) {
	xmlContent := `<domain type='kvm'>
  <vcpu>6</vcpu>
  <cpu mode='host-passthrough' check='none' migratable='on'/>
</domain>`

	updated := ApplyWindowsCPUTopologyToDomainXML(xmlContent, 6)
	if !strings.Contains(updated, "<topology sockets='1' dies='1' cores='6' threads='1'/>") {
		t.Fatalf("未写入 Windows CPU 拓扑:\n%s", updated)
	}
	if strings.Contains(updated, "<cpu mode='host-passthrough' check='none' migratable='on'/>") {
		t.Fatalf("自闭合 CPU 节点未展开:\n%s", updated)
	}
}

func TestApplyWindowsCPUTopologyToDomainXMLReplacesExistingTopology(t *testing.T) {
	xmlContent := `<domain type='kvm'>
  <vcpu>6</vcpu>
  <cpu mode='host-passthrough'>
    <topology sockets='6' cores='1' threads='1'/>
  </cpu>
</domain>`

	updated := ApplyWindowsCPUTopologyToDomainXML(xmlContent, 6)
	if strings.Contains(updated, "sockets='6'") {
		t.Fatalf("旧 CPU 拓扑未被替换:\n%s", updated)
	}
	if !strings.Contains(updated, "<topology sockets='1' dies='1' cores='6' threads='1'/>") {
		t.Fatalf("未写入单插槽多核心拓扑:\n%s", updated)
	}
}

func TestApplyWindowsCPUTopologyToDomainXMLUsesVCPUFromXML(t *testing.T) {
	xmlContent := `<domain type='kvm'>
  <vcpu placement='static'>4</vcpu>
  <features><acpi/></features>
  <devices></devices>
</domain>`

	updated := ApplyWindowsCPUTopologyToDomainXML(xmlContent, 0)
	if !strings.Contains(updated, "<topology sockets='1' dies='1' cores='4' threads='1'/>") {
		t.Fatalf("未从 vcpu 节点推导 CPU 拓扑:\n%s", updated)
	}
}

func TestApplyCPUTopologyModeToDomainXMLHostDefaultRemovesTopology(t *testing.T) {
	xmlContent := `<domain type='kvm'>
  <vcpu>6</vcpu>
  <cpu mode='host-passthrough'>
    <topology sockets='1' dies='1' cores='6' threads='1'/>
  </cpu>
</domain>`

	updated := ApplyCPUTopologyModeToDomainXML(xmlContent, "host_default", "windows", 6)
	if strings.Contains(updated, "<topology") {
		t.Fatalf("host_default 应移除显式 CPU 拓扑:\n%s", updated)
	}
}

func TestApplyCPUTopologyModeToDomainXMLAutoUsesSingleSocketForWindows(t *testing.T) {
	xmlContent := `<domain type='kvm'>
  <vcpu>6</vcpu>
  <cpu mode='host-passthrough' check='none' migratable='on'/>
</domain>`

	updated := ApplyCPUTopologyModeToDomainXML(xmlContent, "auto", "windows", 6)
	if !strings.Contains(updated, "<topology sockets='1' dies='1' cores='6' threads='1'/>") {
		t.Fatalf("auto 模式未为 Windows 写入单插槽拓扑:\n%s", updated)
	}
}
