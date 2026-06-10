package service

import (
	lwpkg "kvm_console/service/lightweight"
)

// lightweight_register.go — 将 service 根包函数注入到 lightweight 子包的 Hook 变量中，
// 供 lightweight 子包通过 Hook 间接调用根包函数，避免循环 import。

func init() {
	// ── VM / User hooks ──
	lwpkg.HookGetUserVMList = GetUserVMList
	lwpkg.HookRefreshVMCacheByNameAsync = RefreshVMCacheByNameAsync
	lwpkg.HookCleanupVMVPCBinding = CleanupVMVPCBinding
	lwpkg.HookEnsureDefaultSecurityGroup = EnsureDefaultSecurityGroup
	lwpkg.HookEnsureDefaultVPCSwitch = EnsureDefaultVPCSwitch
	lwpkg.HookSwitchUsesDirectBridge = SwitchUsesDirectBridge
	lwpkg.HookBindVMToVPCAsAdmin = BindVMToVPCAsAdmin
	lwpkg.HookApplyVPCACLRules = ApplyVPCACLRules
	lwpkg.HookClearVMBandwidth = ClearVMBandwidth
	lwpkg.HookApplyVMNICBandwidth = ApplyVMNICBandwidth
	lwpkg.HookCurrentTrafficMonth = CurrentTrafficMonth
	lwpkg.HookTrafficQuotaBytes = TrafficQuotaBytes
	lwpkg.HookClampTrafficBytes = ClampTrafficBytes
	lwpkg.HookNormalizeVMCPUTopologyMode = NormalizeVMCPUTopologyMode
	lwpkg.HookNormalizeVMCPULimitPercent = NormalizeVMCPULimitPercent
	lwpkg.HookValidateVMCPULimitPercent = ValidateVMCPULimitPercent
	lwpkg.HookNormalizeVMFirstBootRebootMode = NormalizeVMFirstBootRebootMode
	lwpkg.HookGenerateRandomCloneHostname = GenerateRandomCloneHostname
	lwpkg.HookValidateCloneCredentialsForTemplate = ValidateCloneCredentialsForTemplate
	lwpkg.HookNormalizeCloneUsernameForTemplate = NormalizeCloneUsernameForTemplate
	lwpkg.HookGetTemplateMeta = func(templateName string) *lwpkg.TemplateMeta {
		meta := GetTemplateMeta(templateName)
		if meta == nil {
			return nil
		}
		var defaultConfig *lwpkg.TemplateDefaultConfig
		if meta.DefaultConfig != nil {
			defaultConfig = &lwpkg.TemplateDefaultConfig{
				DiskBus:             meta.DefaultConfig.DiskBus,
				VideoModel:          meta.DefaultConfig.VideoModel,
				CPUTopologyMode:     meta.DefaultConfig.CPUTopologyMode,
				FirstBootRebootMode: meta.DefaultConfig.FirstBootRebootMode,
			}
		}
		return &lwpkg.TemplateMeta{
			Type:          meta.Type,
			BootType:      meta.BootType,
			RootPassword:  meta.RootPassword,
			TemplateUser:  meta.TemplateUser,
			NVRAMPath:     meta.NVRAMPath,
			DefaultConfig: defaultConfig,
		}
	}
	lwpkg.HookResolveVMAPICEnabled = ResolveVMAPICEnabled
	lwpkg.HookRemoveVMFromUser = RemoveVMFromUser
	lwpkg.HookAddVMToUser = AddVMToUser
	lwpkg.HookSaveVMCredential = SaveVMCredential
	lwpkg.HookStartVM = StartVM
	lwpkg.HookFixOnReboot = FixOnReboot
	lwpkg.HookIsSMTPConfigured = IsSMTPConfigured
	lwpkg.HookSendEmail = SendEmail
	lwpkg.HookFormatRuntimeQuotaDuration = FormatRuntimeQuotaDuration
	lwpkg.HookShutdownVM = ShutdownVM
	lwpkg.HookDestroyVM = DestroyVM
	lwpkg.HookFirstNonEmpty = FirstNonEmpty

	// Unexported function hooks
	lwpkg.HookGetVMDiskPath = func(vmName string) string {
		info := getVMDiskInfo(vmName)
		return info.path
	}
	lwpkg.HookWaitVMShutdownForDisable = waitVMShutdownForDisable
	lwpkg.HookGetRuntimeActiveVMSetFromHost = getRuntimeActiveVMSetFromHost
}
