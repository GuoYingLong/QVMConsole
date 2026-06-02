package service

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kvm_console/model"
	"kvm_console/utils"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNormalizeSecurityGroupRuleCIDRAndPortRange(t *testing.T) {
	rule, err := normalizeSecurityGroupRule(10, VPCSecurityGroupRuleRequest{
		Direction:   "Ingress",
		Protocol:    "TCP",
		PortStart:   22,
		PortEnd:     2222,
		TargetType:  "cidr",
		TargetValue: "10.0.0.1",
		Remark:      " ssh ",
	})
	if err != nil {
		t.Fatalf("normalizeSecurityGroupRule returned error: %v", err)
	}
	if rule.Direction != "ingress" || rule.Protocol != "tcp" {
		t.Fatalf("unexpected normalized direction/protocol: %+v", rule)
	}
	if rule.PortStart != 22 || rule.PortEnd != 2222 {
		t.Fatalf("unexpected port range: %+v", rule)
	}
	if rule.TargetValue != "10.0.0.1/32" {
		t.Fatalf("unexpected target value: %s", rule.TargetValue)
	}
	if rule.Remark != "ssh" {
		t.Fatalf("unexpected remark: %q", rule.Remark)
	}
}

func TestIsLibvirtDomainMissingResult(t *testing.T) {
	result := &utils.CmdResult{
		Stderr: "error: failed to get domain 'vmpzwg7rsj'",
		Error:  fmt.Errorf("命令执行失败"),
	}
	if !isLibvirtDomainMissingResult(result) {
		t.Fatal("expected libvirt missing-domain error to be detected")
	}

	transient := &utils.CmdResult{
		Stderr: "error: failed to connect to the hypervisor",
		Error:  fmt.Errorf("命令执行失败"),
	}
	if isLibvirtDomainMissingResult(transient) {
		t.Fatal("transient libvirt errors must not be treated as missing domains")
	}
	if isLibvirtDomainMissingResult(&utils.CmdResult{Stdout: "running"}) {
		t.Fatal("successful libvirt result must not be treated as missing domain")
	}
}

func TestNormalizeSecurityGroupRuleRejectsInvalidPort(t *testing.T) {
	_, err := normalizeSecurityGroupRule(10, VPCSecurityGroupRuleRequest{
		Direction:   "ingress",
		Protocol:    "tcp",
		PortStart:   70000,
		TargetType:  "cidr",
		TargetValue: "0.0.0.0/0",
	})
	if err == nil {
		t.Fatal("expected invalid port error")
	}
}

func TestNormalizeSecurityGroupRuleRejectsEmptySwitchTarget(t *testing.T) {
	_, err := normalizeSecurityGroupRule(10, VPCSecurityGroupRuleRequest{
		Direction:  "ingress",
		Protocol:   "tcp",
		PortStart:  22,
		PortEnd:    22,
		TargetType: "switch",
	})
	if err == nil {
		t.Fatal("expected empty switch target error")
	}
}

func TestDeleteAutoSecurityGroupPortForwardRulesOnlyDeletesAutomaticRule(t *testing.T) {
	oldDB := model.DB
	t.Cleanup(func() {
		model.DB = oldDB
	})
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "vpc-auto-forward-rule.db")), &gorm.Config{})
	if err != nil {
		t.Skipf("当前环境无法打开 SQLite 测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.VPCSecurityGroupRule{}); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	model.DB = db
	rules := []model.VPCSecurityGroupRule{
		{SecurityGroupID: 7, Direction: "ingress", Protocol: "tcp", PortStart: 22, PortEnd: 22, TargetType: "cidr", TargetValue: "0.0.0.0/0", Remark: autoPortForwardSecurityGroupRuleNote},
		{SecurityGroupID: 7, Direction: "ingress", Protocol: "tcp", PortStart: 22, PortEnd: 22, TargetType: "cidr", TargetValue: "0.0.0.0/0", Remark: "用户手动放行"},
		{SecurityGroupID: 7, Direction: "ingress", Protocol: "tcp", PortStart: 80, PortEnd: 80, TargetType: "cidr", TargetValue: "0.0.0.0/0", Remark: autoPortForwardSecurityGroupRuleNote},
	}
	if err := db.Create(&rules).Error; err != nil {
		t.Fatalf("写入测试安全组规则失败: %v", err)
	}
	deleted, err := deleteAutoSecurityGroupPortForwardRules(7, "tcp", 22, 22)
	if err != nil {
		t.Fatalf("deleteAutoSecurityGroupPortForwardRules returned error: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected one automatic rule deleted, got %d", deleted)
	}
	var remaining []model.VPCSecurityGroupRule
	if err := db.Order("port_start ASC, remark ASC").Find(&remaining).Error; err != nil {
		t.Fatalf("读取剩余规则失败: %v", err)
	}
	if len(remaining) != 2 {
		t.Fatalf("expected two rules remaining, got %+v", remaining)
	}
	for _, rule := range remaining {
		if rule.PortStart == 22 && rule.Remark == autoPortForwardSecurityGroupRuleNote {
			t.Fatalf("automatic port-forward rule should be deleted: %+v", remaining)
		}
	}
}

func TestVPCSwitchMeterIDStableAndSeparated(t *testing.T) {
	down := vpcSwitchMeterID(42, "down")
	up := vpcSwitchMeterID(42, "up")
	if down == 0 || up == 0 {
		t.Fatalf("meter ids should be non-zero: down=%d up=%d", down, up)
	}
	if down == up {
		t.Fatalf("directional meter ids should differ: %d", down)
	}
	if down != vpcSwitchMeterID(42, "down") {
		t.Fatalf("meter id should be stable")
	}
}

func TestBuildVPCSwitchBandwidthFlowsSupportsDirectionalLimits(t *testing.T) {
	sw := model.VPCSwitch{ID: 7, VLANID: 107, CIDR: "10.200.7.0/24"}
	flows := buildVPCSwitchBandwidthFlows(sw, "9", nil, 22, 10000, 0)
	if len(flows) != 1 {
		t.Fatalf("expected down flow only, got %d: %+v", len(flows), flows)
	}
	joined := strings.Join(flows, "\n")
	if strings.Contains(joined, "meter:22") {
		t.Fatalf("up meter should not be present when up bandwidth is unlimited: %s", joined)
	}
	if !strings.Contains(joined, "in_port=9") || !strings.Contains(joined, "actions=NORMAL") {
		t.Fatalf("down flow should forward traffic from VPC gateway port to switch CIDR (shaping via tc on gw port): %s", joined)
	}
	if strings.Contains(joined, "LOCAL") {
		t.Fatalf("VPC switch bandwidth flow must not use LOCAL port: %s", joined)
	}

	flows = buildVPCSwitchBandwidthFlows(sw, "", []string{"11", "12"}, 22, 0, 5000)
	joined = strings.Join(flows, "\n")
	if strings.Contains(joined, "meter:11") {
		t.Fatalf("down meter should not be present when down bandwidth is unlimited: %s", joined)
	}
	if strings.Contains(joined, "dl_vlan=107") {
		t.Fatalf("up flow should match VM ofport, not VLAN metadata: %s", joined)
	}
	for _, expected := range []string{
		"in_port=11,ip,nw_src=10.200.7.0/24,nw_dst=10.200.7.0/24,actions=NORMAL",
		"in_port=11,ip,nw_src=10.200.7.0/24,actions=meter:22,NORMAL",
		"in_port=12,ip,nw_src=10.200.7.0/24,nw_dst=10.200.7.0/24,actions=NORMAL",
		"in_port=12,ip,nw_src=10.200.7.0/24,actions=meter:22,NORMAL",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected %q in up flow set: %s", expected, joined)
		}
	}
}

func TestBuildVPCSwitchBandwidthFlowsForDirectBridge(t *testing.T) {
	sw := model.VPCSwitch{ID: 9, BridgeName: "brpub0", BridgeMode: BridgeModeDirect}
	flows := buildDirectBridgeSwitchBandwidthFlows(sw, []vpcSwitchVMPortMatch{
		{OFPort: "11", MAC: "52:54:00:aa:bb:01"},
		{OFPort: "12", MAC: "52:54:00:aa:bb:02"},
	}, 91, 92, 10000, 5000)
	joined := strings.Join(flows, "\n")
	for _, expected := range []string{
		"in_port=11,dl_src=52:54:00:aa:bb:01,actions=meter:92,NORMAL",
		"in_port=12,dl_src=52:54:00:aa:bb:02,actions=meter:92,NORMAL",
		"in_port=11,actions=drop",
		"in_port=12,actions=drop",
		"dl_dst=52:54:00:aa:bb:01,actions=meter:91,NORMAL",
		"dl_dst=52:54:00:aa:bb:02,actions=meter:91,NORMAL",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected %q in direct bridge flows: %s", expected, joined)
		}
	}
	if strings.Contains(joined, "nw_src") || strings.Contains(joined, "nw_dst") {
		t.Fatalf("direct bridge flows must not depend on internal CIDR: %s", joined)
	}
}

func TestBuildVPCSwitchBandwidthFlowsForDirectBridgeAllowsForgedSourceWhenEnabled(t *testing.T) {
	sw := model.VPCSwitch{ID: 9, BridgeName: "brpub0", BridgeMode: BridgeModeDirect, AllowMACChange: true, AllowForgedTransmits: true}
	flows := buildDirectBridgeSwitchBandwidthFlows(sw, []vpcSwitchVMPortMatch{
		{OFPort: "11", MAC: "52:54:00:aa:bb:01"},
	}, 91, 92, 10000, 5000)
	joined := strings.Join(flows, "\n")
	if !strings.Contains(joined, "in_port=11,actions=meter:92,NORMAL") {
		t.Fatalf("expected generic up flow when source MAC policy allows changes: %s", joined)
	}
	if strings.Contains(joined, "actions=drop") {
		t.Fatalf("should not add anti-spoof drop when source MAC policy allows changes: %s", joined)
	}
}

func setupVPCQuotaTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	oldDB := model.DB
	t.Cleanup(func() {
		model.DB = oldDB
	})
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), name)), &gorm.Config{})
	if err != nil {
		t.Skipf("当前环境无法打开 SQLite 测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.VPCSwitch{}, &model.VPCSecurityGroup{}); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	model.DB = db
	return db
}

func TestCheckSwitchResourceQuotaAllowsUnlimitedSwitchWhenUserUnlimited(t *testing.T) {
	db := setupVPCQuotaTestDB(t, "vpc-quota-unlimited.db")
	if err := db.Create(&model.User{Username: "admin", Role: "admin"}).Error; err != nil {
		t.Fatalf("写入测试用户失败: %v", err)
	}
	err := checkSwitchResourceQuota("admin", 0, VPCSwitchRequest{
		TrafficDownGB:     0,
		TrafficUpGB:       0,
		BandwidthDownMbps: 0,
		BandwidthUpMbps:   0,
	})
	if err != nil {
		t.Fatalf("expected unlimited user to allow switch unlimited quota, got: %v", err)
	}
}

func TestCheckSwitchResourceQuotaRejectsUnlimitedSwitchWhenUserLimited(t *testing.T) {
	db := setupVPCQuotaTestDB(t, "vpc-quota-limited-zero.db")
	if err := db.Create(&model.User{
		Username:         "test",
		Role:             "user",
		MaxTrafficDown:   10,
		MaxTrafficUp:     10,
		MaxBandwidthDown: 100,
		MaxBandwidthUp:   100,
	}).Error; err != nil {
		t.Fatalf("写入测试用户失败: %v", err)
	}
	err := checkSwitchResourceQuota("test", 0, VPCSwitchRequest{
		TrafficDownGB:     0,
		TrafficUpGB:       1,
		BandwidthDownMbps: 10,
		BandwidthUpMbps:   10,
	})
	if err == nil || !strings.Contains(err.Error(), "下行月流量配额必须大于 0") {
		t.Fatalf("expected finite traffic quota to reject zero switch traffic, got: %v", err)
	}
	err = checkSwitchResourceQuota("test", 0, VPCSwitchRequest{
		TrafficDownGB:     1,
		TrafficUpGB:       1,
		BandwidthDownMbps: 0,
		BandwidthUpMbps:   10,
	})
	if err == nil || !strings.Contains(err.Error(), "下行总带宽必须大于 0") {
		t.Fatalf("expected finite bandwidth quota to reject zero switch bandwidth, got: %v", err)
	}
}

func TestCheckSwitchResourceQuotaRejectsBandwidthOverAllocation(t *testing.T) {
	db := setupVPCQuotaTestDB(t, "vpc-quota-bandwidth-over.db")
	if err := db.Create(&model.User{
		Username:         "test",
		Role:             "user",
		MaxTrafficDown:   20,
		MaxTrafficUp:     20,
		MaxBandwidthDown: 100,
		MaxBandwidthUp:   100,
	}).Error; err != nil {
		t.Fatalf("写入测试用户失败: %v", err)
	}
	if err := db.Create(&model.VPCSwitch{
		Username:          "test",
		Name:              "sw1",
		TrafficDownGB:     5,
		TrafficUpGB:       5,
		BandwidthDownMbps: 80,
		BandwidthUpMbps:   50,
	}).Error; err != nil {
		t.Fatalf("写入测试交换机失败: %v", err)
	}
	err := checkSwitchResourceQuota("test", 0, VPCSwitchRequest{
		TrafficDownGB:     5,
		TrafficUpGB:       5,
		BandwidthDownMbps: 30,
		BandwidthUpMbps:   20,
	})
	if err == nil || !strings.Contains(err.Error(), "下行总带宽配额不足") {
		t.Fatalf("expected bandwidth over allocation error, got: %v", err)
	}
}

func TestDefaultVPCSwitchRequestUsesUserQuota(t *testing.T) {
	req := defaultVPCSwitchRequestForUser(model.User{
		Username:         "test",
		MaxTrafficDown:   10,
		MaxTrafficUp:     0,
		MaxBandwidthDown: 100,
		MaxBandwidthUp:   0,
	})
	if req.Username != "test" || req.Name != defaultVPCSwitchName {
		t.Fatalf("unexpected default switch identity: %+v", req)
	}
	if req.TrafficDownGB != 10 || req.TrafficUpGB != 0 {
		t.Fatalf("unexpected default traffic quota: %+v", req)
	}
	if req.BandwidthDownMbps != 100 || req.BandwidthUpMbps != 0 {
		t.Fatalf("unexpected default bandwidth quota: %+v", req)
	}
}

func TestUpdateVPCSecurityGroupAllowsRemarkUpdateForDefaultGroup(t *testing.T) {
	db := setupVPCQuotaTestDB(t, "vpc-update-default-group.db")
	group := model.VPCSecurityGroup{Username: "admin", Name: "默认安全组", IsDefault: true, Remark: "旧备注"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("写入测试安全组失败: %v", err)
	}
	updated, err := UpdateVPCSecurityGroup("admin", "admin", group.ID, VPCSecurityGroupRequest{
		Name:   group.Name,
		Remark: "新的备注",
	})
	if err != nil {
		t.Fatalf("UpdateVPCSecurityGroup returned error: %v", err)
	}
	if updated.Name != "默认安全组" {
		t.Fatalf("default group name should stay unchanged, got %q", updated.Name)
	}
	if updated.Remark != "新的备注" {
		t.Fatalf("expected updated remark, got %q", updated.Remark)
	}
}

func TestUpdateVPCSecurityGroupRejectsDuplicateName(t *testing.T) {
	db := setupVPCQuotaTestDB(t, "vpc-update-group-duplicate.db")
	groups := []model.VPCSecurityGroup{
		{Username: "test", Name: "group-a"},
		{Username: "test", Name: "group-b"},
	}
	if err := db.Create(&groups).Error; err != nil {
		t.Fatalf("写入测试安全组失败: %v", err)
	}
	_, err := UpdateVPCSecurityGroup("test", "user", groups[1].ID, VPCSecurityGroupRequest{
		Name:   "group-a",
		Remark: "重复名称",
	})
	if err == nil || !strings.Contains(err.Error(), "安全组名称已存在") {
		t.Fatalf("expected duplicate name error, got: %v", err)
	}
}

func TestResolveVPCForVMCreateUsesSingleSwitchAndDefaultSecurityGroup(t *testing.T) {
	db := setupVPCQuotaTestDB(t, "vpc-resolve-default.db")
	if err := db.Create(&model.User{
		Username:         "test",
		Role:             "user",
		MaxTrafficDown:   20,
		MaxTrafficUp:     20,
		MaxBandwidthDown: 10,
		MaxBandwidthUp:   10,
	}).Error; err != nil {
		t.Fatalf("写入测试用户失败: %v", err)
	}
	sw := model.VPCSwitch{
		Username:          "test",
		Name:              defaultVPCSwitchName,
		VLANID:            100,
		CIDR:              "10.200.1.0/24",
		GatewayIP:         "10.200.1.1",
		DHCPStart:         "10.200.1.10",
		DHCPEnd:           "10.200.1.250",
		TrafficDownGB:     20,
		TrafficUpGB:       20,
		BandwidthDownMbps: 10,
		BandwidthUpMbps:   10,
	}
	if err := db.Create(&sw).Error; err != nil {
		t.Fatalf("写入测试交换机失败: %v", err)
	}
	group := model.VPCSecurityGroup{Username: "test", Name: "默认安全组", IsDefault: true}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("写入测试安全组失败: %v", err)
	}
	switchID, groupID, err := ResolveVPCForVMCreate("test", 0, 0)
	if err != nil {
		t.Fatalf("ResolveVPCForVMCreate returned error: %v", err)
	}
	if switchID != sw.ID || groupID != group.ID {
		t.Fatalf("unexpected resolved IDs: switch=%d group=%d", switchID, groupID)
	}
}

func TestResolveVPCForVMCreateRequiresSwitchWhenMultipleWithoutDefault(t *testing.T) {
	db := setupVPCQuotaTestDB(t, "vpc-resolve-multiple.db")
	if err := db.Create(&model.User{Username: "test", Role: "user"}).Error; err != nil {
		t.Fatalf("写入测试用户失败: %v", err)
	}
	switches := []model.VPCSwitch{
		{Username: "test", Name: "sw1", VLANID: 100, CIDR: "10.200.1.0/24", GatewayIP: "10.200.1.1", DHCPStart: "10.200.1.10", DHCPEnd: "10.200.1.250"},
		{Username: "test", Name: "sw2", VLANID: 101, CIDR: "10.200.2.0/24", GatewayIP: "10.200.2.1", DHCPStart: "10.200.2.10", DHCPEnd: "10.200.2.250"},
	}
	if err := db.Create(&switches).Error; err != nil {
		t.Fatalf("写入测试交换机失败: %v", err)
	}
	if err := db.Create(&model.VPCSecurityGroup{Username: "test", Name: "默认安全组", IsDefault: true}).Error; err != nil {
		t.Fatalf("写入测试安全组失败: %v", err)
	}
	_, _, err := ResolveVPCForVMCreate("test", 0, 0)
	if err == nil || !strings.Contains(err.Error(), "请选择要接入的 VPC 交换机") {
		t.Fatalf("expected select switch error, got: %v", err)
	}
}

func TestSetFirstOVSInterfaceVLANTagAddsTag(t *testing.T) {
	xmlText := `<domain type='kvm'>
  <devices>
    <interface type='bridge'>
      <mac address='52:54:00:1a:4c:b4'/>
      <source bridge='br-ovs'/>
      <virtualport type='openvswitch'/>
      <model type='virtio'/>
    </interface>
  </devices>
</domain>`
	got, changed := setFirstOVSInterfaceVLANTag(xmlText, 100)
	if !changed {
		t.Fatal("expected XML to be changed")
	}
	if !strings.Contains(got, "<vlan>") || !strings.Contains(got, "<tag id='100'/>") {
		t.Fatalf("expected vlan tag in XML: %s", got)
	}
}

func TestSetFirstOVSInterfaceVPCRestoresDefaultBridgeFromDirectBridge(t *testing.T) {
	xmlText := `<domain type='kvm'>
  <devices>
    <interface type='bridge'>
      <mac address='52:54:00:1a:4c:b4'/>
      <source bridge='br-test'/>
      <virtualport type='openvswitch'/>
      <model type='virtio'/>
    </interface>
  </devices>
</domain>`
	got, changed := setFirstOVSInterfaceVPC(xmlText, 101)
	if !changed {
		t.Fatal("expected XML to be changed")
	}
	if !strings.Contains(got, "<source bridge='br-ovs'/>") {
		t.Fatalf("expected bridge source restored to br-ovs: %s", got)
	}
	if !strings.Contains(got, "<tag id='101'/>") {
		t.Fatalf("expected vlan tag in XML: %s", got)
	}
}

func TestSetFirstOVSInterfaceVLANTagUpdatesExistingTag(t *testing.T) {
	xmlText := `<domain type='kvm'>
  <devices>
    <interface type='bridge'>
      <source bridge='br-ovs'/>
      <virtualport type='openvswitch'/>
      <vlan>
        <tag id='200'/>
      </vlan>
    </interface>
  </devices>
</domain>`
	got, changed := setFirstOVSInterfaceVLANTag(xmlText, 100)
	if !changed {
		t.Fatal("expected XML to be changed")
	}
	if strings.Contains(got, "id='200'") || !strings.Contains(got, "<tag id='100'/>") {
		t.Fatalf("expected vlan tag update in XML: %s", got)
	}
}

func TestSetFirstOVSInterfaceVLANTagKeepsExistingTargetTag(t *testing.T) {
	xmlText := `<domain type='kvm'>
  <devices>
    <interface type='bridge'>
      <source bridge='br-ovs'/>
      <virtualport type='openvswitch'/>
      <vlan>
        <tag id='100'/>
      </vlan>
    </interface>
  </devices>
</domain>`
	got, changed := setFirstOVSInterfaceVLANTag(xmlText, 100)
	if !changed {
		t.Fatal("expected existing VLAN tag to be accepted")
	}
	if got != xmlText {
		t.Fatalf("expected XML to stay unchanged, got: %s", got)
	}
}

func TestExtractFirstOVSInterfaceBlock(t *testing.T) {
	xmlText := `<domain type='kvm'>
  <devices>
    <interface type='network'>
      <source network='default'/>
    </interface>
    <interface type='bridge'>
      <mac address='52:54:00:1a:4c:b4'/>
      <source bridge='br-ovs'/>
      <virtualport type='openvswitch'>
        <parameters interfaceid='abc'/>
      </virtualport>
      <target dev='vnet1'/>
      <model type='virtio'/>
    </interface>
  </devices>
</domain>`
	block, ok := extractFirstOVSInterfaceBlock(xmlText)
	if !ok {
		t.Fatal("expected OVS bridge interface block")
	}
	if !strings.Contains(block, "<source bridge='br-ovs'/>") || !strings.Contains(block, "interfaceid='abc'") {
		t.Fatalf("unexpected interface block: %s", block)
	}
}

func TestStripRuntimeOnlyInterfaceElements(t *testing.T) {
	block := `<interface type='bridge'>
      <mac address='52:54:00:1a:4c:b4'/>
      <source bridge='br-ovs'/>
      <virtualport type='openvswitch'/>
      <target dev='vnet1'/>
      <model type='virtio'/>
      <alias name='net0'/>
      <address type='pci' domain='0x0000' bus='0x01' slot='0x00' function='0x0'/>
    </interface>`
	got := stripRuntimeOnlyInterfaceElements(block)
	if strings.Contains(got, "<target ") || strings.Contains(got, "<alias ") || strings.Contains(got, "<address ") {
		t.Fatalf("expected runtime-only elements stripped: %s", got)
	}
	if !strings.Contains(got, "<mac address='52:54:00:1a:4c:b4'/>") || !strings.Contains(got, "<model type='virtio'/>") {
		t.Fatalf("expected stable interface elements kept: %s", got)
	}
}

func TestSetFirstOVSInterfaceBridgeRemovesVLAN(t *testing.T) {
	xmlText := `<domain type='kvm'>
  <devices>
    <interface type='bridge'>
      <source bridge='br-ovs'/>
      <virtualport type='openvswitch'/>
      <vlan>
        <tag id='100'/>
      </vlan>
    </interface>
  </devices>
</domain>`
	got, changed := setFirstOVSInterfaceBridge(xmlText, "brpub0")
	if !changed {
		t.Fatal("expected bridge source to be updated")
	}
	got = removeFirstInterfaceVLAN(got)
	if !strings.Contains(got, "<source bridge='brpub0'/>") {
		t.Fatalf("expected bridge source update: %s", got)
	}
	if strings.Contains(got, "<vlan>") {
		t.Fatalf("expected VLAN removed for direct bridge: %s", got)
	}
}

func TestSetFirstOVSInterfaceDirectBridgeAddsBridgeVLAN(t *testing.T) {
	xmlText := `<domain type='kvm'>
  <devices>
    <interface type='bridge'>
      <source bridge='br-ovs'/>
      <virtualport type='openvswitch'/>
    </interface>
  </devices>
</domain>`
	got, changed := setFirstOVSInterfaceDirectBridge(xmlText, "brpub0", 30)
	if !changed {
		t.Fatal("expected bridge VLAN XML to be changed")
	}
	if !strings.Contains(got, "<source bridge='brpub0'/>") || !strings.Contains(got, "<tag id='30'/>") {
		t.Fatalf("expected direct bridge with VLAN tag: %s", got)
	}
}

func TestEffectiveVPCSwitchBandwidthUsesDirectionalPenaltyWhenLimited(t *testing.T) {
	oldDB := model.DB
	t.Cleanup(func() {
		model.DB = oldDB
	})
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "vpc-switch-limit.db")), &gorm.Config{})
	if err != nil {
		t.Skipf("当前环境无法打开 SQLite 测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.VPCSwitchTrafficMonthly{}); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	model.DB = db
	sw := model.VPCSwitch{ID: 8, BandwidthDownMbps: 100, BandwidthUpMbps: 20}
	down, up := effectiveVPCSwitchBandwidth(sw)
	if down != 100 || up != 20 {
		t.Fatalf("expected configured bandwidth, got down=%d up=%d", down, up)
	}
	if err := db.Create(&model.VPCSwitchTrafficMonthly{
		SwitchID:      sw.ID,
		Username:      "test",
		Month:         currentTrafficMonth(),
		IsLimitedDown: true,
	}).Error; err != nil {
		t.Fatalf("写入测试限速状态失败: %v", err)
	}
	down, up = effectiveVPCSwitchBandwidth(sw)
	if down != vpcSwitchTrafficPenaltyMbps || up != 20 {
		t.Fatalf("expected directional down penalty, got down=%d up=%d", down, up)
	}
	if err := db.Model(&model.VPCSwitchTrafficMonthly{}).Where("switch_id = ?", sw.ID).Updates(map[string]interface{}{
		"is_limited_down": false,
		"is_limited_up":   true,
	}).Error; err != nil {
		t.Fatalf("更新测试限速状态失败: %v", err)
	}
	down, up = effectiveVPCSwitchBandwidth(sw)
	if down != 100 || up != vpcSwitchTrafficPenaltyMbps {
		t.Fatalf("expected directional up penalty, got down=%d up=%d", down, up)
	}
}

func TestRebaseVPCSwitchTrafficMonthlyWhenBindingMoves(t *testing.T) {
	oldDB := model.DB
	t.Cleanup(func() {
		model.DB = oldDB
	})
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "vpc-switch-rebase.db")), &gorm.Config{})
	if err != nil {
		t.Skipf("当前环境无法打开 SQLite 测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.VPCSwitch{}, &model.VPCVMBinding{}, &model.VPCSwitchTrafficMonthly{}, &model.VmStatsRecord{}); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	model.DB = db
	switches := []model.VPCSwitch{
		{ID: 1, Username: "test", Name: "old", BridgeName: "br-ovs", BridgeMode: BridgeModeNAT, VLANID: 100, CIDR: "10.200.1.0/24", GatewayIP: "10.200.1.1", DHCPStart: "10.200.1.10", DHCPEnd: "10.200.1.250"},
		{ID: 2, Username: "admin", Name: "bridge", BridgeName: "br-test", BridgeMode: BridgeModeDirect, VLANID: 101, CIDR: "10.200.2.0/24", GatewayIP: "10.200.2.1", DHCPStart: "10.200.2.10", DHCPEnd: "10.200.2.250"},
	}
	if err := db.Create(&switches).Error; err != nil {
		t.Fatalf("写入测试交换机失败: %v", err)
	}
	if err := db.Create(&model.VPCVMBinding{VMName: "vm1", Username: "test", SwitchID: 1, SecurityGroupID: 1}).Error; err != nil {
		t.Fatalf("写入测试绑定失败: %v", err)
	}
	now := time.Now()
	records := []model.VmStatsRecord{
		{VMName: "vm1", NetRxBytes: 100, NetTxBytes: 200, RecordedAt: now.Add(-2 * time.Hour)},
		{VMName: "vm1", NetRxBytes: 1100, NetTxBytes: 1200, RecordedAt: now.Add(-time.Hour)},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("写入测试统计失败: %v", err)
	}
	oldDown, oldUp := AggregateSwitchMonthlyTraffic(1)
	newDown, newUp := AggregateSwitchMonthlyTraffic(2)
	if oldDown != 1000 || oldUp != 1000 || newDown != 0 || newUp != 0 {
		t.Fatalf("unexpected traffic before move: old=%d/%d new=%d/%d", oldDown, oldUp, newDown, newUp)
	}
	if err := db.Model(&model.VPCVMBinding{}).Where("vm_name = ?", "vm1").Updates(map[string]interface{}{"switch_id": 2, "username": "admin"}).Error; err != nil {
		t.Fatalf("更新测试绑定失败: %v", err)
	}
	rebaseVPCSwitchTrafficMonthly(1, oldDown, oldUp)
	rebaseVPCSwitchTrafficMonthly(2, newDown, newUp)
	oldDown, oldUp = AggregateSwitchMonthlyTraffic(1)
	newDown, newUp = AggregateSwitchMonthlyTraffic(2)
	if oldDown != 1000 || oldUp != 1000 || newDown != 0 || newUp != 0 {
		t.Fatalf("unexpected traffic after rebase: old=%d/%d new=%d/%d", oldDown, oldUp, newDown, newUp)
	}
	if err := db.Create(&model.VmStatsRecord{VMName: "vm1", NetRxBytes: 1600, NetTxBytes: 1700, RecordedAt: now}).Error; err != nil {
		t.Fatalf("写入移动后统计失败: %v", err)
	}
	newDown, newUp = AggregateSwitchMonthlyTraffic(2)
	if newDown != 500 || newUp != 500 {
		t.Fatalf("new switch should only count traffic after move, got %d/%d", newDown, newUp)
	}
}

func TestBuildVPCIngressAllowRulesMatchesIPWithoutInterface(t *testing.T) {
	oldDB := model.DB
	t.Cleanup(func() {
		model.DB = oldDB
	})
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "vpc-acl-test.db")), &gorm.Config{})
	if err != nil {
		t.Skipf("当前环境无法打开 SQLite 测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.VPCSecurityGroupRule{}); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	model.DB = db
	if err := db.Create(&model.VPCSecurityGroupRule{
		SecurityGroupID: 9,
		Direction:       "ingress",
		Protocol:        "tcp",
		PortStart:       22,
		PortEnd:         22,
		TargetType:      "cidr",
		TargetValue:     "0.0.0.0/0",
	}).Error; err != nil {
		t.Fatalf("写入测试安全组规则失败: %v", err)
	}

	lines, err := buildVPCIngressAllowRules(model.VPCVMBinding{SecurityGroupID: 9}, "10.200.1.52")
	if err != nil {
		t.Fatalf("buildVPCIngressAllowRules returned error: %v", err)
	}
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "oifname") {
		t.Fatalf("VPC ACL should not depend on bridge oifname: %s", joined)
	}
	if !strings.Contains(joined, "ip daddr 10.200.1.52 ip saddr 0.0.0.0/0 tcp dport 22 accept") {
		t.Fatalf("expected IP based allow rule, got: %s", joined)
	}
}
