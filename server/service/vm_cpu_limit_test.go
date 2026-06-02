package service

import (
	"strings"
	"testing"
)

func TestParseVMCPULimitPercentFromDomainXML(t *testing.T) {
	xml := `<domain type='kvm'>
  <name>demo</name>
  <vcpu>4</vcpu>
  <cputune>
    <period>100000</period>
    <quota>200000</quota>
  </cputune>
</domain>`

	got := ParseVMCPULimitPercentFromDomainXML(xml, 4)
	if got != 50 {
		t.Fatalf("应解析为 50%%，实际为 %d%%", got)
	}
}

func TestParseVMCPULimitPercentUnlimitedFromDomainXML(t *testing.T) {
	xml := `<domain type='kvm'>
  <name>demo</name>
  <vcpu>2</vcpu>
  <cputune>
    <period>100000</period>
    <quota>-1</quota>
  </cputune>
</domain>`

	got := ParseVMCPULimitPercentFromDomainXML(xml, 2)
	if got != VMCPULimitUnlimited {
		t.Fatalf("无限制应解析为 0，实际为 %d", got)
	}
}

func TestApplyVMCPULimitToDomainXMLAddsTuneBlock(t *testing.T) {
	xml := `<domain type='kvm'>
  <name>demo</name>
  <vcpu>2</vcpu>
  <devices></devices>
</domain>`

	updated := ApplyVMCPULimitToDomainXML(xml, 2, 50)
	if !strings.Contains(updated, "<cputune>") {
		t.Fatalf("应写入 cputune 块，实际 XML: %s", updated)
	}
	if !strings.Contains(updated, "<quota>100000</quota>") {
		t.Fatalf("应按 2 vCPU * 50%% 写入 quota=100000，实际 XML: %s", updated)
	}
}

func TestApplyVMCPULimitToDomainXMLRemovesOnlyPeriodAndQuota(t *testing.T) {
	xml := `<domain type='kvm'>
  <name>demo</name>
  <vcpu>2</vcpu>
  <cputune>
    <shares>2048</shares>
    <period>100000</period>
    <quota>100000</quota>
  </cputune>
</domain>`

	updated := ApplyVMCPULimitToDomainXML(xml, 2, VMCPULimitUnlimited)
	if strings.Contains(updated, "<period>") || strings.Contains(updated, "<quota>") {
		t.Fatalf("取消限制后不应保留 period/quota，实际 XML: %s", updated)
	}
	if !strings.Contains(updated, "<shares>2048</shares>") {
		t.Fatalf("取消限制后应保留其他 cputune 配置，实际 XML: %s", updated)
	}
}

func TestApplyVMCPULimitToDomainXMLKeepsPercentWhenVCPUChanges(t *testing.T) {
	xml := `<domain type='kvm'>
  <name>demo</name>
  <vcpu>4</vcpu>
  <cputune>
    <period>100000</period>
    <quota>100000</quota>
  </cputune>
</domain>`

	updated := ApplyVMCPULimitToDomainXML(xml, 4, 25)
	if !strings.Contains(updated, "<quota>100000</quota>") {
		t.Fatalf("4 vCPU * 25%% 应保持 quota=100000，实际 XML: %s", updated)
	}

	updated = ApplyVMCPULimitToDomainXML(updated, 8, 25)
	if !strings.Contains(updated, "<quota>200000</quota>") {
		t.Fatalf("8 vCPU * 25%% 应调整 quota=200000，实际 XML: %s", updated)
	}
}
