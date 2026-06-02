package service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"kvm_console/utils"
)

const (
	VMCPUTopologyAuto         = "auto"
	VMCPUTopologySingleSocket = "single_socket"
	VMCPUTopologyHostDefault  = "host_default"
)

var (
	vmCPUBlockRegexp       = regexp.MustCompile(`(?s)<cpu\b[^>]*(?:/>|>.*?</cpu>)`)
	vmCPUTopologyRegexp    = regexp.MustCompile(`(?s)<topology\b[^>]*/>`)
	vmVCPUValueRegexp      = regexp.MustCompile(`(?s)<vcpu\b[^>]*>\s*([0-9]+)\s*</vcpu>`)
	vmSelfClosingCPUExpr   = regexp.MustCompile(`^<cpu\b[^>]*/>$`)
	vmTopologySocketsRegex = regexp.MustCompile(`\bsockets=['"]([0-9]+)['"]`)
	vmTopologyCoresRegex   = regexp.MustCompile(`\bcores=['"]([0-9]+)['"]`)
	vmTopologyThreadsRegex = regexp.MustCompile(`\bthreads=['"]([0-9]+)['"]`)
)

// NormalizeVMCPUTopologyMode 规范化 CPU 拓扑模式。
func NormalizeVMCPUTopologyMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case VMCPUTopologySingleSocket, VMCPUTopologyHostDefault:
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		return VMCPUTopologyAuto
	}
}

// ApplyCPUTopologyModeToDomainXML 按模式写入 domain XML 的 CPU 拓扑。
func ApplyCPUTopologyModeToDomainXML(xmlStr, mode, osType string, vcpu int) string {
	switch NormalizeVMCPUTopologyMode(mode) {
	case VMCPUTopologySingleSocket:
		return ApplyWindowsCPUTopologyToDomainXML(xmlStr, vcpu)
	case VMCPUTopologyHostDefault:
		return RemoveCPUTopologyFromDomainXML(xmlStr)
	default:
		if strings.EqualFold(strings.TrimSpace(osType), "windows") {
			return ApplyWindowsCPUTopologyToDomainXML(xmlStr, vcpu)
		}
		return xmlStr
	}
}

// RemoveCPUTopologyFromDomainXML 移除显式 CPU 拓扑，让 libvirt/QEMU 使用默认拓扑。
func RemoveCPUTopologyFromDomainXML(xmlStr string) string {
	return vmCPUTopologyRegexp.ReplaceAllString(xmlStr, "")
}

// ParseVMCPUTopologyModeFromDomainXML 从 domain XML 中识别可回填的 CPU 拓扑模式。
func ParseVMCPUTopologyModeFromDomainXML(xmlStr string) string {
	topology := vmCPUTopologyRegexp.FindString(xmlStr)
	if strings.TrimSpace(topology) == "" {
		return VMCPUTopologyAuto
	}
	sockets := parseTopologyAttr(topology, vmTopologySocketsRegex)
	cores := parseTopologyAttr(topology, vmTopologyCoresRegex)
	threads := parseTopologyAttr(topology, vmTopologyThreadsRegex)
	vcpu := ParseVCPUCountFromDomainXML(xmlStr)
	if sockets == 1 && threads == 1 && (vcpu <= 0 || cores == vcpu) {
		return VMCPUTopologySingleSocket
	}
	return VMCPUTopologyHostDefault
}

// ApplyWindowsCPUTopologyToDomainXML 将 Windows 来宾的 vCPU 暴露为单插槽多核心。
func ApplyWindowsCPUTopologyToDomainXML(xmlStr string, vcpu int) string {
	if vcpu <= 0 {
		vcpu = ParseVCPUCountFromDomainXML(xmlStr)
	}
	if vcpu <= 0 {
		return xmlStr
	}

	topology := fmt.Sprintf("<topology sockets='1' dies='1' cores='%d' threads='1'/>", vcpu)
	if vmCPUBlockRegexp.MatchString(xmlStr) {
		return vmCPUBlockRegexp.ReplaceAllStringFunc(xmlStr, func(cpuBlock string) string {
			return applyTopologyToCPUBlock(cpuBlock, topology)
		})
	}

	cpuBlock := fmt.Sprintf("  <cpu mode='host-passthrough' check='none' migratable='on'>\n    %s\n  </cpu>", topology)
	if strings.Contains(xmlStr, "</features>") {
		return strings.Replace(xmlStr, "</features>", "</features>\n"+cpuBlock, 1)
	}
	if strings.Contains(xmlStr, "<devices>") {
		return strings.Replace(xmlStr, "<devices>", cpuBlock+"\n  <devices>", 1)
	}
	return xmlStr
}

// ParseVCPUCountFromDomainXML 从 domain XML 中读取 vCPU 数量。
func ParseVCPUCountFromDomainXML(xmlStr string) int {
	matches := vmVCPUValueRegexp.FindStringSubmatch(xmlStr)
	if len(matches) < 2 {
		return 0
	}
	value, err := strconv.Atoi(strings.TrimSpace(matches[1]))
	if err != nil {
		return 0
	}
	return value
}

func applyTopologyToCPUBlock(cpuBlock, topology string) string {
	trimmed := strings.TrimSpace(cpuBlock)
	if vmSelfClosingCPUExpr.MatchString(trimmed) {
		openTag := strings.TrimSuffix(trimmed, "/>")
		openTag = strings.TrimRight(openTag, " ")
		indent := leadingWhitespace(cpuBlock)
		return fmt.Sprintf("%s>\n%s  %s\n%s</cpu>", openTag, indent, topology, indent)
	}

	if vmCPUTopologyRegexp.MatchString(cpuBlock) {
		return vmCPUTopologyRegexp.ReplaceAllString(cpuBlock, topology)
	}
	if strings.Contains(cpuBlock, "</cpu>") {
		indent := leadingWhitespace(cpuBlock)
		return strings.Replace(cpuBlock, "</cpu>", "  "+topology+"\n"+indent+"</cpu>", 1)
	}
	return cpuBlock
}

func leadingWhitespace(value string) string {
	for i, r := range value {
		if r != ' ' && r != '\t' {
			return value[:i]
		}
	}
	return ""
}

func parseTopologyAttr(topology string, pattern *regexp.Regexp) int {
	matches := pattern.FindStringSubmatch(topology)
	if len(matches) < 2 {
		return 0
	}
	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0
	}
	return value
}

// SetVMCPUTopologyMode 设置虚拟机 CPU 拓扑模式。运行中的虚拟机需要先关机后再修改。
func SetVMCPUTopologyMode(name, mode string) error {
	stateResult := utils.ExecCommand("virsh", "domstate", name)
	if stateResult.Error != nil {
		return fmt.Errorf("获取虚拟机状态失败: %s", stateResult.Stderr)
	}
	state := strings.TrimSpace(stateResult.Stdout)
	if state == "running" || state == "paused" {
		return fmt.Errorf("请先关机后再修改 CPU 拓扑")
	}

	xmlResult := utils.ExecCommand("virsh", "dumpxml", name, "--inactive")
	if xmlResult.Error != nil {
		return fmt.Errorf("获取虚拟机 XML 失败: %s", xmlResult.Stderr)
	}

	xmlStr := xmlResult.Stdout
	osType := detectVMOSType("", xmlStr)
	updated := ApplyCPUTopologyModeToDomainXML(xmlStr, mode, osType, ParseVCPUCountFromDomainXML(xmlStr))

	xmlPath := fmt.Sprintf("/tmp/_cpu-topology-%s.xml", name)
	utils.ExecShell(fmt.Sprintf("cat > '%s' << 'XMLEOF'\n%s\nXMLEOF", xmlPath, updated))
	defineResult := utils.ExecCommand("virsh", "define", xmlPath)
	utils.ExecShell(fmt.Sprintf("rm -f '%s'", xmlPath))
	if defineResult.Error != nil {
		return fmt.Errorf("修改 CPU 拓扑失败: %s", defineResult.Stderr)
	}
	return nil
}
