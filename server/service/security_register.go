package service

import (
	securitypkg "kvm_console/service/security"
)

// security_register.go - 将 service 根包的函数注入到 security 子包的反向依赖 Hook 变量中，
// 供子包通过 Hook 间接调用根包函数，避免循环 import。

func init() {
	securitypkg.InitDeps(&securitypkg.Deps{
		// ---- Cloud type ----
		NormalizeCloudType:     NormalizeCloudType,
		CloudTypeElastic:       CloudTypeElastic,
		IsLightweightCloudType: IsLightweightCloudType,

		// ---- User provisioning ----
		ProvisionSystemUserResources: ProvisionSystemUserResources,

		// ---- VPC defaults ----
		EnsureDefaultSecurityGroup: EnsureDefaultSecurityGroup,
		EnsureDefaultVPCSwitch:     EnsureDefaultVPCSwitch,

		// ---- Validation ----
		ValidateStrongPassword: ValidateStrongPassword,

		// ---- Lightweight cloud ----
		ListLightweightVMRegistrations: func(username string, includeActive bool) ([]securitypkg.LightweightVMRegistrationView, error) {
			views, err := ListLightweightVMRegistrations(username, includeActive)
			if err != nil {
				return nil, err
			}
			result := make([]securitypkg.LightweightVMRegistrationView, len(views))
			for i, v := range views {
				result[i] = securitypkg.LightweightVMRegistrationView(v)
			}
			return result, nil
		},
		FormatLightweightVMRegistrationList: func(regs []securitypkg.LightweightVMRegistrationView) string {
			views := make([]LightweightVMRegistrationView, len(regs))
			for i, r := range regs {
				views[i] = LightweightVMRegistrationView(r)
			}
			return FormatLightweightVMRegistrationList(views)
		},

		// ---- Network constants ----
		BridgeModeNAT: BridgeModeNAT,

		// ---- Maintenance mode ----
		IsMaintenanceModeEnabled: IsMaintenanceModeEnabled,
	})
}