package service

// Host compatibility types - delegate to service/host subpackage
import hostpkg "kvm_console/service/host"

// Type aliases for host subpackage types used by handler layer.
// These make service.XXX and host.XXX interchangeable, ensuring zero changes in handler code.

type HostNodeRequest = hostpkg.HostNodeRequest
type HostNodeView = hostpkg.HostNodeView
type HostKSMProfile = hostpkg.HostKSMProfile
type HostKSMStatus = hostpkg.HostKSMStatus
type HostKSMRuntimeConfig = hostpkg.HostKSMRuntimeConfig
type HostKSMMetrics = hostpkg.HostKSMMetrics
type HostZRAMProfile = hostpkg.HostZRAMProfile
type HostZRAMStatus = hostpkg.HostZRAMStatus
type HostZRAMRuntimeConfig = hostpkg.HostZRAMRuntimeConfig
type HostZRAMPersistentConfig = hostpkg.HostZRAMPersistentConfig
type HostDiskInfo = hostpkg.HostDiskInfo
type MaintenanceModeTaskParams = hostpkg.MaintenanceModeTaskParams
type MaintenanceModeTaskResult = hostpkg.MaintenanceModeTaskResult
