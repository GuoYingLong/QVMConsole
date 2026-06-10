package service

import vncpkg "kvm_console/service/vnc"

// vnc_register.go — 将 service 根包函数注入到 vnc 子包的 Hook 变量中，
// 供 vnc 子包通过 Hook 间接调用根包函数，避免循环 import。

func init() {
	vncpkg.HookStartVM = StartVM
	vncpkg.HookDetectVMOSType = detectVMOSType
}
