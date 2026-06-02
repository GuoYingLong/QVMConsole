package service

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"kvm_console/utils"
)

const vmGuestAgentTargetName = "org.qemu.guest_agent.0"

var (
	vmGuestAgentTargetRegex  = regexp.MustCompile(`<target\b[^>]*name=['"]org\.qemu\.guest_agent\.0['"][^>]*/?>`)
	vmGuestAgentChannelRegex = regexp.MustCompile(`(?s)\n?\s*<channel\b[^>]*>.*?<target\b[^>]*name=['"]org\.qemu\.guest_agent\.0['"][^>]*/?>.*?</channel>`)
)

// VMGuestAgentConfig 虚拟机 QEMU Guest Agent 配置
type VMGuestAgentConfig struct {
	Enabled bool `json:"enabled"`
}

// NormalizeVMGuestAgentConfig 规范化 Guest Agent 配置
func NormalizeVMGuestAgentConfig(cfg *VMGuestAgentConfig) *VMGuestAgentConfig {
	if cfg == nil {
		return &VMGuestAgentConfig{}
	}
	return &VMGuestAgentConfig{
		Enabled: cfg.Enabled,
	}
}

// ParseVMGuestAgentConfigFromDomainXML 从 domain XML 中解析 Guest Agent 配置
func ParseVMGuestAgentConfigFromDomainXML(xmlContent string) *VMGuestAgentConfig {
	return &VMGuestAgentConfig{
		Enabled: vmGuestAgentTargetRegex.MatchString(xmlContent),
	}
}

// ApplyVMGuestAgentConfigToDomainXML 将 Guest Agent 配置写入 domain XML
func ApplyVMGuestAgentConfigToDomainXML(xmlContent string, cfg *VMGuestAgentConfig) (string, error) {
	normalized := NormalizeVMGuestAgentConfig(cfg)
	cleanedXML := vmGuestAgentChannelRegex.ReplaceAllString(xmlContent, "")

	if !normalized.Enabled {
		return cleanedXML, nil
	}

	channelXML := "" +
		"    <channel type='unix'>\n" +
		"      <source mode='bind'/>\n" +
		"      <target type='virtio' name='" + vmGuestAgentTargetName + "'/>\n" +
		"    </channel>\n"

	if strings.Contains(cleanedXML, "</devices>") {
		return strings.Replace(cleanedXML, "</devices>", channelXML+"  </devices>", 1), nil
	}

	return "", fmt.Errorf("写入 QEMU Guest Agent 配置失败：未找到 devices 节点")
}

// SetVMGuestAgentConfig 修改虚拟机 QEMU Guest Agent 配置
func SetVMGuestAgentConfig(name string, cfg *VMGuestAgentConfig) error {
	xmlResult := utils.ExecCommand("virsh", "dumpxml", name, "--inactive")
	if xmlResult.Error != nil {
		return fmt.Errorf("获取虚拟机 XML 失败: %s", xmlResult.Stderr)
	}

	newXML, err := ApplyVMGuestAgentConfigToDomainXML(xmlResult.Stdout, cfg)
	if err != nil {
		return err
	}

	xmlPath := fmt.Sprintf("/tmp/_guest-agent-%s.xml", name)
	if err := os.WriteFile(xmlPath, []byte(newXML), 0644); err != nil {
		return fmt.Errorf("写入 QEMU Guest Agent 配置文件失败: %w", err)
	}
	defer os.Remove(xmlPath)

	defineResult := utils.ExecCommand("virsh", "define", xmlPath)
	if defineResult.Error != nil {
		return fmt.Errorf("设置 QEMU Guest Agent 配置失败: %s", defineResult.Stderr)
	}

	return nil
}
