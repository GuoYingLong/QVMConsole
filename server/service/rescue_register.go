package service

import rescuepkg "kvm_console/service/rescue"

// rescue_register.go — 将 service 根包函数注入到 rescue 子包的 Hook 变量中，
// 供 rescue 子包通过 Hook 间接调用根包函数，避免循环 import。

func init() {
	rescuepkg.HookEnsureVMNotMigrating = EnsureVMNotMigrating
	rescuepkg.HookDestroyVM = DestroyVM
	rescuepkg.HookSetVMBootOrder = SetVMBootOrder
	rescuepkg.HookStartVM = StartVM
	rescuepkg.HookSetVMNicModel = SetVMNicModel
}
