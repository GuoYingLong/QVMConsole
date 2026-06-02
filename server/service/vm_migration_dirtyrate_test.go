package service

import (
	"os"
	"os/exec"
	"testing"
)

func TestReadDomainDirtyRateAlreadyMeasuringIntegration(t *testing.T) {
	vmName := os.Getenv("KVM_CONSOLE_DIRTYRATE_TEST_VM")
	if vmName == "" {
		t.Skip("未设置 KVM_CONSOLE_DIRTYRATE_TEST_VM，跳过 dirty-rate 集成测试")
	}
	_ = exec.Command("virsh", "domdirtyrate-calc", vmName, "--seconds", "10").Run()
	rate, stats, err := readDomainDirtyRateMiB(vmName)
	if err != nil {
		t.Fatalf("测量中再次读取 dirty-rate 不应失败: %v; stats=%v", err, stats)
	}
	if _, ok := stats["dirtyrate.megabytes_per_second"]; !ok {
		t.Fatalf("应读取 dirtyrate.megabytes_per_second，got stats=%v rate=%f", stats, rate)
	}
}
