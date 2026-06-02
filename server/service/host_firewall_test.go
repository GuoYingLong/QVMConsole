package service

import "testing"

func TestParseUFWAddedRulesSingleAndRange(t *testing.T) {
	text := `Added user rules (see 'ufw status' for running firewall):
ufw allow 22/tcp comment 'kvm-console:protected:ssh'
ufw deny 8000:8010/udp comment 'test-range'
`
	rules := parseUFWAddedRules(text)
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d: %+v", len(rules), rules)
	}
	if rules[0].Action != "allow" || rules[0].Protocol != "tcp" || rules[0].PortStart != 22 || rules[0].PortEnd != 22 {
		t.Fatalf("unexpected single rule: %+v", rules[0])
	}
	if rules[1].Action != "deny" || rules[1].Protocol != "udp" || rules[1].PortStart != 8000 || rules[1].PortEnd != 8010 {
		t.Fatalf("unexpected range rule: %+v", rules[1])
	}
}

func TestParseUFWAddedRuleWithSourceCIDR(t *testing.T) {
	rule, ok := parseUFWAddedRuleLine("ufw allow from 203.0.113.0/24 to any port 5900:5999 proto tcp comment 'kvm-console:vnc-default'")
	if !ok {
		t.Fatal("expected source rule to parse")
	}
	if rule.SourceCIDR != "203.0.113.0/24" || rule.PortStart != 5900 || rule.PortEnd != 5999 || rule.Protocol != "tcp" {
		t.Fatalf("unexpected source rule: %+v", rule)
	}
	if !rule.ManagedByPanel {
		t.Fatalf("expected panel managed rule: %+v", rule)
	}
}

func TestNormalizeHostFirewallRuleRequestsExpandsBoth(t *testing.T) {
	rules := normalizeHostFirewallRuleRequests([]HostFirewallRuleRequest{{
		Action:    "allow",
		Protocol:  "both",
		PortStart: 10000,
		PortEnd:   10010,
	}})
	if len(rules) != 2 {
		t.Fatalf("expected tcp and udp rules, got %+v", rules)
	}
	if rules[0].Protocol == rules[1].Protocol {
		t.Fatalf("expected different protocols: %+v", rules)
	}
}

func TestMarkHostFirewallProtection(t *testing.T) {
	rule := HostFirewallRule{
		Action:    "allow",
		Protocol:  "tcp",
		PortStart: 22,
		PortEnd:   22,
	}
	markHostFirewallProtection(&rule, []int{22}, []int{8090})
	if !rule.Protected || rule.ProtectedReason != "SSH 端口" {
		t.Fatalf("expected ssh protection: %+v", rule)
	}

	wide := HostFirewallRule{
		Action:    "allow",
		Protocol:  "tcp",
		PortStart: 1,
		PortEnd:   65535,
	}
	markHostFirewallProtection(&wide, []int{22}, []int{8090})
	if wide.Protected {
		t.Fatalf("wide user rule should not be locked as protected: %+v", wide)
	}
}

func TestAddHostFirewallVNCDefaultShape(t *testing.T) {
	rule := HostFirewallRule{
		Action:         "allow",
		Protocol:       "tcp",
		PortStart:      5900,
		PortEnd:        5999,
		Comment:        hostFirewallVNCComment,
		ManagedByPanel: true,
	}
	args := buildUFWRuleArgs(rule, false)
	got := joinArgs(args)
	want := "allow 5900:5999/tcp comment kvm-console:vnc-default"
	if got != want {
		t.Fatalf("unexpected ufw args: got %q want %q", got, want)
	}
}

func TestMergeHostFirewallRulesKeepsPortForwardRecommendation(t *testing.T) {
	protected := buildProtectedHostFirewallRules([]int{22}, []int{8090})
	portForward := HostFirewallRule{Action: "allow", Protocol: "tcp", PortStart: 10000, PortEnd: 10000, Comment: hostFirewallPortForwardPrefix, ManagedByPanel: true}
	rules := mergeHostFirewallRules(protected, []HostFirewallRule{portForward})
	if len(rules) != 3 {
		t.Fatalf("expected protected rules plus port forward rule, got %+v", rules)
	}
	found := false
	for _, rule := range rules {
		if rule.PortStart == 10000 && rule.Protocol == "tcp" {
			found = true
		}
	}
	if !found {
		t.Fatalf("port forward recommendation missing: %+v", rules)
	}
}

func joinArgs(args []string) string {
	out := ""
	for i, arg := range args {
		if i > 0 {
			out += " "
		}
		out += arg
	}
	return out
}
