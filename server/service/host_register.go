package service

import (
	hostpkg "kvm_console/service/host"
)

// host_register.go — 将 service 根包函数注入到 host 子包的 Hook 变量中，
// 供 host 子包通过 Hook 间接调用根包函数，避免循环 import。

func init() {
	hostpkg.HookRemoteSSHExec = remoteSSHExec
	hostpkg.HookCallNodeAPI = CallNodeAPI
	hostpkg.HookShutdownVM = ShutdownVM
	hostpkg.HookDestroyVM = DestroyVM
	hostpkg.HookWaitVMShutdownForDisable = waitVMShutdownForDisable
	hostpkg.HookClearRuntimeCachesForMaintenance = clearRuntimeCachesForMaintenance
}
