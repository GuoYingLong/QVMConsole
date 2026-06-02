package service

import (
	"strings"
	"testing"
)

func TestValidateBridgeName(t *testing.T) {
	if err := validateBridgeName("brpub0"); err != nil {
		t.Fatalf("expected valid bridge name: %v", err)
	}
	if err := validateBridgeName("br-kvm-plan-test"); err == nil {
		t.Fatal("expected long bridge name to be rejected")
	}
	if err := validateBridgeName("br pub"); err == nil {
		t.Fatal("expected bridge name with space to be rejected")
	}
}

func TestBuildOVSInterfaceXMLForBridge(t *testing.T) {
	xml := BuildOVSInterfaceXMLForBridge("52:54:00:aa:bb:cc", "virtio", "brpub0")
	for _, want := range []string{
		"<source bridge='brpub0'/>",
		"<virtualport type='openvswitch'/>",
		"<model type='virtio'/>",
	} {
		if !strings.Contains(xml, want) {
			t.Fatalf("interface XML missing %q: %s", want, xml)
		}
	}
}

func TestParseResolvectlDNSServers(t *testing.T) {
	text := `Link 2 (ens33): 223.5.5.5 223.6.6.6
Global: 1.1.1.1
Link 20 (br-test): 223.5.5.5`
	got := parseResolvectlDNSServers(text)
	want := []string{"223.5.5.5", "223.6.6.6", "1.1.1.1"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected DNS servers: got=%v want=%v", got, want)
	}
}

func TestBuildBridgeRestoreScriptMigratesHostIP(t *testing.T) {
	script := buildBridgeRestoreScriptContent("br-test", "ens33", true)
	for _, want := range []string{
		`HOST_ADDRS="$(ip -4 -o addr show dev "$UPLINK" scope global`,
		`ovs-vsctl --may-exist add-port "$BRIDGE" "$UPLINK"`,
		`ip addr flush dev "$UPLINK"`,
		`ip addr replace "$addr" dev "$BRIDGE"`,
		`ip route replace "$HOST_GW" dev "$BRIDGE" scope link`,
		`ip route replace default via "$HOST_GW" dev "$BRIDGE"`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("restore script missing %q: %s", want, script)
		}
	}
}

func TestBuildBridgeRestoreScriptWithoutHostIPMigration(t *testing.T) {
	script := buildBridgeRestoreScriptContent("br-test", "ens33", false)
	if strings.Contains(script, "HOST_ADDRS") || strings.Contains(script, "ip addr flush dev") {
		t.Fatalf("restore script should not migrate host IP when disabled: %s", script)
	}
	if !strings.Contains(script, `ovs-vsctl --may-exist add-port "$BRIDGE" "$UPLINK"`) {
		t.Fatalf("restore script missing add-port: %s", script)
	}
}

func TestBridgeHostIPRollbackShellMigratesBackToUplink(t *testing.T) {
	script := bridgeHostIPRollbackShell()
	for _, want := range []string{
		`HOST_ADDRS="$(ip -4 -o addr show dev "$BRIDGE" scope global`,
		`HOST_GW="$(ip -4 route show default dev "$BRIDGE"`,
		`ip link set "$UPLINK" up`,
		`ip addr flush dev "$BRIDGE"`,
		`ip addr replace "$addr" dev "$UPLINK"`,
		`ip route replace "$HOST_GW" dev "$UPLINK" scope link`,
		`ip route replace default via "$HOST_GW" dev "$UPLINK"`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("rollback script missing %q: %s", want, script)
		}
	}
}
