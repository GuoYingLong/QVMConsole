package service

// OVS compatibility types - type aliases to service/ovs subpackage
// Maintains backward compatibility for callers using service.XXX types
import ovspkg "kvm_console/service/ovs"

// ── Type aliases ──

type OVSStaticHost = ovspkg.OVSStaticHost
type OVSDHCPLease = ovspkg.OVSDHCPLease
type ovsRuntimeInterface = ovspkg.OvsRuntimeInterface

// ── Diagnostic type aliases ──

type OVSStatus = ovspkg.OVSStatus
type OVSServiceStatus = ovspkg.OVSServiceStatus
type OVSRuleStatus = ovspkg.OVSRuleStatus
type OVSCommandFailure = ovspkg.OVSCommandFailure
type OVSPortList = ovspkg.OVSPortList
type OVSPortStatus = ovspkg.OVSPortStatus
type OVSLeaseStatus = ovspkg.OVSLeaseStatus
type OVSStaticHostInfo = ovspkg.OVSStaticHostInfo
type OVSDHCPLeaseInfo = ovspkg.OVSDHCPLeaseInfo
type OVSLeaseConflict = ovspkg.OVSLeaseConflict
type OVSCheckResult = ovspkg.OVSCheckResult
type VMNetworkRuntimeStatus = ovspkg.VMNetworkRuntimeStatus
type VMNetworkInterface = ovspkg.VMNetworkInterface
type OVSBandwidthReadStatus = ovspkg.OVSBandwidthReadStatus