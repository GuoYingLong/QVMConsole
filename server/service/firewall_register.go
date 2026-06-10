package service

import (
	fwpkg "kvm_console/service/firewall"
	netpkg "kvm_console/service/network"
	ovspkg "kvm_console/service/ovs"
)

// firewall_register.go — 将 service 根包函数注入到 firewall 子包的 Hook 变量中，
// 供 firewall 子包通过 Hook 间接调用根包函数，避免循环 import。

func init() {
	fwpkg.HookOvsBridgeName = ovspkg.OvsBridgeName
	fwpkg.HookUseOVSNetwork = ovspkg.UseOVSNetwork
	fwpkg.HookVPCGatewayPortName = VPCGatewayPortName
	fwpkg.HookListLivePortForwardsFromIPTables = func() ([]fwpkg.PortForwardRule, error) {
		rules, err := listLivePortForwardsFromIPTables()
		if err != nil {
			return nil, err
		}
		result := make([]fwpkg.PortForwardRule, len(rules))
		for i, r := range rules {
			result[i] = fwpkg.PortForwardRule{
				Protocol: r.Protocol,
				HostPort: r.HostPort,
				DestIP:   r.DestIP,
				DestPort: r.DestPort,
			}
		}
		return result, nil
	}

	// ── Update network hooks to call firewall subpackage directly ──
	netpkg.HookGetFirewallPolicy = func() (*netpkg.FirewallPolicy, error) {
		policy, err := fwpkg.GetFirewallPolicy()
		if err != nil {
			return nil, err
		}
		return &netpkg.FirewallPolicy{
			PortForwardExemptions: policy.PortForwardExemptions,
		}, nil
	}
	netpkg.HookSetPortForwardFirewallExemption = func(key string, exempt bool) (*netpkg.FirewallPolicy, error) {
		policy, err := fwpkg.SetPortForwardFirewallExemption(key, exempt)
		if err != nil {
			return nil, err
		}
		if policy == nil {
			return nil, nil
		}
		return &netpkg.FirewallPolicy{
			PortForwardExemptions: policy.PortForwardExemptions,
		}, nil
	}
	netpkg.HookClearPortForwardFirewallExemption = fwpkg.ClearPortForwardFirewallExemption
	netpkg.HookEnsureHostFirewallPortForwardRule = fwpkg.EnsureHostFirewallPortForwardRule
	netpkg.HookDeleteHostFirewallPortForwardRule = fwpkg.DeleteHostFirewallPortForwardRule
}
