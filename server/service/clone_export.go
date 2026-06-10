package service

// Clone export adapters - export unexported functions/types for clone Deps injection
import (
	"context"
	"time"

	clonepkg "kvm_console/service/clone"
	ovspkg "kvm_console/service/ovs"
)

// GetTemplateMetaForClone returns a clone-compatible TemplateMeta
func GetTemplateMetaForClone(templateName string) *clonepkg.TemplateMeta {
	meta := GetTemplateMeta(templateName)
	if meta == nil {
		return nil
	}
	result := &clonepkg.TemplateMeta{
		Type:         meta.Type,
		BootType:     meta.BootType,
		RootPassword: meta.RootPassword,
		TemplateUser: meta.TemplateUser,
		NVRAMPath:    meta.NVRAMPath,
	}
	if meta.DefaultConfig != nil {
		result.DefaultConfig = &clonepkg.TemplateDefaultConfig{
			DiskBus:             meta.DefaultConfig.DiskBus,
			VideoModel:          meta.DefaultConfig.VideoModel,
			CPUTopologyMode:     meta.DefaultConfig.CPUTopologyMode,
			FirstBootRebootMode: meta.DefaultConfig.FirstBootRebootMode,
		}
	}
	return result
}

// ListAllVPCStaticHostsForClone converts service OVSStaticHost to clone.OVSStaticHost
func ListAllVPCStaticHostsForClone() ([]clonepkg.OVSStaticHost, error) {
	hosts, err := ovspkg.ListAllVPCStaticHosts()
	if err != nil {
		return nil, err
	}
	result := make([]clonepkg.OVSStaticHost, len(hosts))
	for i, h := range hosts {
		result[i] = clonepkg.OVSStaticHost{MAC: h.MAC, IP: h.IP}
	}
	return result, nil
}

// WaitForVMShutOff exports the unexported waitForVMShutOff for clone Deps
func WaitForVMShutOff(ctx context.Context, name string, timeout time.Duration) (bool, error) {
	return waitForVMShutOff(ctx, name, timeout)
}

// GetVMDiskInfoForClone exports getVMDiskInfo result for clone Deps
func GetVMDiskInfoForClone(name string) clonepkg.VMDiskInfoResult {
	info := getVMDiskInfo(name)
	return clonepkg.VMDiskInfoResult{
		Path:   info.path,
		Device: info.device,
		Size:   info.size,
	}
}

// InjectPCIERootPortsExported exports injectPCIERootPorts for clone Deps
func InjectPCIERootPortsExported(xmlContent string, portCount int) string {
	return injectPCIERootPorts(xmlContent, portCount)
}

// EnsureTemplatePathExported exports ensureTemplatePath for clone Deps
func EnsureTemplatePathExported(templateName string) (string, error) {
	return ensureTemplatePath(templateName)
}

// PrepareFnOSSystemDiskExpansionExported exports prepareFnOSSystemDiskExpansion for clone Deps
func PrepareFnOSSystemDiskExpansionExported(ctx context.Context, cloneDisk string, progressFn func(int, string)) error {
	return prepareFnOSSystemDiskExpansion(ctx, cloneDisk, progressFn)
}

// PrepareWindowsSystemDiskExpansionExported exports prepareWindowsSystemDiskExpansion for clone Deps
func PrepareWindowsSystemDiskExpansionExported(ctx context.Context, cloneDisk string, progressFn func(int, string)) error {
	return prepareWindowsSystemDiskExpansion(ctx, cloneDisk, progressFn)
}