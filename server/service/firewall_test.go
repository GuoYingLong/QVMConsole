package service

import (
	"path/filepath"
	"strings"
	"testing"

	"kvm_console/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBuildFirewallRulesGlobalRegions(t *testing.T) {
	oldDB := model.DB
	model.DB = nil
	t.Cleanup(func() { model.DB = oldDB })
	policy := defaultFirewallPolicy()
	policy.OutboundEnabled = true
	policy.InboundEnabled = true
	policy.OutboundAllowedRegions = []string{"cn"}
	policy.InboundAllowedRegions = []string{"cn"}
	policy.Regions = []FirewallRegion{
		{Code: "cn", Name: "中国大陆", CIDRs: []string{"1.0.1.0/24", "1.0.2.0/23"}},
	}

	rules, err := BuildFirewallRules(policy)
	if err != nil {
		t.Fatalf("BuildFirewallRules returned error: %v", err)
	}
	if !strings.Contains(rules, "table inet kvm_console_fw") {
		t.Fatalf("rules should contain kvm_console_fw table: %s", rules)
	}
	if !strings.Contains(rules, "iifname \"br-ovs\" ip saddr 192.168.122.0/24 ip daddr != @out_allowed4 reject") {
		t.Fatalf("rules should contain outbound reject rule: %s", rules)
	}
	if !strings.Contains(rules, "oifname \"br-ovs\" ip daddr 192.168.122.0/24 ip saddr != @in_allowed4 reject") {
		t.Fatalf("rules should contain inbound reject rule: %s", rules)
	}
}

func TestBuildFirewallRulesCoversVPCSwitchScopes(t *testing.T) {
	oldDB := model.DB
	t.Cleanup(func() { model.DB = oldDB })
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "firewall-vpc.db")), &gorm.Config{})
	if err != nil {
		t.Skipf("当前环境无法打开 SQLite 测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.VPCSwitch{}); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	model.DB = db
	if err := db.Create(&model.VPCSwitch{ID: 7, Username: "test", Name: "sw", VLANID: 107, CIDR: "10.200.7.0/24", GatewayIP: "10.200.7.1", DHCPStart: "10.200.7.10", DHCPEnd: "10.200.7.250"}).Error; err != nil {
		t.Fatalf("写入测试交换机失败: %v", err)
	}
	policy := defaultFirewallPolicy()
	policy.OutboundEnabled = true
	policy.InboundEnabled = true
	policy.OutboundAllowedRegions = []string{"cn"}
	policy.InboundAllowedRegions = []string{"cn"}
	policy.Regions = []FirewallRegion{{Code: "cn", Name: "中国大陆", CIDRs: []string{"1.0.1.0/24"}}}

	rules, err := BuildFirewallRules(policy)
	if err != nil {
		t.Fatalf("BuildFirewallRules returned error: %v", err)
	}
	if !strings.Contains(rules, "iifname \"vpcsw7\" ip saddr 10.200.7.0/24 ip daddr != @out_allowed4 reject") {
		t.Fatalf("rules should contain VPC outbound reject rule: %s", rules)
	}
	if !strings.Contains(rules, "oifname \"vpcsw7\" ip daddr 10.200.7.0/24 ip saddr != @in_allowed4 reject") {
		t.Fatalf("rules should contain VPC inbound reject rule: %s", rules)
	}
}

func TestNormalizeFirewallPolicyMigratesVirbr0ToOVS(t *testing.T) {
	policy := normalizeFirewallPolicy(&FirewallPolicy{Bridge: "virbr0", VMSubnet: "192.168.122.0/24"})
	if policy.Bridge != "br-ovs" {
		t.Fatalf("expected br-ovs bridge, got %s", policy.Bridge)
	}
}

func TestNormalizeCIDRList(t *testing.T) {
	got := normalizeCIDRList([]string{"1.1.1.1", "1.1.1.0/24", "bad", "1.1.1.1/32"})
	want := []string{"1.1.1.0/24"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected CIDR list: got %v want %v", got, want)
	}
}

func TestNormalizeCIDRListRemovesContainedPrivateSubnet(t *testing.T) {
	got := normalizeCIDRList([]string{"192.168.0.0/16", "192.168.122.0/24", "10.0.0.0/8"})
	want := []string{"10.0.0.0/8", "192.168.0.0/16"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected CIDR list: got %v want %v", got, want)
	}
}

func TestPortForwardStableKey(t *testing.T) {
	rule := PortForwardRule{
		Protocol: "TCP",
		HostPort: "10000",
		DestIP:   "192.168.122.12",
		DestPort: "22",
	}
	if got := rule.StableKey(); got != "tcp|10000|192.168.122.12|22" {
		t.Fatalf("unexpected stable key: %s", got)
	}
}

func TestNormalizeOverrideModeInboundOnly(t *testing.T) {
	if got := normalizeOverrideMode("inbound_only"); got != "inbound_only" {
		t.Fatalf("unexpected override mode: %s", got)
	}
}
