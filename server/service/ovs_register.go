package service

import ovspkg "kvm_console/service/ovs"

// ovs_register.go — 将 service 根包函数注入到 ovs 子包的 Hook 变量中，
// 供 ovs 子包通过 Hook 间接调用根包函数，避免循环 import。

func init() {
	ovspkg.HookEnsureOVSBridgeExists = EnsureOVSBridgeExists
	ovspkg.HookBuildOVSVirtInstallNetworkArgForBridge = BuildOVSVirtInstallNetworkArgForBridge
	ovspkg.HookBuildOVSInterfaceXMLForBridge = BuildOVSInterfaceXMLForBridge
}