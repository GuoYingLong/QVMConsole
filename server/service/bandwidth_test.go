package service

import (
	"strings"
	"testing"
)

func TestOVSBandwidthMaxRateBps(t *testing.T) {
	if got := ovsBandwidthMaxRateBps(MbpsToKBps(10)); got != 10000000 {
		t.Fatalf("10Mbps should convert to 10000000bps, got %d", got)
	}
	if got := ovsBandwidthMaxRateBps(0); got != 0 {
		t.Fatalf("zero limit should stay zero, got %d", got)
	}
}

func TestOVSBandwidthQueueKey(t *testing.T) {
	if got := ovsBandwidthQueueKey(5551); got != "5551" {
		t.Fatalf("unexpected queue key: %s", got)
	}
}

func TestOVSBandwidthRateKbit(t *testing.T) {
	if got := ovsBandwidthRateKbit(MbpsToKBps(10)); got != 10000 {
		t.Fatalf("10Mbps should convert to 10000kbit, got %d", got)
	}
}

func TestOVSBandwidthMeterArg(t *testing.T) {
	if got := ovsBandwidthMeterArg(555101); got != "meter=555101" {
		t.Fatalf("unexpected meter arg: %s", got)
	}
}

func TestTCRateAndBurstForOneMbps(t *testing.T) {
	oneMbpsKBps := MbpsToKBps(1)
	if got := tcRateKbit(oneMbpsKBps); got != 1000 {
		t.Fatalf("1Mbps should convert to 1000kbit for tc, got %d", got)
	}
	if got := tcBurstBytes(oneMbpsKBps); got != 15360 {
		t.Fatalf("1Mbps should keep minimum tc burst for MTU safety, got %d", got)
	}
	if got := tcIFBTxQueueLen(); got != 100 {
		t.Fatalf("IFB queue length should stay small to avoid low-rate bufferbloat, got %d", got)
	}
}

func TestTCUploadIFBName(t *testing.T) {
	if got := tcUploadIFBName("vnet8"); got != "ifb-vnet8" {
		t.Fatalf("unexpected short IFB name: %s", got)
	}
	longName := tcUploadIFBName("vnet123456789012345")
	if longName == "" || len(longName) > 15 || !strings.HasPrefix(longName, "ifb") {
		t.Fatalf("long IFB name should be non-empty, <=15 chars and ifb-prefixed, got %q", longName)
	}
}

func TestBuildOVSBandwidthFlowsOnlyLimitsExternalTraffic(t *testing.T) {
	flows := buildOVSBandwidthFlows("0xabc", "7", "192.168.122.10", "192.168.122.0/24", 101, 102, 10000, 5000)
	joined := strings.Join(flows, "\n")

	required := []string{
		"priority=220,in_port=7,ip,nw_src=192.168.122.10,nw_dst=192.168.122.0/24,actions=NORMAL",
		"priority=120,in_port=7,ip,nw_src=192.168.122.10,actions=meter:102,output:LOCAL",
		"priority=220,in_port=LOCAL,ip,nw_src=192.168.122.0/24,nw_dst=192.168.122.10,actions=NORMAL",
		"priority=120,in_port=LOCAL,ip,nw_dst=192.168.122.10,actions=set_queue:101,output:7,pop_queue",
	}
	for _, item := range required {
		if !strings.Contains(joined, item) {
			t.Fatalf("expected flow %q in:\n%s", item, joined)
		}
	}
}

func TestBuildOVSBandwidthFlowsSkipsUnlimitedDirection(t *testing.T) {
	flows := buildOVSBandwidthFlows("0xabc", "7", "192.168.122.10", "192.168.122.0/24", 101, 102, 0, 5000)
	joined := strings.Join(flows, "\n")
	if strings.Contains(joined, "in_port=LOCAL") {
		t.Fatalf("down direction should not be installed when down rate is zero:\n%s", joined)
	}
	if !strings.Contains(joined, "meter:102") {
		t.Fatalf("up direction should still be metered:\n%s", joined)
	}
}

func TestBuildOVSVPCBandwidthFlowsUsesGatewayPort(t *testing.T) {
	flows := buildOVSVPCBandwidthFlows("0xabc", "11", "10", "10.200.1.52", "10.200.1.0/24", 101, 102, 10000, 5000)
	joined := strings.Join(flows, "\n")
	if !strings.Contains(joined, "output:10") {
		t.Fatalf("upload flow should output to VPC gateway port:\n%s", joined)
	}
	if !strings.Contains(joined, "in_port=10") || !strings.Contains(joined, "output:11") {
		t.Fatalf("download flow should enter from VPC gateway and output to VM port:\n%s", joined)
	}
	if strings.Contains(joined, "LOCAL") {
		t.Fatalf("VPC VM bandwidth flow must not use LOCAL port:\n%s", joined)
	}
}
