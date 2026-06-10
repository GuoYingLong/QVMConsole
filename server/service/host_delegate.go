package service

// Host function delegates - forward to service/host subpackage
// Maintains backward compatibility for callers using service.XXX()
import (
	"context"

	"kvm_console/model"
	hostpkg "kvm_console/service/host"
)

// ── Exported delegates (used by handler and other packages) ──

func ListHostNodes() ([]hostpkg.HostNodeView, error) {
	return hostpkg.ListHostNodes()
}

func GetHostNode(id uint) (*model.HostNode, error) {
	return hostpkg.GetHostNode(id)
}

func CreateHostNode(req hostpkg.HostNodeRequest) (*hostpkg.HostNodeView, error) {
	return hostpkg.CreateHostNode(req)
}

func UpdateHostNode(id uint, req hostpkg.HostNodeRequest) (*hostpkg.HostNodeView, error) {
	return hostpkg.UpdateHostNode(id, req)
}

func DeleteHostNode(id uint) error {
	return hostpkg.DeleteHostNode(id)
}

func ProbeHostNode(id uint) (*hostpkg.HostNodeView, error) {
	return hostpkg.ProbeHostNode(id)
}

func BuildHostNodeView(node model.HostNode) hostpkg.HostNodeView {
	return hostpkg.BuildHostNodeView(node)
}

func GetHostKSMProfiles() []hostpkg.HostKSMProfile {
	return hostpkg.GetHostKSMProfiles()
}

func GetHostKSMStatus() *hostpkg.HostKSMStatus {
	return hostpkg.GetHostKSMStatus()
}

func SetHostKSMProfile(profileKey string) (*hostpkg.HostKSMStatus, error) {
	return hostpkg.SetHostKSMProfile(profileKey)
}

func GetHostZRAMProfiles() []hostpkg.HostZRAMProfile {
	return hostpkg.GetHostZRAMProfiles()
}

func ApplyHostZRAMPersistentProfile() error {
	return hostpkg.ApplyHostZRAMPersistentProfile()
}

func GetHostZRAMStatus() *hostpkg.HostZRAMStatus {
	return hostpkg.GetHostZRAMStatus()
}

func SetHostZRAMProfile(profileKey string) (*hostpkg.HostZRAMStatus, error) {
	return hostpkg.SetHostZRAMProfile(profileKey)
}

func GetHostDiskInfos() ([]hostpkg.HostDiskInfo, error) {
	return hostpkg.GetHostDiskInfos()
}

func IsMaintenanceModeEnabled() bool {
	return hostpkg.IsMaintenanceModeEnabled()
}

func EnsureMaintenanceModeDisabled(action string) error {
	return hostpkg.EnsureMaintenanceModeDisabled(action)
}

func EnterMaintenanceMode(ctx context.Context, params *hostpkg.MaintenanceModeTaskParams, progressFn func(int, string)) (*hostpkg.MaintenanceModeTaskResult, error) {
	return hostpkg.EnterMaintenanceMode(ctx, params, progressFn)
}

func ExitMaintenanceMode(ctx context.Context, params *hostpkg.MaintenanceModeTaskParams, progressFn func(int, string)) (*hostpkg.MaintenanceModeTaskResult, error) {
	return hostpkg.ExitMaintenanceMode(ctx, params, progressFn)
}

func ParseMaintenanceServiceUnits(raw string) []string {
	return hostpkg.ParseMaintenanceServiceUnits(raw)
}

func IsMaintenanceModeError(err error) bool {
	return hostpkg.IsMaintenanceModeError(err)
}

func IsLibvirtUnavailableError(err error) bool {
	return hostpkg.IsLibvirtUnavailableError(err)
}

func FirstNonEmpty(values ...string) string {
	return hostpkg.FirstNonEmpty(values...)
}

// ── Unexported delegates (used by other service files within service root) ──

func decryptHostNodeSSHPassword(node model.HostNode) (string, error) {
	return hostpkg.DecryptHostNodeSSHPassword(node)
}

func decryptHostNodeAPIKey(node model.HostNode) (string, error) {
	return hostpkg.DecryptHostNodeAPIKey(node)
}

func encryptNodeSecret(plainText string) (string, error) {
	return hostpkg.EncryptNodeSecret(plainText)
}

func decryptNodeSecret(cipherText string) (string, error) {
	return hostpkg.DecryptNodeSecret(cipherText)
}

func buildNodeSecretKey() []byte {
	return hostpkg.BuildNodeSecretKey()
}

func collectHostDiskIOBytes() (int64, int64, error) {
	return hostpkg.CollectHostDiskIOBytes()
}

func isLibvirtUnavailableText(text string) bool {
	return hostpkg.IsLibvirtUnavailableText(text)
}
