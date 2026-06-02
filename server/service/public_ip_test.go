package service

import (
	"strings"
	"testing"

	"kvm_console/model"
)

func TestNormalizePublicIPMode(t *testing.T) {
	cases := map[string]string{
		"":               PublicIPModeNAT,
		"nat":            PublicIPModeNAT,
		"1:1 NAT":        PublicIPModeNAT,
		"classic":        PublicIPModeClassicRoute,
		"classic_route":  PublicIPModeClassicRoute,
		"classic_bridge": PublicIPModeClassicBridge,
	}
	for input, want := range cases {
		if got := NormalizePublicIPMode(input); got != want {
			t.Fatalf("NormalizePublicIPMode(%q)=%q, want %q", input, got, want)
		}
	}
}

func TestBuildPublicIPNATCommands(t *testing.T) {
	ipRow := model.PublicIP{IP: "203.0.113.10", CIDR: "203.0.113.8/29", UplinkIF: "ens3"}
	req := PublicIPBindRequest{VMName: "vm1", VMPrivateIP: "10.200.1.20", Mode: PublicIPModeNAT}
	commands := buildPublicIPNATCommands(ipRow, req)
	joined := strings.Join(commands, "\n")
	if !strings.Contains(joined, "-j DNAT --to-destination '10.200.1.20'") {
		t.Fatalf("DNAT command missing: %s", joined)
	}
	if !strings.Contains(joined, "-I POSTROUTING 1") || !strings.Contains(joined, "--to-source '203.0.113.10'") {
		t.Fatalf("SNAT should be inserted before MASQUERADE: %s", joined)
	}
	if !strings.Contains(joined, "ip addr replace '203.0.113.10/29' dev 'ens3'") {
		t.Fatalf("host public address command missing: %s", joined)
	}
}

func TestPublicIPPrefixSupportsDottedMask(t *testing.T) {
	got := publicIPPrefix(model.PublicIP{IP: "203.0.113.10", CIDR: "255.255.255.248"})
	if got != 29 {
		t.Fatalf("expected /29 from dotted mask, got /%d", got)
	}
}

func TestPublicIPFlowCookieUsesManagedPrefix(t *testing.T) {
	cookie := publicIPFlowCookie("203.0.113.10")
	if !strings.HasPrefix(cookie, publicIPFlowPrefix) {
		t.Fatalf("cookie should use public IP prefix, got %s", cookie)
	}
	if len(strings.TrimPrefix(cookie, "0x")) != 16 {
		t.Fatalf("cookie should be 64-bit hex, got %s", cookie)
	}
}

func TestCleanupPublicIPRulesUsesLineNumbers(t *testing.T) {
	script := cleanupPublicIPRulesShell()
	if !strings.Contains(script, "--line-numbers") {
		t.Fatalf("cleanup should delete iptables rules by line number: %s", script)
	}
	if strings.Contains(script, "iptables -S") {
		t.Fatalf("cleanup should not parse iptables -S output because quoted comments break deletion: %s", script)
	}
}

func TestCleanupPublicIPHostAddressesRemovesManagedPoolIP(t *testing.T) {
	script := cleanupPublicIPHostAddressesShell([]model.PublicIP{
		{IP: "203.0.113.10", CIDR: "203.0.113.8/29", UplinkIF: "ens3"},
	})
	if !strings.Contains(script, "ip addr del '203.0.113.10/29' dev 'ens3'") {
		t.Fatalf("cleanup should remove host address managed by public IP pool: %s", script)
	}
}
