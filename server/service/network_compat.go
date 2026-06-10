package service

// Network compatibility types and delegates - delegate to service/network subpackage
import netpkg "kvm_console/service/network"

type PortForwardRule = netpkg.PortForwardRule
type PortForwardAddParams = netpkg.PortForwardAddParams
type PortForwardUpdateParams = netpkg.PortForwardUpdateParams
type PortForwardAutoAddParams = netpkg.PortForwardAutoAddParams

// listLivePortForwardsFromIPTables delegates to network.ListLivePortForwardsFromIPTables
func listLivePortForwardsFromIPTables() ([]PortForwardRule, error) {
	return netpkg.ListLivePortForwardsFromIPTables()
}

// ListPortForwards delegates to network.ListPortForwards
func ListPortForwards() ([]PortForwardRule, error) {
	return netpkg.ListPortForwards()
}

// findLivePortForwardByStableKey delegates to network.FindLivePortForwardByStableKey
func findLivePortForwardByStableKey(ruleKey string) (*PortForwardRule, error) {
	return netpkg.FindLivePortForwardByStableKey(ruleKey)
}

// deleteLivePortForwardByStableKey delegates to network.DeleteLivePortForwardByStableKey
func deleteLivePortForwardByStableKey(ruleKey string, preserveProbeState bool) error {
	return netpkg.DeleteLivePortForwardByStableKey(ruleKey, preserveProbeState)
}

// AddPortForward delegates to network.AddPortForward
func AddPortForward(params *PortForwardAddParams) error {
	return netpkg.AddPortForward(params)
}

// getHostIP delegates to network.GetHostIP
func getHostIP() string {
	return netpkg.GetHostIP()
}

// buildPortForwardAccessAddress delegates to network.BuildPortForwardAccessAddress
func buildPortForwardAccessAddress(hostIP, hostPort string) string {
	return netpkg.BuildPortForwardAccessAddress(hostIP, hostPort)
}

// GetUserPortForwardUsage delegates to network.GetUserPortForwardUsage
func GetUserPortForwardUsage(username string) int {
	return netpkg.GetUserPortForwardUsage(username)
}

// RemoveVPCPortForwardAcceptRules delegates to network.RemoveVPCPortForwardAcceptRules
func RemoveVPCPortForwardAcceptRules() {
	netpkg.RemoveVPCPortForwardAcceptRules()
}

// SavePortForwardRules delegates to network.SavePortForwardRules
func SavePortForwardRules() error {
	return netpkg.SavePortForwardRules()
}

// removePortForwardsForCIDR delegates to network.RemovePortForwardsForCIDR
func removePortForwardsForCIDR(cidr string) {
	netpkg.RemovePortForwardsForCIDR(cidr)
}

// cleanupOVSStaticHostsForVMs delegates to network.CleanupOVSStaticHostsForVMs
func cleanupOVSStaticHostsForVMs(vmNames []string) {
	netpkg.CleanupOVSStaticHostsForVMs(vmNames)
}
