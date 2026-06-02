package service

import (
	"strings"
	"testing"
)

func TestIPTablesCheckLineForAddLine(t *testing.T) {
	addLine := "iptables -t nat -A PREROUTING -d 192.168.11.18/32 -p tcp --dport 10000 -j DNAT --to-destination 10.200.2.110:22"
	got, ok := iptablesCheckLineForAddLine(addLine)
	if !ok {
		t.Fatal("expected add line to be converted")
	}
	want := "iptables -t nat -C PREROUTING -d 192.168.11.18/32 -p tcp --dport 10000 -j DNAT --to-destination 10.200.2.110:22"
	if got != want {
		t.Fatalf("unexpected check line: got %q want %q", got, want)
	}
}

func TestIdempotentIPTablesAddLine(t *testing.T) {
	got := idempotentIPTablesAddLine("iptables -A FORWARD -d 10.200.2.110 -p tcp --dport 22 -j ACCEPT")
	if !strings.Contains(got, "iptables -C FORWARD") {
		t.Fatalf("expected check command in line: %s", got)
	}
	if !strings.Contains(got, "|| iptables -A FORWARD") {
		t.Fatalf("expected add command fallback in line: %s", got)
	}
}

func TestIdempotentIPTablesAddLineKeepsNatTableForDNAT(t *testing.T) {
	got := idempotentIPTablesAddLine("iptables -t nat -A PREROUTING -d 192.168.11.18/32 -p tcp --dport 10000 -j DNAT --to-destination 10.200.2.110:22")
	if !strings.Contains(got, "iptables -t nat -C PREROUTING") {
		t.Fatalf("expected nat check command in line: %s", got)
	}
	if !strings.Contains(got, "|| iptables -t nat -A PREROUTING") {
		t.Fatalf("expected nat add command fallback in line: %s", got)
	}
}

func TestNormalizePortForwardIPTablesLineFixesLegacyDNAT(t *testing.T) {
	got := normalizePortForwardIPTablesLine("iptables -C PREROUTING -d 192.168.11.18/32 -p tcp --dport 10000 -j DNAT --to-destination 10.200.2.110:22 2>/dev/null || iptables -A PREROUTING -d 192.168.11.18/32 -p tcp --dport 10000 -j DNAT --to-destination 10.200.2.110:22")
	if !strings.Contains(got, "iptables -t nat -C PREROUTING") || !strings.Contains(got, "|| iptables -t nat -A PREROUTING") {
		t.Fatalf("expected legacy DNAT line to include nat table: %s", got)
	}
}

func TestCheckRequestedPortForwardHostPortAvailableIgnoresCurrentRule(t *testing.T) {
	oldList := listPortForwardRulesForAvailability
	oldCanListen := canListenOnHostPort
	t.Cleanup(func() {
		listPortForwardRulesForAvailability = oldList
		canListenOnHostPort = oldCanListen
	})

	currentRule := &PortForwardRule{
		Protocol: "tcp",
		HostPort: "18080",
		DestIP:   "10.0.0.8",
		DestPort: "80",
	}
	listPortForwardRulesForAvailability = func() ([]PortForwardRule, error) {
		return []PortForwardRule{*currentRule}, nil
	}
	canListenOnHostPort = func(protocol string, port int) bool {
		return true
	}

	if err := CheckRequestedPortForwardHostPortAvailable("18080", "tcp", currentRule); err != nil {
		t.Fatalf("同一条规则复用原宿主机端口时不应报冲突: %v", err)
	}
}

func TestCheckRequestedPortForwardHostPortAvailableDetectsOtherRuleConflict(t *testing.T) {
	oldList := listPortForwardRulesForAvailability
	oldCanListen := canListenOnHostPort
	t.Cleanup(func() {
		listPortForwardRulesForAvailability = oldList
		canListenOnHostPort = oldCanListen
	})

	listPortForwardRulesForAvailability = func() ([]PortForwardRule, error) {
		return []PortForwardRule{
			{Protocol: "tcp", HostPort: "18081", DestIP: "10.0.0.9", DestPort: "22"},
		}, nil
	}
	canListenOnHostPort = func(protocol string, port int) bool {
		return true
	}

	err := CheckRequestedPortForwardHostPortAvailable("18081", "tcp", nil)
	if err == nil {
		t.Fatal("宿主机端口被其他转发规则占用时应返回错误")
	}
	if !strings.Contains(err.Error(), "已存在端口转发规则") {
		t.Fatalf("错误信息应说明已有端口转发占用，got=%v", err)
	}
}

func TestCheckRequestedPortForwardHostPortAvailableUsesCurrentProtocolWhenOmitted(t *testing.T) {
	oldList := listPortForwardRulesForAvailability
	oldCanListen := canListenOnHostPort
	t.Cleanup(func() {
		listPortForwardRulesForAvailability = oldList
		canListenOnHostPort = oldCanListen
	})

	currentRule := &PortForwardRule{
		Protocol: "udp",
		HostPort: "18082",
		DestIP:   "10.0.0.10",
		DestPort: "53",
	}
	listPortForwardRulesForAvailability = func() ([]PortForwardRule, error) {
		return []PortForwardRule{*currentRule}, nil
	}
	canListenOnHostPort = func(protocol string, port int) bool {
		return true
	}

	if err := CheckRequestedPortForwardHostPortAvailable("18082", "", currentRule); err != nil {
		t.Fatalf("编辑时未显式传协议，应继承当前规则协议继续校验: %v", err)
	}
}

func TestCheckRequestedPortForwardHostPortAvailableDetectsTCPListener(t *testing.T) {
	oldList := listPortForwardRulesForAvailability
	oldCanListen := canListenOnHostPort
	t.Cleanup(func() {
		listPortForwardRulesForAvailability = oldList
		canListenOnHostPort = oldCanListen
	})

	listPortForwardRulesForAvailability = func() ([]PortForwardRule, error) {
		return nil, nil
	}
	canListenOnHostPort = func(protocol string, port int) bool {
		return false
	}

	err := CheckRequestedPortForwardHostPortAvailable("18082", "tcp", nil)
	if err == nil {
		t.Fatal("宿主机监听已占用端口时应返回错误")
	}
	if !strings.Contains(err.Error(), "宿主机监听占用") {
		t.Fatalf("错误信息应说明宿主机监听占用，got=%v", err)
	}
}
