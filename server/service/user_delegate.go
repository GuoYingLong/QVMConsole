package service

// User function delegates - forward to service/user subpackage
// Maintains backward compatibility for callers using service.XXX()

import (
	"time"

	"kvm_console/model"
	userpkg "kvm_console/service/user"
)

// ── Core user delegates ──

func ListUsers() ([]VMUserInfo, error) {
	return userpkg.ListUsers()
}

func CreateSystemUser(username, password, role string, maxCPU, maxMemory, maxDisk, maxVM, maxStorage, maxRuntimeHours int, enablePortForward bool, maxPortForwards, maxSnapshots int, maxBandwidthUp, maxBandwidthDown, maxTrafficDown, maxTrafficUp float64) error {
	return userpkg.CreateSystemUser(username, password, role, maxCPU, maxMemory, maxDisk, maxVM, maxStorage, maxRuntimeHours, enablePortForward, maxPortForwards, maxSnapshots, maxBandwidthUp, maxBandwidthDown, maxTrafficDown, maxTrafficUp)
}

func ProvisionSystemUserResources(user *model.User, password string) error {
	return userpkg.ProvisionSystemUserResources(user, password)
}

func FindVMOwner(vmName string) string {
	return userpkg.FindVMOwner(vmName)
}

func UpdateUserStatus(username, targetStatus string) error {
	return userpkg.UpdateUserStatus(username, targetStatus)
}

func DisableUserAccount(username string, progressFn func(int, string)) (*UserStatusChangeResult, error) {
	return userpkg.DisableUserAccount(username, progressFn)
}

func DeleteSystemUser(username string, progressFn func(int, string)) error {
	return userpkg.DeleteSystemUser(username, progressFn)
}

// ── VM assignment delegates ──

func AssignVMsToUser(username string, vmNames []string) error {
	return userpkg.AssignVMsToUser(username, vmNames)
}

func AssignVMsToUserWithQuotas(username string, vmNames []string, lightweightQuotas []LightweightVMQuotaRequest) error {
	userQuotas := make([]userpkg.LightweightVMQuotaRequest, len(lightweightQuotas))
	for i, q := range lightweightQuotas {
		userQuotas[i] = userpkg.LightweightVMQuotaRequest{
			VMName:            q.VMName,
			TrafficDownGB:     q.TrafficDownGB,
			TrafficUpGB:       q.TrafficUpGB,
			BandwidthDownMbps: q.BandwidthDownMbps,
			BandwidthUpMbps:   q.BandwidthUpMbps,
			MaxPortForwards:   q.MaxPortForwards,
			MaxSnapshots:      q.MaxSnapshots,
			MaxRuntimeHours:   q.MaxRuntimeHours,
		}
	}
	return userpkg.AssignVMsToUserWithQuotas(username, vmNames, userQuotas)
}

// ── Quota delegates ──

func GetUserVMList(username string) []string {
	return userpkg.GetUserVMList(username)
}

func GetUserQuotaUsage(username string) (*QuotaUsage, error) {
	return userpkg.GetUserQuotaUsage(username)
}

func CheckQuota(username string, reqCPU, reqMemoryGB, reqDiskGB int) error {
	return userpkg.CheckQuota(username, reqCPU, reqMemoryGB, reqDiskGB)
}

func CheckQuotaForEdit(username string, deltaCPU, deltaMemoryGB, deltaDiskGB int) error {
	return userpkg.CheckQuotaForEdit(username, deltaCPU, deltaMemoryGB, deltaDiskGB)
}

func CheckQuotaForStart(username string, vmName string) error {
	return userpkg.CheckQuotaForStart(username, vmName)
}

func AddVMToUser(username, vmName string) error {
	return userpkg.AddVMToUser(username, vmName)
}

func RemoveVMFromUser(username, vmName string) error {
	return userpkg.RemoveVMFromUser(username, vmName)
}

func UserOwnsVM(username, vmName string) bool {
	return userpkg.UserOwnsVM(username, vmName)
}

func UpdateUserQuota(username string, maxCPU, maxMemory, maxDisk, maxVM, maxStorage, maxRuntimeHours int, enablePortForward bool, maxPortForwards, maxSnapshots int, maxBandwidthUp, maxBandwidthDown, maxTrafficDown, maxTrafficUp float64, maxPublicIPs int) error {
	return userpkg.UpdateUserQuota(username, maxCPU, maxMemory, maxDisk, maxVM, maxStorage, maxRuntimeHours, enablePortForward, maxPortForwards, maxSnapshots, maxBandwidthUp, maxBandwidthDown, maxTrafficDown, maxTrafficUp, maxPublicIPs)
}

func GetRunningVMsResourceUsage(username string) (runningCPU int, runningMemoryMB int, err error) {
	return userpkg.GetRunningVMsResourceUsage(username)
}

func GetVMCPUAndMemory(vmName string) (cpu int, memMB int) {
	return userpkg.GetVMCPUAndMemory(vmName)
}

func GetVMDiskDevCapacityGB(vmName, dev string) int {
	return userpkg.GetVMDiskDevCapacityGB(vmName, dev)
}

// ── Storage delegates ──

func GetUserISODir(username string) string {
	return userpkg.GetUserISODir(username)
}

func GetUserShareDir(username string) string {
	return userpkg.GetUserShareDir(username)
}

func GetUserDiskDir(username string) string {
	return userpkg.GetUserDiskDir(username)
}

func InitUserStorage(username string) error {
	return userpkg.InitUserStorage(username)
}

func IsStorageInitialized(username string) bool {
	return userpkg.IsStorageInitialized(username)
}

func GetUserStorageInfo(username string) (*UserStorageInfo, error) {
	return userpkg.GetUserStorageInfo(username)
}

func CheckStorageQuota(username string, additionalBytes int64) error {
	return userpkg.CheckStorageQuota(username, additionalBytes)
}

func IsStorageReadonly(username string) bool {
	return userpkg.IsStorageReadonly(username)
}

func ListUserFiles(username, category string) ([]UserFileInfo, error) {
	return userpkg.ListUserFiles(username, category)
}

func DeleteUserFile(username, category, filename string) error {
	return userpkg.DeleteUserFile(username, category, filename)
}

func GetUserFilePath(username, category, filename string) (string, error) {
	return userpkg.GetUserFilePath(username, category, filename)
}

func GetUserISOs(username string) []ISOFileInfo {
	userResult := userpkg.GetUserISOs(username)
	result := make([]ISOFileInfo, len(userResult))
	for i, iso := range userResult {
		result[i] = ISOFileInfo{
			Name:      iso.Name,
			Path:      iso.Path,
			Size:      iso.Size,
			SizeBytes: iso.SizeBytes,
			Pool:      iso.Pool,
			OSType:    iso.OSType,
			OSVariant: iso.OSVariant,
			MinDisk:   iso.MinDisk,
		}
	}
	return result
}

func MountStorageToVM(username, vmName, category string, readonly bool) error {
	return userpkg.MountStorageToVM(username, vmName, category, readonly)
}

func UnmountStorageFromVM(vmName, tag string) error {
	return userpkg.UnmountStorageFromVM(vmName, tag)
}

func FormatBytesPublic(bytes int64) string {
	return userpkg.FormatBytesPublic(bytes)
}

// ── SSH delegates ──

func SetUserSSH(username string, enabled bool) error {
	return userpkg.SetUserSSH(username, enabled)
}

func GetUserSSHStatus(username string) (bool, error) {
	return userpkg.GetUserSSHStatus(username)
}

func SyncSSHDenyConfig() {
	userpkg.SyncSSHDenyConfig()
}

// ── Runtime quota delegates ──

func InitializeUserRuntimeQuotaTracker() {
	userpkg.InitializeUserRuntimeQuotaTracker()
}

func BuildUserRuntimeQuotaSnapshot(user *model.User, observedAt time.Time) UserRuntimeQuotaSnapshot {
	return userpkg.BuildUserRuntimeQuotaSnapshot(user, observedAt)
}

func FormatRuntimeQuotaDuration(seconds int64) string {
	return userpkg.FormatRuntimeQuotaDuration(seconds)
}

func CheckRuntimeQuotaAvailable(username string) error {
	return userpkg.CheckRuntimeQuotaAvailable(username)
}

func CheckRuntimeQuotaAvailableForUser(user *model.User, observedAt time.Time) error {
	return userpkg.CheckRuntimeQuotaAvailableForUser(user, observedAt)
}

func SyncAllUserRuntimeQuotaStates(observedAt time.Time) {
	userpkg.SyncAllUserRuntimeQuotaStates(observedAt)
}

func SyncAllUserRuntimeQuotaStatesWithActiveVMs(activeVMs map[string]struct{}, observedAt time.Time) {
	userpkg.SyncAllUserRuntimeQuotaStatesWithActiveVMs(activeVMs, observedAt)
}

func SyncUserRuntimeQuotaState(username string, observedAt time.Time) {
	userpkg.SyncUserRuntimeQuotaState(username, observedAt)
}

func EnforceUserRuntimeQuotaShutdown(username string, progressFn func(int, string)) (*RuntimeQuotaShutdownResult, error) {
	return userpkg.EnforceUserRuntimeQuotaShutdown(username, progressFn)
}

// ── Unexported function delegates (used by register files) ──

func waitVMShutdownForDisable(vmName string, timeout time.Duration) bool {
	return userpkg.WaitVMShutdownForDisable(vmName, timeout)
}

func getRuntimeActiveVMSetFromHost() (map[string]struct{}, error) {
	return userpkg.GetRuntimeActiveVMSetFromHost()
}
