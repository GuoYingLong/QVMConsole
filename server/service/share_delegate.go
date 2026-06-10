package service

// Share function delegates - forward to service/share subpackage
// Maintains backward compatibility for callers using service.XXX()

import sharepkg "kvm_console/service/share"

// ── Exported delegates ──

func ListShares(vmName string) ([]ShareInfo, error) {
	return sharepkg.ListShares(vmName)
}

func ListSharesInactive(vmName string) ([]ShareInfo, error) {
	return sharepkg.ListSharesInactive(vmName)
}

func AddShare(vmName, hostPath, tag, securityModel string, readonly bool) error {
	return sharepkg.AddShare(vmName, hostPath, tag, securityModel, readonly)
}

func RemoveShare(vmName, tag string) error {
	return sharepkg.RemoveShare(vmName, tag)
}
