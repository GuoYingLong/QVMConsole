package service

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/digitalocean/go-libvirt"

	"kvm_console/logger"
	"kvm_console/service/ip_resolver"
	"kvm_console/service/libvirt_rpc"
	"kvm_console/service/vm/memory"
	"kvm_console/service/vm_xml"
	"kvm_console/utils"
)

// ==================== 虚拟机详情查询 ====================

// GetVM 获取单个虚拟机详情
func GetVM(name string) (*VmDetail, error) {
	// 检查虚拟机是否存在并获取基本信息
	vcpu, maxMemKB, usedMemKB, autostart, err := libvirt_rpc.GetDomainInfoRPC(name)
	if err != nil {
		return nil, fmt.Errorf("虚拟机不存在: %s", name)
	}

	vm := &VmDetail{}
	vm.Name = name
	if remark, err := GetVMRemark(name); err == nil {
		vm.Remark = remark
	}

	// 状态
	state, err := libvirt_rpc.GetDomainStateRPC(name)
	if err != nil {
		return nil, fmt.Errorf("获取虚拟机状态失败: %w", err)
	}
	vm.Status = state
	UpdateVMRuntimeState(name, vm.Status, time.Now())

	// 基本信息（从 RPC 获取的结构化数据）
	vm.VCPU = vcpu
	vm.MaxMemory = int(maxMemKB) / 1024
	vm.Memory = int(usedMemKB) / 1024
	vm.Autostart = autostart

	// 创建时间
	xmlPath := fmt.Sprintf("/etc/libvirt/qemu/%s.xml", name)
	if ts := utils.GetFileCreateTime(xmlPath); ts > 0 {
		vm.CreatedAt = time.Unix(ts, 0).Format("2006-01-02 15:04:05")
	}

	// IP（无论是否运行都尝试获取，可从静态绑定中兜底获取）
	vm.IP = ip_resolver.GetVMIP(name, vm.Status == "running")
	vm.IPStatus = ip_resolver.GetVMIPStatus(name, vm.Status == "running")
	vm.PublicIPs = ListPublicIPAttachmentsForVM(name)

	// 磁盘
	diskInfo := getVMDiskInfo(name)
	vm.DiskPath = diskInfo.path
	vm.DiskSize = diskInfo.size
	vm.Template = diskInfo.template

	// 检查系统盘完整性（仅检查第一块非 cdrom 磁盘）
	if diskInfo.path != "" {
		if _, err := os.Stat(diskInfo.path); err != nil {
			unhealthy := false
			vm.DiskHealthy = &unhealthy
			logger.App.Warn("虚拟机磁盘文件缺失", "vm", name, "path", diskInfo.path)
		} else {
			healthy := true
			vm.DiskHealthy = &healthy
		}
	}

	// 网络
	netInfo := getVMNetworkInfo(name)
	vm.Network = netInfo.network
	vm.NicModel = netInfo.nicModel
	vm.MacAddress = netInfo.mac

	// VNC 信息
	if vm.Status == "running" || vm.Status == "paused" {
		vncResult := utils.ExecCommand("virsh", "vncdisplay", name)
		if vncResult.Error == nil {
			vm.VNCPort = strings.TrimSpace(vncResult.Stdout)
		}
	}

	// 获取 XML 判断系统类型、引导顺序和可引导设备
	xmlStr, err := libvirt_rpc.GetDomainXMLRPC(name, libvirt.DomainXMLInactive)
	if err != nil {
		return nil, fmt.Errorf("获取虚拟机 XML 失败: %w", err)
	}
	vm.UUID = vm_xml.ParseDomainUUIDFromXML(xmlStr)

	// 使用持久化配置的 vCPU 覆盖在线值，确保界面显示的 vCPU 与用户配置一致
	// (libvirt 不支持在线修改 vCPU 最大值，热添加超限时持久化已更新但在线未变)
	if configVCPU := ParseVCPUCountFromDomainXML(xmlStr); configVCPU > 0 {
		vm.VCPU = configVCPU
	}
	vm.OSType = detectVMOSType(vm.Template, xmlStr)
	vm.BootType = vm_xml.ParseVMBootTypeFromDomainXML(xmlStr)
	vm.Arch = vm_xml.ParseVMArchFromDomainXML(xmlStr)
	vm.MachineType = vm_xml.ParseVMMachineTypeFromDomainXML(xmlStr)
	if count, err := GetVMPCIERootPorts(name); err == nil {
		vm.PCIERootPorts = count
	}
	vm.VideoModel = vm_xml.ParseVMVideoModelFromDomainXML(xmlStr)
	vm.CPUTopologyMode = ParseVMCPUTopologyModeFromDomainXML(xmlStr)
	vm.CPULimitPercent = ParseVMCPULimitPercentFromDomainXML(xmlStr, vm.VCPU)
	vm.CPUAffinity = ParseCPUAffinityFromDomainXML(xmlStr)
	vm.APIC = ParseVMAPICFromDomainXML(xmlStr)
	vm.PAE = vm_xml.ParseVMPAEFromDomainXML(xmlStr)
	vm.RTCOffset = ParseRTCOffsetFromDomainXML(xmlStr)
	vm.RTCStartDate = ParseRTCStartDateFromDomainXML(xmlStr)
	vm.GuestAgent = vm_xml.ParseVMGuestAgentConfigFromDomainXML(xmlStr)
	vm.SMBIOS1 = vm_xml.ParseSMBIOS1ConfigFromDomainXML(xmlStr)
	memInfo := memory.GetVMMemoryDynamicInfo(name, xmlStr, vm.Status)
	applyMemoryDynamicInfoToVMInfo(&vm.VmInfo, memInfo)
	if memInfo != nil {
		vm.MemoryObservationUntil = memInfo.ObservationUntil
		vm.MemoryManualPauseUntil = memInfo.ManualPauseUntil
	}

	// 解析引导顺序（OS 级别 <boot dev='xxx'/>）
	bootDevRe := regexp.MustCompile(`<boot dev='([^']+)'/>`)
	bootMatches := bootDevRe.FindAllStringSubmatch(xmlStr, -1)
	for _, m := range bootMatches {
		vm.BootOrder = append(vm.BootOrder, m[1])
	}
	if len(vm.BootOrder) == 0 {
		vm.BootOrder = []string{"hd"}
	}

	// 解析所有可引导设备
	vm.BootDevices = parseBootDevices(xmlStr, vm.BootOrder)
	vm.Freeze = isVMFreezeEnabled(xmlStr)

	// 获取带宽详情
	vm.BandwidthIn, vm.BandwidthOut = GetVMBandwidthMbps(name)
	if bwDetail, err := GetVMBandwidth(name); err == nil {
		vm.Bandwidth = bwDetail
	}
	if quota, err := GetLightweightVMQuota(name); err == nil {
		vm.LightweightQuota = quota
	}

	// 检查是否处于救援模式
	vm.InRescue = IsInRescueMode(name)
	runtimeInfo := GetVMRuntimeInfo(name, vm.Status)
	vm.ContinuousRuntimeSeconds = runtimeInfo.ContinuousRuntimeSeconds
	vm.ContinuousRunningSince = runtimeInfo.ContinuousRunningSince

	// 从缓存获取实时资源数据（后台采集器每10秒更新，不阻塞SSE推送）
	if vm.Status == "running" {
		vm.Stats = GetCachedStats(name)
	}

	// 读取已保存的虚拟机登录凭据
	if credential, err := GetVMCredential(name); err == nil {
		vm.Credential = credential
	}
	vm.Locked = IsVMLocked(name)
	if HookApplyVMUnderMigrationStatus != nil {
			HookApplyVMUnderMigrationStatus(&vm.VmInfo)
		}

	return vm, nil
}

// ==================== 详情辅助函数 ====================

func detectVMOSType(templateName, xmlStr string) string {
	if templateName != "" {
		if meta := GetTemplateMeta(templateName); meta != nil {
			switch strings.ToLower(strings.TrimSpace(meta.Type)) {
			case "fnos":
				return "fnos"
			case "windows":
				return "windows"
			case "linux":
				return "linux"
			}
		}
	}

	if strings.Contains(xmlStr, "firmware='efi'") &&
		strings.Contains(xmlStr, "hyperv") {
		return "windows"
	}
	return "linux"
}

func isVMFreezeEnabled(content string) bool {
	content = strings.ToLower(content)
	return strings.Contains(content, `freeze="yes"`) ||
		strings.Contains(content, `freeze="true"`) ||
		strings.Contains(content, `freeze='yes'`) ||
		strings.Contains(content, `freeze='true'`)
}

// getVMDiskInfo 获取虚拟机磁盘信息
func getVMDiskInfo(name string) diskInfoResult {
	info := diskInfoResult{}

	// 通过 RPC 获取 XML 并解析磁盘信息
	xmlStr, err := libvirt_rpc.GetDomainXMLRPC(name, 0)
	if err != nil {
		return info
	}

	disks := libvirt_rpc.ParseDisksFromDomainXML(xmlStr)
	for _, disk := range disks {
		if disk.Source != "" && disk.Source != "-" {
			info.device = disk.Target
			info.path = disk.Source
			break
		}
	}

	if info.path == "" {
		return info
	}

	// 获取磁盘配置容量（默认展示虚拟机设置大小，而非实际占用）
	qemuInfoResult := utils.ExecShell(fmt.Sprintf("qemu-img info --output=json -U %s 2>/dev/null", utils.ShellSingleQuote(info.path)))
	if qemuInfoResult.Error == nil {
		info.size = ParseQemuInfoGB(qemuInfoResult.Stdout, "virtual-size")
		if info.size != "-" && info.size != "" {
			info.size += " GB"
		}
		backing := strings.TrimSpace(ParseQemuInfoStr(qemuInfoResult.Stdout, "backing-filename"))
		if backing != "" {
			parts := strings.Split(backing, "/")
			templateFile := parts[len(parts)-1]
			info.template = strings.TrimSuffix(templateFile, ".qcow2")
		}
	}

	// 获取 backing file（模板来源）
	if info.template == "" {
		backingResult := utils.ExecShell(fmt.Sprintf("qemu-img info -U %s 2>/dev/null | grep 'backing file:' | awk '{print $3}'", utils.ShellSingleQuote(info.path)))
		if backingResult.Error == nil {
			backing := strings.TrimSpace(backingResult.Stdout)
			if backing != "" {
				parts := strings.Split(backing, "/")
				templateFile := parts[len(parts)-1]
				info.template = strings.TrimSuffix(templateFile, ".qcow2")
			}
		}
	}

	return info
}

// getVMNetworkInfo 获取虚拟机网络信息
func getVMNetworkInfo(name string) netInfoResult {
	info := netInfoResult{network: "unknown"}

	// 通过 RPC 获取 XML 并解析网卡信息
	xmlStr, err := libvirt_rpc.GetDomainXMLRPC(name, 0)
	if err != nil {
		return info
	}

	ifaces := libvirt_rpc.ParseInterfacesFromDomainXML(xmlStr)
	for _, iface := range ifaces {
		switch iface.Type {
		case "network":
			info.network = "nat"
		case "bridge":
			info.network = "bridge"
		default:
			info.network = iface.Type
		}
		info.nicModel = iface.Model
		info.mac = iface.MAC
		break
	}

	return info
}

// parseBootDevices 从 XML 中解析所有可引导设备
func parseBootDevices(xmlStr string, bootOrder []string) []BootDevice {
	var devices []BootDevice

	// 构建 boot order set（用于标记哪些设备类型被启用）
	bootOrderSet := make(map[string]int) // dev_type -> order
	for i, dev := range bootOrder {
		bootOrderSet[dev] = i + 1
	}

	// 解析磁盘设备
	diskRe := regexp.MustCompile(`(?s)<disk type='[^']*' device='([^']*)'>(.*?)</disk>`)
	sourceFileRe := regexp.MustCompile(`<source file='([^']*)'`)
	targetRe := regexp.MustCompile(`<target dev='([^']*)' bus='([^']*)'`)

	diskMatches := diskRe.FindAllStringSubmatch(xmlStr, -1)
	for _, m := range diskMatches {
		deviceType := m[1] // disk 或 cdrom
		deviceContent := m[2]

		bd := BootDevice{}
		if deviceType == "cdrom" {
			bd.Type = "cdrom"
		} else {
			bd.Type = "disk"
		}

		// 获取文件路径
		if sm := sourceFileRe.FindStringSubmatch(deviceContent); len(sm) > 1 {
			bd.File = sm[1]
		}

		// 获取设备名和总线
		if tm := targetRe.FindStringSubmatch(deviceContent); len(tm) > 2 {
			bd.Device = tm[1]
			bd.Bus = tm[2]
		}

		// 根据 OS 级别 boot order 判断是否启用及顺序
		// disk -> hd, cdrom -> cdrom
		bootKey := "hd"
		if bd.Type == "cdrom" {
			bootKey = "cdrom"
		}
		if order, ok := bootOrderSet[bootKey]; ok {
			bd.Enabled = true
			bd.Order = order
		}

		devices = append(devices, bd)
	}

	// 解析网络接口
	ifRe := regexp.MustCompile(`(?s)<interface type='[^']*'>(.*?)</interface>`)
	macRe := regexp.MustCompile(`<mac address='([^']*)'`)

	ifMatches := ifRe.FindAllStringSubmatch(xmlStr, -1)
	for _, m := range ifMatches {
		ifContent := m[1]
		bd := BootDevice{
			Type: "network",
		}
		if mm := macRe.FindStringSubmatch(ifContent); len(mm) > 1 {
			bd.File = mm[1]
		}

		if order, ok := bootOrderSet["network"]; ok {
			bd.Enabled = true
			bd.Order = order
		}

		devices = append(devices, bd)
	}

	return devices
}
