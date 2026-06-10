package service

import (
	"kvm_console/model"
	userpkg "kvm_console/service/user"
)

// user_pkg_register.go — 将 service 根包函数注入到 user 子包的 Hook 变量中，
// 供 user 子包通过 Hook 间接调用根包函数，避免循环 import。

func init() {
	// ── Cloud / Lightweight VM hooks ──
	userpkg.HookNormalizeCloudType = NormalizeCloudType
	userpkg.HookIsLightweightCloudType = IsLightweightCloudType
	userpkg.HookIsLightweightCloudUser = IsLightweightCloudUser
	userpkg.HookIsLightweightCloudVM = IsLightweightCloudVM
	userpkg.HookNormalizeLightweightVMQuotaRequest = func(req userpkg.LightweightVMQuotaRequest) userpkg.LightweightVMQuotaRequest {
		serviceReq := LightweightVMQuotaRequest{
			VMName:            req.VMName,
			TrafficDownGB:     req.TrafficDownGB,
			TrafficUpGB:       req.TrafficUpGB,
			BandwidthDownMbps: req.BandwidthDownMbps,
			BandwidthUpMbps:   req.BandwidthUpMbps,
			MaxPortForwards:   req.MaxPortForwards,
			MaxSnapshots:      req.MaxSnapshots,
			MaxRuntimeHours:   req.MaxRuntimeHours,
		}
		result := NormalizeLightweightVMQuotaRequest(serviceReq)
		return userpkg.LightweightVMQuotaRequest{
			VMName:            result.VMName,
			TrafficDownGB:     result.TrafficDownGB,
			TrafficUpGB:       result.TrafficUpGB,
			BandwidthDownMbps: result.BandwidthDownMbps,
			BandwidthUpMbps:   result.BandwidthUpMbps,
			MaxPortForwards:   result.MaxPortForwards,
			MaxSnapshots:      result.MaxSnapshots,
			MaxRuntimeHours:   result.MaxRuntimeHours,
		}
	}
	userpkg.HookDefaultLightweightVMQuota = func(vmName string) userpkg.LightweightVMQuotaRequest {
		result := defaultLightweightVMQuota(vmName)
		return userpkg.LightweightVMQuotaRequest{
			VMName:            result.VMName,
			TrafficDownGB:     result.TrafficDownGB,
			TrafficUpGB:       result.TrafficUpGB,
			BandwidthDownMbps: result.BandwidthDownMbps,
			BandwidthUpMbps:   result.BandwidthUpMbps,
			MaxPortForwards:   result.MaxPortForwards,
			MaxSnapshots:      result.MaxSnapshots,
			MaxRuntimeHours:   result.MaxRuntimeHours,
		}
	}
	userpkg.HookFillLightweightVMQuotaRuntime = fillLightweightVMQuotaRuntime
	userpkg.HookListLightweightVMRegistrations = func(username string, includeActive bool) ([]userpkg.LightweightVMRegistrationView, error) {
		regs, err := ListLightweightVMRegistrations(username, includeActive)
		if err != nil {
			return nil, err
		}
		result := make([]userpkg.LightweightVMRegistrationView, len(regs))
		for i, r := range regs {
			result[i] = userpkg.LightweightVMRegistrationView{
				ID:                   r.ID,
				Username:             r.Username,
				VMName:               r.VMName,
				Template:             r.Template,
				TemplateType:         r.TemplateType,
				VCPU:                 r.VCPU,
				RAM:                  r.RAM,
				DiskSize:             r.DiskSize,
				DiskBus:              r.DiskBus,
				Hostname:             r.Hostname,
				Autostart:            r.Autostart,
				Freeze:               r.Freeze,
				APIC:                 r.APIC,
				PAE:                  r.PAE,
				RTCOffset:            r.RTCOffset,
				RTCStartDate:         r.RTCStartDate,
				VideoModel:           r.VideoModel,
				CPUTopologyMode:      r.CPUTopologyMode,
				CPULimitPercent:      r.CPULimitPercent,
				CPUAffinity:          r.CPUAffinity,
				FirstBootRebootMode:  r.FirstBootRebootMode,
				NicModel:             r.NicModel,
				StoragePoolID:        r.StoragePoolID,
				PreserveFnOSDeviceID: r.PreserveFnOSDeviceID,
				FnOSDeviceID:         r.FnOSDeviceID,
				SwitchID:             r.SwitchID,
				SwitchName:           r.SwitchName,
			}
		}
		return result, nil
	}
	userpkg.HookUpsertLightweightVMQuota = func(username string, req userpkg.LightweightVMQuotaRequest) (*model.LightweightVMQuota, error) {
		serviceReq := LightweightVMQuotaRequest{
			VMName:            req.VMName,
			TrafficDownGB:     req.TrafficDownGB,
			TrafficUpGB:       req.TrafficUpGB,
			BandwidthDownMbps: req.BandwidthDownMbps,
			BandwidthUpMbps:   req.BandwidthUpMbps,
			MaxPortForwards:   req.MaxPortForwards,
			MaxSnapshots:      req.MaxSnapshots,
			MaxRuntimeHours:   req.MaxRuntimeHours,
		}
		return UpsertLightweightVMQuota(username, serviceReq)
	}
	userpkg.HookEnsureLightweightVMNetwork = EnsureLightweightVMNetwork
	userpkg.HookCleanupLightweightVMResources = CleanupLightweightVMResources
	userpkg.HookCleanupVMVPCBinding = CleanupVMVPCBinding

	// ── Network / Security hooks ──
	userpkg.HookEnsureDefaultSecurityGroup = EnsureDefaultSecurityGroup
	userpkg.HookEnsureDefaultVPCSwitch = EnsureDefaultVPCSwitch
	userpkg.HookCleanupUserNetworkResources = CleanupUserNetworkResources

	// ── VM cache hooks ──
	userpkg.HookSyncVMCacheOwnersForAssignment = SyncVMCacheOwnersForAssignment
	userpkg.HookUpdateVMCacheOwner = UpdateVMCacheOwner
	userpkg.HookSyncVMCacheOwner = SyncVMCacheOwner

	// ── VM lifecycle hooks ──
	userpkg.HookShutdownVM = ShutdownVM
	userpkg.HookDestroyVM = DestroyVM

	// ── Storage hooks ──
	userpkg.HookGetStorageMountPoint = GetStorageMountPoint
	userpkg.HookEnsureStorageFilesystem = EnsureStorageFilesystem
	userpkg.HookSetupUserProject = SetupUserProject
	userpkg.HookGetProjectID = getProjectID
	userpkg.HookSetUserStorageQuota = SetUserStorageQuota
	userpkg.HookRemoveUserStorageQuota = RemoveUserStorageQuota
	userpkg.HookGetUserStorageUsage = func(username string) (*userpkg.StorageQuotaInfo, error) {
		info, err := GetUserStorageUsage(username)
		if err != nil {
			return nil, err
		}
		if info == nil {
			return nil, nil
		}
		return &userpkg.StorageQuotaInfo{
			UsedBytes:  info.UsedBytes,
			LimitBytes: info.LimitBytes,
		}, nil
	}
	userpkg.HookInferOSFromISO = inferOSFromISO
	userpkg.HookBuildISOInfo = func(filePath, poolName string) userpkg.ISOFileInfo {
		info := buildISOInfo(filePath, poolName)
		return userpkg.ISOFileInfo{
			Name:      info.Name,
			Path:      info.Path,
			Size:      info.Size,
			SizeBytes: info.SizeBytes,
			Pool:      info.Pool,
			OSType:    info.OSType,
			OSVariant: info.OSVariant,
			MinDisk:   info.MinDisk,
		}
	}
	userpkg.HookAddShare = AddShare
	userpkg.HookRemoveShare = RemoveShare

	// ── Network / Traffic / Bandwidth hooks ──
	userpkg.HookGetUserPublicIPUsage = GetUserPublicIPUsage
	userpkg.HookGetUserPortForwardUsage = GetUserPortForwardUsage
	userpkg.HookGetUserTrafficUsage = func(username string) *userpkg.TrafficUsageInfo {
		info := GetUserTrafficUsage(username)
		if info == nil {
			return nil
		}
		return &userpkg.TrafficUsageInfo{
			MaxTrafficDown:    info.MaxTrafficDown,
			MaxTrafficUp:      info.MaxTrafficUp,
			UsedTrafficDown:   info.UsedTrafficDown,
			UsedTrafficUp:     info.UsedTrafficUp,
			UsedTrafficDownGB: info.UsedTrafficDownGB,
			UsedTrafficUpGB:   info.UsedTrafficUpGB,
			IsLimitedDown:     info.IsLimitedDown,
			IsLimitedUp:       info.IsLimitedUp,
		}
	}
	userpkg.HookCheckTrafficAfterQuotaUpdate = CheckTrafficAfterQuotaUpdate
	userpkg.HookRebalanceUserBandwidth = RebalanceUserBandwidth

	// ── Maintenance / Email hooks ──
	userpkg.HookIsMaintenanceModeEnabled = IsMaintenanceModeEnabled
	userpkg.HookIsLibvirtUnavailableText = isLibvirtUnavailableText
	userpkg.HookIsLibvirtUnavailableError = IsLibvirtUnavailableError
	userpkg.HookSendEmail = SendEmail
}
