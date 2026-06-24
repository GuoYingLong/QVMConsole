package arch

// ==================== aarch64 Profile（占位框架） ====================
//
// 本文件为 ARM 架构适配的框架占位，实际 ARM 兼容待后续实现。
// 现阶段仅提供基本参数定义，供架构检测和前端展示使用。

type aarch64Profile struct{}

func (p *aarch64Profile) Arch() string                    { return ArchAarch64 }
func (p *aarch64Profile) DisplayName() string             { return "aarch64 (ARM64)" }
func (p *aarch64Profile) EmulatorPath() string            { return "/usr/bin/qemu-system-aarch64" }
func (p *aarch64Profile) DefaultMachineType() string      { return "virt" }
func (p *aarch64Profile) SupportedMachineTypes() []string { return []string{"virt"} }
func (p *aarch64Profile) DefaultBootType() string         { return "uefi" }
func (p *aarch64Profile) SupportedBootTypes() []string    { return []string{"uefi"} }
func (p *aarch64Profile) DefaultCPUMode() string          { return "host-passthrough" }
func (p *aarch64Profile) SupportedDiskBus() []string      { return []string{"virtio", "scsi"} }
func (p *aarch64Profile) SupportedNicModels() []string    { return []string{"virtio"} }
func (p *aarch64Profile) SupportsBIOS() bool              { return false }
func (p *aarch64Profile) SupportsSecureBoot() bool        { return false }
func (p *aarch64Profile) SupportsPAE() bool               { return false }
func (p *aarch64Profile) SupportsAPIC() bool              { return false }
func (p *aarch64Profile) DefaultWatchdogModel() string    { return "diag288" }

func (p *aarch64Profile) DefaultCPUModel(virtType string) string {
	if virtType == "qemu" {
		return "cortex-a72"
	}
	return ""
}

func (p *aarch64Profile) UEFIFirmwarePath(secureBoot bool) string {
	_ = secureBoot // ARM 暂不支持安全引导
	return "/usr/share/AAVMF/AAVMF_CODE.fd"
}

func (p *aarch64Profile) UEFIVarsTemplatePath(secureBoot bool) string {
	_ = secureBoot
	return "/usr/share/AAVMF/AAVMF_VARS.fd"
}

func init() {
	RegisterProfile(&aarch64Profile{})
}
