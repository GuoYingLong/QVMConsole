package service

import (
	publicippkg "kvm_console/service/public_ip"
	ovspkg "kvm_console/service/ovs"
)

// public_ip_register.go — 将 service 根包函数注入到 public_ip 子包的 Hook 变量中，
// 供 public_ip 子包通过 Hook 间接调用根包函数，避免循环 import。

func init() {
	// ── VM / User hooks ──
	publicippkg.HookFindVMOwner = FindVMOwner

	// ── OVS / Network hooks ──
	publicippkg.HookGetVPCLeaseIPForVM = ovspkg.GetVPCLeaseIPForVM
	publicippkg.HookGetOVSStaticHostByVMName = func(vmName string) (publicippkg.OVSStaticHost, bool) {
		host, ok := ovspkg.GetOVSStaticHostByVMName(vmName)
		return publicippkg.OVSStaticHost{VMName: host.VMName, MAC: host.MAC, IP: host.IP}, ok
	}
	publicippkg.HookGetVMNetworkRuntimeStatus = func(vmName string) (*publicippkg.VMNetworkRuntimeStatus, error) {
		status, err := GetVMNetworkRuntimeStatus(vmName)
		if err != nil {
			return nil, err
		}
		if status == nil {
			return nil, nil
		}
		ifaces := make([]publicippkg.VMNetworkInterface, len(status.Interfaces))
		for i, iface := range status.Interfaces {
			ifaces[i] = publicippkg.VMNetworkInterface{IP: iface.IP}
		}
		return &publicippkg.VMNetworkRuntimeStatus{Interfaces: ifaces}, nil
	}
	publicippkg.HookIsVPCManagedIP = IsVPCManagedIP
	publicippkg.HookApplyVPCACLRules = ApplyVPCACLRules
	publicippkg.HookOvsUplink = ovspkg.OvsUplink
	publicippkg.HookOvsBridgeName = ovspkg.OvsBridgeName
	publicippkg.HookOvsGatewayIP = ovspkg.OvsGatewayIP
	publicippkg.HookGetOVSInterfaceOfPort = getOVSInterfaceOfPort
	publicippkg.HookParseVirshDomiflistOutput = func(text string) []publicippkg.OVSRuntimeInterface {
		rows := parseVirshDomiflistOutput(text)
		result := make([]publicippkg.OVSRuntimeInterface, len(rows))
		for i, r := range rows {
			result[i] = publicippkg.OVSRuntimeInterface{
				Name:   r.Name,
				Type:   r.Type,
				Source: r.Source,
				Model:  r.Model,
				MAC:    r.MAC,
			}
		}
		return result
	}
	publicippkg.HookWriteFileIfChanged = ovspkg.WriteFileIfChanged
}
