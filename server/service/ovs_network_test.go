package service

import (
	"strings"
	"testing"

	"kvm_console/config"
	"kvm_console/model"
)

func TestBuildOVSVirtInstallNetworkArg(t *testing.T) {
	old := config.GlobalConfig
	config.GlobalConfig = &config.Config{NetworkBackend: "ovs", OVSBridge: "br-test", SubnetPrefix: "192.168.122"}
	defer func() { config.GlobalConfig = old }()

	got := BuildOVSVirtInstallNetworkArg("virtio")
	want := "--network 'bridge=br-test,virtualport.type=openvswitch,model=virtio'"
	if got != want {
		t.Fatalf("unexpected network arg: got %q want %q", got, want)
	}
}

func TestBuildOVSInterfaceXML(t *testing.T) {
	old := config.GlobalConfig
	config.GlobalConfig = &config.Config{NetworkBackend: "ovs", OVSBridge: "br-test", SubnetPrefix: "192.168.122"}
	defer func() { config.GlobalConfig = old }()

	xml := BuildOVSInterfaceXML("52:54:00:aa:bb:cc", "virtio")
	for _, want := range []string{
		"<interface type='bridge'>",
		"<source bridge='br-test'/>",
		"<virtualport type='openvswitch'/>",
		"<model type='virtio'/>",
	} {
		if !strings.Contains(xml, want) {
			t.Fatalf("interface XML missing %q: %s", want, xml)
		}
	}
}

func TestBuildOVSInterfaceXMLWithVLAN(t *testing.T) {
	old := config.GlobalConfig
	config.GlobalConfig = &config.Config{NetworkBackend: "ovs", OVSBridge: "br-test", SubnetPrefix: "192.168.122"}
	defer func() { config.GlobalConfig = old }()

	xml := BuildOVSInterfaceXMLWithVLAN("52:54:00:aa:bb:cc", "virtio", 101)
	for _, want := range []string{
		"<source bridge='br-test'/>",
		"<vlan>",
		"<tag id='101'/>",
		"<virtualport type='openvswitch'/>",
	} {
		if !strings.Contains(xml, want) {
			t.Fatalf("interface XML missing %q: %s", want, xml)
		}
	}
}

func TestParseOVSStaticHostsText(t *testing.T) {
	hosts := ParseOVSStaticHostsText(`
# comment
52:54:00:aa:bb:cc,192.168.122.10,vm-a
52:54:00:dd:ee:ff,set:vm-b,192.168.122.11,vm-b
`)
	if len(hosts) != 2 {
		t.Fatalf("unexpected host count: %d", len(hosts))
	}
	if hosts[0].MAC != "52:54:00:aa:bb:cc" || hosts[0].IP != "192.168.122.10" || hosts[0].VMName != "vm-a" {
		t.Fatalf("unexpected first host: %+v", hosts[0])
	}
	if hosts[1].IP != "192.168.122.11" || hosts[1].VMName != "vm-b" {
		t.Fatalf("unexpected second host: %+v", hosts[1])
	}
}

func TestParseOVSDHCPLeasesText(t *testing.T) {
	leases := ParseOVSDHCPLeasesText("1777330000 52:54:00:aa:bb:cc 192.168.122.20 ubuntu *\n")
	if len(leases) != 1 {
		t.Fatalf("unexpected lease count: %d", len(leases))
	}
	if leases[0].MAC != "52:54:00:aa:bb:cc" || leases[0].IP != "192.168.122.20" || leases[0].Hostname != "ubuntu" || leases[0].ExpiryUnix != 1777330000 || strings.Contains(leases[0].ExpiryTime, "1777330000") {
		t.Fatalf("unexpected lease: %+v", leases[0])
	}
}

func TestNewerOVSDHCPLeasePrefersLatestExpiry(t *testing.T) {
	oldLease := OVSDHCPLease{IP: "10.200.3.134", ExpiryUnix: 1777584856}
	newLease := OVSDHCPLease{IP: "10.200.3.135", ExpiryUnix: 1777584907}
	if got := newerOVSDHCPLease(oldLease, newLease); got.IP != "10.200.3.135" {
		t.Fatalf("应选择过期时间更新的租约，got %+v", got)
	}
}

func TestBuildOVSStaticHostsForUpsertRejectsDuplicateIP(t *testing.T) {
	_, err := buildOVSStaticHostsForUpsert([]OVSStaticHost{
		{VMName: "vm-a", MAC: "52:54:00:aa:bb:cc", IP: "192.168.122.10"},
	}, OVSStaticHost{VMName: "vm-b", MAC: "52:54:00:dd:ee:ff", IP: "192.168.122.10"})
	if err == nil || !strings.Contains(err.Error(), "IP 地址") {
		t.Fatalf("expected duplicate IP error, got %v", err)
	}
}

func TestBuildOVSStaticHostsForUpsertRejectsDuplicateMAC(t *testing.T) {
	_, err := buildOVSStaticHostsForUpsert([]OVSStaticHost{
		{VMName: "vm-a", MAC: "52:54:00:aa:bb:cc", IP: "192.168.122.10"},
	}, OVSStaticHost{VMName: "vm-b", MAC: "52:54:00:aa:bb:cc", IP: "192.168.122.11"})
	if err == nil || !strings.Contains(err.Error(), "MAC 地址") {
		t.Fatalf("expected duplicate MAC error, got %v", err)
	}
}

func TestBuildOVSStaticHostsForUpsertAllowsSameVMToChangeMAC(t *testing.T) {
	got, err := buildOVSStaticHostsForUpsert([]OVSStaticHost{
		{VMName: "vm-a", MAC: "52:54:00:aa:bb:cc", IP: "192.168.122.10"},
	}, OVSStaticHost{VMName: "vm-a", MAC: "52:54:00:dd:ee:ff", IP: "192.168.122.10"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].MAC != "52:54:00:dd:ee:ff" || got[0].IP != "192.168.122.10" {
		t.Fatalf("unexpected hosts: %+v", got)
	}
}

func TestNormalizeIPForVPCSupportsLastOctet(t *testing.T) {
	sw := model.VPCSwitch{CIDR: "10.200.7.0/24", GatewayIP: "10.200.7.1"}
	ip, err := normalizeIPForVPC("52", sw)
	if err != nil {
		t.Fatalf("normalizeIPForVPC returned error: %v", err)
	}
	if ip != "10.200.7.52" {
		t.Fatalf("unexpected normalized IP: %s", ip)
	}
	if _, err := normalizeIPForVPC("10.200.8.52", sw); err == nil {
		t.Fatal("expected out-of-cidr error")
	}
}
