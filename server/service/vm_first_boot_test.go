package service

import (
	"strings"
	"testing"
)

func TestNormalizeVMFirstBootRebootMode(t *testing.T) {
	if got := NormalizeVMFirstBootRebootMode("cold"); got != VMFirstBootRebootCold {
		t.Fatalf("NormalizeVMFirstBootRebootMode(cold) = %q", got)
	}
	if got := NormalizeVMFirstBootRebootMode("unknown"); got != VMFirstBootRebootNormal {
		t.Fatalf("NormalizeVMFirstBootRebootMode(unknown) = %q", got)
	}
}

func TestApplyFirstBootRebootModeToDomainXML(t *testing.T) {
	xml := "<domain><on_poweroff>destroy</on_poweroff><on_reboot>restart</on_reboot><on_crash>destroy</on_crash></domain>"
	got := ApplyFirstBootRebootModeToDomainXML(xml, VMFirstBootRebootCold)
	if !strings.Contains(got, "<on_reboot>destroy</on_reboot>") {
		t.Fatalf("expected cold reboot mode to use destroy, got: %s", got)
	}

	got = ApplyFirstBootRebootModeToDomainXML(got, VMFirstBootRebootNormal)
	if !strings.Contains(got, "<on_reboot>restart</on_reboot>") {
		t.Fatalf("expected normal reboot mode to use restart, got: %s", got)
	}
}
