package service

// Lightweight function delegates - forward to service/lightweight subpackage
// Maintains backward compatibility for callers using service.XXX()

import (
	"context"
	"time"

	"kvm_console/model"
	lwpkg "kvm_console/service/lightweight"
)

// ── Cloud type delegates ──

func NormalizeCloudType(value string) string {
	return lwpkg.NormalizeCloudType(value)
}

func IsLightweightCloudType(value string) bool {
	return lwpkg.IsLightweightCloudType(value)
}

func IsLightweightCloudUser(username string) bool {
	return lwpkg.IsLightweightCloudUser(username)
}

func IsLightweightCloudVM(vmName string) bool {
	return lwpkg.IsLightweightCloudVM(vmName)
}

func UpdateUserCloudProfile(username, cloudType string, dedicatedVPCSwitchID uint) error {
	return lwpkg.UpdateUserCloudProfile(username, cloudType, dedicatedVPCSwitchID)
}

// ── Quota delegates ──

func NormalizeLightweightVMQuotaRequest(req LightweightVMQuotaRequest) LightweightVMQuotaRequest {
	return lwpkg.NormalizeLightweightVMQuotaRequest(req)
}

func UpsertLightweightVMQuota(username string, req LightweightVMQuotaRequest) (*model.LightweightVMQuota, error) {
	return lwpkg.UpsertLightweightVMQuota(username, req)
}

func GetLightweightVMQuota(vmName string) (*model.LightweightVMQuota, error) {
	return lwpkg.GetLightweightVMQuota(vmName)
}

// defaultLightweightVMQuota delegates to lightweight.DefaultLightweightVMQuota
func defaultLightweightVMQuota(vmName string) LightweightVMQuotaRequest {
	return lwpkg.DefaultLightweightVMQuota(vmName)
}

// fillLightweightVMQuotaRuntime delegates to lightweight.FillLightweightVMQuotaRuntime
func fillLightweightVMQuotaRuntime(quota *model.LightweightVMQuota) *model.LightweightVMQuota {
	return lwpkg.FillLightweightVMQuotaRuntime(quota)
}

// ── Network delegates ──

func EnsureLightweightVMNetwork(username, vmName string) error {
	return lwpkg.EnsureLightweightVMNetwork(username, vmName)
}

// ── Traffic delegates ──

func AggregateLightweightVMMonthlyTraffic(vmName string) (downBytes, upBytes int64) {
	return lwpkg.AggregateLightweightVMMonthlyTraffic(vmName)
}

func IsLightweightVMTrafficLimited(vmName string) (downLimited, upLimited bool) {
	return lwpkg.IsLightweightVMTrafficLimited(vmName)
}

func ApplyLightweightVMBandwidth(vmName string) error {
	return lwpkg.ApplyLightweightVMBandwidth(vmName)
}

func CheckAndApplyLightweightVMTrafficLimit(quota model.LightweightVMQuota) {
	lwpkg.CheckAndApplyLightweightVMTrafficLimit(quota)
}

func CheckAllLightweightVMTrafficQuota() {
	lwpkg.CheckAllLightweightVMTrafficQuota()
}

func CheckLightweightVMTrafficAfterQuotaUpdate(vmName string) {
	lwpkg.CheckLightweightVMTrafficAfterQuotaUpdate(vmName)
}

func ResetAllLightweightVMTraffic() {
	lwpkg.ResetAllLightweightVMTraffic()
}

func GetLightweightVMPortForwardUsage(vmName string) int {
	return lwpkg.GetLightweightVMPortForwardUsage(vmName)
}

func CheckLightweightVMPortForwardQuota(username, vmName string, delta int) error {
	return lwpkg.CheckLightweightVMPortForwardQuota(username, vmName, delta)
}

func CheckLightweightVMSnapshotQuota(username, vmName string, delta int) error {
	return lwpkg.CheckLightweightVMSnapshotQuota(username, vmName, delta)
}

// ── Resource cleanup delegates ──

func CleanupLightweightVMResources(vmName string) {
	lwpkg.CleanupLightweightVMResources(vmName)
}

// ── Registration delegates ──

func CreateLightweightVMRegistrations(username string, reqs []LightweightVMRegistrationRequest, createdBy string) ([]LightweightVMRegistrationView, error) {
	return lwpkg.CreateLightweightVMRegistrations(username, reqs, createdBy)
}

func ListLightweightVMRegistrations(username string, includeActive bool) ([]LightweightVMRegistrationView, error) {
	return lwpkg.ListLightweightVMRegistrations(username, includeActive)
}

func DeleteLightweightVMRegistration(username string, id uint) error {
	return lwpkg.DeleteLightweightVMRegistration(username, id)
}

func RemoveLightweightVMRegistrationByVMName(username string, vmName string) error {
	return lwpkg.RemoveLightweightVMRegistrationByVMName(username, vmName)
}

func UpdateLightweightVMQuotaByVMName(username string, req LightweightVMQuotaRequest) (*model.LightweightVMQuota, *LightweightVMRegistrationView, error) {
	return lwpkg.UpdateLightweightVMQuotaByVMName(username, req)
}

func BuildLightweightVMRegistrationView(reg model.LightweightVMRegistration) LightweightVMRegistrationView {
	return lwpkg.BuildLightweightVMRegistrationView(reg)
}

func FormatLightweightVMRegistrationList(regs []LightweightVMRegistrationView) string {
	return lwpkg.FormatLightweightVMRegistrationList(regs)
}

func NormalizeVMNicModel(value string) string {
	return lwpkg.NormalizeVMNicModel(value)
}

// ── Provision delegates ──

func BuildLightweightVMProvisionParams(regID uint, username string, credential LightweightVMConfirmRequest) (*LightweightVMProvisionParams, error) {
	return lwpkg.BuildLightweightVMProvisionParams(regID, username, credential)
}

func MarkLightweightVMRegistrationTask(regID uint, taskID uint) {
	lwpkg.MarkLightweightVMRegistrationTask(regID, taskID)
}

func ParseLightweightVMProvisionParams(jsonStr string) (*LightweightVMProvisionParams, error) {
	return lwpkg.ParseLightweightVMProvisionParams(jsonStr)
}

func ProvisionLightweightVMRegistration(ctx context.Context, params *LightweightVMProvisionParams, progressFn func(int, string)) (*CloneResult, error) {
	return lwpkg.ProvisionLightweightVMRegistration(ctx, params, progressFn)
}

// ── Runtime quota delegates ──

func InitializeLightweightRuntimeQuotaTracker() {
	lwpkg.InitializeLightweightRuntimeQuotaTracker()
}

func BuildLightweightVMRuntimeQuotaSnapshot(quota *model.LightweightVMQuota, observedAt time.Time) LightweightVMRuntimeQuotaSnapshot {
	return lwpkg.BuildLightweightVMRuntimeQuotaSnapshot(quota, observedAt)
}

func CheckLightweightVMRuntimeQuotaAvailable(vmName string) error {
	return lwpkg.CheckLightweightVMRuntimeQuotaAvailable(vmName)
}

func CheckLightweightVMRuntimeQuotaAvailableForQuota(quota *model.LightweightVMQuota, observedAt time.Time) error {
	return lwpkg.CheckLightweightVMRuntimeQuotaAvailableForQuota(quota, observedAt)
}

func SyncAllLightweightVMRuntimeQuotaStates(observedAt time.Time) {
	lwpkg.SyncAllLightweightVMRuntimeQuotaStates(observedAt)
}

func SyncLightweightVMRuntimeQuotaState(vmName string, observedAt time.Time) {
	lwpkg.SyncLightweightVMRuntimeQuotaState(vmName, observedAt)
}

func EnforceLightweightVMRuntimeQuotaShutdown(vmName string, progressFn func(int, string)) (*LightweightRuntimeQuotaShutdownResult, error) {
	return lwpkg.EnforceLightweightVMRuntimeQuotaShutdown(vmName, progressFn)
}

// ── Unexported function delegates ──

func syncAllLightweightVMRuntimeQuotaStatesWithActiveVMs(activeVMs map[string]struct{}, observedAt time.Time) {
	lwpkg.SyncAllLightweightVMRuntimeQuotaStatesWithActiveVMs(activeVMs, observedAt)
}
