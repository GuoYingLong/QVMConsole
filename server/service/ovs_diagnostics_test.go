package service

import "testing"

func TestParseOVSOfctlShow(t *testing.T) {
	text := `OFPT_FEATURES_REPLY (OF1.3) (xid=0x2): dpid:00004282b25a1048
OFPST_PORT_DESC reply (OF1.3) (xid=0x3):
 1(vnet0): addr:fe:54:00:aa:bb:cc
     config:     0
     state:      LIVE
 LOCAL(br-ovs): addr:42:82:b2:5a:10:48
`
	ports := parseOVSOfctlShow(text)
	if len(ports) != 2 {
		t.Fatalf("expected 2 ports, got %d", len(ports))
	}
	if ports[0].Name != "vnet0" || ports[0].OFPort != "1" {
		t.Fatalf("unexpected first port: %+v", ports[0])
	}
	if ports[1].Name != "br-ovs" || ports[1].OFPort != "LOCAL" {
		t.Fatalf("unexpected local port: %+v", ports[1])
	}
}

func TestParseOVSInterfaceTypeCSV(t *testing.T) {
	types := parseOVSInterfaceTypeCSV("br-ovs,internal\nvnet0,\n")
	if types["br-ovs"] != "internal" {
		t.Fatalf("expected internal bridge type, got %q", types["br-ovs"])
	}
	if types["vnet0"] != "system" {
		t.Fatalf("expected empty interface type to become system, got %q", types["vnet0"])
	}
}

func TestParseVirshDomiflistOutput(t *testing.T) {
	text := ` Interface   Type     Source   Model    MAC
-----------------------------------------------------------
 vnet0       bridge   br-ovs   virtio   52:54:00:1a:4c:b4
 -           bridge   br-ovs   virtio   52:54:00:aa:bb:cc
`
	ifaces := parseVirshDomiflistOutput(text)
	if len(ifaces) != 2 {
		t.Fatalf("expected 2 interfaces, got %d", len(ifaces))
	}
	if ifaces[0].Name != "vnet0" || ifaces[0].Source != "br-ovs" || ifaces[0].MAC != "52:54:00:1a:4c:b4" {
		t.Fatalf("unexpected running interface: %+v", ifaces[0])
	}
	if ifaces[1].Name != "-" {
		t.Fatalf("expected shut off interface target to be preserved, got %+v", ifaces[1])
	}
}

func TestCorrelateOVSPortsWithVMAndIP(t *testing.T) {
	ports := []OVSPortStatus{{Name: "vnet0", OFPort: "7"}}
	types := map[string]string{"vnet0": "system"}
	vmIfaces := map[string]ovsRuntimeInterface{
		"vnet0": {
			Name: "vnet0",
			MAC:  "52:54:00:1a:4c:b4",
		},
		"52:54:00:1a:4c:b4": {
			Name: "linuxtest",
			MAC:  "52:54:00:1a:4c:b4",
		},
	}
	staticHosts := []OVSStaticHost{{VMName: "linuxtest", MAC: "52:54:00:1a:4c:b4", IP: "192.168.122.218"}}
	result := correlateOVSPorts(ports, types, vmIfaces, staticHosts, nil, "br-ovs")
	if len(result.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(result.Ports))
	}
	port := result.Ports[0]
	if port.VMName != "linuxtest" || port.IP != "192.168.122.218" || port.IPSource != "static" {
		t.Fatalf("unexpected correlated port: %+v", port)
	}
	if len(port.Issues) != 0 {
		t.Fatalf("unexpected port issues: %+v", port.Issues)
	}
}

func TestCorrelateOVSPortsDetectsAbnormalVMPort(t *testing.T) {
	ports := []OVSPortStatus{{Name: "vnet9", OFPort: "-1"}}
	result := correlateOVSPorts(ports, nil, nil, nil, nil, "br-ovs")
	if len(result.Ports[0].Issues) != 3 {
		t.Fatalf("expected ofport/vm/ip issues, got %+v", result.Ports[0].Issues)
	}
}

func TestDetectOVSLeaseConflicts(t *testing.T) {
	staticHosts := []OVSStaticHost{{VMName: "vm-a", MAC: "52:54:00:aa:bb:cc", IP: "192.168.122.10"}}
	leases := []OVSDHCPLease{
		{MAC: "52:54:00:dd:ee:ff", IP: "192.168.122.10", Hostname: "other"},
		{MAC: "52:54:00:aa:bb:cc", IP: "192.168.122.11", Hostname: "vm-a"},
	}
	conflicts := detectOVSLeaseConflicts(staticHosts, leases)
	if len(conflicts) != 2 {
		t.Fatalf("expected 2 conflicts, got %+v", conflicts)
	}
	if conflicts[0].Type != "ip_conflict" || conflicts[1].Type != "mac_conflict" {
		t.Fatalf("unexpected conflict types: %+v", conflicts)
	}
}

func TestHasOVSBandwidthFlow(t *testing.T) {
	flows := " cookie=0xabc, duration=1.0s, table=0, priority=100 actions=NORMAL\n"
	if !hasOVSBandwidthFlow(flows, "0xabc") {
		t.Fatal("expected flow cookie to be detected")
	}
	if hasOVSBandwidthFlow(flows, "0xdef") {
		t.Fatal("unexpected flow cookie match")
	}
}

func TestBuildOVSRepairSuggestions(t *testing.T) {
	status := &OVSStatus{
		BridgeExists:       false,
		BridgeHasGateway:   false,
		OpenVSwitchService: OVSServiceStatus{Active: false},
		DNSMasqService:     OVSServiceStatus{Active: true},
		IPForwardEnabled:   false,
		NATRule:            OVSRuleStatus{Exists: false},
		ForwardOutRule:     OVSRuleStatus{Exists: true},
		ForwardReturnRule:  OVSRuleStatus{Exists: true},
	}
	suggestions := buildOVSRepairSuggestions(status)
	if len(suggestions) != 4 {
		t.Fatalf("expected 4 suggestions, got %+v", suggestions)
	}
}
