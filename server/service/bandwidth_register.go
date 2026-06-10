package service

import (
	bwpkg "kvm_console/service/bandwidth"
	ovspkg "kvm_console/service/ovs"
)

// bandwidth_register.go — 将 service 根包函数注入到 bandwidth 子包的 Hook 变量中，
// 供 bandwidth 子包通过 Hook 间接调用根包函数，避免循环 import。

func init() {
	// ── OVS / Network hooks — now delegate to ovs subpackage ──
	bwpkg.HookGetOVSStaticHostByVMName = func(vmName string) (bwpkg.OVSStaticHost, bool) {
		host, ok := ovspkg.GetOVSStaticHostByVMName(vmName)
		return bwpkg.OVSStaticHost{VMName: host.VMName, MAC: host.MAC, IP: host.IP}, ok
	}
	bwpkg.HookListAllVPCStaticHosts = func() ([]bwpkg.OVSStaticHost, error) {
		hosts, err := ovspkg.ListAllVPCStaticHosts()
		if err != nil {
			return nil, err
		}
		result := make([]bwpkg.OVSStaticHost, len(hosts))
		for i, h := range hosts {
			result[i] = bwpkg.OVSStaticHost{VMName: h.VMName, MAC: h.MAC, IP: h.IP}
		}
		return result, nil
	}
	bwpkg.HookGetOVSStaticIPByMAC = ovspkg.GetOVSStaticIPByMAC
	bwpkg.HookGetOVSLeaseIPByMAC = ovspkg.GetOVSLeaseIPByMAC
	bwpkg.HookUseOVSNetwork = ovspkg.UseOVSNetwork
	bwpkg.HookOvsBridgeName = ovspkg.OvsBridgeName
	bwpkg.HookOvsSubnetCIDR = ovspkg.OvsSubnetCIDR
	bwpkg.HookVPCGatewayPortName = VPCGatewayPortName

	// ── VM / User hooks ──
	bwpkg.HookGetUserVMList = GetUserVMList
	bwpkg.HookIsLightweightCloudVM = IsLightweightCloudVM
	bwpkg.HookIsUserTrafficLimited = IsUserTrafficLimited
	bwpkg.HookInferVPCSwitchForVM = InferVPCSwitchForVM
	bwpkg.HookApplyVPCSwitchBandwidth = ApplyVPCSwitchBandwidth
	bwpkg.HookRefreshVMCacheByNameAsync = RefreshVMCacheByNameAsync
}