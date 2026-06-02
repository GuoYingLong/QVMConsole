package service

import (
	"testing"

	"kvm_console/model"
)

func TestPortForwardWhitelistMatchAdmin(t *testing.T) {
	set := &portForwardWhitelistSet{
		user: map[string]bool{},
		vm:   map[string]bool{},
	}

	if got := set.Match("admin", "demo", false); got != PortForwardWhitelistScopeAdmin {
		t.Fatalf("owner 为 admin 时应命中管理员白名单，实际为 %q", got)
	}

	if got := set.Match("test", "demo", true); got != PortForwardWhitelistScopeAdmin {
		t.Fatalf("管理员创建的转发应命中管理员白名单，实际为 %q", got)
	}
}

func TestPortForwardWhitelistMatchUserAndVM(t *testing.T) {
	set := &portForwardWhitelistSet{
		user: map[string]bool{"test": true},
		vm:   map[string]bool{"demo": true},
	}

	if got := set.Match("test", "other", false); got != model.PortForwardWhitelistScopeUser {
		t.Fatalf("用户白名单应优先命中 user，实际为 %q", got)
	}

	if got := set.Match("other", "demo", false); got != model.PortForwardWhitelistScopeVM {
		t.Fatalf("虚拟机白名单应命中 vm，实际为 %q", got)
	}
}
