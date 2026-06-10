package service

// hooks.go - 子包通过 init() 注册实现函数，避免循环 import
// 根包通过这些变量调用子包函数（当子包函数尚未迁移时为 nil，
// 迁移后由子包 init() 赋值）。

// VPC Hooks - 由 service/network/vpc 子包注册
var (
	HookApplyVPCBindingRuntime func(vmName string) error
)

// Migration Hooks - 由 service/vm/migration 子包注册
var (
	HookEnsureVMNotMigrating        func(vmName, action string) error
	HookApplyVMUnderMigrationStatus func(vm *VmInfo)
	HookDetectMigrationModeFromState func(state string) string
	HookMigrationModeLive           string
)

// Memory Hooks - 由 service/vm/memory 子包注册
var (
	HookApplyPendingVMMemoryConfig func(vmName string) error
	HookGetVMMemoryDynamicInfo     func(name, xmlStr, state string) any
)

// maxInt 返回两个 int 中的较大值。
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// EnsureVMNotMigrating delegates to HookEnsureVMNotMigrating for handler layer compatibility
func EnsureVMNotMigrating(vmName, action string) error {
	if HookEnsureVMNotMigrating != nil {
		return HookEnsureVMNotMigrating(vmName, action)
	}
	return nil
}
