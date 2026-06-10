package service

// Firewall function delegates - forward to service/firewall subpackage
// Maintains backward compatibility for callers using service.XXX()
import (
	"context"

	fwpkg "kvm_console/service/firewall"
)

// ── Type aliases ──

type FirewallPolicy = fwpkg.FirewallPolicy
type FirewallRegion = fwpkg.FirewallRegion
type FirewallVMOverride = fwpkg.FirewallVMOverride
type FirewallStatus = fwpkg.FirewallStatus
type FirewallImportParams = fwpkg.FirewallImportParams
type FirewallGeoUpdateParams = fwpkg.FirewallGeoUpdateParams
type FirewallOperationParams = fwpkg.FirewallOperationParams
type HostFirewallRule = fwpkg.HostFirewallRule
type HostFirewallStatus = fwpkg.HostFirewallStatus
type HostFirewallRuleRequest = fwpkg.HostFirewallRuleRequest
type HostFirewallEnableRequest = fwpkg.HostFirewallEnableRequest
type HostFirewallConnection = fwpkg.HostFirewallConnection
type HostFirewallConnectionPreview = fwpkg.HostFirewallConnectionPreview
type HostFirewallCloseConnectionsRequest = fwpkg.HostFirewallCloseConnectionsRequest

// ── VM firewall policy delegates ──

func GetFirewallPolicy() (*FirewallPolicy, error) {
	return fwpkg.GetFirewallPolicy()
}

func SaveFirewallPolicy(policy *FirewallPolicy) error {
	return fwpkg.SaveFirewallPolicy(policy)
}

func ValidateFirewallPolicy(policy *FirewallPolicy) error {
	return fwpkg.ValidateFirewallPolicy(policy)
}

func GetFirewallStatus() (*FirewallStatus, error) {
	return fwpkg.GetFirewallStatus()
}

func PreviewFirewallRules(policy *FirewallPolicy) (string, error) {
	return fwpkg.PreviewFirewallRules(policy)
}

func ApplyFirewallPolicy(policy *FirewallPolicy, progress func(int, string)) error {
	return fwpkg.ApplyFirewallPolicy(policy, progress)
}

func DisableFirewall(progress func(int, string)) error {
	return fwpkg.DisableFirewall(progress)
}

func RollbackFirewall(progress func(int, string)) error {
	return fwpkg.RollbackFirewall(progress)
}

// ── VM firewall rules delegates ──

func BuildFirewallRules(policy *FirewallPolicy) (string, error) {
	return fwpkg.BuildFirewallRules(policy)
}

func ImportFirewallRegionCIDRs(params FirewallImportParams) (*FirewallPolicy, error) {
	return fwpkg.ImportFirewallRegionCIDRs(params)
}

func UpdateFirewallGeoIP(ctx context.Context, params FirewallGeoUpdateParams, progress func(int, string)) error {
	return fwpkg.UpdateFirewallGeoIP(ctx, params, progress)
}

// ── Firewall exemption delegates ──

func SetPortForwardFirewallExemption(key string, exempt bool) (*FirewallPolicy, error) {
	return fwpkg.SetPortForwardFirewallExemption(key, exempt)
}

func ClearPortForwardFirewallExemption(key string) error {
	return fwpkg.ClearPortForwardFirewallExemption(key)
}

// ── Host firewall delegates ──

func GetHostFirewallStatus() (*HostFirewallStatus, error) {
	return fwpkg.GetHostFirewallStatus()
}

func ListHostFirewallRules() ([]HostFirewallRule, error) {
	return fwpkg.ListHostFirewallRules()
}

func PreviewEnableHostFirewall(req HostFirewallEnableRequest) (*HostFirewallStatus, error) {
	return fwpkg.PreviewEnableHostFirewall(req)
}

func EnableHostFirewall(req HostFirewallEnableRequest, progress func(int, string)) error {
	return fwpkg.EnableHostFirewall(req, progress)
}

func DisableHostFirewall(progress func(int, string)) error {
	return fwpkg.DisableHostFirewall(progress)
}

// ── Host firewall rule management delegates ──

func AddHostFirewallRule(req HostFirewallRuleRequest) (*HostFirewallRule, error) {
	return fwpkg.AddHostFirewallRule(req)
}

func UpdateHostFirewallRule(id string, req HostFirewallRuleRequest) (*HostFirewallRule, error) {
	return fwpkg.UpdateHostFirewallRule(id, req)
}

func DeleteHostFirewallRule(id string) error {
	return fwpkg.DeleteHostFirewallRule(id)
}

func FindHostFirewallRule(id string) (HostFirewallRule, error) {
	return fwpkg.FindHostFirewallRule(id)
}

func BuildHostFirewallRecommendedRules() []HostFirewallRule {
	return fwpkg.BuildHostFirewallRecommendedRules()
}

func AddHostFirewallVNCDefaultRule() (*HostFirewallRule, error) {
	return fwpkg.AddHostFirewallVNCDefaultRule()
}

// ── Host firewall port forward delegates ──

func EnsureHostFirewallPortForwardRule(hostPort, protocol, comment string) error {
	return fwpkg.EnsureHostFirewallPortForwardRule(hostPort, protocol, comment)
}

func DeleteHostFirewallPortForwardRule(hostPort, protocol string) error {
	return fwpkg.DeleteHostFirewallPortForwardRule(hostPort, protocol)
}

func IsHostFirewallActive() bool {
	return fwpkg.IsHostFirewallActive()
}

// ── Host firewall connection delegates ──

func PreviewHostFirewallConnections(mode string) (*HostFirewallConnectionPreview, error) {
	return fwpkg.PreviewHostFirewallConnections(mode)
}

func CloseHostFirewallConnections(mode string) (int, error) {
	return fwpkg.CloseHostFirewallConnections(mode)
}

// ── Unexported function delegates ──

func getFirewallVMIP(vmName string) string {
	return fwpkg.GetFirewallVMIP(vmName)
}
