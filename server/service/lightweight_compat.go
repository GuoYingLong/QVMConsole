package service

import lwpkg "kvm_console/service/lightweight"

// Lightweight compatibility types and constants - delegate to service/lightweight subpackage

const (
	CloudTypeElastic     = lwpkg.CloudTypeElastic
	CloudTypeLightweight = lwpkg.CloudTypeLightweight
)

type LightweightVMQuotaRequest = lwpkg.LightweightVMQuotaRequest
type LightweightVMRegistrationRequest = lwpkg.LightweightVMRegistrationRequest
type LightweightVMConfirmRequest = lwpkg.LightweightVMConfirmRequest
type LightweightVMProvisionParams = lwpkg.LightweightVMProvisionParams
type LightweightVMRegistrationView = lwpkg.LightweightVMRegistrationView
type LightweightVMRuntimeQuotaSnapshot = lwpkg.LightweightVMRuntimeQuotaSnapshot
type LightweightRuntimeQuotaShutdownResult = lwpkg.LightweightRuntimeQuotaShutdownResult
